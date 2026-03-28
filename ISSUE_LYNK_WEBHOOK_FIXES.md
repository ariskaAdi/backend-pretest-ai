# ISSUE: Lynk Webhook — Bug Fix & Missing Tests

## Status
`open`

## Priority
`high`

## Assignee
_unassigned_

---

## Background

Hasil review implementasi Lynk webhook (`POST /api/v1/webhook/lynk`) menemukan:

1. **1 bug kritis** — race condition yang bisa menyebabkan quota double-added
2. **1 bug minor** — product name matching case-sensitive, mudah miss
3. **1 gap dokumentasi** — `doc/webhook/lynk.md` kurang detail
4. **1 missing** — tidak ada unit test sama sekali untuk lynk

Semua fitur lain sudah punya test (user, quiz, summary, module), lynk belum.

---

## Task 1 — Fix Bug Kritis: Race Condition di `ProcessWebhook`

### File: `internal/service/lynk_service.go`

### Masalah

Urutan operasi saat ini:

```
1. FindByTransactionID  ← cek duplikat
2. UpdateQuotaAndRole   ← quota bertambah di DB
3. UpdateRole
4. CreateTransaction    ← ⚠️ jika ini GAGAL...
```

Jika `CreateTransaction` (langkah 4) gagal karena DB error atau network issue, quota **sudah bertambah** di step 2 tapi tidak ada record di tabel `lynk_transactions`. Saat Lynk **retry** webhook yang sama, `FindByTransactionID` tidak menemukan record → quota **ditambah lagi** (double-added).

### Fix

Wrap seluruh operasi dalam satu **database transaction** agar semua step berhasil atau semua dibatalkan (atomic).

Ganti implementasi `ProcessWebhook` menjadi:

```go
func (s *lynkService) ProcessWebhook(ctx context.Context, payload dto.LynkWebhookPayload) error {
	// 1. Validasi status
	if payload.Status != "success" {
		log.Printf("[lynk_service] skipping webhook with status: %s, transaction_id: %s", payload.Status, payload.TransactionID)
		return nil
	}

	// 2. Idempotency check (di luar transaction agar cepat)
	existing, err := s.lynkRepo.FindByTransactionID(ctx, payload.TransactionID)
	if err != nil {
		return fmt.Errorf("gagal cek idempotency: %w", err)
	}
	if existing != nil {
		return ErrTransactionAlreadyProcessed
	}

	// 3. Mapping produk → quota
	quizQuota, summarizeQuota := mapProductToQuota(payload.ProductName)
	if quizQuota == 0 && summarizeQuota == 0 {
		log.Printf("[lynk_service] unknown product: %s, skipping quota update", payload.ProductName)
	}

	// 4. Jalankan semua DB operation dalam satu transaction
	if err := s.lynkRepo.ProcessInTransaction(ctx, payload, quizQuota, summarizeQuota); err != nil {
		return fmt.Errorf("gagal proses webhook: %w", err)
	}

	log.Printf("[lynk_service] success process webhook for %s, product: %s, tx_id: %s",
		payload.Email, payload.ProductName, payload.TransactionID)
	return nil
}
```

Tambahkan method `ProcessInTransaction` ke `LynkRepository` interface di `internal/repository/lynk_repo.go`:

```go
type LynkRepository interface {
	CreateTransaction(ctx context.Context, tx *domain.LynkTransaction) error
	FindByTransactionID(ctx context.Context, transactionID string) (*domain.LynkTransaction, error)
	ProcessInTransaction(ctx context.Context, payload dto.LynkWebhookPayload, quizQuota int, summarizeQuota int) error // ← TAMBAH
}
```

Implementasi `ProcessInTransaction`:

```go
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
```

> ⚠️ Setelah menambah `ProcessInTransaction`, method `ProcessWebhook` di `lynk_service.go` tidak lagi memanggil `userRepo` secara langsung untuk bagian ini. Pastikan import `userRepo` di service masih digunakan — jika tidak, hapus dari struct untuk menghindari confusion.

---

## Task 2 — Fix Bug Minor: `mapProductToQuota` Case-Sensitive

### File: `internal/service/lynk_service.go`

### Masalah

```go
func mapProductToQuota(productName string) (int, int) {
	switch productName {
	case "Paket 4x":   // ← exact match only
		return 4, 4
	case "Paket 10x":  // ← exact match only
		return 10, 10
	default:
		return 0, 0
	}
}
```

Jika Lynk mengirim `"paket 4x"`, `"PAKET 4X"`, atau ada spasi ekstra, seluruh quota update di-skip tanpa log error yang jelas.

### Fix

Gunakan `strings.EqualFold` untuk case-insensitive matching dan trim spasi:

```go
import "strings"

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
```

---

## Task 3 — Lengkapi `doc/webhook/lynk.md`

### File: `doc/webhook/lynk.md`

Dokumentasi saat ini terlalu singkat. Tambahkan section berikut:

### 3a. Response Format

```markdown
## Response

Semua response menggunakan format standar `APIResponse`.

### Success
**Status:** `200 OK`
\```json
{
  "status": "success",
  "message": "webhook processed successfully",
  "data": null
}
\```

### Already Processed (idempotent)
**Status:** `200 OK`
\```json
{
  "status": "success",
  "message": "transaksi sudah diproses sebelumnya",
  "data": null
}
\```

### Invalid Secret
**Status:** `401 Unauthorized`
\```json
{
  "status": "error",
  "message": "invalid webhook secret",
  "data": null
}
\```

### Bad Request
**Status:** `400 Bad Request`
\```json
{
  "status": "error",
  "message": "format request tidak valid",
  "data": null
}
\```
```

### 3b. Environment Setup

```markdown
## Environment Setup

Tambahkan ke file `.env`:
\```
LYNK_WEBHOOK_SECRET=isi_dengan_secret_dari_lynk_dashboard
\```

Cara mendapatkan secret:
1. Login ke Lynk.id dashboard
2. Buka menu Webhook Settings
3. Copy secret key dan paste ke `.env`
```

### 3c. Contoh Payload Non-Success

```markdown
## Contoh Payload Gagal (akan di-skip, tidak update quota)

\```json
{
  "email": "user@email.com",
  "product_name": "Paket 4x",
  "amount": 10000,
  "status": "failed",
  "transaction_id": "abc789"
}
\```

> Payload dengan `status` selain `"success"` akan diterima dengan `200 OK` tapi tidak memproses quota.
```

### 3d. Testing Lokal

```markdown
## Testing Lokal

1. Jalankan server lokal
2. Expose via ngrok: `ngrok http 8080`
3. Set webhook URL di Lynk dashboard ke: `https://<ngrok-url>/api/v1/webhook/lynk`
4. Set `X-Webhook-Secret` di Lynk dashboard sama dengan `LYNK_WEBHOOK_SECRET` di `.env`
5. Trigger test payment dari Lynk dashboard
```

---

## Task 4 — Buat Unit Test untuk Lynk

### Struktur folder baru

```
test/
└── lynk/
    ├── handler/
    │   └── lynk_handler_test.go
    └── service/
        └── lynk_service_test.go
```

---

### File: `test/lynk/service/lynk_service_test.go`

Ikuti pola yang sama dengan `test/quiz/service/quiz_service_test.go`. Gunakan mock untuk `LynkRepository` dan `UserRepository`.

```go
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

// implementasi method UserRepository yang dibutuhkan service...
// (salin dari test lain yang sudah ada MockUserRepository)

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
```

---

### File: `test/lynk/handler/lynk_handler_test.go`

Ikuti pola yang sama dengan `test/quiz/handler/quiz_handler_test.go`.

```go
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"backend-pretest-ai/config"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/handler"
	"backend-pretest-ai/internal/service"
)

// --- Mock ---

type MockLynkService struct {
	mock.Mock
}

func (m *MockLynkService) ProcessWebhook(ctx context.Context, payload dto.LynkWebhookPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

// --- Helpers ---

func setupApp(lynkSvc service.LynkService) *fiber.App {
	app := fiber.New()
	h := handler.NewLynkHandler(lynkSvc)
	app.Post("/webhook/lynk", h.HandleWebhook)
	return app
}

func makeRequest(app *fiber.App, secret string, payload any) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhook/lynk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}
	resp, _ := app.Test(req)
	rec := httptest.NewRecorder()
	rec.WriteHeader(resp.StatusCode)
	return rec
}

// --- Test Cases ---

func TestHandleWebhook_Success(t *testing.T) {
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	payload := dto.LynkWebhookPayload{
		Email: "user@test.com", ProductName: "Paket 4x",
		Amount: 10000, Status: "success", TransactionID: "tx-001",
	}
	lynkSvc.On("ProcessWebhook", mock.Anything, payload).Return(nil)

	rec := makeRequest(app, "test-secret", payload)
	assert.Equal(t, 200, rec.Code)
}

func TestHandleWebhook_InvalidSecret_Returns401(t *testing.T) {
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	rec := makeRequest(app, "wrong-secret", dto.LynkWebhookPayload{})
	assert.Equal(t, 401, rec.Code)
	lynkSvc.AssertNotCalled(t, "ProcessWebhook")
}

func TestHandleWebhook_NoSecret_Returns401(t *testing.T) {
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	rec := makeRequest(app, "", dto.LynkWebhookPayload{})
	assert.Equal(t, 401, rec.Code)
}

func TestHandleWebhook_DuplicateTransaction_Returns200(t *testing.T) {
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	payload := dto.LynkWebhookPayload{TransactionID: "tx-dup", Status: "success"}
	lynkSvc.On("ProcessWebhook", mock.Anything, payload).Return(service.ErrTransactionAlreadyProcessed)

	rec := makeRequest(app, "test-secret", payload)
	// Harus tetap 200 agar Lynk tidak retry
	assert.Equal(t, 200, rec.Code)
}

func TestHandleWebhook_InvalidBody_Returns400(t *testing.T) {
	config.Cfg.App.LynkWebhookSecret = "test-secret"
	lynkSvc := new(MockLynkService)
	app := setupApp(lynkSvc)

	req := httptest.NewRequest("POST", "/webhook/lynk", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", "test-secret")
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}
```

---

## Ringkasan Perubahan Per File

| File | Perubahan |
|---|---|
| `internal/service/lynk_service.go` | Refactor `ProcessWebhook` — delegate DB ops ke `ProcessInTransaction`, fix case-sensitive product mapping |
| `internal/repository/lynk_repo.go` | Tambah `ProcessInTransaction` ke interface + implementasi pakai `db.Transaction()` |
| `doc/webhook/lynk.md` | Tambah response format, environment setup, contoh payload non-success, cara testing lokal |
| `test/lynk/service/lynk_service_test.go` | Buat baru — 6 test cases |
| `test/lynk/handler/lynk_handler_test.go` | Buat baru — 5 test cases |

---

## Definition of Done

- [ ] `ProcessWebhook` menggunakan DB transaction — quota dan `lynk_transactions` insert dalam satu atomic operation
- [ ] `mapProductToQuota` case-insensitive (pakai `strings.EqualFold`)
- [ ] `doc/webhook/lynk.md` memiliki response format, environment setup, dan cara testing lokal
- [ ] `test/lynk/service/lynk_service_test.go` ada dan semua test pass
- [ ] `test/lynk/handler/lynk_handler_test.go` ada dan semua test pass
- [ ] `go test ./test/lynk/...` tidak ada error
