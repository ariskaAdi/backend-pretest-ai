package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockModuleRepository reuse dari module_service_test.go
type MockModuleRepository struct {
	mock.Mock
}

func (m *MockModuleRepository) Create(ctx context.Context, module *domain.Module) error {
	args := m.Called(ctx, module)
	return args.Error(0)
}

func (m *MockModuleRepository) FindByID(ctx context.Context, id string) (*domain.Module, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Module), args.Error(1)
}

func (m *MockModuleRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Module, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.Module), args.Error(1)
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

func (m *MockModuleRepository) MarkSummarizeFailed(ctx context.Context, moduleID string) error {
	args := m.Called(ctx, moduleID)
	return args.Error(0)
}

func (m *MockModuleRepository) UpdateSummarizeStatus(ctx context.Context, moduleID string, isSummarized bool, summarizeFailed bool) error {
	args := m.Called(ctx, moduleID, isSummarized, summarizeFailed)
	return args.Error(0)
}

func TestGetByModuleID(t *testing.T) {
	mockRepo := new(MockModuleRepository)
	srv := service.NewSummaryService(mockRepo)
	ctx := context.Background()
	userID := "user-123"
	moduleID := "mod-123"

	t.Run("Modul tidak ditemukan", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, moduleID).Return(nil, nil).Once()
		
		resp, err := srv.GetByModuleID(ctx, userID, moduleID)
		
		assert.ErrorIs(t, err, service.ErrModuleNotFound)
		assert.Nil(t, resp)
	})

	t.Run("Modul milik user lain", func(t *testing.T) {
		module := &domain.Module{ID: moduleID, UserID: "other-user"}
		mockRepo.On("FindByID", ctx, moduleID).Return(module, nil).Once()
		
		resp, err := srv.GetByModuleID(ctx, userID, moduleID)
		
		assert.ErrorIs(t, err, service.ErrNotModuleOwner)
		assert.Nil(t, resp)
	})

	t.Run("Summary belum siap (is_summarized=false)", func(t *testing.T) {
		module := &domain.Module{ID: moduleID, UserID: userID, IsSummarized: false}
		mockRepo.On("FindByID", ctx, moduleID).Return(module, nil).Once()
		
		resp, err := srv.GetByModuleID(ctx, userID, moduleID)
		
		assert.ErrorIs(t, err, service.ErrSummaryNotReady)
		assert.Nil(t, resp)
	})

	t.Run("Summary belum siap (summary=kosong)", func(t *testing.T) {
		module := &domain.Module{ID: moduleID, UserID: userID, IsSummarized: true, Summary: ""}
		mockRepo.On("FindByID", ctx, moduleID).Return(module, nil).Once()
		
		resp, err := srv.GetByModuleID(ctx, userID, moduleID)
		
		assert.ErrorIs(t, err, service.ErrSummaryNotReady)
		assert.Nil(t, resp)
	})

	t.Run("Sukses", func(t *testing.T) {
		module := &domain.Module{
			ID:           moduleID,
			UserID:       userID,
			Title:        "Matematika",
			Summary:      "Ringkasan MTK",
			IsSummarized: true,
			UpdatedAt:    time.Now(),
		}
		mockRepo.On("FindByID", ctx, moduleID).Return(module, nil).Once()
		
		resp, err := srv.GetByModuleID(ctx, userID, moduleID)
		
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Matematika", resp.ModuleTitle)
		assert.Equal(t, "Ringkasan MTK", resp.Summary)
		assert.True(t, resp.IsSummarized)
	})
}

func TestUpdateManual(t *testing.T) {
	mockRepo := new(MockModuleRepository)
	srv := service.NewSummaryService(mockRepo)
	ctx := context.Background()
	userID := "user-123"
	moduleID := "mod-123"
	req := dto.UpdateSummaryRequest{Summary: "Ringkasan Baru"}

	t.Run("Modul tidak ditemukan", func(t *testing.T) {
		mockRepo.On("FindByID", ctx, moduleID).Return(nil, nil).Once()
		
		resp, err := srv.UpdateManual(ctx, userID, moduleID, req)
		
		assert.ErrorIs(t, err, service.ErrModuleNotFound)
		assert.Nil(t, resp)
	})

	t.Run("Modul milik user lain", func(t *testing.T) {
		module := &domain.Module{ID: moduleID, UserID: "other-user"}
		mockRepo.On("FindByID", ctx, moduleID).Return(module, nil).Once()
		
		resp, err := srv.UpdateManual(ctx, userID, moduleID, req)
		
		assert.ErrorIs(t, err, service.ErrNotModuleOwner)
		assert.Nil(t, resp)
	})

	t.Run("DB error saat update", func(t *testing.T) {
		module := &domain.Module{ID: moduleID, UserID: userID}
		mockRepo.On("FindByID", ctx, moduleID).Return(module, nil).Once()
		mockRepo.On("UpdateSummaryManual", ctx, moduleID, req.Summary).Return(errors.New("db error")).Once()
		
		resp, err := srv.UpdateManual(ctx, userID, moduleID, req)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Sukses", func(t *testing.T) {
		module := &domain.Module{ID: moduleID, UserID: userID, Title: "IPAS", IsSummarized: true}
		mockRepo.On("FindByID", ctx, moduleID).Return(module, nil).Once()
		mockRepo.On("UpdateSummaryManual", ctx, moduleID, req.Summary).Return(nil).Once()
		
		resp, err := srv.UpdateManual(ctx, userID, moduleID, req)
		
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, req.Summary, resp.Summary)
		assert.True(t, resp.IsSummarized)
	})
}
