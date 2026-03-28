package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/repository"
)

var (
	ErrTransactionAlreadyProcessed = errors.New("transaksi sudah diproses")
)

type LynkService interface {
	ProcessWebhook(ctx context.Context, payload dto.LynkWebhookPayload) error
}

type lynkService struct {
	lynkRepo repository.LynkRepository
	// userRepo tidak lagi dipanggil langsung oleh service karena sudah didelegasikan ke lynkRepo.ProcessInTransaction
}

func NewLynkService(lynkRepo repository.LynkRepository, userRepo repository.UserRepository) LynkService {
	return &lynkService{
		lynkRepo: lynkRepo,
	}
}

func (s *lynkService) ProcessWebhook(ctx context.Context, payload dto.LynkWebhookPayload) error {
	// 1. Validasi status (hanya proses yang success)
	if payload.Status != "success" {
		log.Printf("[lynk_service] skipping webhook with status: %s, transaction_id: %s", payload.Status, payload.TransactionID)
		return nil
	}

	// 2. Idempotency check
	existing, err := s.lynkRepo.FindByTransactionID(ctx, payload.TransactionID)
	if err != nil {
		return fmt.Errorf("gagal cek idempotency: %w", err)
	}
	if existing != nil {
		return ErrTransactionAlreadyProcessed
	}

	// 3. Mapping produk -> quota
	quizQuota, summarizeQuota := mapProductToQuota(payload.ProductName)
	if quizQuota == 0 && summarizeQuota == 0 {
		log.Printf("[lynk_service] unknown product: %s, skipping quota update", payload.ProductName)
	}

	// 4. Jalankan semua DB operation dalam satu transaction
	if err := s.lynkRepo.ProcessInTransaction(ctx, payload, quizQuota, summarizeQuota); err != nil {
		return fmt.Errorf("gagal proses webhook: %w", err)
	}

	log.Printf("[lynk_service] success process webhook for %s, product: %s, tx_id: %s", payload.Email, payload.ProductName, payload.TransactionID)
	return nil
}

// mapProductToQuota merepresentasikan business logic untuk merubah nama produk menjadi jumlah quota
func mapProductToQuota(productName string) (int, int) {
	name := strings.TrimSpace(productName)
	switch {
	case strings.EqualFold(name, "Paket 4x"):
		return 4, 4
	case strings.EqualFold(name, "Paket 10x"):
		return 10, 10
	default:
		return 0, 0
	}
}
