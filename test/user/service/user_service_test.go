package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
)

// MockUserRepository is a mock of the UserRepository interface
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) UpdateOTP(ctx context.Context, userID string, otp string) error {
	args := m.Called(ctx, userID, otp)
	return args.Error(0)
}

func (m *MockUserRepository) VerifyUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateEmail(ctx context.Context, userID string, newEmail string) error {
	args := m.Called(ctx, userID, newEmail)
	return args.Error(0)
}

func TestUserService_Register_EmailExists(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := service.NewUserService(mockRepo)

	req := dto.RegisterRequest{
		Email: "existing@example.com",
	}

	mockRepo.On("FindByEmail", mock.Anything, req.Email).Return(&domain.User{Email: req.Email}, nil)

	err := svc.Register(context.Background(), req)

	assert.Error(t, err)
	assert.Equal(t, service.ErrEmailAlreadyExists, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_VerifyOTP_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := service.NewUserService(mockRepo)

	req := dto.VerifyOTPRequest{
		Email: "test@example.com",
		OTP:   "123456",
	}

	user := &domain.User{
		ID:    "user-1",
		Email: req.Email,
		OTP:   req.OTP,
	}

	mockRepo.On("FindByEmail", mock.Anything, req.Email).Return(user, nil)
	mockRepo.On("VerifyUser", mock.Anything, user.ID).Return(nil)

	err := svc.VerifyOTP(context.Background(), req)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
