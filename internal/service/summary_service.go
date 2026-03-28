package service

import (
	"context"
	"errors"
	"time"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/repository"
)

var (
	ErrSummaryNotReady = errors.New("summary belum tersedia, modul masih diproses")
)

type SummaryServiceContract interface {
	GetByModuleID(ctx context.Context, userID string, moduleID string) (*dto.SummaryResponse, error)
	UpdateManual(ctx context.Context, userID string, moduleID string, req dto.UpdateSummaryRequest) (*dto.SummaryResponse, error)
}

type SummaryService struct {
	moduleRepo repository.ModuleRepositoryContract
}

func NewSummaryService(moduleRepo repository.ModuleRepositoryContract) *SummaryService {
	return &SummaryService{moduleRepo: moduleRepo}
}

// GetByModuleID — ambil summary dari modul
func (s *SummaryService) GetByModuleID(ctx context.Context, userID string, moduleID string) (*dto.SummaryResponse, error) {
	module, err := s.moduleRepo.FindByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, ErrModuleNotFound
	}
	if module.UserID != userID {
		return nil, ErrNotModuleOwner
	}
	if !module.IsSummarized || module.Summary == "" {
		return nil, ErrSummaryNotReady
	}

	return toSummaryResponse(module), nil
}

// UpdateManual — user edit summary secara manual
func (s *SummaryService) UpdateManual(ctx context.Context, userID string, moduleID string, req dto.UpdateSummaryRequest) (*dto.SummaryResponse, error) {
	module, err := s.moduleRepo.FindByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, ErrModuleNotFound
	}
	if module.UserID != userID {
		return nil, ErrNotModuleOwner
	}

	if err := s.moduleRepo.UpdateSummaryManual(ctx, moduleID, req.Summary); err != nil {
		return nil, err
	}

	// Update local untuk response
	module.Summary = req.Summary
	module.UpdatedAt = time.Now()

	return toSummaryResponse(module), nil
}

func toSummaryResponse(module *domain.Module) *dto.SummaryResponse {
	return &dto.SummaryResponse{
		ModuleID:     module.ID,
		ModuleTitle:  module.Title,
		Summary:      module.Summary,
		IsSummarized:    module.IsSummarized,
		SummarizeFailed: module.SummarizeFailed,
		UpdatedAt:       module.UpdatedAt.Format(time.RFC3339),
	}
}
