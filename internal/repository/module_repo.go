package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/domain"
)

type ModuleRepositoryContract interface {
	Create(ctx context.Context, module *domain.Module) error
	FindByID(ctx context.Context, id string) (*domain.Module, error)
	FindByUserID(ctx context.Context, userID string) ([]domain.Module, error)
	UpdateSummary(ctx context.Context, moduleID string, summary string) error
	UpdateSummaryManual(ctx context.Context, moduleID string, summary string) error
	UpdateSummarizeStatus(ctx context.Context, moduleID string, isSummarized bool, summarizeFailed bool) error
	MarkSummarizeFailed(ctx context.Context, moduleID string) error
	Delete(ctx context.Context, id string) error
}

type ModuleRepository struct {
	db *gorm.DB
}

func NewModuleRepository() *ModuleRepository {
	return &ModuleRepository{db: config.DB}
}

func (r *ModuleRepository) Create(ctx context.Context, module *domain.Module) error {
	return r.db.WithContext(ctx).Create(module).Error
}

func (r *ModuleRepository) FindByID(ctx context.Context, id string) (*domain.Module, error) {
	var module domain.Module
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&module).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &module, err
}

func (r *ModuleRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Module, error) {
	var modules []domain.Module
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&modules).Error
	return modules, err
}

func (r *ModuleRepository) UpdateSummary(ctx context.Context, moduleID string, summary string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Module{}).
		Where("id = ?", moduleID).
		Updates(map[string]any{
			"summary":       summary,
			"is_summarized": true,
		}).Error
}

// UpdateSummaryManual — dipanggil user, is_summarized tetap true
func (r *ModuleRepository) UpdateSummaryManual(ctx context.Context, moduleID string, summary string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Module{}).
		Where("id = ?", moduleID).
		Update("summary", summary).Error
}

func (r *ModuleRepository) MarkSummarizeFailed(ctx context.Context, moduleID string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Module{}).
		Where("id = ?", moduleID).
		Update("summarize_failed", true).Error
}

func (r *ModuleRepository) UpdateSummarizeStatus(ctx context.Context, moduleID string, isSummarized bool, summarizeFailed bool) error {
	return r.db.WithContext(ctx).
		Model(&domain.Module{}).
		Where("id = ?", moduleID).
		Updates(map[string]any{
			"is_summarized":    isSummarized,
			"summarize_failed": summarizeFailed,
		}).Error
}

func (r *ModuleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Module{}).Error
}
