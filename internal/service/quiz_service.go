package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/repository"
	pkgai "backend-pretest-ai/pkg/ai"
)

var (
	ErrQuizNotFound        = errors.New("quiz tidak ditemukan")
	ErrNotQuizOwner        = errors.New("kamu tidak memiliki akses ke quiz ini")
	ErrModuleNotSummarized = errors.New("modul belum selesai diproses, coba beberapa saat lagi")
	ErrQuizAlreadyDone     = errors.New("quiz ini sudah dikerjakan")
	ErrAnswerCountMismatch = errors.New("jumlah jawaban tidak sesuai dengan jumlah soal")
	ErrInvalidQuestionID   = errors.New("terdapat question_id yang tidak valid")
)

type QuizServiceContract interface {
	Generate(ctx context.Context, userID string, req dto.GenerateQuizRequest) (*dto.QuizResponse, error)
	Submit(ctx context.Context, userID string, quizID string, req dto.SubmitAnswerRequest) (*dto.QuizResultResponse, error)
	GetHistory(ctx context.Context, userID string) ([]dto.QuizHistoryResponse, error)
	GetHistoryByModule(ctx context.Context, userID string, moduleID string) ([]dto.QuizHistoryResponse, error)
	GetResult(ctx context.Context, userID string, quizID string) (*dto.QuizResultResponse, error)
	Retry(ctx context.Context, userID string, quizID string) (*dto.QuizResponse, error)
}

type QuizAI interface {
	GenerateQuiz(summary string, numQuestions int) (*pkgai.GenerateQuizOutput, error)
}

type QuizService struct {
	quizRepo   repository.QuizRepositoryContract
	moduleRepo repository.ModuleRepositoryContract
	aiClient   QuizAI
}

func NewQuizService(quizRepo repository.QuizRepositoryContract, moduleRepo repository.ModuleRepositoryContract, aiClient QuizAI) *QuizService {
	return &QuizService{
		quizRepo:   quizRepo,
		moduleRepo: moduleRepo,
		aiClient:   aiClient,
	}
}

// Generate — buat quiz baru dari summary modul via Genkit
func (s *QuizService) Generate(ctx context.Context, userID string, req dto.GenerateQuizRequest) (*dto.QuizResponse, error) {
	// Ambil modul, validasi ownership
	module, err := s.moduleRepo.FindByID(ctx, req.ModuleID)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, ErrModuleNotFound
	}
	if module.UserID != userID {
		return nil, ErrNotModuleOwner
	}

	// Summary harus sudah ada
	if !module.IsSummarized || module.Summary == "" {
		return nil, ErrModuleNotSummarized
	}

	// Call Genkit generate quiz
	result, err := s.aiClient.GenerateQuiz(module.Summary, req.NumQuestions)
	if err != nil {
		return nil, fmt.Errorf("gagal generate quiz: %w", err)
	}

	// Bangun domain questions dari hasil AI
	questions := make([]domain.Question, 0, len(result.Questions))
	for _, q := range result.Questions {
		optionsJSON, err := json.Marshal(q.Options)
		if err != nil {
			return nil, fmt.Errorf("gagal parse options: %w", err)
		}
		questions = append(questions, domain.Question{
			Text:          q.Question,
			Options:       string(optionsJSON),
			CorrectAnswer: q.Answer,
		})
	}

	quiz := &domain.Quiz{
		UserID:       userID,
		ModuleID:     req.ModuleID,
		NumQuestions: req.NumQuestions,
		Status:       domain.QuizStatusPending,
		Questions:    questions,
	}

	if err := s.quizRepo.Create(ctx, quiz); err != nil {
		return nil, fmt.Errorf("gagal menyimpan quiz: %w", err)
	}

	return toQuizResponse(quiz, module.Title), nil
}

// Submit — user kirim jawaban, hitung skor
func (s *QuizService) Submit(ctx context.Context, userID string, quizID string, req dto.SubmitAnswerRequest) (*dto.QuizResultResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz == nil {
		return nil, ErrQuizNotFound
	}
	if quiz.UserID != userID {
		return nil, ErrNotQuizOwner
	}
	if quiz.Status == domain.QuizStatusCompleted {
		return nil, ErrQuizAlreadyDone
	}
	if len(req.Answers) != len(quiz.Questions) {
		return nil, ErrAnswerCountMismatch
	}

	// Buat map questionID → jawaban user
	answerMap := make(map[string]string, len(req.Answers))
	for _, a := range req.Answers {
		answerMap[a.QuestionID] = a.Answer
	}

	// Hitung skor dan isi user_answer
	correct := 0
	for i := range quiz.Questions {
		answer, ok := answerMap[quiz.Questions[i].ID]
		if !ok {
			return nil, ErrInvalidQuestionID
		}
		quiz.Questions[i].UserAnswer = answer
		if answer == quiz.Questions[i].CorrectAnswer {
			correct++
		}
	}

	score := (correct * 100) / len(quiz.Questions)

	if err := s.quizRepo.SaveAnswersAndScore(ctx, quizID, quiz.Questions, score); err != nil {
		return nil, fmt.Errorf("gagal menyimpan hasil quiz: %w", err)
	}

	quiz.Score = &score
	quiz.Status = domain.QuizStatusCompleted

	return toQuizResultResponse(quiz), nil
}

// GetHistory — riwayat semua quiz user
func (s *QuizService) GetHistory(ctx context.Context, userID string) ([]dto.QuizHistoryResponse, error) {
	quizzes, err := s.quizRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toHistoryList(quizzes), nil
}

// GetHistoryByModule — riwayat quiz user untuk modul tertentu
func (s *QuizService) GetHistoryByModule(ctx context.Context, userID string, moduleID string) ([]dto.QuizHistoryResponse, error) {
	quizzes, err := s.quizRepo.FindByUserIDAndModuleID(ctx, userID, moduleID)
	if err != nil {
		return nil, err
	}
	return toHistoryList(quizzes), nil
}

// GetResult — lihat hasil quiz yang sudah dikerjakan
func (s *QuizService) GetResult(ctx context.Context, userID string, quizID string) (*dto.QuizResultResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz == nil {
		return nil, ErrQuizNotFound
	}
	if quiz.UserID != userID {
		return nil, ErrNotQuizOwner
	}
	return toQuizResultResponse(quiz), nil
}

// Retry — buat quiz baru dari modul yang sama (soal berbeda karena AI generate ulang)
func (s *QuizService) Retry(ctx context.Context, userID string, quizID string) (*dto.QuizResponse, error) {
	// Ambil quiz lama sebagai referensi modul dan jumlah soal
	oldQuiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if oldQuiz == nil {
		return nil, ErrQuizNotFound
	}
	if oldQuiz.UserID != userID {
		return nil, ErrNotQuizOwner
	}

	// Generate quiz baru dari modul yang sama, jumlah soal sama
	return s.Generate(ctx, userID, dto.GenerateQuizRequest{
		ModuleID:     oldQuiz.ModuleID,
		NumQuestions: oldQuiz.NumQuestions,
	})
}

// --- Mapper helpers ---

func toQuizResponse(quiz *domain.Quiz, moduleTitle string) *dto.QuizResponse {
	questions := make([]dto.QuestionResponse, 0, len(quiz.Questions))
	for _, q := range quiz.Questions {
		var options []string
		_ = json.Unmarshal([]byte(q.Options), &options)
		questions = append(questions, dto.QuestionResponse{
			ID:      q.ID,
			Text:    q.Text,
			Options: options,
		})
	}
	return &dto.QuizResponse{
		ID:           quiz.ID,
		ModuleID:     quiz.ModuleID,
		ModuleTitle:  moduleTitle,
		NumQuestions: quiz.NumQuestions,
		Status:       string(quiz.Status),
		Questions:    questions,
		CreatedAt:    quiz.CreatedAt.Format(time.RFC3339),
	}
}

func toQuizResultResponse(quiz *domain.Quiz) *dto.QuizResultResponse {
	questions := make([]dto.QuestionResultResponse, 0, len(quiz.Questions))
	for _, q := range quiz.Questions {
		var options []string
		_ = json.Unmarshal([]byte(q.Options), &options)
		questions = append(questions, dto.QuestionResultResponse{
			ID:            q.ID,
			Text:          q.Text,
			Options:       options,
			CorrectAnswer: q.CorrectAnswer,
			UserAnswer:    q.UserAnswer,
			IsCorrect:     q.UserAnswer == q.CorrectAnswer,
		})
	}
	score := 0
	if quiz.Score != nil {
		score = *quiz.Score
	}
	return &dto.QuizResultResponse{
		ID:           quiz.ID,
		ModuleID:     quiz.ModuleID,
		ModuleTitle:  quiz.Module.Title,
		NumQuestions: quiz.NumQuestions,
		Score:        score,
		Status:       string(quiz.Status),
		Questions:    questions,
		CreatedAt:    quiz.CreatedAt.Format(time.RFC3339),
	}
}

func toHistoryList(quizzes []domain.Quiz) []dto.QuizHistoryResponse {
	result := make([]dto.QuizHistoryResponse, 0, len(quizzes))
	for _, q := range quizzes {
		result = append(result, dto.QuizHistoryResponse{
			ID:           q.ID,
			ModuleID:     q.ModuleID,
			ModuleTitle:  q.Module.Title,
			NumQuestions: q.NumQuestions,
			Score:        q.Score,
			Status:       string(q.Status),
			CreatedAt:    q.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
}
