package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/repository"
	pkgai "backend-pretest-ai/pkg/ai"
)

var (
	ErrQuizNotFound           = errors.New("quiz tidak ditemukan")
	ErrNotQuizOwner           = errors.New("kamu tidak memiliki akses ke quiz ini")
	ErrModuleNotSummarized    = errors.New("modul belum selesai diproses, coba beberapa saat lagi")
	ErrQuizAlreadyDone        = errors.New("quiz ini sudah dikerjakan")
	ErrInsufficientQuizQuota  = errors.New("kuota quiz habis, silakan beli paket terlebih dahulu")
	ErrQuizCannotBeCancelled  = errors.New("hanya quiz yang belum dikerjakan yang dapat dibatalkan")
)

type QuizServiceContract interface {
	Generate(ctx context.Context, userID string, req dto.GenerateQuizRequest) (*dto.QuizResponse, error)
	Submit(ctx context.Context, userID string, quizID string, req dto.SubmitAnswerRequest) (*dto.QuizResultResponse, error)
	GetHistory(ctx context.Context, userID string) ([]dto.QuizHistoryResponse, error)
	GetHistoryByModule(ctx context.Context, userID string, moduleID string) ([]dto.QuizHistoryResponse, error)
	GetResult(ctx context.Context, userID string, quizID string) (*dto.QuizResultResponse, error)
	Retry(ctx context.Context, userID string, quizID string) (*dto.QuizResponse, error)
	Cancel(ctx context.Context, userID string, quizID string) error
	Explain(ctx context.Context, userID string, quizID string) (*dto.QuizResultResponse, error)
}

type QuizAI interface {
	GenerateQuiz(summary string, numQuestions int) (*pkgai.GenerateQuizOutput, error)
	ExplainQuiz(wrongQuestions []pkgai.WrongQuestion, summary string) (*pkgai.ExplainQuizOutput, error)
}

type QuizService struct {
	quizRepo   repository.QuizRepositoryContract
	moduleRepo repository.ModuleRepositoryContract
	userRepo   repository.UserRepository
	aiClient   QuizAI
}

func NewQuizService(quizRepo repository.QuizRepositoryContract, moduleRepo repository.ModuleRepositoryContract, userRepo repository.UserRepository, aiClient QuizAI) *QuizService {
	return &QuizService{
		quizRepo:   quizRepo,
		moduleRepo: moduleRepo,
		userRepo:   userRepo,
		aiClient:   aiClient,
	}
}

// Generate — buat quiz baru dari summary modul via Genkit
func (s *QuizService) Generate(ctx context.Context, userID string, req dto.GenerateQuizRequest) (*dto.QuizResponse, error) {
	// Cek role user — admin bypass quota
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Deduct quiz quota (skip untuk admin)
	quotaDeducted := false
	if user.Role != domain.RoleAdmin {
		if err := s.userRepo.DeductQuizQuota(ctx, userID); err != nil {
			if errors.Is(err, repository.ErrQuotaInsufficient) {
				return nil, ErrInsufficientQuizQuota
			}
			return nil, err
		}
		quotaDeducted = true
	}

	// Jika ada error setelah quota dikurangi, kembalikan quota
	var quizCreated bool
	defer func() {
		if quotaDeducted && !quizCreated {
			if restoreErr := s.userRepo.RestoreQuizQuota(context.Background(), userID); restoreErr != nil {
				log.Printf("[quiz_service] gagal restore quiz quota untuk user %s: %v", userID, restoreErr)
			}
		}
	}()

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

	quizCreated = true
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
	if quiz.Status != domain.QuizStatusPending {
		return nil, ErrQuizAlreadyDone
	}
	// Buat map questionID → jawaban user
	answerMap := make(map[string]string, len(req.Answers))
	for _, a := range req.Answers {
		answerMap[a.QuestionID] = a.Answer
	}

	// Hitung skor; soal yang tidak ada di answerMap → UserAnswer = "" → pasti salah
	correct := 0
	for i := range quiz.Questions {
		answer := answerMap[quiz.Questions[i].ID] // "" jika tidak dijawab
		quiz.Questions[i].UserAnswer = answer
		if answer != "" && answer == quiz.Questions[i].CorrectAnswer {
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

// Cancel — batalkan quiz pending dan kembalikan quiz quota ke user
func (s *QuizService) Cancel(ctx context.Context, userID string, quizID string) error {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return err
	}
	if quiz == nil {
		return ErrQuizNotFound
	}
	if quiz.UserID != userID {
		return ErrNotQuizOwner
	}
	if quiz.Status != domain.QuizStatusPending {
		return ErrQuizCannotBeCancelled
	}

	if err := s.quizRepo.UpdateStatus(ctx, quizID, domain.QuizStatusCancelled); err != nil {
		return fmt.Errorf("gagal membatalkan quiz: %w", err)
	}

	// Kembalikan quota (skip untuk admin)
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Printf("[quiz_service] gagal ambil user saat restore quota, quiz %s: %v", quizID, err)
		return nil
	}
	if user != nil && user.Role != domain.RoleAdmin {
		if restoreErr := s.userRepo.RestoreQuizQuota(ctx, userID); restoreErr != nil {
			log.Printf("[quiz_service] gagal restore quiz quota untuk user %s: %v", userID, restoreErr)
		}
	}

	return nil
}

// Explain — generate AI explanation for wrong answers and persist them
func (s *QuizService) Explain(ctx context.Context, userID string, quizID string) (*dto.QuizResultResponse, error) {
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
	if quiz.Status != domain.QuizStatusCompleted {
		return nil, fmt.Errorf("quiz belum selesai dikerjakan")
	}

	// Collect wrong questions that don't have explanation yet
	var wrongQuestions []pkgai.WrongQuestion
	for _, q := range quiz.Questions {
		if q.UserAnswer != q.CorrectAnswer && q.Explanation == "" {
			var options []string
			_ = json.Unmarshal([]byte(q.Options), &options)
			wrongQuestions = append(wrongQuestions, pkgai.WrongQuestion{
				ID:            q.ID,
				Question:      q.Text,
				Options:       options,
				CorrectAnswer: q.CorrectAnswer,
				UserAnswer:    q.UserAnswer,
			})
		}
	}

	if len(wrongQuestions) == 0 {
		return toQuizResultResponse(quiz), nil
	}

	// Get module summary for context
	module, err := s.moduleRepo.FindByID(ctx, quiz.ModuleID)
	if err != nil || module == nil {
		return nil, fmt.Errorf("gagal mengambil data modul")
	}

	result, err := s.aiClient.ExplainQuiz(wrongQuestions, module.Summary)
	if err != nil {
		return nil, fmt.Errorf("gagal generate penjelasan: %w", err)
	}

	// Build map and persist
	explanationMap := make(map[string]string, len(result.Explanations))
	for _, e := range result.Explanations {
		explanationMap[e.ID] = e.Explanation
	}
	if err := s.quizRepo.SaveExplanations(ctx, explanationMap); err != nil {
		return nil, fmt.Errorf("gagal menyimpan penjelasan: %w", err)
	}

	// Patch in-memory questions so response reflects new explanations
	for i := range quiz.Questions {
		if exp, ok := explanationMap[quiz.Questions[i].ID]; ok {
			quiz.Questions[i].Explanation = exp
		}
	}

	return toQuizResultResponse(quiz), nil
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
			Explanation:   q.Explanation,
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
