package flows

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Semua generate teks memakai Gemini. Model dicoba berurutan: kalau model
// pertama sedang overload (503) atau kena rate limit (429), pindah ke berikutnya.
// Bisa dioverride lewat env GEMINI_MODEL (model utama).
var geminiFallbackModels = []string{"gemini-3.5-flash", "gemini-3.5-flash-lite", "gemini-3.8-flash"}

// geminiHTTPClient punya batas waktu per request: saat Gemini overload, request
// kadang menggantung tanpa jawaban dan membuat seluruh flow macet.
var geminiHTTPClient = &http.Client{Timeout: 120 * time.Second}

func geminiModels() []string {
	if m := os.Getenv("GEMINI_MODEL"); m != "" {
		return append([]string{m}, geminiFallbackModels...)
	}
	return geminiFallbackModels
}

// generateText memanggil Gemini dengan retry + fallback antar model Gemini.
// Free tier dibatasi ~5 request/menit per model, jadi saat 429 kita tunggu
// sesuai retryDelay yang dikirim Gemini (bukan retry cepat lalu menyerah).
func generateText(prompt string) (string, error) {
	const maxAttempts = 6
	backoff := []time.Duration{3 * time.Second, 8 * time.Second, 15 * time.Second, 30 * time.Second, 45 * time.Second}
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		var wait time.Duration
		for _, model := range geminiModels() {
			result, err := generateWithGemini(model, prompt)
			if err == nil {
				return result, nil
			}
			lastErr = err
			if !isGeminiRetryable(err) {
				return "", err
			}
			d := geminiRetryDelay(err)
			log.Printf("[provider] Gemini %s %s, coba model berikutnya", model, geminiErrSummary(err, d))
			// Ambil jeda terkecil yang diminta: model mana pun yang lebih dulu siap.
			if d > 0 && (wait == 0 || d < wait) {
				wait = d
			}
		}
		if attempt == maxAttempts-1 {
			break
		}
		if wait == 0 {
			wait = backoff[attempt]
		}
		if wait > 65*time.Second {
			wait = 65 * time.Second
		}
		log.Printf("[provider] semua model sibuk, tunggu %s (percobaan %d/%d)", wait.Round(time.Second), attempt+1, maxAttempts)
		time.Sleep(wait)
	}
	return "", lastErr
}

var (
	reRetryDelay = regexp.MustCompile(`"retryDelay":\s*"([0-9.]+)s"`)
	reRetryIn    = regexp.MustCompile(`retry in ([0-9.]+)s`)
)

// geminiRetryDelay membaca jeda yang diminta Gemini pada error 429 (0 kalau tidak ada).
func geminiRetryDelay(err error) time.Duration {
	var he *geminiHTTPError
	if !errors.As(err, &he) || he.Status != http.StatusTooManyRequests {
		return 0
	}
	for _, re := range []*regexp.Regexp{reRetryDelay, reRetryIn} {
		if m := re.FindStringSubmatch(he.Body); m != nil {
			if secs, perr := strconv.ParseFloat(m[1], 64); perr == nil {
				return time.Duration(secs*float64(time.Second)) + time.Second
			}
		}
	}
	return 0
}

func geminiErrSummary(err error, delay time.Duration) string {
	var he *geminiHTTPError
	if !errors.As(err, &he) {
		return err.Error()
	}
	if he.Status == http.StatusTooManyRequests {
		return fmt.Sprintf("kena rate limit 429 (diminta tunggu %s)", delay.Round(time.Second))
	}
	return fmt.Sprintf("sibuk %d", he.Status)
}

type geminiHTTPError struct {
	Status int
	Body   string
}

func (e *geminiHTTPError) Error() string {
	return fmt.Sprintf("Gemini error %d: %s", e.Status, e.Body)
}

func isGeminiRetryable(err error) bool {
	var he *geminiHTTPError
	if errors.As(err, &he) {
		return he.Status == http.StatusServiceUnavailable || he.Status == http.StatusTooManyRequests ||
			he.Status == http.StatusInternalServerError || he.Status == http.StatusGatewayTimeout
	}
	// Timeout / koneksi putus saat Gemini overload juga layak dicoba ulang.
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// --- Native Gemini API (thinking disabled) ---

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int                   `json:"maxOutputTokens"`
	Temperature     float64               `json:"temperature"`
	ThinkingConfig  *geminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

type geminiThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func generateWithGemini(model, prompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY tidak di-set")
	}

	payload := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			MaxOutputTokens: 8192,
			Temperature:     0.3,
		},
	}
	// Model "-lite" menolak thinkingConfig (400 INVALID_ARGUMENT).
	if !strings.HasSuffix(model, "-lite") {
		payload.GenerationConfig.ThinkingConfig = &geminiThinkingConfig{ThinkingBudget: 0}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gagal encode request: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, apiKey,
	)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := geminiHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal call Gemini: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", &geminiHTTPError{Status: resp.StatusCode, Body: string(respBody)}
	}

	var result geminiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("gagal parse Gemini response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("tidak ada response dari Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
