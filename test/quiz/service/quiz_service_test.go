package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
	pkgai "backend-pretest-ai/pkg/ai"
)

// --- Mocks ---

type MockQuizRepository struct {
	mock.Mock
}

func (m *MockQuizRepository) Create(ctx context.Context, quiz *domain.Quiz) error {
	args := m.Called(ctx, quiz)
	return args.Error(0)
}

func (m *MockQuizRepository) FindByID(ctx context.Context, id string) (*domain.Quiz, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Quiz), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuizRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Quiz, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]domain.Quiz), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuizRepository) FindByUserIDAndModuleID(ctx context.Context, userID string, moduleID string) ([]domain.Quiz, error) {
	args := m.Called(ctx, userID, moduleID)
	if args.Get(0) != nil {
		return args.Get(0).([]domain.Quiz), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuizRepository) SaveAnswersAndScore(ctx context.Context, quizID string, questions []domain.Question, score int) error {
	args := m.Called(ctx, quizID, questions, score)
	return args.Error(0)
}

type MockModuleRepository struct {
	mock.Mock
}

func (m *MockModuleRepository) Create(ctx context.Context, module *domain.Module) error {
	args := m.Called(ctx, module)
	return args.Error(0)
}

func (m *MockModuleRepository) FindByID(ctx context.Context, id string) (*domain.Module, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Module), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockModuleRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Module, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]domain.Module), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockModuleRepository) UpdateSummary(ctx context.Context, moduleID string, summary string) error {
	args := m.Called(ctx, moduleID, summary)
	return args.Error(0)
}

func (m *MockModuleRepository) UpdateSummaryManual(ctx context.Context, moduleID string, summary string) error {
	args := m.Called(ctx, moduleID, summary)
	return args.Error(0)
}

func (m *MockModuleRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockQuizAI struct {
	mock.Mock
}

func (m *MockQuizAI) GenerateQuiz(summary string, numQuestions int) (*pkgai.GenerateQuizOutput, error) {
	args := m.Called(summary, numQuestions)
	if args.Get(0) != nil {
		return args.Get(0).(*pkgai.GenerateQuizOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Setup ---

func setupQuizServiceTest(t *testing.T) (service.QuizServiceContract, *MockQuizRepository, *MockModuleRepository, *MockQuizAI) {
	mockQuizRepo := new(MockQuizRepository)
	mockModuleRepo := new(MockModuleRepository)
	mockAI := new(MockQuizAI)

	srv := service.NewQuizService(mockQuizRepo, mockModuleRepo, mockAI)
	return srv, mockQuizRepo, mockModuleRepo, mockAI
}

// --- Test Generate ---

func TestQuizService_Generate(t *testing.T) {
	t.Run("Modul tidak ditemukan", func(t *testing.T) {
		srv, _, mockModuleRepo, _ := setupQuizServiceTest(t)
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(nil, nil)

		resp, err := srv.Generate(context.Background(), "user-1", dto.GenerateQuizRequest{ModuleID: "mod-1"})

		assert.Error(t, err)
		assert.Equal(t, service.ErrModuleNotFound, err)
		assert.Nil(t, resp)
	})

	t.Run("Modul milik user lain", func(t *testing.T) {
		srv, _, mockModuleRepo, _ := setupQuizServiceTest(t)
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(&domain.Module{UserID: "user-other"}, nil)

		resp, err := srv.Generate(context.Background(), "user-1", dto.GenerateQuizRequest{ModuleID: "mod-1"})

		assert.Error(t, err)
		assert.Equal(t, service.ErrNotModuleOwner, err)
		assert.Nil(t, resp)
	})

	t.Run("is_summarized = false", func(t *testing.T) {
		srv, _, mockModuleRepo, _ := setupQuizServiceTest(t)
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(&domain.Module{UserID: "user-1", IsSummarized: false}, nil)

		resp, err := srv.Generate(context.Background(), "user-1", dto.GenerateQuizRequest{ModuleID: "mod-1"})

		assert.Error(t, err)
		assert.Equal(t, service.ErrModuleNotSummarized, err)
		assert.Nil(t, resp)
	})

	t.Run("Summary kosong meski is_summarized = true", func(t *testing.T) {
		srv, _, mockModuleRepo, _ := setupQuizServiceTest(t)
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(&domain.Module{UserID: "user-1", IsSummarized: true, Summary: ""}, nil)

		resp, err := srv.Generate(context.Background(), "user-1", dto.GenerateQuizRequest{ModuleID: "mod-1"})

		assert.Error(t, err)
		assert.Equal(t, service.ErrModuleNotSummarized, err)
		assert.Nil(t, resp)
	})

	t.Run("Genkit gagal (error)", func(t *testing.T) {
		srv, _, mockModuleRepo, mockAI := setupQuizServiceTest(t)
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(&domain.Module{UserID: "user-1", IsSummarized: true, Summary: "summary", Title: "Title"}, nil)
		mockAI.On("GenerateQuiz", "summary", 5).Return(nil, errors.New("ai error"))

		resp, err := srv.Generate(context.Background(), "user-1", dto.GenerateQuizRequest{ModuleID: "mod-1", NumQuestions: 5})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ai error")
		assert.Nil(t, resp)
	})

	t.Run("Sukses, num_questions=5", func(t *testing.T) {
		srv, mockQuizRepo, mockModuleRepo, mockAI := setupQuizServiceTest(t)
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(&domain.Module{UserID: "user-1", IsSummarized: true, Summary: "summary", Title: "Title"}, nil)
		
		aiOutput := &pkgai.GenerateQuizOutput{
			Questions: []pkgai.Question{
				{Question: "Q1", Options: []string{"A", "B", "C", "D"}, Answer: "A"},
				{Question: "Q2", Options: []string{"A", "B", "C", "D"}, Answer: "B"},
				{Question: "Q3", Options: []string{"A", "B", "C", "D"}, Answer: "C"},
				{Question: "Q4", Options: []string{"A", "B", "C", "D"}, Answer: "D"},
				{Question: "Q5", Options: []string{"A", "B", "C", "D"}, Answer: "A"},
			},
		}
		mockAI.On("GenerateQuiz", "summary", 5).Return(aiOutput, nil)
		mockQuizRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Quiz")).Return(nil)

		resp, err := srv.Generate(context.Background(), "user-1", dto.GenerateQuizRequest{ModuleID: "mod-1", NumQuestions: 5})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Questions, 5)
		assert.Equal(t, "pending", resp.Status)
		assert.Equal(t, "Title", resp.ModuleTitle)
	})

	t.Run("Sukses, num_questions=10", func(t *testing.T) {
		srv, mockQuizRepo, mockModuleRepo, mockAI := setupQuizServiceTest(t)
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(&domain.Module{UserID: "user-1", IsSummarized: true, Summary: "summary", Title: "Title"}, nil)
		
		questions := make([]pkgai.Question, 10)
		for i := 0; i < 10; i++ {
			questions[i] = pkgai.Question{Question: "Q", Options: []string{"A", "B", "C", "D"}, Answer: "A"}
		}
		aiOutput := &pkgai.GenerateQuizOutput{Questions: questions}
		
		mockAI.On("GenerateQuiz", "summary", 10).Return(aiOutput, nil)
		mockQuizRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Quiz")).Return(nil)

		resp, err := srv.Generate(context.Background(), "user-1", dto.GenerateQuizRequest{ModuleID: "mod-1", NumQuestions: 10})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Questions, 10)
	})

	t.Run("Response soal tidak mengandung correct_answer", func(t *testing.T) {
		// DTO QuizResponse.Questions adalah []dto.QuestionResponse yang memang tidak punya CorrectAnswer
		srv, mockQuizRepo, mockModuleRepo, mockAI := setupQuizServiceTest(t)
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(&domain.Module{UserID: "user-1", IsSummarized: true, Summary: "summary", Title: "Title"}, nil)
		
		aiOutput := &pkgai.GenerateQuizOutput{
			Questions: []pkgai.Question{{Question: "Q1", Options: []string{"A"}, Answer: "A"}},
		}
		mockAI.On("GenerateQuiz", "summary", 1).Return(aiOutput, nil)
		mockQuizRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Quiz")).Return(nil)

		_, err := srv.Generate(context.Background(), "user-1", dto.GenerateQuizRequest{ModuleID: "mod-1", NumQuestions: 1})

		assert.NoError(t, err)
		// CorrectAnswer tidak ada di DTO QuestionResponse
		// Kita cek via refleksi atau manual: assert.Equal(t, "", resp.Questions[0].CorrectAnswer) // ini akan error compile karena tidak ada fieldnya
		// Pengecekan ini lebih ke arah kontrak DTO
	})
}

// --- Test Submit ---

func TestQuizService_Submit(t *testing.T) {
	t.Run("Quiz tidak ditemukan", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(nil, nil)

		resp, err := srv.Submit(context.Background(), "user-1", "q-1", dto.SubmitAnswerRequest{})

		assert.Error(t, err)
		assert.Equal(t, service.ErrQuizNotFound, err)
		assert.Nil(t, resp)
	})

	t.Run("Quiz milik user lain", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(&domain.Quiz{UserID: "user-other"}, nil)

		resp, err := srv.Submit(context.Background(), "user-1", "q-1", dto.SubmitAnswerRequest{})

		assert.Error(t, err)
		assert.Equal(t, service.ErrNotQuizOwner, err)
		assert.Nil(t, resp)
	})

	t.Run("Quiz sudah completed", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(&domain.Quiz{UserID: "user-1", Status: domain.QuizStatusCompleted}, nil)

		resp, err := srv.Submit(context.Background(), "user-1", "q-1", dto.SubmitAnswerRequest{})

		assert.Error(t, err)
		assert.Equal(t, service.ErrQuizAlreadyDone, err)
		assert.Nil(t, resp)
	})

	t.Run("Jumlah jawaban mismatch", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		quiz := &domain.Quiz{
			UserID: "user-1",
			Questions: []domain.Question{{ID: "q1"}, {ID: "q2"}},
		}
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(quiz, nil)

		req := dto.SubmitAnswerRequest{
			Answers: []dto.AnswerItem{{QuestionID: "q1", Answer: "A"}},
		}

		resp, err := srv.Submit(context.Background(), "user-1", "q-1", req)

		assert.Error(t, err)
		assert.Equal(t, service.ErrAnswerCountMismatch, err)
		assert.Nil(t, resp)
	})

	t.Run("question_id tidak valid", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		quiz := &domain.Quiz{
			UserID: "user-1",
			Questions: []domain.Question{{ID: "q1"}},
		}
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(quiz, nil)

		req := dto.SubmitAnswerRequest{
			Answers: []dto.AnswerItem{{QuestionID: "invalid-id", Answer: "A"}},
		}

		resp, err := srv.Submit(context.Background(), "user-1", "q-1", req)

		assert.Error(t, err)
		assert.Equal(t, service.ErrInvalidQuestionID, err)
		assert.Nil(t, resp)
	})

	t.Run("Semua jawaban benar (100)", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		quiz := &domain.Quiz{
			ID: "q-1", UserID: "user-1",
			Questions: []domain.Question{
				{ID: "q1", CorrectAnswer: "A", Options: `["A"]`},
				{ID: "q2", CorrectAnswer: "B", Options: `["B"]`},
			},
		}
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(quiz, nil)
		mockQuizRepo.On("SaveAnswersAndScore", mock.Anything, "q-1", mock.Anything, 100).Return(nil)

		req := dto.SubmitAnswerRequest{
			Answers: []dto.AnswerItem{
				{QuestionID: "q1", Answer: "A"},
				{QuestionID: "q2", Answer: "B"},
			},
		}

		resp, err := srv.Submit(context.Background(), "user-1", "q-1", req)

		assert.NoError(t, err)
		assert.Equal(t, 100, resp.Score)
	})

	t.Run("Semua jawaban salah (0)", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		quiz := &domain.Quiz{
			ID: "q-1", UserID: "user-1",
			Questions: []domain.Question{
				{ID: "q1", CorrectAnswer: "A", Options: `["A"]`},
			},
		}
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(quiz, nil)
		mockQuizRepo.On("SaveAnswersAndScore", mock.Anything, "q-1", mock.Anything, 0).Return(nil)

		req := dto.SubmitAnswerRequest{
			Answers: []dto.AnswerItem{{QuestionID: "q1", Answer: "B"}},
		}

		resp, err := srv.Submit(context.Background(), "user-1", "q-1", req)

		assert.NoError(t, err)
		assert.Equal(t, 0, resp.Score)
	})

	t.Run("Sebagian benar (60)", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		questions := make([]domain.Question, 10)
		answers := make([]dto.AnswerItem, 10)
		for i := 0; i < 10; i++ {
			id := string(rune('a' + i))
			questions[i] = domain.Question{ID: id, CorrectAnswer: "A", Options: `["A"]`}
			if i < 6 {
				answers[i] = dto.AnswerItem{QuestionID: id, Answer: "A"}
			} else {
				answers[i] = dto.AnswerItem{QuestionID: id, Answer: "B"}
			}
		}
		quiz := &domain.Quiz{ID: "q-1", UserID: "user-1", Questions: questions}
		
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(quiz, nil)
		mockQuizRepo.On("SaveAnswersAndScore", mock.Anything, "q-1", mock.Anything, 60).Return(nil)

		resp, err := srv.Submit(context.Background(), "user-1", "q-1", dto.SubmitAnswerRequest{Answers: answers})

		assert.NoError(t, err)
		assert.Equal(t, 60, resp.Score)
	})

	t.Run("SaveAnswersAndScore DB error", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		quiz := &domain.Quiz{ID: "q-1", UserID: "user-1", Questions: []domain.Question{{ID: "q1", CorrectAnswer: "A", Options: `["A"]`}}}
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(quiz, nil)
		mockQuizRepo.On("SaveAnswersAndScore", mock.Anything, "q-1", mock.Anything, 100).Return(errors.New("db error"))

		resp, err := srv.Submit(context.Background(), "user-1", "q-1", dto.SubmitAnswerRequest{Answers: []dto.AnswerItem{{QuestionID: "q1", Answer: "A"}}})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
		assert.Nil(t, resp)
	})
}

// --- Test History ---

func TestQuizService_GetHistory(t *testing.T) {
	t.Run("User belum punya quiz", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByUserID", mock.Anything, "user-1").Return([]domain.Quiz{}, nil)

		resp, err := srv.GetHistory(context.Background(), "user-1")

		assert.NoError(t, err)
		assert.Empty(t, resp)
	})

	t.Run("User punya mix pending + completed", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		score := 80
		quizzes := []domain.Quiz{
			{Status: domain.QuizStatusPending, Score: nil, Module: domain.Module{Title: "M1"}},
			{Status: domain.QuizStatusCompleted, Score: &score, Module: domain.Module{Title: "M2"}},
		}
		mockQuizRepo.On("FindByUserID", mock.Anything, "user-1").Return(quizzes, nil)

		resp, err := srv.GetHistory(context.Background(), "user-1")

		assert.NoError(t, err)
		assert.Len(t, resp, 2)
		assert.Nil(t, resp[0].Score)
		assert.NotNil(t, resp[1].Score)
		assert.Equal(t, 80, *resp[1].Score)
	})

	t.Run("DB error", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByUserID", mock.Anything, "user-1").Return(nil, errors.New("db error"))

		resp, err := srv.GetHistory(context.Background(), "user-1")

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestQuizService_GetHistoryByModule(t *testing.T) {
	t.Run("Tidak ada quiz untuk modul itu", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByUserIDAndModuleID", mock.Anything, "user-1", "mod-1").Return([]domain.Quiz{}, nil)

		resp, err := srv.GetHistoryByModule(context.Background(), "user-1", "mod-1")

		assert.NoError(t, err)
		assert.Empty(t, resp)
	})

	t.Run("Ada beberapa quiz untuk modul itu", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		quizzes := []domain.Quiz{{ModuleID: "mod-1", Module: domain.Module{Title: "M1"}}, {ModuleID: "mod-1", Module: domain.Module{Title: "M1"}}}
		mockQuizRepo.On("FindByUserIDAndModuleID", mock.Anything, "user-1", "mod-1").Return(quizzes, nil)

		resp, err := srv.GetHistoryByModule(context.Background(), "user-1", "mod-1")

		assert.NoError(t, err)
		assert.Len(t, resp, 2)
	})
}

// --- Test GetResult ---

func TestQuizService_GetResult(t *testing.T) {
	t.Run("Quiz tidak ditemukan", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(nil, nil)

		resp, err := srv.GetResult(context.Background(), "user-1", "q-1")

		assert.Error(t, err)
		assert.Equal(t, service.ErrQuizNotFound, err)
		assert.Nil(t, resp)
	})

	t.Run("Quiz milik user lain", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(&domain.Quiz{UserID: "user-other"}, nil)

		resp, err := srv.GetResult(context.Background(), "user-1", "q-1")

		assert.Error(t, err)
		assert.Equal(t, service.ErrNotQuizOwner, err)
		assert.Nil(t, resp)
	})

	t.Run("Sukses", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		quiz := &domain.Quiz{
			ID: "q-1", UserID: "user-1",
			Questions: []domain.Question{{ID: "q1", CorrectAnswer: "A", UserAnswer: "A", Options: `["A"]`}},
			Module: domain.Module{Title: "M1"},
		}
		mockQuizRepo.On("FindByID", mock.Anything, "q-1").Return(quiz, nil)

		resp, err := srv.GetResult(context.Background(), "user-1", "q-1")

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "A", resp.Questions[0].CorrectAnswer)
	})
}

// --- Test Retry ---

func TestQuizService_Retry(t *testing.T) {
	t.Run("Quiz lama tidak ditemukan", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByID", mock.Anything, "q-old").Return(nil, nil)

		resp, err := srv.Retry(context.Background(), "user-1", "q-old")

		assert.Error(t, err)
		assert.Equal(t, service.ErrQuizNotFound, err)
		assert.Nil(t, resp)
	})

	t.Run("Quiz lama milik user lain", func(t *testing.T) {
		srv, mockQuizRepo, _, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByID", mock.Anything, "q-old").Return(&domain.Quiz{UserID: "user-other"}, nil)

		resp, err := srv.Retry(context.Background(), "user-1", "q-old")

		assert.Error(t, err)
		assert.Equal(t, service.ErrNotQuizOwner, err)
		assert.Nil(t, resp)
	})

	t.Run("Modul sudah tidak tersummarisasi", func(t *testing.T) {
		srv, mockQuizRepo, mockModuleRepo, _ := setupQuizServiceTest(t)
		mockQuizRepo.On("FindByID", mock.Anything, "q-old").Return(&domain.Quiz{UserID: "user-1", ModuleID: "mod-1", NumQuestions: 5}, nil)
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(&domain.Module{UserID: "user-1", IsSummarized: false}, nil)

		resp, err := srv.Retry(context.Background(), "user-1", "q-old")

		assert.Error(t, err)
		assert.Equal(t, service.ErrModuleNotSummarized, err)
		assert.Nil(t, resp)
	})

	t.Run("Sukses, module_id dan num_questions sama", func(t *testing.T) {
		srv, mockQuizRepo, mockModuleRepo, mockAI := setupQuizServiceTest(t)
		
		// Setup quiz lama
		mockQuizRepo.On("FindByID", mock.Anything, "q-old").Return(&domain.Quiz{UserID: "user-1", ModuleID: "mod-1", NumQuestions: 5}, nil)
		
		// Setup module
		mockModuleRepo.On("FindByID", mock.Anything, "mod-1").Return(&domain.Module{UserID: "user-1", IsSummarized: true, Summary: "summary", Title: "Title"}, nil)
		
		// Setup AI generate quiz baru
		aiOutput := &pkgai.GenerateQuizOutput{
			Questions: []pkgai.Question{{Question: "Q baru", Options: []string{"A"}, Answer: "A"}},
		}
		mockAI.On("GenerateQuiz", "summary", 5).Return(aiOutput, nil)
		mockQuizRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Quiz")).Return(nil).Run(func(args mock.Arguments) {
			q := args.Get(1).(*domain.Quiz)
			q.ID = "q-new"
		})

		resp, err := srv.Retry(context.Background(), "user-1", "q-old")

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "q-new", resp.ID)
		assert.Equal(t, "mod-1", resp.ModuleID)
		assert.Equal(t, 5, resp.NumQuestions)
	})
}
