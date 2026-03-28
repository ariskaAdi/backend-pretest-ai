# ISSUE: Backend — Genkit Summarize Silent Failure

## Status
`open`

## Priority
`high`

## Assignee
_unassigned_

---

## Background

Ketika user upload PDF, `module_service.go` menjalankan summarize AI secara **async via goroutine**. Jika proses itu gagal (misalnya karena model AI deprecated, Genkit down, timeout, dll), error hanya di-`log.Printf` dan **tidak disimpan ke database**.

Akibatnya:
- Modul tetap bertahan di status `is_summarized: false` selamanya
- Frontend terus polling tiap 5 detik menunggu perubahan yang tidak akan pernah terjadi
- User melihat badge **"Sedang diproses..."** tanpa batas waktu
- Tidak ada cara bagi user untuk tahu bahwa prosesnya gagal
- Tidak ada cara untuk retry tanpa delete dan upload ulang

### Contoh Error yang Sudah Terjadi

```
2026/03/26 15:25:34 ERROR request end reqID=1 err="gagal generate summary:
Error 404, Message: models/gemini-1.5-flash is not found for API version v1beta,
or is not supported for generateContent."
```

Model `gemini-1.5-flash` deprecated oleh Google — **sudah diperbaiki** dengan mengganti ke `gemini-2.0-flash` di `genkit/main.go`. Namun masalah desain di bawah ini tetap ada dan **harus diselesaikan** agar kejadian serupa tidak menyebabkan stuck tanpa feedback.

---

## Akar Masalah

Lihat `internal/service/module_service.go` baris 123–133:

```go
// Trigger summarize ke Genkit — async, tidak block response
go func() {
    result, err := s.aiClient.Summarize(rawText)
    if err != nil {
        log.Printf("[module_service] gagal summarize modul %s: %v", module.ID, err)
        return  // ← error dibuang, DB tidak diupdate, user tidak tahu
    }
    if err := s.moduleRepo.UpdateSummary(context.Background(), module.ID, result.Summary); err != nil {
        log.Printf("[module_service] gagal simpan summary modul %s: %v", module.ID, err)
    }
}()
```

Tidak ada mekanisme untuk menyimpan status gagal ke DB.

---

## Yang Harus Dilakukan

### Task 1 — Tambah Field `SummarizeFailed` ke Domain

**File: `internal/domain/module.go`**

```go
type Module struct {
    ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    UserID         string    `gorm:"type:uuid;not null;index"`
    User           User      `gorm:"foreignKey:UserID"`
    Title          string    `gorm:"type:varchar(255);not null"`
    FileURL        string    `gorm:"type:varchar(500);not null"`
    RawText        string    `gorm:"type:text"`
    Summary        string    `gorm:"type:text"`
    IsSummarized   bool      `gorm:"default:false"`
    SummarizeFailed bool     `gorm:"default:false"` // ← TAMBAH INI
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

---

### Task 2 — Tambah Method Repo untuk Update Status Gagal

**File: `internal/repository/module_repo.go`**

Tambahkan ke `ModuleRepositoryContract` interface:

```go
MarkSummarizeFailed(ctx context.Context, moduleID string) error
```

Implementasi:

```go
func (r *ModuleRepository) MarkSummarizeFailed(ctx context.Context, moduleID string) error {
    return r.db.WithContext(ctx).
        Model(&domain.Module{}).
        Where("id = ?", moduleID).
        Update("summarize_failed", true).Error
}
```

---

### Task 3 — Update Goroutine di Module Service

**File: `internal/service/module_service.go`**

Ganti goroutine yang ada dengan yang menyimpan status gagal:

```go
go func() {
    result, err := s.aiClient.Summarize(rawText)
    if err != nil {
        log.Printf("[module_service] gagal summarize modul %s: %v", module.ID, err)
        // Simpan status gagal ke DB agar frontend bisa mendeteksi
        if dbErr := s.moduleRepo.MarkSummarizeFailed(context.Background(), module.ID); dbErr != nil {
            log.Printf("[module_service] gagal update status failed modul %s: %v", module.ID, dbErr)
        }
        return
    }
    if err := s.moduleRepo.UpdateSummary(context.Background(), module.ID, result.Summary); err != nil {
        log.Printf("[module_service] gagal simpan summary modul %s: %v", module.ID, err)
    }
    log.Printf("[module_service] summary modul %s selesai", module.ID)
}()
```

---

### Task 4 — Expose `summarize_failed` di DTO dan Response

**File: `internal/dto/module_dto.go`**

Tambahkan field ke `ModuleResponse` dan `ModuleDetailResponse`:

```go
type ModuleResponse struct {
    ID              string `json:"id"`
    Title           string `json:"title"`
    FileURL         string `json:"file_url"`
    IsSummarized    bool   `json:"is_summarized"`
    SummarizeFailed bool   `json:"summarize_failed"` // ← TAMBAH INI
    CreatedAt       string `json:"created_at"`
}
```

Update semua tempat yang membuat `ModuleResponse` di `module_service.go` agar menyertakan field baru:

```go
return &dto.ModuleResponse{
    ID:              module.ID,
    Title:           module.Title,
    FileURL:         module.FileURL,
    IsSummarized:    module.IsSummarized,
    SummarizeFailed: module.SummarizeFailed, // ← TAMBAH INI
    CreatedAt:       module.CreatedAt.Format(time.RFC3339),
}, nil
```

---

### Task 5 — Tambah Endpoint Retry Summarize

User perlu cara untuk mencoba ulang summarize tanpa harus hapus dan upload ulang modul.

**Route baru:** `POST /api/v1/modules/:id/retry-summarize`

**File: `internal/handler/module_handler.go`**

```go
// POST /api/v1/modules/:id/retry-summarize
func (h *ModuleHandler) RetrySummarize(c *fiber.Ctx) error {
    userID := c.Locals("userID").(string)
    moduleID := c.Params("id")

    if err := h.moduleService.RetrySummarize(c.Context(), userID, moduleID); err != nil {
        if errors.Is(err, service.ErrModuleNotFound) {
            return response.NotFound(c, err.Error())
        }
        if errors.Is(err, service.ErrNotModuleOwner) {
            return response.Forbidden(c, err.Error())
        }
        if errors.Is(err, service.ErrAlreadySummarized) {
            return response.BadRequest(c, err.Error())
        }
        return response.InternalError(c, "gagal memulai ulang proses summarize")
    }

    return response.OK(c, "proses summarize dimulai ulang", nil)
}
```

**File: `internal/service/module_service.go`**

Tambahkan error baru dan method baru:

```go
ErrAlreadySummarized = errors.New("modul sudah memiliki summary")
```

Tambahkan ke `ModuleServiceContract` interface:

```go
RetrySummarize(ctx context.Context, userID string, moduleID string) error
```

Implementasi:

```go
func (s *ModuleService) RetrySummarize(ctx context.Context, userID string, moduleID string) error {
    module, err := s.moduleRepo.FindByID(ctx, moduleID)
    if err != nil {
        return err
    }
    if module == nil {
        return ErrModuleNotFound
    }
    if module.UserID != userID {
        return ErrNotModuleOwner
    }
    if module.IsSummarized {
        return ErrAlreadySummarized
    }

    // Reset status gagal dulu
    if err := s.moduleRepo.MarkSummarizeFailed(ctx, moduleID); err != nil {
        // reset ke false sebelum retry
    }

    go func() {
        result, err := s.aiClient.Summarize(module.RawText)
        if err != nil {
            log.Printf("[module_service] retry summarize gagal modul %s: %v", moduleID, err)
            if dbErr := s.moduleRepo.MarkSummarizeFailed(context.Background(), moduleID); dbErr != nil {
                log.Printf("[module_service] gagal update status failed modul %s: %v", moduleID, dbErr)
            }
            return
        }
        if err := s.moduleRepo.UpdateSummary(context.Background(), moduleID, result.Summary); err != nil {
            log.Printf("[module_service] gagal simpan summary retry modul %s: %v", moduleID, err)
        }
    }()

    return nil
}
```

**File: `internal/router/router.go`**

```go
modules.Post("/:id/retry-summarize", moduleHandler.RetrySummarize)
```

---

### Task 6 — Migration Database

Tambahkan kolom baru ke tabel `modules`:

```sql
ALTER TABLE modules ADD COLUMN summarize_failed BOOLEAN DEFAULT FALSE;
```

Atau buat file migration baru jika menggunakan migration tool:

```
migrations/XXXX_add_summarize_failed_to_modules.sql
```

---

## Alur Setelah Fix

```
User upload PDF
    └── goroutine summarize AI
            ├── SUCCESS → is_summarized = true, summarize_failed = false
            └── FAIL    → summarize_failed = true, is_summarized = false (tetap)
                            └── Frontend deteksi summarize_failed = true
                                    └── Tampilkan badge "Gagal diproses"
                                        Tombol "Coba Lagi" → POST /modules/:id/retry-summarize
```

---

## Dampak ke Frontend

Setelah backend menambahkan field `summarize_failed`, frontend perlu update:

- `ModuleCard` dan `ModuleDetail`: tambahkan kondisi ketiga selain `is_summarized true/false`
- Badge state: `success` (tersedia), `warning` (diproses), **`danger` (gagal)**
- Tombol **"Coba Lagi"** pada modul yang `summarize_failed = true`
- Hentikan polling jika `summarize_failed = true` (tidak perlu terus poll)

Ubah kondisi polling di `useModulesQuery`:

```ts
// Sebelum: hanya cek is_summarized
query.state.data?.some(m => !m.is_summarized) ? 5000 : false

// Sesudah: berhenti polling jika sudah gagal
query.state.data?.some(m => !m.is_summarized && !m.summarize_failed) ? 5000 : false
```

---

## Ringkasan Perubahan Per File

| File | Perubahan |
|---|---|
| `genkit/main.go` | ✅ Sudah diperbaiki — ganti `gemini-1.5-flash` → `gemini-2.0-flash` |
| `internal/domain/module.go` | Tambah field `SummarizeFailed bool` |
| `internal/repository/module_repo.go` | Tambah `MarkSummarizeFailed()` ke interface + implementasi |
| `internal/service/module_service.go` | Update goroutine upload, tambah `RetrySummarize()`, tambah `ErrAlreadySummarized` |
| `internal/dto/module_dto.go` | Tambah `SummarizeFailed bool` ke `ModuleResponse` dan `ModuleDetailResponse` |
| `internal/handler/module_handler.go` | Tambah `RetrySummarize()` handler |
| `internal/router/router.go` | Tambah route `POST /modules/:id/retry-summarize` |
| `migrations/` | Tambah migration kolom `summarize_failed` |

---

## Endpoint Baru

| Method | Path | Auth | Deskripsi |
|---|---|---|---|
| POST | `/api/v1/modules/:id/retry-summarize` | ✅ | Coba ulang summarize yang gagal |

---

## Definition of Done

- [ ] Field `summarize_failed` tersedia di domain, repo, DTO, dan response API
- [ ] Goroutine di `Upload()` menyimpan status gagal ke DB jika Genkit error
- [ ] Endpoint `POST /modules/:id/retry-summarize` berfungsi
- [ ] Polling frontend berhenti jika `summarize_failed = true`
- [ ] Frontend menampilkan badge "Gagal" dan tombol "Coba Lagi" jika failed
- [ ] Migration database sudah dibuat
- [ ] Unit test untuk skenario summarize gagal ditambahkan
