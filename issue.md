# ISSUE: Unit Test — Summary Case

## Status
`open`

## Priority
`medium`

## Assignee
_unassigned_

---

## Background & Alur Summary Case

Summary **tidak punya tabel sendiri** — data summary disimpan langsung di tabel `modules` (field `summary`, `is_summarized`). Summary case hanya menyediakan akses dan editing terhadap summary yang sudah ada.

---

### Dari Mana Summary Berasal?

Summary di-generate otomatis oleh AI saat user upload PDF di **case module**:

```
User upload PDF
    └── module_service.Upload()
            └── goroutine async
                    └── Genkit summarizeModule
                            └── modules.summary = "..."
                            └── modules.is_summarized = true
```

Summary case hanya membaca dan mengedit hasil tersebut. Tidak ada proses AI di summary case.

---

### Alur 1 — GET Summary `GET /api/v1/summary/:moduleId`

```
SummaryService.GetByModuleID()
    │
    ├── FindByID(moduleID)
    │       ├── nil              → ErrModuleNotFound → 404
    │       └── UserID != userID → ErrNotModuleOwner → 401
    │
    ├── Cek is_summarized == true && summary != ""
    │       └── false/kosong → ErrSummaryNotReady → 400
    │               "summary belum tersedia, modul masih diproses"
    │
    └── Return SummaryResponse
            ├── module_id
            ├── module_title
            ├── summary      ← teks ringkasan dari AI
            ├── is_summarized: true
            └── updated_at
```

> Kalau user baru saja upload PDF dan langsung hit endpoint ini, kemungkinan summary belum selesai karena AI masih proses di background. Client perlu polling atau cek `is_summarized` dari `GET /modules/:id`.

---

### Alur 2 — Edit Summary `PUT /api/v1/summary/:moduleId`

```
Client PUT /api/v1/summary/:moduleId
{
    "summary": "Ringkasan yang sudah diedit user..."
}

SummaryService.UpdateManual()
    │
    ├── FindByID(moduleID)
    │       ├── nil              → ErrModuleNotFound → 404
    │       └── UserID != userID → ErrNotModuleOwner → 401
    │
    ├── moduleRepo.UpdateSummaryManual()
    │       └── UPDATE modules SET summary = ? WHERE id = ?
    │           (is_summarized tetap true, tidak diubah)
    │
    └── Return SummaryResponse dengan summary yang baru
```

> `UpdateSummaryManual` berbeda dari `UpdateSummary` (yang dipakai AI). AI mengeset `is_summarized = true`, sedangkan edit manual hanya update field `summary` saja — `is_summarized` tidak disentuh karena sudah true.

---

### Kenapa Tidak Ada Re-Summarize?

Re-summarize (generate ulang summary dari AI) sengaja tidak dimasukkan ke scope ini. Kalau user merasa summary AI kurang bagus, cukup edit manual. Ini lebih simpel dan tidak membuang token Gemini.

---

## Endpoint

| Method | Path | Auth | Fungsi |
|---|---|---|---|
| GET | `/api/v1/summary/:moduleId` | ✅ | Ambil summary modul |
| PUT | `/api/v1/summary/:moduleId` | ✅ | Edit summary manual |

---

## Task 1 — Interface sudah ada

`SummaryServiceContract` sudah didefinisikan di `internal/service/summary_service.go`:

```go
type SummaryServiceContract interface {
    GetByModuleID(ctx context.Context, userID string, moduleID string) (*dto.SummaryResponse, error)
    UpdateManual(ctx context.Context, userID string, moduleID string, req dto.UpdateSummaryRequest) (*dto.SummaryResponse, error)
}
```

`SummaryHandler` sudah menggunakan interface ini. `SummaryService` menggunakan `ModuleRepositoryContract` yang sudah ada — tidak perlu repository baru.

---

## Task 2 — Unit Test: `test/summary/service/summary_service_test.go`

### Setup mock:
Gunakan `MockModuleRepository` yang sudah dibuat di `ISSUE_MODULE_TEST.md` — tidak perlu buat ulang.

### Test case `GetByModuleID()`:

| # | Skenario | Expected |
|---|---|---|
| 1 | Modul tidak ditemukan | `ErrModuleNotFound` |
| 2 | Modul milik user lain | `ErrNotModuleOwner` |
| 3 | `is_summarized = false` | `ErrSummaryNotReady` |
| 4 | `is_summarized = true` tapi `summary = ""` | `ErrSummaryNotReady` |
| 5 | Sukses | return `SummaryResponse` dengan summary terisi |
| 6 | Sukses | `module_title` ikut ter-include di response |

### Test case `UpdateManual()`:

| # | Skenario | Expected |
|---|---|---|
| 1 | Modul tidak ditemukan | `ErrModuleNotFound` |
| 2 | Modul milik user lain | `ErrNotModuleOwner` |
| 3 | DB error saat update | return error |
| 4 | Sukses | return `SummaryResponse` dengan summary baru |
| 5 | Sukses | `is_summarized` tetap `true` setelah edit |
| 6 | Summary kurang dari 10 karakter | validasi di handler, bukan service |

---

## Task 3 — Unit Test: `test/summary/handler/summary_handler_test.go`

| Endpoint | Skenario | Expected HTTP |
|---|---|---|
| `GET /summary/:moduleId` | Modul tidak ditemukan | `404` |
| `GET /summary/:moduleId` | Bukan pemilik | `401` |
| `GET /summary/:moduleId` | Summary belum siap | `400` |
| `GET /summary/:moduleId` | Sukses | `200` + summary data |
| `PUT /summary/:moduleId` | Body kosong | `400` |
| `PUT /summary/:moduleId` | Summary < 10 karakter | `400` |
| `PUT /summary/:moduleId` | Modul tidak ditemukan | `404` |
| `PUT /summary/:moduleId` | Sukses | `200` + summary baru |

---

## Struktur File

```
test  /
├── summary/
│   ├── service/
│   │   └── summary_service_test.go   ← buat baru
│   └── handler/
│       └── summary_handler_test.go   ← buat baru
```

---


## Task 4 — Dokumentasi Swagger: `doc/summary/swagger.yml`

buat dokumentasi detail untuk summary case di folder tsb

## Struktur File

```
doc  /
├── summary/
│   └── summary.yml   ← buat baru
```

---

## Catatan Penting

- Summary service **reuse** `ModuleRepositoryContract` — tidak ada repository baru
- Mock yang dipakai adalah `MockModuleRepository` dari issue module, tidak perlu duplikasi
- `UpdateSummaryManual` hanya update field `summary`, tidak mengubah `is_summarized`
- Validasi panjang minimum summary (`min=10`) ada di DTO/handler, bukan service

```bash
go test ./internal/service/... -v -race
go test ./internal/handler/... -v
```

---

## Ekspektasi Coverage

| Package | Target |
|---|---|
| `test/summary/service/summary_service_test.go` | ≥ 80% |
| `test/summary/handler/summary_handler_test.go` | ≥ 60% |

---

## Definition of Done

- [ ] Semua test case di tabel atas diimplementasi
- [ ] Mock reuse dari `MockModuleRepository` (tidak duplikasi)
- [ ] `go test ./... -race` lulus tanpa error
- [ ] Coverage `summary_service` minimal 80%
- [ ] Dokumentasi summary case selesai

