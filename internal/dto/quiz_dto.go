package dto

// --- Request ---

type GenerateQuizRequest struct {
	ModuleID     string `json:"module_id"  validate:"required,uuid"`
	NumQuestions int    `json:"num_questions" validate:"required,oneof=20 40 50"`
}

type SubmitAnswerRequest struct {
	Answers []AnswerItem `json:"answers" validate:"dive"`
}

type AnswerItem struct {
	QuestionID string `json:"question_id" validate:"required,uuid"`
	Answer     string `json:"answer"      validate:"required,oneof=A B C D"`
}

// --- Response ---

type QuestionResponse struct {
	ID      string   `json:"id"`
	Text    string   `json:"text"`
	Options []string `json:"options"`
}

type QuestionResultResponse struct {
	ID            string   `json:"id"`
	Text          string   `json:"text"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correct_answer"`
	UserAnswer    string   `json:"user_answer"`
	IsCorrect     bool     `json:"is_correct"`
	Explanation   string   `json:"explanation,omitempty"`
}

type ExplainQuizRequest struct {
	QuizID string `json:"quiz_id" validate:"required,uuid"`
}

type QuizResponse struct {
	ID           string             `json:"id"`
	ModuleID     string             `json:"module_id"`
	ModuleTitle  string             `json:"module_title"`
	NumQuestions int                `json:"num_questions"`
	Status       string             `json:"status"`
	Questions    []QuestionResponse `json:"questions"`
	CreatedAt    string             `json:"created_at"`
}

type QuizResultResponse struct {
	ID           string                   `json:"id"`
	ModuleID     string                   `json:"module_id"`
	ModuleTitle  string                   `json:"module_title"`
	NumQuestions int                      `json:"num_questions"`
	Score        int                      `json:"score"`
	Status       string                   `json:"status"`
	Questions    []QuestionResultResponse `json:"questions"`
	CreatedAt    string                   `json:"created_at"`
}

type QuizHistoryResponse struct {
	ID           string `json:"id"`
	ModuleID     string `json:"module_id"`
	ModuleTitle  string `json:"module_title"`
	NumQuestions int    `json:"num_questions"`
	Score        *int   `json:"score"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}
