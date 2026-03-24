# Pretest AI — Backend

Backend service untuk aplikasi **Pretest AI**, platform belajar mahasiswa/pelajar yang membantu membuat ringkasan modul dan soal latihan secara otomatis menggunakan AI.

---

## Tech Stack

| Layer | Teknologi |
|---|---|
| Language | Go 1.22 |
| HTTP Framework | Fiber v2 |
| ORM | GORM |
| Database | PostgreSQL 16 |
| File Storage | Cloudflare R2 |
| AI Service | Genkit (Google AI — Gemini 1.5 Flash) |
| Auth | JWT (golang-jwt/jwt v5) |
| Email | SMTP (net/smtp) |

---

## Arsitektur

```
┌─────────────────────────────────────────────────────┐
│                   Client (Frontend)                  │
└───────────────────────┬─────────────────────────────┘
                        │ HTTP
                        ▼
┌─────────────────────────────────────────────────────┐
│              Backend Fiber  :8080                    │
│                                                      │
│  Handler → Service → Repository → PostgreSQL         │
│                │                                     │
│                └──── pkg/storage → Cloudflare R2     │
│                └──── pkg/ai ─────► Genkit :3400      │
└─────────────────────────────────────────────────────┘
                        │ HTTP
                        ▼
┌─────────────────────────────────────────────────────┐
│              Genkit AI Service  :3400                │
│                                                      │
│   flows/summarize → Gemini API                       │
│   flows/generate_quiz → Gemini API                   │
└─────────────────────────────────────────────────────┘
```

### Struktur Folder

```
.
├── cmd/server/main.go          # Entry point + graceful shutdown
├── config/
│   ├── config.go               # Load env ke struct Config
│   └── database.go             # Init koneksi PostgreSQL (GORM)
├── internal/
│   ├── domain/                 # GORM entity (User, Module, Quiz, Question)
│   ├── dto/                    # Request & Response struct per domain
│   ├── handler/                # HTTP handler (Fiber)
│   ├── service/                # Business logic + interface contract
│   ├── repository/             # Query DB + interface contract
│   ├── middleware/             # Auth JWT, Logger
│   └── router/                 # Register semua route
├── pkg/
│   ├── ai/genkit.go            # HTTP client ke Genkit service
│   ├── jwt/jwt.go              # Generate & parse JWT
│   ├── mailer/mailer.go        # SMTP email sender
│   ├── pdf/extractor.go        # Extract teks dari PDF
│   ├── response/response.go    # Standard HTTP response helper
│   └── storage/r2.go           # Cloudflare R2 client
├── genkit/                     # AI service (module terpisah)
│   ├── flows/
│   │   ├── summarize.go        # Flow: summarize PDF text
│   │   └── generate_quiz.go    # Flow: generate soal pilihan ganda
│   └── main.go                 # Entry point Genkit
├── migrations/                 # SQL migration files
├── Makefile                    # Shortcut commands
└── .env.example                # Template environment variables
```

---

## Prasyarat

- Go 1.22+
- Docker (untuk PostgreSQL)
- Node.js (untuk Genkit CLI, opsional)
- Akun Cloudflare (R2)
- Google AI API Key (Gemini)

---

## Setup & Menjalankan

### 1. Clone & Persiapan

```bash
git clone <repo-url>
cd ut-studypal

# Buat .env dari template
make env

# Edit .env sesuai konfigurasi kamu
```

### 2. Isi `.env`

```env
# App
APP_PORT=8080
APP_ENV=development

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=pretestai
DB_SSLMODE=disable

# Cloudflare R2
R2_ACCOUNT_ID=your_account_id
R2_ACCESS_KEY=your_access_key_id
R2_SECRET_KEY=your_secret_access_key
R2_BUCKET_NAME=pdf-assets
R2_PUBLIC_URL=https://pub-xxx.r2.dev

# Genkit
GENKIT_URL=http://localhost:3400
GENKIT_PORT=3400
GENKIT_TIMEOUT_SECONDS=60
GEMINI_API_KEY=your_gemini_api_key

# JWT
JWT_SECRET=your_secret_key
JWT_EXPIRE_HOURS=24

# Mailer (Gmail SMTP)
MAIL_HOST=smtp.gmail.com
MAIL_PORT=587
MAIL_USERNAME=your@gmail.com
MAIL_PASSWORD=your_app_password
MAIL_FROM=noreply@pretestai.com
```

> **GEMINI_API_KEY** — dapatkan gratis di [aistudio.google.com/app/apikey](https://aistudio.google.com/app/apikey)
>
> **MAIL_PASSWORD** — untuk Gmail, gunakan App Password (bukan password biasa). Aktifkan 2FA dulu di Google Account → Security → App Passwords.

### 3. Jalankan PostgreSQL

```bash
docker run -d \
  --name postgres-db \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=pretestai \
  -p 5432:5432 \
  -v postgres_data:/var/lib/postgresql/data \
  postgres:16
```

### 4. Jalankan Migration

```bash
make migrate
```

### 5. Install Dependencies

```bash
make tidy
```

### 6. Jalankan Service

Butuh **dua terminal**:

```bash
# Terminal 1 — Genkit AI Service
make run-genkit

# Terminal 2 — Backend Fiber
make run
```

---

## API Endpoints

Base URL: `http://localhost:8080/api/v1`

Semua endpoint yang membutuhkan auth menggunakan header:
```
Authorization: Bearer <token>
```

### Auth

| Method | Endpoint | Auth | Deskripsi |
|---|---|---|---|
| POST | `/auth/register` | ❌ | Daftar akun baru, kirim OTP ke email |
| POST | `/auth/verify-otp` | ❌ | Verifikasi OTP untuk aktivasi akun |
| POST | `/auth/login` | ❌ | Login, return JWT token |
| POST | `/auth/logout` | ✅ | Logout (stateless) |

### User

| Method | Endpoint | Auth | Deskripsi |
|---|---|---|---|
| POST | `/user/email/request-update` | ✅ | Request ganti email, kirim OTP ke email baru |
| POST | `/user/email/verify-update` | ✅ | Konfirmasi OTP, update email |

### Module

| Method | Endpoint | Auth | Deskripsi |
|---|---|---|---|
| POST | `/modules` | ✅ | Upload PDF (multipart/form-data) |
| GET | `/modules` | ✅ | List semua modul milik user |
| GET | `/modules/:id` | ✅ | Detail modul + status summary |
| DELETE | `/modules/:id` | ✅ | Hapus modul |

### Summary

| Method | Endpoint | Auth | Deskripsi |
|---|---|---|---|
| GET | `/summary/:moduleId` | ✅ | Ambil ringkasan modul |
| PUT | `/summary/:moduleId` | ✅ | Edit ringkasan secara manual |

### Quiz

| Method | Endpoint | Auth | Deskripsi |
|---|---|---|---|
| POST | `/quiz` | ✅ | Generate quiz dari summary modul |
| POST | `/quiz/:id/submit` | ✅ | Submit jawaban, dapat skor |
| POST | `/quiz/:id/retry` | ✅ | Buat quiz baru dari modul yang sama |
| GET | `/quiz/history` | ✅ | Riwayat semua quiz user |
| GET | `/quiz/history/module/:moduleId` | ✅ | Riwayat quiz per modul |
| GET | `/quiz/:id/result` | ✅ | Lihat hasil quiz yang sudah dikerjakan |

---

## Alur Utama

### Upload PDF & Summarize

```
1. User upload PDF → POST /modules
2. Backend:
   a. Validasi file (harus .pdf, max 20MB)
   b. Extract teks dari PDF
   c. Upload file asli ke Cloudflare R2
   d. Simpan metadata ke DB (url, raw_text)
   e. Goroutine async → Genkit summarize → simpan summary ke DB
3. Response langsung balik ke user (tidak nunggu AI)
4. User cek is_summarized via GET /modules/:id
```

### Generate & Kerjakan Quiz

```
1. Pastikan modul sudah is_summarized = true
2. POST /quiz → { module_id, num_questions: 5|10|20 }
3. Backend call Genkit generateQuiz → soal dikirim ke user (tanpa jawaban)
4. User kerjakan → POST /quiz/:id/submit → { answers: [...] }
5. Backend hitung skor → simpan ke DB → return hasil + jawaban benar
6. Mau ulang? → POST /quiz/:id/retry → soal baru dari modul yang sama
```

---

## Request & Response Examples

### Register

```bash
POST /api/v1/auth/register
Content-Type: application/json

{
  "name": "Ariska Adi",
  "email": "adi@example.com",
  "password": "password123"
}
```

```json
{
  "success": true,
  "message": "registrasi berhasil, cek email kamu untuk verifikasi OTP"
}
```

### Upload PDF

```bash
POST /api/v1/modules
Authorization: Bearer <token>
Content-Type: multipart/form-data

title: Modul Hukum Perdata
file: [file.pdf]
```

```json
{
  "success": true,
  "message": "modul berhasil diupload, proses ringkasan sedang berjalan",
  "data": {
    "id": "uuid",
    "title": "Modul Hukum Perdata",
    "file_url": "https://pub-xxx.r2.dev/modules/userid_file.pdf",
    "is_summarized": false,
    "created_at": "2026-03-21T18:00:00Z"
  }
}
```

### Generate Quiz

```bash
POST /api/v1/quiz
Authorization: Bearer <token>
Content-Type: application/json

{
  "module_id": "uuid-modul",
  "num_questions": 10
}
```

```json
{
  "success": true,
  "message": "quiz berhasil dibuat",
  "data": {
    "id": "uuid-quiz",
    "module_id": "uuid-modul",
    "module_title": "Modul Hukum Perdata",
    "num_questions": 10,
    "status": "pending",
    "questions": [
      {
        "id": "uuid-q1",
        "text": "Apa yang dimaksud dengan subjek hukum?",
        "options": [
          "A. Segala sesuatu yang dapat menjadi pendukung hak dan kewajiban",
          "B. Aturan yang mengatur hubungan antar manusia",
          "C. Lembaga yang berwenang membuat hukum",
          "D. Sanksi bagi pelanggar hukum"
        ]
      }
    ]
  }
}
```

---

## Makefile Commands

```bash
make help          # Tampilkan semua command
make run           # Jalankan backend
make run-genkit    # Jalankan Genkit AI service
make build         # Build binary ke ./bin/server
make tidy          # go mod tidy backend + genkit
make migrate       # Jalankan semua SQL migration
make migrate-fresh # Drop semua tabel + migrate ulang
make db-shell      # Masuk ke psql shell
make test          # Semua unit test + race detector
make test-cover    # Test + generate coverage.html
make test-service  # Test hanya layer service
make test-handler  # Test hanya layer handler
make env           # Buat .env dari .env.example
```

---

## Database Schema

```
users
  id (uuid), name, email, password (bcrypt), role (guest/member/admin),
  otp, is_verified, created_at, updated_at

modules
  id (uuid), user_id → users, title, file_url, raw_text, summary,
  is_summarized, created_at, updated_at

quizzes
  id (uuid), user_id → users, module_id → modules, num_questions,
  score (null = belum dikerjakan), status (pending/completed),
  created_at, updated_at

questions
  id (uuid), quiz_id → quizzes, text, options (jsonb),
  correct_answer, user_answer (null = belum dijawab),
  created_at, updated_at
```

---

## Unit Test

```bash
# Semua test
make test

# Dengan coverage report
make test-cover
# Buka coverage.html di browser

# Race condition check
go test ./... -race
```

Target coverage per package:

| Package | Target |
|---|---|
| `internal/service` | ≥ 80% |
| `internal/handler` | ≥ 60% |
| `pkg/jwt` | ≥ 90% |

Lihat issue terkait untuk panduan penulisan unit test:
- `ISSUE.md` — overview unit test
- `ISSUE_MODULE_TEST.md` — unit test case module
- `ISSUE_QUIZ_TEST.md` — unit test case quiz
- `ISSUE_SUMMARY_TEST.md` — unit test case summary
- `ISSUE_GENKIT.md` — dokumentasi & pengembangan Genkit

---

## Catatan Penting

- **Jangan commit `.env`** — sudah ada di `.gitignore`
- **Gemini free tier** — 250 request/hari, 10 RPM. Cukup untuk development
- **Logout stateless** — token tidak di-blacklist. Implementasi blacklist (Redis) bisa ditambahkan nanti jika dibutuhkan
- **PDF harus text-based** — PDF hasil scan/gambar tidak bisa diekstrak teksnya
- **Summary async** — setelah upload, tunggu beberapa detik sebelum summary tersedia. Cek `is_summarized` dari `GET /modules/:id`
