package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/domain"
)

type QuizRepositoryContract interface {
	Create(ctx context.Context, quiz *domain.Quiz) error
	FindByID(ctx context.Context, id string) (*domain.Quiz, error)
	FindByUserID(ctx context.Context, userID string) ([]domain.Quiz, error)
	FindByUserIDAndModuleID(ctx context.Context, userID string, moduleID string) ([]domain.Quiz, error)
	SaveAnswersAndScore(ctx context.Context, quizID string, questions []domain.Question, score int) error
	UpdateStatus(ctx context.Context, quizID string, status domain.QuizStatus) error
	SaveExplanations(ctx context.Context, explanations map[string]string) error
}

type QuizRepository struct {
	db *gorm.DB
}

func NewQuizRepository() *QuizRepository {
	return &QuizRepository{db: config.DB}
}

func (r *QuizRepository) Create(ctx context.Context, quiz *domain.Quiz) error {
	return r.db.WithContext(ctx).Create(quiz).Error
}

func (r *QuizRepository) FindByID(ctx context.Context, id string) (*domain.Quiz, error) {
	var quiz domain.Quiz
	err := r.db.WithContext(ctx).
		Preload("Questions").
		Preload("Module").
		Where("id = ?", id).
		First(&quiz).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &quiz, err
}

func (r *QuizRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Quiz, error) {
	var quizzes []domain.Quiz
	err := r.db.WithContext(ctx).
		Preload("Module").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&quizzes).Error
	return quizzes, err
}

func (r *QuizRepository) FindByUserIDAndModuleID(ctx context.Context, userID string, moduleID string) ([]domain.Quiz, error) {
	var quizzes []domain.Quiz
	err := r.db.WithContext(ctx).
		Preload("Module").
		Where("user_id = ? AND module_id = ?", userID, moduleID).
		Order("created_at DESC").
		Find(&quizzes).Error
	return quizzes, err
}

func (r *QuizRepository) UpdateStatus(ctx context.Context, quizID string, status domain.QuizStatus) error {
	return r.db.WithContext(ctx).
		Model(&domain.Quiz{}).
		Where("id = ?", quizID).
		Update("status", status).Error
}

func (r *QuizRepository) SaveExplanations(ctx context.Context, explanations map[string]string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for questionID, explanation := range explanations {
			if err := tx.Model(&domain.Question{}).
				Where("id = ?", questionID).
				Update("explanation", explanation).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *QuizRepository) SaveAnswersAndScore(ctx context.Context, quizID string, questions []domain.Question, score int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update tiap jawaban user per soal
		for _, q := range questions {
			if err := tx.Model(&domain.Question{}).
				Where("id = ?", q.ID).
				Update("user_answer", q.UserAnswer).Error; err != nil {
				return err
			}
		}
		// Update score dan status quiz
		return tx.Model(&domain.Quiz{}).
			Where("id = ?", quizID).
			Updates(map[string]any{
				"score":  score,
				"status": domain.QuizStatusCompleted,
			}).Error
	})
}
