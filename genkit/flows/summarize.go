package flows

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type SummarizeInput struct {
	PdfText string `json:"pdf_text"`
}

type SummarizeOutput struct {
	Summary string `json:"summary"`
}

func RegisterSummarizeFlow(g *genkit.Genkit) {
	genkit.DefineFlow(g, "summarizeModule",
		func(ctx context.Context, input *SummarizeInput) (*SummarizeOutput, error) {
			if input.PdfText == "" {
				return nil, fmt.Errorf("pdf_text tidak boleh kosong")
			}

			// Potong teks kalau terlalu panjang — Gemini ada batas token
			text := input.PdfText
			if len(text) > 12000 {
				text = text[:12000]
			}

			resp, err := genkit.GenerateText(ctx, g,
				ai.WithPrompt(buildSummarizePrompt(text)),
			)
			if err != nil {
				return nil, fmt.Errorf("gagal generate summary: %w", err)
			}

			return &SummarizeOutput{Summary: resp}, nil
		},
	)
}

func buildSummarizePrompt(text string) string {
	return fmt.Sprintf(`Kamu adalah asisten belajar untuk mahasiswa Universitas Terbuka.

Tugasmu adalah membuat ringkasan dari modul kuliah berikut agar mudah dipahami mahasiswa.

Panduan ringkasan:
- Tulis dalam Bahasa Indonesia yang jelas dan mudah dipahami
- Fokus pada konsep utama, definisi penting, dan poin kunci
- Gunakan paragraf yang terstruktur
- Panjang ringkasan: 3-5 paragraf
- Jangan sertakan hal yang tidak relevan

Isi modul:
%s

Tulis ringkasan sekarang:`, text)
}
