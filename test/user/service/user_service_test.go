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

func (m *MockUserRepository) UpdateQuotaAndRole(ctx context.Context, email string, quizQuota int, summarizeQuota int) error {
	args := m.Called(ctx, email, quizQuota, summarizeQuota)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateRole(ctx context.Context, email string, role domain.Role) error {
	args := m.Called(ctx, email, role)
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

func TestUserService_GetMe_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := service.NewUserService(mockRepo)

	userID := "user-123"
	user := &domain.User{
		ID:         userID,
		Name:       "Test User",
		Email:      "test@example.com",
		Role:       domain.RoleGuest,
		IsVerified: true,
	}

	mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

	res, err := svc.GetMe(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, user.ID, res.ID)
	assert.Equal(t, user.Name, res.Name)
	assert.Equal(t, user.Email, res.Email)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetMe_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := service.NewUserService(mockRepo)

	userID := "non-existent"
	mockRepo.On("FindByID", mock.Anything, userID).Return(nil, nil)

	res, err := svc.GetMe(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, service.ErrUserNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_ResendOTP_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := service.NewUserService(mockRepo)

	req := dto.ResendOTPRequest{
		Email: "unverified@example.com",
	}

	user := &domain.User{
		ID:         "user-1",
		Email:      req.Email,
		IsVerified: false,
	}

	mockRepo.On("FindByEmail", mock.Anything, req.Email).Return(user, nil)
	mockRepo.On("UpdateOTP", mock.Anything, user.ID, mock.Anything).Return(nil)

	err := svc.ResendOTP(context.Background(), req)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_ResendOTP_AlreadyVerified(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := service.NewUserService(mockRepo)

	req := dto.ResendOTPRequest{
		Email: "verified@example.com",
	}

	user := &domain.User{
		ID:         "user-1",
		Email:      req.Email,
		IsVerified: true,
	}

	mockRepo.On("FindByEmail", mock.Anything, req.Email).Return(user, nil)

	err := svc.ResendOTP(context.Background(), req)

	assert.Error(t, err)
	assert.Equal(t, service.ErrAlreadyVerified, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_ResendOTP_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := service.NewUserService(mockRepo)

	req := dto.ResendOTPRequest{
		Email: "unknown@example.com",
	}

	mockRepo.On("FindByEmail", mock.Anything, req.Email).Return(nil, nil)

	err := svc.ResendOTP(context.Background(), req)

	assert.Error(t, err)
	assert.Equal(t, service.ErrUserNotFound, err)
	mockRepo.AssertExpectations(t)
}
