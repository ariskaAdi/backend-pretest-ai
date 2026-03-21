package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/domain"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{db: config.DB}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) UpdateOTP(ctx context.Context, userID string, otp string) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Update("otp", otp).Error
}

func (r *UserRepository) VerifyUser(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"is_verified": true,
			"otp":         "",
		}).Error
}

func (r *UserRepository) UpdateEmail(ctx context.Context, userID string, newEmail string) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"email":       newEmail,
			"is_verified": true,
			"otp":         "",
		}).Error
}
