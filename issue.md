# 📋 ISSUE: Dokumentasi & Pengembangan — Genkit AI Service

## Status
`open`

## Priority
`medium`

## Assignee
_unassigned_

---

## Overview

Genkit adalah **service AI terpisah** yang berjalan di port `3400`. Backend utama (Fiber, port `8080`) berkomunikasi dengan Genkit melalui HTTP POST. Genkit bertugas menerima input teks, memprosesnya ke Gemini API, lalu mengembalikan hasil dalam format JSON.

```
┌─────────────────┐         HTTP POST          ┌──────────────────┐
│  Backend Fiber  │ ─────────────────────────► │  Genkit Service  │
│   :8080         │ ◄───────────────────────── │   :3400          │
└─────────────────┘       JSON response        └────────┬─────────┘
                                                        │
                                                        │ HTTPS
                                                        ▼
                                               ┌──────────────────┐
                                               │   Gemini API     │
                                               │  (Google AI)     │
                                               └──────────────────┘
```

---

## Struktur Folder Genkit

```
genkit/
├── flows/
│   ├── summarize.go        ← Flow: summarize PDF text → ringkasan modul
│   └── generate_quiz.go    ← Flow: generate soal dari summary
├── prompts/
│   ├── summarize.prompt    ← (opsional) template prompt summarize
│   └── quiz.prompt         ← (opsional) template prompt quiz
├── go.mod                  ← module terpisah dari backend utama
└── main.go                 ← entry point, register flows, jalankan server
```

> **Catatan:** `genkit/` adalah Go module tersendiri (`module ut-studypal/genkit`), terpisah dari `module ut-studypal` di root. Harus `go mod tidy` terpisah.

---

## Cara Menjalankan

### Prasyarat
1. Punya **GEMINI_API_KEY** — dapatkan gratis di [aistudio.google.com/app/apikey](https://aistudio.google.com/app/apikey)
2. Sudah ada file `.env` di root project dengan `GEMINI_API_KEY=...`

### Jalankan Genkit
```bash
# Dari root project
make run-genkit

# Atau manual
cd genkit/
go mod tidy
go run main.go
```

### Jalankan Developer UI (opsional, untuk testing flow)
```bash
# Install Genkit CLI dulu
npm install -g genkit-cli

# Dari folder genkit/
genkit start -- go run main.go
# Developer UI tersedia di http://localhost:4000
```

---

## Alur Flow: `summarizeModule`

### Endpoint
```
POST http://localhost:3400/summarizeModule
```

### Request
```json
{
  "data": {
    "pdf_text": "Isi teks dari PDF yang sudah diekstrak..."
  }
}
```

### Proses Internal
```
Terima pdf_text
    ├── Validasi: tidak boleh kosong
    ├── Potong teks kalau > 12.000 karakter (batas aman token Gemini)
    └── Kirim ke Gemini dengan prompt:
            "Buat ringkasan modul kuliah dalam Bahasa Indonesia,
             3-5 paragraf, fokus pada konsep utama..."
            └── Gemini proses → return teks ringkasan
```

### Response
```json
{
  "result": {
    "summary": "Modul ini membahas konsep dasar..."
  }
}
```

### Dipanggil dari
`internal/service/module_service.go` → fungsi `Upload()` → **goroutine async**

```go
go func() {
    result, err := pkgai.Client.Summarize(rawText)
    // simpan ke DB setelah selesai
}()
```

> Dipanggil async karena proses AI bisa memakan 5–30 detik. Upload PDF tidak perlu nunggu summary selesai.

---

## Alur Flow: `generateQuiz`

### Endpoint
```
POST http://localhost:3400/generateQuiz
```

### Request
```json
{
  "data": {
    "summary": "Ringkasan modul yang sudah ada...",
    "num_questions": 10
  }
}
```

### Proses Internal
```
Terima summary + num_questions
    ├── Validasi: summary tidak kosong, num_questions > 0
    └── Kirim ke Gemini dengan prompt:
            "Buat {num_questions} soal pilihan ganda dari materi berikut.
             Format JSON: {questions: [{question, options[4], answer}]}
             Kembalikan HANYA JSON tanpa markdown..."
            └── Gemini generate soal
            └── Parse JSON response
            └── Validasi: tiap soal punya 4 pilihan + jawaban
            └── Normalisasi jawaban ke huruf kapital (A/B/C/D)
```

### Response
```json
{
  "result": {
    "questions": [
      {
        "question": "Apa yang dimaksud dengan Pancasila?",
        "options": [
          "A. Dasar negara Indonesia",
          "B. Lagu kebangsaan",
          "C. Bendera negara",
          "D. Semboyan negara"
        ],
        "answer": "A"
      }
    ]
  }
}
```

### Dipanggil dari
`internal/service/quiz_service.go` → fungsi `Generate()` → **synchronous** (user menunggu soal selesai di-generate)

---

## Komunikasi Backend → Genkit (`pkg/ai/genkit.go`)

Backend tidak langsung pakai Genkit SDK — komunikasi lewat HTTP biasa menggunakan `net/http`. Ini memungkinkan Genkit di-deploy terpisah (misal di Cloud Run) tanpa mengubah kode backend.

```go
// Helper generic untuk semua flow
func (g *genkitClient) call(flow string, input any, output any) error {
    body := json.Marshal({"data": input})
    resp := http.Post(baseURL + "/" + flow, body)
    // unwrap {"result": ...}
    json.Unmarshal(resp.result, &output)
}

// Penggunaan spesifik
func (g *genkitClient) Summarize(pdfText string) (*SummarizeOutput, error)
func (g *genkitClient) GenerateQuiz(summary string, numQuestions int) (*GenerateQuizOutput, error)
```

---

## Model yang Dipakai

| Model | Keterangan |
|---|---|
| `gemini-2.5-flash` | Model yang aktif dipakai saat ini |
| ~~`gemini-2.0-flash`~~ | Deprecated sejak 3 Maret 2026, jangan dipakai |

---

## Rate Limit Gemini Free Tier

| Limit | Nilai |
|---|---|
| Request per menit (RPM) | 10 |
| Request per hari (RPD) | 250 |
| Token per menit (TPM) | 250.000 |

Cukup untuk development. Untuk production, aktifkan billing di Google Cloud untuk limit yang jauh lebih tinggi.

---

## Task untuk Tim

### Task 1 — Uji Manual Kedua Flow

Sebelum integrasi penuh, test masing-masing flow secara manual:

```bash
# Test summarizeModule
curl -X POST http://localhost:3400/summarizeModule \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "pdf_text": "Pancasila adalah dasar negara Republik Indonesia yang terdiri dari 5 sila..."
    }
  }'

# Test generateQuiz
curl -X POST http://localhost:3400/generateQuiz \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "summary": "Pancasila adalah dasar negara yang memiliki 5 sila...",
      "num_questions": 5
    }
  }'
```

Pastikan response sesuai format yang diharapkan.

### Task 2 — Dokumentasi Prompt

Isi file `genkit/prompts/summarize.prompt` dan `genkit/prompts/quiz.prompt` dengan versi final prompt yang sudah diuji dan terbukti menghasilkan output terbaik. File ini sebagai referensi tim, bukan dipakai langsung oleh kode.

Format dokumentasi prompt:
```
# Prompt: summarizeModule
# Model: gemini-2.5-flash
# Last updated: YYYY-MM-DD
# Author: ...

[System context]
...

[User prompt template]
...

[Expected output format]
...

[Known issues / edge cases]
...
```

### Task 3 — Error Handling Edge Cases

Beberapa kondisi yang perlu diverifikasi:

| Kondisi | Perilaku saat ini | Yang diharapkan |
|---|---|---|
| PDF text sangat panjang (> 50.000 karakter) | Dipotong di 12.000 | Verifikasi tidak kehilangan konteks penting |
| Gemini return JSON tidak valid | `parseQuizResponse` return error | Backend log error, quiz tidak tersimpan |
| Gemini timeout | HTTP client timeout (60 detik) | Error dikembalikan ke service |
| Gemini return kurang dari `num_questions` soal | Lolos validasi | Perlu dicek — apakah perlu retry? |

### Task 4 — Pertimbangan Deployment Terpisah

Saat ini Genkit dan Backend dijalankan manual di dua terminal. Untuk production, pertimbangkan:

```
Opsi A: Satu server
    Backend Fiber :8080
    Genkit :3400
    → Jalankan keduanya via Makefile atau supervisor

Opsi B: Deploy terpisah
    Backend → VPS / Railway
    Genkit  → Google Cloud Run (native support Genkit)
    → Update GENKIT_URL di .env ke URL Cloud Run
```

---

## Graceful Shutdown

Kedua service (Backend dan Genkit) sudah mengimplementasi graceful shutdown:

```
Ctrl+C / SIGTERM diterima
    ├── Backend Fiber
    │       ├── Tunggu request yang sedang berjalan selesai
    │       └── Tutup koneksi PostgreSQL
    └── Genkit
            ├── Cancel context (hentikan request Gemini yang sedang jalan)
            └── Shutdown HTTP server
```

Ini penting agar proses summarize yang sedang berjalan di goroutine tidak terpotong di tengah jalan.

---

## Definition of Done

- [ ] Kedua flow berhasil ditest manual via curl
- [ ] `genkit/prompts/summarize.prompt` dan `quiz.prompt` terisi dokumentasi prompt final
- [ ] Edge case di Task 3 sudah diverifikasi dan didokumentasikan
- [ ] Keputusan deployment (Task 4) sudah disepakati tim
- [ ] Tidak ada API key yang ter-commit ke repository (pastikan `.env` ada di `.gitignore`)
