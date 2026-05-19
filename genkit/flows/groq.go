package flows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const groqAPIURL = "https://api.groq.com/openai/v1/chat/completions"
const groqModel = "meta-llama/llama-4-scout-17b-16e-instruct"
const groqVisionModel = "llama-3.2-11b-vision-preview"

// --- Text model ---

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type groqChoice struct {
	Message groqMessage `json:"message"`
}

type groqResponse struct {
	Choices []groqChoice `json:"choices"`
}

func generateWithGroq(prompt string) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY tidak di-set")
	}

	payload := groqRequest{
		Model: groqModel,
		Messages: []groqMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   3000,
		Temperature: 0.3,
	}

	return doGroqRequest(apiKey, payload)
}

// --- Vision model ---

type groqVisionContent struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	ImageURL *groqVisionImage  `json:"image_url,omitempty"`
}

type groqVisionImage struct {
	URL string `json:"url"`
}

type groqVisionMessage struct {
	Role    string              `json:"role"`
	Content []groqVisionContent `json:"content"`
}

type groqVisionRequest struct {
	Model    string              `json:"model"`
	Messages []groqVisionMessage `json:"messages"`
}

// GenerateWithGroqVision mengirim satu gambar (base64) ke vision model dan kembalikan teks
func GenerateWithGroqVision(imageBase64 string, mimeType string) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY tidak di-set")
	}

	payload := groqVisionRequest{
		Model: groqVisionModel,
		Messages: []groqVisionMessage{
			{
				Role: "user",
				Content: []groqVisionContent{
					{
						Type: "image_url",
						ImageURL: &groqVisionImage{
							URL: fmt.Sprintf("data:%s;base64,%s", mimeType, imageBase64),
						},
					},
					{
						Type: "text",
						Text: "Ekstrak SEMUA teks dari gambar ini persis seperti yang tertulis. Pertahankan struktur paragraf, judul, dan daftar. Jangan tambahkan penjelasan — hanya teks dari gambar.",
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gagal encode vision request: %w", err)
	}

	return doGroqHTTP(apiKey, body)
}

// --- Shared HTTP helpers ---

func doGroqRequest(apiKey string, payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gagal encode request: %w", err)
	}
	return doGroqHTTP(apiKey, body)
}

func doGroqHTTP(apiKey string, body []byte) (string, error) {
	const maxRetries = 4
	delays := []time.Duration{3 * time.Second, 8 * time.Second, 15 * time.Second, 30 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err, retry := doGroqHTTPOnce(apiKey, body)
		if !retry {
			return result, err
		}
		if attempt < maxRetries {
			time.Sleep(delays[attempt])
		}
	}
	return "", fmt.Errorf("Groq API rate limit, sudah retry %d kali", maxRetries)
}

func doGroqHTTPOnce(apiKey string, body []byte) (string, error, bool) {
	req, err := http.NewRequest(http.MethodPost, groqAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gagal buat request: %w", err), false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal call Groq API: %w", err), false
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gagal baca response: %w", err), false
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", nil, true // retry
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Groq API error %d: %s", resp.StatusCode, string(respBody)), false
	}

	// Kedua model (text & vision) punya response format yang sama
	var groqResp groqResponse
	if err := json.Unmarshal(respBody, &groqResp); err != nil {
		return "", fmt.Errorf("gagal parse response Groq: %w", err), false
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("tidak ada response dari Groq"), false
	}

	return groqResp.Choices[0].Message.Content, nil, false
}
