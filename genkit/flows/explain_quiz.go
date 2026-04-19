package flows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

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

func RegisterExplainQuizFlow(g *genkit.Genkit) *core.Flow[*ExplainQuizInput, *ExplainQuizOutput, struct{}] {
	return genkit.DefineFlow(g, "explainQuiz",
		func(ctx context.Context, input *ExplainQuizInput) (*ExplainQuizOutput, error) {
			if len(input.WrongQuestions) == 0 {
				return &ExplainQuizOutput{Explanations: []QuestionExplanation{}}, nil
			}

			resp, err := generateWithGroq(buildExplainPrompt(input.WrongQuestions, input.Summary))
			if err != nil {
				return nil, fmt.Errorf("failed to generate explanations: %w", err)
			}

			output, err := parseExplainResponse(resp)
			if err != nil {
				return nil, fmt.Errorf("failed to parse explanation response: %w", err)
			}

			return output, nil
		},
	)
}

func buildExplainPrompt(questions []WrongQuestion, summary string) string {
	qJSON, _ := json.Marshal(questions)
	return fmt.Sprintf(`Kamu adalah tutor akademik yang membantu mahasiswa memahami kesalahan mereka dalam ujian.

TUGAS: Berikan penjelasan singkat dan informatif untuk setiap soal yang dijawab salah oleh mahasiswa.

FORMAT PENJELASAN PER SOAL:
- Jelaskan mengapa jawaban mahasiswa SALAH (1 kalimat)
- Jelaskan mengapa jawaban yang BENAR adalah benar, dengan konsep dari materi (2-3 kalimat)
- Sebutkan konsep atau bagian dari materi yang relevan sebagai referensi

ATURAN:
- Bahasa Indonesia yang jelas dan akademis
- Maksimal 4-5 kalimat per soal — padat, langsung ke inti
- Jangan ulang teks soal atau pilihan, langsung ke penjelasan
- Gunakan pengetahuan dari ringkasan materi yang diberikan
- Jika penjelasan mengandung formula matematika/fisika/kimia, gunakan LaTeX: $...$ untuk inline, $$...$$ untuk display

Ringkasan Materi:
%s

Soal yang dijawab salah (format JSON):
%s

Kembalikan HANYA JSON tanpa penjelasan, tanpa markdown, tanpa backtick:
{
  "explanations": [
    {
      "id": "question-uuid-here",
      "explanation": "Penjelasan singkat mengapa jawaban salah dan konsep yang benar..."
    }
  ]
}`, summary, string(qJSON))
}

func parseExplainResponse(raw string) (*ExplainQuizOutput, error) {
	cleaned := cleanJSON(raw)

	var output ExplainQuizOutput
	if err := json.Unmarshal([]byte(cleaned), &output); err != nil {
		return nil, fmt.Errorf("JSON tidak valid: %w", err)
	}

	return &output, nil
}
