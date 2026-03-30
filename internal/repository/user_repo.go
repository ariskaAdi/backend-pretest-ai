package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/domain"
)

// ErrQuotaInsufficient dikembalikan ketika quota user habis dan tidak bisa dikurangi lagi.
var ErrQuotaInsufficient = errors.New("quota tidak mencukupi")

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	UpdateOTP(ctx context.Context, userID string, otp string) error
	VerifyUser(ctx context.Context, userID string) error
	UpdateEmail(ctx context.Context, userID string, newEmail string) error
	UpdateQuotaAndRole(ctx context.Context, email string, quizQuota int, summarizeQuota int) error
	UpdateRole(ctx context.Context, email string, role domain.Role) error
	DeductQuizQuota(ctx context.Context, userID string) error
	DeductSummarizeQuota(ctx context.Context, userID string) error
	RestoreQuizQuota(ctx context.Context, userID string) error
	RestoreSummarizeQuota(ctx context.Context, userID string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository() UserRepository {
	return &userRepository{db: config.DB}
}

// NewTestUserRepository is for testing only
func NewTestUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *userRepository) UpdateOTP(ctx context.Context, userID string, otp string) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Update("otp", otp).Error
}

func (r *userRepository) VerifyUser(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"is_verified": true,
			"otp":         "",
		}).Error
}

func (r *userRepository) UpdateEmail(ctx context.Context, userID string, newEmail string) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"email":       newEmail,
			"is_verified": true,
			"otp":         "",
		}).Error
}

func (r *userRepository) UpdateQuotaAndRole(ctx context.Context, email string, quizQuota int, summarizeQuota int) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("email = ?", email).
		Updates(map[string]any{
			"quiz_quota":      gorm.Expr("quiz_quota + ?", quizQuota),
			"summarize_quota": gorm.Expr("summarize_quota + ?", summarizeQuota),
		}).Error
}

func (r *userRepository) UpdateRole(ctx context.Context, email string, role domain.Role) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("email = ?", email).
		Update("role", role).Error
}

func (r *userRepository) DeductQuizQuota(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ? AND quiz_quota > 0", userID).
		UpdateColumn("quiz_quota", gorm.Expr("quiz_quota - 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrQuotaInsufficient
	}
	return nil
}

func (r *userRepository) DeductSummarizeQuota(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ? AND summarize_quota > 0", userID).
		UpdateColumn("summarize_quota", gorm.Expr("summarize_quota - 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrQuotaInsufficient
	}
	return nil
}

func (r *userRepository) RestoreQuizQuota(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		UpdateColumn("quiz_quota", gorm.Expr("quiz_quota + 1")).Error
}

func (r *userRepository) RestoreSummarizeQuota(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		UpdateColumn("summarize_quota", gorm.Expr("summarize_quota + 1")).Error
}
