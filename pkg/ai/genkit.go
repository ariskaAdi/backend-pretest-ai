package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	appconfig "backend-pretest-ai/config"
)

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
