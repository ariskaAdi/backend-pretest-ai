package flows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const groqAPIURL = "https://api.groq.com/openai/v1/chat/completions"
const groqModel = "llama-3.3-70b-versatile"
const groqVisionModel = "llama-3.2-11b-vision-preview"

// --- Text model ---

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
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
	req, err := http.NewRequest(http.MethodPost, groqAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gagal buat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal call Groq API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gagal baca response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Groq API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Kedua model (text & vision) punya response format yang sama
	var groqResp groqResponse
	if err := json.Unmarshal(respBody, &groqResp); err != nil {
		return "", fmt.Errorf("gagal parse response Groq: %w", err)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("tidak ada response dari Groq")
	}

	return groqResp.Choices[0].Message.Content, nil
}
