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
)

// MockUserService is a mock of the UserService interface
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Register(ctx context.Context, req dto.RegisterRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserService) VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LoginResponse), args.Error(1)
}

func (m *MockUserService) RequestUpdateEmail(ctx context.Context, userID string, req dto.UpdateEmailRequest) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *MockUserService) VerifyUpdateEmail(ctx context.Context, userID string, req dto.VerifyUpdateEmailRequest) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func TestUserHandler_Register_Success(t *testing.T) {
	mockService := new(MockUserService)
	h := handler.NewUserHandler(mockService)

	app := fiber.New()
	app.Post("/register", h.Register)

	reqBody := dto.RegisterRequest{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	mockService.On("Register", mock.Anything, reqBody).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	mockService.AssertExpectations(t)
}

func TestUserHandler_Register_ValidationError(t *testing.T) {
	mockService := new(MockUserService)
	h := handler.NewUserHandler(mockService)

	app := fiber.New()
	app.Post("/register", h.Register)

	reqBody := dto.RegisterRequest{
		Name:     "", // Invalid
		Email:    "invalid-email",
		Password: "123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
