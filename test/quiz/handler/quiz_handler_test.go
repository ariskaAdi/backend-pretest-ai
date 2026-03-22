package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/handler"
	"backend-pretest-ai/internal/service"
)

// --- Mocks ---

type MockQuizService struct {
	mock.Mock
}

func (m *MockQuizService) Generate(ctx context.Context, userID string, req dto.GenerateQuizRequest) (*dto.QuizResponse, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) != nil {
		return args.Get(0).(*dto.QuizResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuizService) Submit(ctx context.Context, userID string, quizID string, req dto.SubmitAnswerRequest) (*dto.QuizResultResponse, error) {
	args := m.Called(ctx, userID, quizID, req)
	if args.Get(0) != nil {
		return args.Get(0).(*dto.QuizResultResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuizService) GetHistory(ctx context.Context, userID string) ([]dto.QuizHistoryResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]dto.QuizHistoryResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuizService) GetHistoryByModule(ctx context.Context, userID string, moduleID string) ([]dto.QuizHistoryResponse, error) {
	args := m.Called(ctx, userID, moduleID)
	if args.Get(0) != nil {
		return args.Get(0).([]dto.QuizHistoryResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuizService) GetResult(ctx context.Context, userID string, quizID string) (*dto.QuizResultResponse, error) {
	args := m.Called(ctx, userID, quizID)
	if args.Get(0) != nil {
		return args.Get(0).(*dto.QuizResultResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuizService) Retry(ctx context.Context, userID string, quizID string) (*dto.QuizResponse, error) {
	args := m.Called(ctx, userID, quizID)
	if args.Get(0) != nil {
		return args.Get(0).(*dto.QuizResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Setup ---

func setupApp(h *handler.QuizHandler) *fiber.App {
	app := fiber.New()
	
	// Mock middleware auth
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "user1")
		return c.Next()
	})

	quiz := app.Group("/quiz")
	quiz.Post("/", h.Generate)
	quiz.Post("/:id/submit", h.Submit)
	quiz.Post("/:id/retry", h.Retry)
	quiz.Get("/history", h.GetHistory)
	quiz.Get("/history/module/:moduleId", h.GetHistoryByModule)
	quiz.Get("/:id/result", h.GetResult)

	return app
}

// --- Tests ---

func TestQuizHandler_Generate(t *testing.T) {
	t.Run("Body invalid", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		req := httptest.NewRequest(http.MethodPost, "/quiz", nil) // Body nil
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("num_questions bukan 5/10/20", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		body, _ := json.Marshal(dto.GenerateQuizRequest{ModuleID: "some-uuid", NumQuestions: 7})
		req := httptest.NewRequest(http.MethodPost, "/quiz", bytes.NewReader(body))
		req.Header.Add("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Modul tidak ditemukan", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Generate", mock.Anything, "user1", mock.Anything).Return(nil, service.ErrModuleNotFound)

		body, _ := json.Marshal(dto.GenerateQuizRequest{ModuleID: "00000000-0000-0000-0000-000000000000", NumQuestions: 5})
		req := httptest.NewRequest(http.MethodPost, "/quiz", bytes.NewReader(body))
		req.Header.Add("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Modul belum disummarisasi", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Generate", mock.Anything, "user1", mock.Anything).Return(nil, service.ErrModuleNotSummarized)

		body, _ := json.Marshal(dto.GenerateQuizRequest{ModuleID: "00000000-0000-0000-0000-000000000000", NumQuestions: 5})
		req := httptest.NewRequest(http.MethodPost, "/quiz", bytes.NewReader(body))
		req.Header.Add("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Generate", mock.Anything, "user1", mock.Anything).Return(&dto.QuizResponse{ID: "q1"}, nil)

		body, _ := json.Marshal(dto.GenerateQuizRequest{ModuleID: "00000000-0000-0000-0000-000000000000", NumQuestions: 5})
		req := httptest.NewRequest(http.MethodPost, "/quiz", bytes.NewReader(body))
		req.Header.Add("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}

func TestQuizHandler_Submit(t *testing.T) {
	t.Run("Jawaban tidak lengkap (validation error)", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		body, _ := json.Marshal(dto.SubmitAnswerRequest{Answers: []dto.AnswerItem{}}) // min=1
		req := httptest.NewRequest(http.MethodPost, "/quiz/q1/submit", bytes.NewReader(body))
		req.Header.Add("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Quiz sudah dikerjakan", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Submit", mock.Anything, "user1", "q1", mock.Anything).Return(nil, service.ErrQuizAlreadyDone)

		body, _ := json.Marshal(dto.SubmitAnswerRequest{Answers: []dto.AnswerItem{{QuestionID: "00000000-0000-0000-0000-000000000000", Answer: "A"}}})
		req := httptest.NewRequest(http.MethodPost, "/quiz/q1/submit", bytes.NewReader(body))
		req.Header.Add("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Submit", mock.Anything, "user1", "q1", mock.Anything).Return(&dto.QuizResultResponse{ID: "q1", Score: 100}, nil)

		body, _ := json.Marshal(dto.SubmitAnswerRequest{Answers: []dto.AnswerItem{{QuestionID: "00000000-0000-0000-0000-000000000000", Answer: "A"}}})
		req := httptest.NewRequest(http.MethodPost, "/quiz/q1/submit", bytes.NewReader(body))
		req.Header.Add("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestQuizHandler_Retry(t *testing.T) {
	t.Run("Quiz tidak ditemukan", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Retry", mock.Anything, "user1", "invalid-id").Return(nil, service.ErrQuizNotFound)

		req := httptest.NewRequest(http.MethodPost, "/quiz/invalid-id/retry", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Retry", mock.Anything, "user1", "q1").Return(&dto.QuizResponse{ID: "q2"}, nil)

		req := httptest.NewRequest(http.MethodPost, "/quiz/q1/retry", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}

func TestQuizHandler_GetHistory(t *testing.T) {
	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("GetHistory", mock.Anything, "user1").Return([]dto.QuizHistoryResponse{}, nil)

		req := httptest.NewRequest(http.MethodGet, "/quiz/history", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestQuizHandler_GetHistoryByModule(t *testing.T) {
	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("GetHistoryByModule", mock.Anything, "user1", "mod1").Return([]dto.QuizHistoryResponse{}, nil)

		req := httptest.NewRequest(http.MethodGet, "/quiz/history/module/mod1", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestQuizHandler_GetResult(t *testing.T) {
	t.Run("Quiz tidak ditemukan", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("GetResult", mock.Anything, "user1", "invalid-id").Return(nil, service.ErrQuizNotFound)

		req := httptest.NewRequest(http.MethodGet, "/quiz/invalid-id/result", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockQuizService)
		h := handler.NewQuizHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("GetResult", mock.Anything, "user1", "q1").Return(&dto.QuizResultResponse{ID: "q1"}, nil)

		req := httptest.NewRequest(http.MethodGet, "/quiz/q1/result", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
