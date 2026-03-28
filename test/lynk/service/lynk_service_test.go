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

// --- Mocks ---

type MockLynkRepository struct {
	mock.Mock
}

func (m *MockLynkRepository) CreateTransaction(ctx context.Context, tx *domain.LynkTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockLynkRepository) FindByTransactionID(ctx context.Context, transactionID string) (*domain.LynkTransaction, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.LynkTransaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockLynkRepository) ProcessInTransaction(ctx context.Context, payload dto.LynkWebhookPayload, quizQuota int, summarizeQuota int) error {
	args := m.Called(ctx, payload, quizQuota, summarizeQuota)
	return args.Error(0)
}

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

// --- Test Cases ---

func TestProcessWebhook_Success(t *testing.T) {
	lynkRepo := new(MockLynkRepository)
	userRepo := new(MockUserRepository)
	svc := service.NewLynkService(lynkRepo, userRepo)

	payload := dto.LynkWebhookPayload{
		Email:         "user@test.com",
		ProductName:   "Paket 4x",
		Amount:        10000,
		Status:        "success",
		TransactionID: "tx-001",
	}

	lynkRepo.On("FindByTransactionID", mock.Anything, "tx-001").Return(nil, nil)
	lynkRepo.On("ProcessInTransaction", mock.Anything, payload, 4, 4).Return(nil)

	err := svc.ProcessWebhook(context.Background(), payload)
	assert.NoError(t, err)
	lynkRepo.AssertExpectations(t)
}

func TestProcessWebhook_StatusFailed_Skip(t *testing.T) {
	lynkRepo := new(MockLynkRepository)
	userRepo := new(MockUserRepository)
	svc := service.NewLynkService(lynkRepo, userRepo)

	payload := dto.LynkWebhookPayload{
		Status:        "failed",
		TransactionID: "tx-002",
	}

	err := svc.ProcessWebhook(context.Background(), payload)
	assert.NoError(t, err)
	// Tidak boleh ada call ke repo
	lynkRepo.AssertNotCalled(t, "FindByTransactionID")
	lynkRepo.AssertNotCalled(t, "ProcessInTransaction")
}

func TestProcessWebhook_DuplicateTransaction(t *testing.T) {
	lynkRepo := new(MockLynkRepository)
	userRepo := new(MockUserRepository)
	svc := service.NewLynkService(lynkRepo, userRepo)

	payload := dto.LynkWebhookPayload{
		Status:        "success",
		TransactionID: "tx-003",
	}

	existing := &domain.LynkTransaction{TransactionID: "tx-003"}
	lynkRepo.On("FindByTransactionID", mock.Anything, "tx-003").Return(existing, nil)

	err := svc.ProcessWebhook(context.Background(), payload)
	assert.ErrorIs(t, err, service.ErrTransactionAlreadyProcessed)
	lynkRepo.AssertNotCalled(t, "ProcessInTransaction")
}

func TestProcessWebhook_UnknownProduct_StillSavesTransaction(t *testing.T) {
	lynkRepo := new(MockLynkRepository)
	userRepo := new(MockUserRepository)
	svc := service.NewLynkService(lynkRepo, userRepo)

	payload := dto.LynkWebhookPayload{
		Email:         "user@test.com",
		ProductName:   "Produk Tidak Dikenal",
		Amount:        99000,
		Status:        "success",
		TransactionID: "tx-004",
	}

	lynkRepo.On("FindByTransactionID", mock.Anything, "tx-004").Return(nil, nil)
	// quota 0,0 karena produk tidak dikenal
	lynkRepo.On("ProcessInTransaction", mock.Anything, payload, 0, 0).Return(nil)

	err := svc.ProcessWebhook(context.Background(), payload)
	assert.NoError(t, err)
}

func TestProcessWebhook_Paket10x(t *testing.T) {
	lynkRepo := new(MockLynkRepository)
	userRepo := new(MockUserRepository)
	svc := service.NewLynkService(lynkRepo, userRepo)

	payload := dto.LynkWebhookPayload{
		Email:         "user@test.com",
		ProductName:   "Paket 10x",
		Amount:        20000,
		Status:        "success",
		TransactionID: "tx-005",
	}

	lynkRepo.On("FindByTransactionID", mock.Anything, "tx-005").Return(nil, nil)
	lynkRepo.On("ProcessInTransaction", mock.Anything, payload, 10, 10).Return(nil)

	err := svc.ProcessWebhook(context.Background(), payload)
	assert.NoError(t, err)
}

func TestProcessWebhook_CaseInsensitiveProductName(t *testing.T) {
	lynkRepo := new(MockLynkRepository)
	userRepo := new(MockUserRepository)
	svc := service.NewLynkService(lynkRepo, userRepo)

	payload := dto.LynkWebhookPayload{
		Email:         "user@test.com",
		ProductName:   "paket 4x", // lowercase
		Amount:        10000,
		Status:        "success",
		TransactionID: "tx-006",
	}

	lynkRepo.On("FindByTransactionID", mock.Anything, "tx-006").Return(nil, nil)
	lynkRepo.On("ProcessInTransaction", mock.Anything, payload, 4, 4).Return(nil)

	err := svc.ProcessWebhook(context.Background(), payload)
	assert.NoError(t, err)
}
