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

type MockSummaryService struct {
	mock.Mock
}

func (m *MockSummaryService) GetByModuleID(ctx context.Context, userID string, moduleID string) (*dto.SummaryResponse, error) {
	args := m.Called(ctx, userID, moduleID)
	if args.Get(0) != nil {
		return args.Get(0).(*dto.SummaryResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSummaryService) UpdateManual(ctx context.Context, userID string, moduleID string, req dto.UpdateSummaryRequest) (*dto.SummaryResponse, error) {
	args := m.Called(ctx, userID, moduleID, req)
	if args.Get(0) != nil {
		return args.Get(0).(*dto.SummaryResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Helpers ---

func setupSummaryApp(h *handler.SummaryHandler) *fiber.App {
	app := fiber.New()
	
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "user1")
		return c.Next()
	})

	summary := app.Group("/summary")
	summary.Get("/:moduleId", h.GetByModuleID)
	summary.Put("/:moduleId", h.UpdateManual)

	return app
}

// --- Tests ---

func TestSummaryHandler_GetByModuleID(t *testing.T) {
	t.Run("Modul tidak ditemukan (404)", func(t *testing.T) {
		mockSvc := new(MockSummaryService)
		h := handler.NewSummaryHandler(mockSvc)
		app := setupSummaryApp(h)

		mockSvc.On("GetByModuleID", mock.Anything, "user1", "mod-99").Return(nil, service.ErrModuleNotFound)

		req := httptest.NewRequest(http.MethodGet, "/summary/mod-99", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Bukan pemilik (401)", func(t *testing.T) {
		mockSvc := new(MockSummaryService)
		h := handler.NewSummaryHandler(mockSvc)
		app := setupSummaryApp(h)

		mockSvc.On("GetByModuleID", mock.Anything, "user1", "mod-99").Return(nil, service.ErrNotModuleOwner)

		req := httptest.NewRequest(http.MethodGet, "/summary/mod-99", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Summary belum siap (400)", func(t *testing.T) {
		mockSvc := new(MockSummaryService)
		h := handler.NewSummaryHandler(mockSvc)
		app := setupSummaryApp(h)

		mockSvc.On("GetByModuleID", mock.Anything, "user1", "mod-99").Return(nil, service.ErrSummaryNotReady)

		req := httptest.NewRequest(http.MethodGet, "/summary/mod-99", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Sukses (200)", func(t *testing.T) {
		mockSvc := new(MockSummaryService)
		h := handler.NewSummaryHandler(mockSvc)
		app := setupSummaryApp(h)

		res := &dto.SummaryResponse{ModuleID: "mod-1", Summary: "Tests"}
		mockSvc.On("GetByModuleID", mock.Anything, "user1", "mod-1").Return(res, nil)

		req := httptest.NewRequest(http.MethodGet, "/summary/mod-1", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestSummaryHandler_UpdateManual(t *testing.T) {
	t.Run("Body kosong (400)", func(t *testing.T) {
		mockSvc := new(MockSummaryService)
		h := handler.NewSummaryHandler(mockSvc)
		app := setupSummaryApp(h)

		req := httptest.NewRequest(http.MethodPut, "/summary/mod-1", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Summary terlalu pendek (400)", func(t *testing.T) {
		mockSvc := new(MockSummaryService)
		h := handler.NewSummaryHandler(mockSvc)
		app := setupSummaryApp(h)

		data := dto.UpdateSummaryRequest{Summary: "short"}
		body, _ := json.Marshal(data)
		req := httptest.NewRequest(http.MethodPut, "/summary/mod-1", bytes.NewReader(body))
		req.Header.Add("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Sukses (200)", func(t *testing.T) {
		mockSvc := new(MockSummaryService)
		h := handler.NewSummaryHandler(mockSvc)
		app := setupSummaryApp(h)

		data := dto.UpdateSummaryRequest{Summary: "This is a long enough summary."}
		body, _ := json.Marshal(data)
		
		res := &dto.SummaryResponse{ModuleID: "mod-1", Summary: data.Summary}
		mockSvc.On("UpdateManual", mock.Anything, "user1", "mod-1", data).Return(res, nil)

		req := httptest.NewRequest(http.MethodPut, "/summary/mod-1", bytes.NewReader(body))
		req.Header.Add("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
