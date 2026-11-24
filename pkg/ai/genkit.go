package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	appconfig "backend-pretest-ai/config"
)

// Model Gemini untuk OCR gambar (PDF berbasis scan). Override via env GEMINI_VISION_MODEL.
const defaultGeminiVisionModel = "gemini-3.5-flash"

type genkitClient struct {
	baseURL    string
	httpClient *http.Client
}

var Client *genkitClient

func InitGenkit() {
	cfg := appconfig.Cfg.Genkit

	Client = &genkitClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
	}
}

// call adalah helper generic untuk POST ke Genkit flow endpoint
func (g *genkitClient) call(flow string, input any, output any) error {
	body, err := json.Marshal(map[string]any{"data": input})
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	resp, err := g.httpClient.Post(
		fmt.Sprintf("%s/%s", g.baseURL, flow),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("failed to call genkit flow %q: %w", flow, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("genkit flow %q returned status %d: %s", flow, resp.StatusCode, string(raw))
	}

	// Genkit membungkus response dalam { "result": ... }
	var wrapper struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return fmt.Errorf("failed to decode genkit response: %w", err)
	}

	return json.Unmarshal(wrapper.Result, output)
}

// --- Summarize ---

type SummarizeInput struct {
	PdfText string `json:"pdf_text"`
}

type SummarizeOutput struct {
	Summary string `json:"summary"`
}

func (g *genkitClient) Summarize(pdfText string) (*SummarizeOutput, error) {
	var output SummarizeOutput
	if err := g.call("summarizeModule", SummarizeInput{PdfText: pdfText}, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

// --- Generate Quiz ---

type GenerateQuizInput struct {
	RawText      string `json:"raw_text"`
	NumQuestions int    `json:"num_questions"`
}

type DiagramData struct {
	Type    string `json:"type"`    // "svg"
	Content string `json:"content"` // SVG string
}

type Question struct {
	Question string       `json:"question"`
	Options  []string     `json:"options"`
	Answer   string       `json:"answer"`
	Diagram  *DiagramData `json:"diagram,omitempty"`
}

type GenerateQuizOutput struct {
	Questions []Question `json:"questions"`
}

func (g *genkitClient) GenerateQuiz(rawText string, numQuestions int) (*GenerateQuizOutput, error) {
	var output GenerateQuizOutput
	err := g.call("generateQuiz", GenerateQuizInput{
		RawText:      rawText,
		NumQuestions: numQuestions,
	}, &output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

// --- Explain Quiz ---

type WrongQuestion struct {
	ID            string   `json:"id"`
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correct_answer"`
	UserAnswer    string   `json:"user_answer"`
}

type ExplainQuizInput struct {
	WrongQuestions []WrongQuestion `json:"wrong_questions"`
	Summary        string          `json:"summary"`
}

type QuestionExplanation struct {
	ID          string `json:"id"`
	Explanation string `json:"explanation"`
}

type ExplainQuizOutput struct {
	Explanations []QuestionExplanation `json:"explanations"`
}

// VisionExtract mengirim gambar base64 ke Gemini dan kembalikan teks hasil OCR.
// Dipanggil via pdfpkg.VisionExtractor saat PDF berbasis gambar.
func (g *genkitClient) VisionExtract(imageBase64 string, mimeType string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY tidak di-set")
	}
	model := os.Getenv("GEMINI_VISION_MODEL")
	if model == "" {
		model = defaultGeminiVisionModel
	}

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{"inline_data": map[string]string{"mime_type": mimeType, "data": imageBase64}},
					{"text": "Ekstrak SEMUA teks dari gambar ini persis seperti yang tertulis. Pertahankan struktur paragraf, judul, dan daftar. Jangan tambahkan penjelasan — hanya teks dari gambar."},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature":    0,
			"thinkingConfig": map[string]int{"thinkingBudget": 0},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gagal encode vision request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	delays := []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := g.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("gagal call Gemini vision: %w", err)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if (resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests) && attempt < len(delays) {
			time.Sleep(delays[attempt])
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Gemini vision error %d: %s", resp.StatusCode, string(raw))
		}

		var result struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal(raw, &result); err != nil || len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("gagal parse vision response")
		}
		return result.Candidates[0].Content.Parts[0].Text, nil
	}
}

func (g *genkitClient) ExplainQuiz(wrongQuestions []WrongQuestion, summary string) (*ExplainQuizOutput, error) {
	var output ExplainQuizOutput
	err := g.call("explainQuiz", ExplainQuizInput{
		WrongQuestions: wrongQuestions,
		Summary:        summary,
	}, &output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}
