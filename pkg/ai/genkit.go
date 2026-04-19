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

const groqAPIURL = "https://api.groq.com/openai/v1/chat/completions"
const groqVisionModel = "llama-3.2-11b-vision-preview"

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
	Summary      string `json:"summary"`
	NumQuestions int    `json:"num_questions"`
}

type Question struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Answer   string   `json:"answer"`
}

type GenerateQuizOutput struct {
	Questions []Question `json:"questions"`
}

func (g *genkitClient) GenerateQuiz(summary string, numQuestions int) (*GenerateQuizOutput, error) {
	var output GenerateQuizOutput
	err := g.call("generateQuiz", GenerateQuizInput{
		Summary:      summary,
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

// VisionExtract mengirim gambar base64 ke Groq vision model dan kembalikan teks hasil OCR.
// Dipanggil via pdfpkg.VisionExtractor saat PDF berbasis gambar.
func (g *genkitClient) VisionExtract(imageBase64 string, mimeType string) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY tidak di-set")
	}

	payload := map[string]any{
		"model": groqVisionModel,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", mimeType, imageBase64),
						},
					},
					{
						"type": "text",
						"text": "Ekstrak SEMUA teks dari gambar ini persis seperti yang tertulis. Pertahankan struktur paragraf, judul, dan daftar. Jangan tambahkan penjelasan — hanya teks dari gambar.",
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gagal encode vision request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, groqAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal call Groq vision: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Groq vision error %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Choices) == 0 {
		return "", fmt.Errorf("gagal parse vision response")
	}

	return result.Choices[0].Message.Content, nil
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
