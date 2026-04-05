package repository

import (
	"context"

	"gorm.io/gorm"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/domain"
)

type ReviewRepository interface {
	Create(ctx context.Context, review *domain.Review) error
	GetAll(ctx context.Context) ([]domain.Review, error)
	GetByID(ctx context.Context, id string) (*domain.Review, error)
	ExistsByUserID(ctx context.Context, userID string) (bool, error)
	Update(ctx context.Context, review *domain.Review) error
	Delete(ctx context.Context, id string) error
}

type reviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository() ReviewRepository {
	return &reviewRepository{db: config.DB}
}

func (r *reviewRepository) Create(ctx context.Context, review *domain.Review) error {
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *reviewRepository) GetAll(ctx context.Context) ([]domain.Review, error) {
	var reviews []domain.Review
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

func (r *reviewRepository) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Review{}).Where("user_id = ?", userID).Count(&count).Error
	return count > 0, err
}

func (r *reviewRepository) GetByID(ctx context.Context, id string) (*domain.Review, error) {
	var review domain.Review
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

func (r *reviewRepository) Update(ctx context.Context, review *domain.Review) error {
	return r.db.WithContext(ctx).Save(review).Error
}

func (r *reviewRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Review{}).Error
}
