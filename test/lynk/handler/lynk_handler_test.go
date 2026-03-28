package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/handler"
	"backend-pretest-ai/internal/service"
)

// --- Mock ---

type MockLynkService struct {
	mock.Mock
}

func (m *MockLynkService) ProcessWebhook(ctx context.Context, payload dto.LynkWebhookPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

// --- Helpers ---

func setupApp(lynkSvc service.LynkService) *fiber.App {
	app := fiber.New()
	h := handler.NewLynkHandler(lynkSvc)
	app.Post("/webhook/lynk", h.HandleWebhook)
	return app
}

func makeRequest(app *fiber.App, secret string, payload any) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhook/lynk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}
	
	// Create a recorder
	rec := httptest.NewRecorder()
	
	// Use fiber.App's Test method directly
	resp, _ := app.Test(req)
	
	// Copy status code and body to recorder
	rec.Code = resp.StatusCode
	return rec
}

// --- Test Cases ---

func TestHandleWebhook_Success(t *testing.T) {
	config.Cfg = &config.Config{}
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	payload := dto.LynkWebhookPayload{
		Email: "user@test.com", ProductName: "Paket 4x",
		Amount: 10000, Status: "success", TransactionID: "tx-001",
	}
	lynkSvc.On("ProcessWebhook", mock.Anything, payload).Return(nil)

	rec := makeRequest(app, "test-secret", payload)
	assert.Equal(t, 200, rec.Code)
}

func TestHandleWebhook_InvalidSecret_Returns401(t *testing.T) {
	config.Cfg = &config.Config{}
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	rec := makeRequest(app, "wrong-secret", dto.LynkWebhookPayload{})
	assert.Equal(t, 401, rec.Code)
	lynkSvc.AssertNotCalled(t, "ProcessWebhook")
}

func TestHandleWebhook_NoSecret_Returns401(t *testing.T) {
	config.Cfg = &config.Config{}
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	rec := makeRequest(app, "", dto.LynkWebhookPayload{})
	assert.Equal(t, 401, rec.Code)
}

func TestHandleWebhook_DuplicateTransaction_Returns(t *testing.T) {
	config.Cfg = &config.Config{}
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	payload := dto.LynkWebhookPayload{TransactionID: "tx-dup", Status: "success"}
	lynkSvc.On("ProcessWebhook", mock.Anything, payload).Return(service.ErrTransactionAlreadyProcessed)

	rec := makeRequest(app, "test-secret", payload)
	// Harus tetap 200 agar Lynk tidak retry
	assert.Equal(t, 200, rec.Code)
}

func TestHandleWebhook_InvalidBody_Returns400(t *testing.T) {
	config.Cfg = &config.Config{}
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	req := httptest.NewRequest("POST", "/webhook/lynk", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", "test-secret")
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}
