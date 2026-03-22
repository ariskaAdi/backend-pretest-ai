package flows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type GenerateQuizInput struct {
	Summary      string `json:"summary"`
	NumQuestions int    `json:"num_questions"`
}

type QuizQuestion struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Answer   string   `json:"answer"`
}

type GenerateQuizOutput struct {
	Questions []QuizQuestion `json:"questions"`
}

func RegisterGenerateQuizFlow(g *genkit.Genkit) {
	genkit.DefineFlow(g, "generateQuiz",
		func(ctx context.Context, input *GenerateQuizInput) (*GenerateQuizOutput, error) {
			if input.Summary == "" {
				return nil, fmt.Errorf("summary tidak boleh kosong")
			}
			if input.NumQuestions <= 0 {
				return nil, fmt.Errorf("num_questions harus lebih dari 0")
			}

			resp, err := genkit.GenerateText(ctx, g,
				ai.WithPrompt(buildQuizPrompt(input.Summary, input.NumQuestions)),
			)
			if err != nil {
				return nil, fmt.Errorf("gagal generate quiz: %w", err)
			}

			// Parse JSON dari response AI
			output, err := parseQuizResponse(resp, input.NumQuestions)
			if err != nil {
				return nil, fmt.Errorf("gagal parse response quiz: %w", err)
			}

			return output, nil
		},
	)
}

func buildQuizPrompt(summary string, numQuestions int) string {
	return fmt.Sprintf(`Kamu adalah asisten pembuat soal ujian untuk mahasiswa Universitas Terbuka.

Buatkan %d soal pilihan ganda berdasarkan materi berikut.

Aturan soal:
- Setiap soal memiliki tepat 4 pilihan jawaban (A, B, C, D)
- Hanya ada 1 jawaban yang benar
- Soal menguji pemahaman konsep, bukan hafalan
- Tingkat kesulitan bervariasi (mudah, sedang, sulit)
- Tulis dalam Bahasa Indonesia

Materi:
%s

Kembalikan HANYA JSON tanpa penjelasan apapun, tanpa markdown, tanpa backtick, dengan format persis seperti ini:
{
  "questions": [
    {
      "question": "Teks pertanyaan di sini?",
      "options": ["A. pilihan satu", "B. pilihan dua", "C. pilihan tiga", "D. pilihan empat"],
      "answer": "A"
    }
  ]
}`, numQuestions, summary)
}

func parseQuizResponse(raw string, expectedCount int) (*GenerateQuizOutput, error) {
	// Bersihkan kalau AI masih tambahkan markdown
	cleaned := cleanJSON(raw)

	var output GenerateQuizOutput
	if err := json.Unmarshal([]byte(cleaned), &output); err != nil {
		return nil, fmt.Errorf("JSON tidak valid: %w", err)
	}

	if len(output.Questions) == 0 {
		return nil, fmt.Errorf("tidak ada soal yang dihasilkan")
	}

	// Validasi tiap soal
	for i, q := range output.Questions {
		if q.Question == "" {
			return nil, fmt.Errorf("soal #%d tidak memiliki teks", i+1)
		}
		if len(q.Options) != 4 {
			return nil, fmt.Errorf("soal #%d harus memiliki tepat 4 pilihan", i+1)
		}
		if q.Answer == "" {
			return nil, fmt.Errorf("soal #%d tidak memiliki jawaban", i+1)
		}
		// Normalisasi answer ke huruf kapital
		output.Questions[i].Answer = normalizeAnswer(q.Answer)
	}

	return &output, nil
}

func cleanJSON(s string) string {
	// Hapus markdown code block kalau ada
	s = trimPrefix(s, "```json")
	s = trimPrefix(s, "```")
	s = trimSuffix(s, "```")

	// Cari posisi { pertama dan } terakhir
	start := -1
	end := -1
	for i, c := range s {
		if c == '{' && start == -1 {
			start = i
		}
		if c == '}' {
			end = i
		}
	}
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}

func normalizeAnswer(answer string) string {
	if len(answer) == 0 {
		return ""
	}
	// Ambil karakter pertama saja, jadikan uppercase
	c := answer[0]
	if c >= 'a' && c <= 'd' {
		return string(c - 32) // lowercase ke uppercase
	}
	return string(c)
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}
