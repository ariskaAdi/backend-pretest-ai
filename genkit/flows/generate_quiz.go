package flows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

type GenerateQuizInput struct {
	RawText      string `json:"raw_text"`
	NumQuestions int    `json:"num_questions"`
}

type DiagramData struct {
	Type    string `json:"type"`    // "svg"
	Content string `json:"content"` // SVG string
}

type QuizQuestion struct {
	Question string       `json:"question"`
	Options  []string     `json:"options"`
	Answer   string       `json:"answer"`
	Diagram  *DiagramData `json:"diagram,omitempty"`
}

type GenerateQuizOutput struct {
	Questions []QuizQuestion `json:"questions"`
}

// maxChunkSize adalah batas karakter per chunk sebelum dikirim ke AI.
// ~8000 char ≈ 2000 token konten — cukup detail tanpa membebani model.
const maxChunkSize = 6000

func RegisterGenerateQuizFlow(g *genkit.Genkit) *core.Flow[*GenerateQuizInput, *GenerateQuizOutput, struct{}] {
	return genkit.DefineFlow(g, "generateQuiz",
		func(ctx context.Context, input *GenerateQuizInput) (*GenerateQuizOutput, error) {
			if input.RawText == "" {
				return nil, fmt.Errorf("raw_text tidak boleh kosong")
			}
			if input.NumQuestions <= 0 {
				return nil, fmt.Errorf("num_questions harus lebih dari 0")
			}

			chunks := splitIntoChunks(input.RawText, maxChunkSize)
			distribution := distributeQuestions(input.NumQuestions, len(chunks))

			var allQuestions []QuizQuestion
			for i, chunk := range chunks {
				n := distribution[i]
				if n == 0 {
					continue
				}

				resp, err := generateText(buildQuizPrompt(chunk, n))
				if err != nil {
					return nil, fmt.Errorf("gagal generate soal chunk %d: %w", i+1, err)
				}

				output, err := parseQuizResponse(resp, n)
				if err != nil {
					return nil, fmt.Errorf("gagal parse soal chunk %d: %w", i+1, err)
				}

				allQuestions = append(allQuestions, output.Questions...)
			}

			if len(allQuestions) > input.NumQuestions {
				allQuestions = allQuestions[:input.NumQuestions]
			}

			return &GenerateQuizOutput{Questions: allQuestions}, nil
		},
	)
}

// splitIntoChunks memotong teks menjadi beberapa bagian dengan ukuran maksimal chunkSize.
// Potongan dilakukan di batas paragraf (baris kosong) agar konteks tidak terpotong di tengah kalimat.
func splitIntoChunks(text string, chunkSize int) []string {
	text = strings.TrimSpace(text)
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= chunkSize {
			chunks = append(chunks, strings.TrimSpace(text))
			break
		}

		// Cari paragraph break (\n\n) terdekat mundur dari posisi chunkSize
		end := chunkSize
		found := false
		for pos := chunkSize; pos > chunkSize/2; pos-- {
			if pos+1 < len(text) && text[pos] == '\n' && text[pos+1] == '\n' {
				end = pos + 2
				found = true
				break
			}
		}
		// Kalau tidak ketemu paragraph break, cari newline biasa
		if !found {
			for pos := chunkSize; pos > chunkSize/2; pos-- {
				if text[pos] == '\n' {
					end = pos + 1
					found = true
					break
				}
			}
		}
		// Hard cut kalau tidak ada newline sama sekali
		if !found {
			end = chunkSize
		}

		chunks = append(chunks, strings.TrimSpace(text[:end]))
		text = strings.TrimSpace(text[end:])
	}

	return chunks
}

// distributeQuestions membagi total soal secara merata ke setiap chunk.
// Sisa dibagi ke chunk-chunk pertama agar total tetap tepat.
func distributeQuestions(total, numChunks int) []int {
	if numChunks == 0 {
		return nil
	}
	base := total / numChunks
	remainder := total % numChunks
	dist := make([]int, numChunks)
	for i := range dist {
		dist[i] = base
		if i < remainder {
			dist[i]++
		}
	}
	return dist
}

func buildQuizPrompt(text string, numQuestions int) string {
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
- JANGAN sertakan field "diagram" — semua soal hanya berupa teks

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
      "question": "Teks pertanyaan?",
      "options": ["A. pilihan satu", "B. pilihan dua", "C. pilihan tiga", "D. pilihan empat"],
      "answer": "A"
    }
  ]
}`, numQuestions, text)
}

func parseQuizResponse(raw string, numQuestions int) (*GenerateQuizOutput, error) {
	cleaned := cleanJSON(raw)

	var output GenerateQuizOutput
	if err := json.Unmarshal([]byte(cleaned), &output); err != nil {
		return nil, fmt.Errorf("JSON tidak valid: %w", err)
	}

	if len(output.Questions) == 0 {
		return nil, fmt.Errorf("tidak ada soal yang dihasilkan")
	}

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
		output.Questions[i].Answer = normalizeAnswer(q.Answer)
	}

	if numQuestions > 0 && len(output.Questions) > numQuestions {
		output.Questions = output.Questions[:numQuestions]
	}

	return &output, nil
}

func cleanJSON(s string) string {
	s = trimPrefix(s, "```json")
	s = trimPrefix(s, "```")
	s = trimSuffix(s, "```")

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
	c := answer[0]
	if c >= 'a' && c <= 'd' {
		return string(c - 32)
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
