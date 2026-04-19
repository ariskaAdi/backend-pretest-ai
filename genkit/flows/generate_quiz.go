package flows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/core"
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

func RegisterGenerateQuizFlow(g *genkit.Genkit) *core.Flow[*GenerateQuizInput, *GenerateQuizOutput, struct{}] {
	return genkit.DefineFlow(g, "generateQuiz",
		func(ctx context.Context, input *GenerateQuizInput) (*GenerateQuizOutput, error) {
			if input.Summary == "" {
				return nil, fmt.Errorf("summary should not be empty")
			}
			if input.NumQuestions <= 0 {
				return nil, fmt.Errorf("num_questions harus lebih dari 0")
			}

			resp, err := generateWithGroq(buildQuizPrompt(input.Summary, input.NumQuestions))
			if err != nil {
				return nil, fmt.Errorf("failed to generate quiz: %w", err)
			}

			// Parse JSON dari response AI
			output, err := parseQuizResponse(resp, input.NumQuestions)
			if err != nil {
				return nil, fmt.Errorf("failed to parse response: %w", err)
			}

			return output, nil
		},
	)
}

func buildQuizPrompt(summary string, numQuestions int) string {
	return fmt.Sprintf(`Kamu adalah pembuat soal ujian perguruan tinggi yang berpengalaman. Tugasmu membuat soal ujian berkualitas tinggi yang benar-benar menguji pemahaman mahasiswa terhadap materi.

TUGAS: Buat tepat %[1]d soal pilihan ganda berdasarkan materi di bawah ini.

DISTRIBUSI SOAL YANG WAJIB DIPENUHI:
- Pastikan soal tersebar merata ke SEMUA topik/bab yang ada dalam materi
- Jangan hanya fokus pada satu topik — setiap topik utama harus terwakili
- Distribusi tingkat kesulitan: 30%% mudah, 50%% sedang, 20%% sulit

KRITERIA SOAL BERKUALITAS TINGGI:
1. Soal harus menguji PEMAHAMAN MENDALAM, bukan sekadar hafalan istilah
2. Gunakan variasi tipe pertanyaan:
   - Konsep dan definisi ("Apa yang dimaksud dengan...")
   - Perbedaan antar konsep ("Perbedaan antara X dan Y adalah...")
   - Penerapan ("Dalam kasus berikut, metode yang tepat adalah...")
   - Analisis ("Mengapa... / Apa yang terjadi jika...")
   - Identifikasi ("Manakah yang BUKAN termasuk...")
3. Pilihan jawaban yang SALAH (distractor) harus:
   - Masuk akal dan plausibel, bukan jawaban absurd
   - Merupakan konsep yang ada dalam materi namun digunakan pada konteks yang salah
   - Menguji apakah mahasiswa benar-benar paham, bukan hanya menebak
4. Hindari pertanyaan trivial seperti "Apa kepanjangan dari..." atau "Siapa yang mencetuskan..."
5. Kalimat soal harus jelas, tidak ambigu, dan tidak mengandung clue jawaban

ATURAN FORMAT:
- Setiap soal memiliki TEPAT 4 pilihan (A, B, C, D)
- Hanya 1 jawaban benar
- Tulis dalam Bahasa Indonesia yang baik dan akademis
- Jika materi mengandung formula matematika/fisika/kimia, gunakan notasi LaTeX:
  * Inline formula: $x^2 + y^2 = r^2$
  * Display formula: $$\int_a^b f(x)\,dx$$
  * Himpunan: $A = \{x \mid x \in \mathbb{N}\}$, Interval: $(-\infty, 2] \cup [3, \infty)$
  * Jika materi bukan sains/matematika, tulis teks biasa tanpa LaTeX

DISTRIBUSI JAWABAN BENAR (WAJIB DIIKUTI KETAT):
- Jawaban benar HARUS terdistribusi merata: sekitar 25%% soal jawab A, 25%% jawab B, 25%% jawab C, 25%% jawab D
- Untuk %[1]d soal: masing-masing A, B, C, D harus muncul sebagai jawaban benar kurang lebih sama banyak
- DILARANG membuat lebih dari 2 soal berturut-turut dengan jawaban yang sama
- DILARANG mendominasi satu huruf — jika sudah 3 soal jawaban A, soal berikutnya HARUS B, C, atau D
- Letakkan jawaban benar di posisi pilihan yang berbeda-beda (tidak selalu pilihan pertama)

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
}`, numQuestions, summary) // %[1]d = numQuestions (reused), %[2]s = summary
}

func parseQuizResponse(raw string, numQuestions int) (*GenerateQuizOutput, error) {
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

	// Trim ke jumlah yang diminta — AI kadang menghasilkan lebih
	if numQuestions > 0 && len(output.Questions) > numQuestions {
		output.Questions = output.Questions[:numQuestions]
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
