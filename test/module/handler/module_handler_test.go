package handler_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/handler"
	"backend-pretest-ai/internal/service"
)

// --- Mocks ---

type MockModuleService struct {
	mock.Mock
}

func (m *MockModuleService) Upload(ctx context.Context, userID string, fileHeader *multipart.FileHeader, req dto.UploadModuleRequest) (*dto.ModuleResponse, error) {
	args := m.Called(ctx, userID, fileHeader, req)
	if args.Get(0) != nil {
		return args.Get(0).(*dto.ModuleResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockModuleService) GetAll(ctx context.Context, userID string) ([]dto.ModuleResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]dto.ModuleResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockModuleService) GetByID(ctx context.Context, userID string, moduleID string) (*dto.ModuleDetailResponse, error) {
	args := m.Called(ctx, userID, moduleID)
	if args.Get(0) != nil {
		return args.Get(0).(*dto.ModuleDetailResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockModuleService) Delete(ctx context.Context, userID string, moduleID string) error {
	args := m.Called(ctx, userID, moduleID)
	return args.Error(0)
}

func (m *MockModuleService) RetrySummarize(ctx context.Context, userID string, moduleID string) error {
	args := m.Called(ctx, userID, moduleID)
	return args.Error(0)
}

// --- Helpers ---

func setupApp(h *handler.ModuleHandler) *fiber.App {
	app := fiber.New()
	
	// Mock middleware auth
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "user1")
		return c.Next()
	})

	modules := app.Group("/modules")
	modules.Post("/", h.Upload)
	modules.Get("/", h.GetAll)
	modules.Get("/:id", h.GetByID)
	modules.Delete("/:id", h.Delete)

	return app
}

func createMultipartRequest(url string, title string, filename string, content []byte) *http.Request {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	if title != "" {
		_ = writer.WriteField("title", title)
	}

	if filename != "" {
		part, _ := writer.CreateFormFile("file", filename)
		_, _ = part.Write(content)
	}
	
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	return req
}

// --- Tests ---

func TestModuleHandler_Upload(t *testing.T) {
	t.Run("Tidak ada file", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		req := createMultipartRequest("/modules", "Valid Title", "", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Title kosong", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		req := createMultipartRequest("/modules", "", "test.pdf", []byte("pdf"))
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("File bukan PDF", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Upload", mock.Anything, "user1", mock.Anything, mock.Anything).
			Return(nil, service.ErrInvalidFileType)

		req := createMultipartRequest("/modules", "Valid Title", "test.txt", []byte("txt"))
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		modResp := &dto.ModuleResponse{
			ID:              "mod-1",
			Title:           "Valid Title",
			FileURL:         "http://test.com/file.pdf",
			IsSummarized:    false,
			SummarizeFailed: false,
			CreatedAt:       time.Now().Format(time.RFC3339),
		}

		mockSvc.On("Upload", mock.Anything, "user1", mock.Anything, mock.MatchedBy(func(req dto.UploadModuleRequest) bool {
			return req.Title == "Valid Title"
		})).Return(modResp, nil)

		req := createMultipartRequest("/modules", "Valid Title", "test.pdf", []byte("pdf"))
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		mockSvc.AssertExpectations(t)
	})
}

func TestModuleHandler_GetAll(t *testing.T) {
	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		listResp := []dto.ModuleResponse{
			{ID: "1", Title: "Mod 1"},
			{ID: "2", Title: "Mod 2"},
		}

		mockSvc.On("GetAll", mock.Anything, "user1").Return(listResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/modules", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockSvc.AssertExpectations(t)
	})
}

func TestModuleHandler_GetByID(t *testing.T) {
	t.Run("Tidak ditemukan", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("GetByID", mock.Anything, "user1", "mod-99").Return(nil, service.ErrModuleNotFound)

		req := httptest.NewRequest(http.MethodGet, "/modules/mod-99", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Bukan pemilik", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("GetByID", mock.Anything, "user1", "mod-99").Return(nil, service.ErrNotModuleOwner)

		req := httptest.NewRequest(http.MethodGet, "/modules/mod-99", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		detailResp := &dto.ModuleDetailResponse{ID: "mod-99", Title: "Mod 99"}

		mockSvc.On("GetByID", mock.Anything, "user1", "mod-99").Return(detailResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/modules/mod-99", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestModuleHandler_Delete(t *testing.T) {
	t.Run("Tidak ditemukan", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Delete", mock.Anything, "user1", "mod-99").Return(service.ErrModuleNotFound)

		req := httptest.NewRequest(http.MethodDelete, "/modules/mod-99", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Sukses", func(t *testing.T) {
		mockSvc := new(MockModuleService)
		h := handler.NewModuleHandler(mockSvc)
		app := setupApp(h)

		mockSvc.On("Delete", mock.Anything, "user1", "mod-99").Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/modules/mod-99", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
