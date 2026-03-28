package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
)

type LynkRepository interface {
	CreateTransaction(ctx context.Context, tx *domain.LynkTransaction) error
	FindByTransactionID(ctx context.Context, transactionID string) (*domain.LynkTransaction, error)
	ProcessInTransaction(ctx context.Context, payload dto.LynkWebhookPayload, quizQuota int, summarizeQuota int) error
}

type lynkRepository struct {
	db *gorm.DB
}

func NewLynkRepository() LynkRepository {
	return &lynkRepository{db: config.DB}
}

func NewTestLynkRepository(db *gorm.DB) LynkRepository {
	return &lynkRepository{db: db}
}

func (r *lynkRepository) CreateTransaction(ctx context.Context, tx *domain.LynkTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *lynkRepository) FindByTransactionID(ctx context.Context, transactionID string) (*domain.LynkTransaction, error) {
	var tx domain.LynkTransaction
	err := r.db.WithContext(ctx).Where("transaction_id = ?", transactionID).First(&tx).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &tx, err
}

func (r *lynkRepository) ProcessInTransaction(ctx context.Context, payload dto.LynkWebhookPayload, quizQuota int, summarizeQuota int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// a. Simpan transaksi dulu (jika gagal, seluruh block dibatalkan)
		lynkTx := &domain.LynkTransaction{
			TransactionID: payload.TransactionID,
			Email:         payload.Email,
			ProductName:   payload.ProductName,
			Amount:        payload.Amount,
			Status:        payload.Status,
		}
		if err := tx.Create(lynkTx).Error; err != nil {
			return fmt.Errorf("gagal simpan transaksi: %w", err)
		}

		if quizQuota == 0 && summarizeQuota == 0 {
			// Produk tidak dikenal, skip quota update tapi transaksi tetap tersimpan
			return nil
		}

		// b. Cek user ada
		var user domain.User
		if err := tx.Where("email = ?", payload.Email).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("[lynk_repo] user not found for email: %s, skipping quota update", payload.Email)
				return nil
			}
			return fmt.Errorf("gagal cari user: %w", err)
		}

		// c. Update quota (atomic)
		if err := tx.Model(&domain.User{}).
			Where("email = ?", payload.Email).
			Updates(map[string]any{
				"quiz_quota":      gorm.Expr("quiz_quota + ?", quizQuota),
				"summarize_quota": gorm.Expr("summarize_quota + ?", summarizeQuota),
			}).Error; err != nil {
			return fmt.Errorf("gagal update quota: %w", err)
		}

		// d. Update role ke member jika masih guest
		if user.Role == domain.RoleGuest {
			if err := tx.Model(&domain.User{}).
				Where("email = ?", payload.Email).
				Update("role", domain.RoleMember).Error; err != nil {
				return fmt.Errorf("gagal update role: %w", err)
			}
		}

		return nil
	})
}
