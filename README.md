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
| POST | `/auth/resend-otp` | ❌ | Kirim ulang OTP ke email |
| POST | `/auth/login` | ❌ | Login, return JWT token |
| POST | `/auth/logout` | ✅ | Logout (stateless) |

### Webhook

| Method | Endpoint | Auth | Deskripsi |
|---|---|---|---|
| POST | `/webhook/lynk` | ❌ | Webhook Lynk.id (Secret validated) |

### User

| Method | Endpoint | Auth | Deskripsi |
|---|---|---|---|
| GET | `/user/me` | ✅ | Ambil data profil user yang login |
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

## Quota & Monetisasi

Aplikasi menggunakan sistem quota per user untuk membatasi penggunaan layanan AI.

| Role | Quiz Quota | Summarize Quota | Cara Mendapat |
|---|---|---|---|
| **guest** | 1 | 1 | Otomatis saat register |
| **member** | accumulate | accumulate | Beli paket di Lynk.id |
| **admin** | unlimited | unlimited | - |

Webhook dari Lynk.id akan otomatis menambah quota dan mengupdate role setelah pembayaran berhasil.

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

## Environments

| Environment | Branch | Database | Deployment |
|---|---|---|---|
| **Development** | lokal | Docker PostgreSQL lokal | `make run` native |
| **Staging** | `develop` | Supabase (managed) | GitHub Actions → 1 VPS staging |
| **Production** | `main` | Self-hosted PostgreSQL | GitHub Actions → 1 VPS production |

---

## Docker & CI/CD

### Arsitektur Production (1 VPS)

```
Internet :80/:443
    ↓
Nginx (Docker) ─── rate limiting
    ├── /        → frontend  :3000
    └── /api/    → backend   :8080
                      ↓
                  genkit     :3400  (internal, tidak expose)
                  postgres   :5432  (internal, tidak expose)
```

Semua service jalan via `docker-compose.yml` dalam 1 VPS. Nginx sebagai reverse proxy sekaligus rate limiter.

Images di **GitHub Container Registry (ghcr.io)**:

| Service | Image | Tag Staging | Tag Production |
|---|---|---|---|
| Backend | `ghcr.io/<owner>/pretest-backend` | `:develop` | `:latest` |
| Genkit | `ghcr.io/<owner>/pretest-genkit` | `:develop` | `:latest` |
| Frontend | `ghcr.io/<owner>/pretest-frontend` | — | `:latest` |

### Development — PostgreSQL Lokal

```bash
docker compose -f docker-compose.dev.yml up -d   # nyalakan postgres
make run-genkit                                    # terminal 1
make run                                           # terminal 2
```

`.env` development:
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=pretestai
DB_SSLMODE=disable
```

### Staging — Supabase

`.env` di server staging:
```env
DB_HOST=db.xxxx.supabase.co
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_supabase_password
DB_NAME=postgres
DB_SSLMODE=require
```

### Production — Setup Pertama Kali di VPS

**0. Setup DNS domain**

Sebelum setup SSL, domain harus sudah diarahkan ke IP VPS. Di provider domain (contoh: Rumahweb), masuk ke DNS Management dan tambahkan:

```
Type : A  |  Name : @    |  Value : IP_VPS  |  TTL : 3600
Type : A  |  Name : www  |  Value : IP_VPS  |  TTL : 3600
```

Verifikasi DNS sudah propagate (tunggu 5-15 menit):
```powershell
# Di PowerShell lokal
nslookup yourdomain.com
# Harus muncul IP VPS kamu
```

**1. Install Docker**
```bash
curl -fsSL https://get.docker.com | sh
```

**2. Buat folder & .env**
```bash
mkdir -p /opt/pretest
# Isi .env — wajib ada semua variabel ini:
cat > /opt/pretest/.env << 'EOF'
APP_PORT=8080
APP_ENV=production

# Images (CI akan update otomatis)
BACKEND_IMAGE=ghcr.io/<owner>/pretest-backend:latest
GENKIT_IMAGE=ghcr.io/<owner>/pretest-genkit:latest
FRONTEND_IMAGE=ghcr.io/<owner>/pretest-frontend:latest

# DB — DB_HOST wajib "postgres" (nama service docker-compose)
DB_HOST=postgres
DB_PORT=5432
DB_USER=pretestai
DB_PASSWORD=strong_password_here
DB_NAME=pretestai
DB_SSLMODE=disable

# Services
GROQ_API_KEY=your_groq_key
GENKIT_URL=http://genkit:3400
GENKIT_PORT=3400
GENKIT_TIMEOUT_SECONDS=60

NEXT_PUBLIC_API_URL=https://yourdomain.com

# JWT, Mailer, R2, dsb...
EOF
```

**3. Login ke ghcr.io**
```bash
# Buat PAT di GitHub → Settings → Developer Settings → scope: read:packages
echo "YOUR_PAT" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

**4. Setup SSL dengan Certbot (sebelum jalankan nginx)**
```bash
# Install certbot
apt install certbot -y

# Buat folder untuk ACME challenge
mkdir -p /var/www/certbot

# Dapatkan sertifikat (nginx harus belum jalan di port 80)
certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com

# Ganti YOUR_DOMAIN di nginx config
sed -i 's/YOUR_DOMAIN/yourdomain.com/g' /opt/pretest/nginx/conf.d/pretest.conf
```

**5. Jalankan semua service**
```bash
cd /opt/pretest
docker compose up -d
```

**6. Auto-renew SSL (cron)**
```bash
# Tambah ke crontab
echo "0 3 * * * certbot renew --quiet && docker compose -f /opt/pretest/docker-compose.yml restart nginx" | crontab -
```

### CI/CD Pipeline

```
push ke develop              push ke main
      ↓                            ↓
[build]                      [build]
  backend:develop              backend:latest
  genkit:develop               genkit:latest
      ↓                            ↓
[staging]                    [production]
  SSH → docker run             SCP docker-compose.yml
  (Supabase DB)                    + nginx/conf.d/
                               SSH → docker compose up
                               (self-hosted postgres)
```

```
push ke main (frontend repo)
      ↓
[build] frontend:latest
      ↓
[production]
  SSH → docker compose pull frontend
      → docker compose up -d frontend
```

### GitHub Environments & Secrets

Buat di repo → **Settings → Environments**:

**Environment: `staging`**

| Secret | Keterangan |
|---|---|
| `SERVER_HOST` | IP VPS staging |
| `SERVER_USER` | User SSH (misal: `ubuntu`) |
| `SERVER_SSH_KEY` | Private key SSH |
| `GROQ_API_KEY` | API key Groq |

**Environment: `production`** (backend repo & frontend repo)

| Secret | Keterangan |
|---|---|
| `SERVER_HOST` | IP VPS production |
| `SERVER_USER` | User SSH |
| `SERVER_SSH_KEY` | Private key SSH |
| `GROQ_API_KEY` | API key Groq (backend repo) |
| `NEXT_PUBLIC_API_URL` | URL domain production (frontend repo) |

### Rate Limiting Nginx

Konfigurasi di `nginx/conf.d/pretest.conf`:

| Zone | Limit | Endpoint |
|---|---|---|
| `general` | 20 req/detik | API umum, halaman frontend |
| `auth` | 5 req/menit | `/api/v1/auth/` (login, register, OTP) |
| `upload` | 3 req/menit | `/api/v1/modules` (upload PDF) |
| webhook | tanpa limit | `/api/v1/webhook/` (sudah ada secret validation) |

---

## Catatan Penting

- **Jangan commit `.env`** — sudah ada di `.gitignore`
- **Gemini free tier** — 250 request/hari, 10 RPM. Cukup untuk development
- **Logout stateless** — token tidak di-blacklist. Implementasi blacklist (Redis) bisa ditambahkan nanti jika dibutuhkan
- **PDF harus text-based** — PDF hasil scan/gambar tidak bisa diekstrak teksnya
- **Summary async** — setelah upload, tunggu beberapa detik sebelum summary tersedia. Cek `is_summarized` dari `GET /modules/:id`

---

## DevOps Guide — Deployment dari Nol

Dokumentasi ini merangkum seluruh proses membangun pipeline deployment production-grade untuk project Go + Next.js. Bisa dijadikan referensi untuk project serupa di masa depan.

---

### 1. Konsep Dasar

#### Docker
Docker membungkus aplikasi beserta semua dependensinya menjadi sebuah **image** yang bisa dijalankan di server manapun secara konsisten.

```
Source code + Dockerfile  →  docker build  →  Image
Image                     →  docker run    →  Container (proses yang berjalan)
```

**Multi-stage build** — teknik untuk menghasilkan image sekecil mungkin:
```dockerfile
# Stage 1: build (pakai image besar dengan compiler)
FROM golang:1.25-alpine AS builder
RUN go build -o /app .

# Stage 2: run (hanya ambil binary-nya, buang compiler)
FROM alpine:3.20
COPY --from=builder /app .
```
Hasilnya image Go bisa sekecil ~15MB dibanding ~300MB kalau tidak pakai multi-stage.

#### Docker Compose
Menjalankan beberapa container sekaligus dan menghubungkannya dalam satu network internal.

```yaml
services:
  backend:
    image: my-backend
  postgres:
    image: postgres:16
```

Container bisa saling akses via nama service: backend bisa connect ke postgres cukup dengan `DB_HOST=postgres`.

#### CI/CD (GitHub Actions)
Otomasi proses build dan deploy setiap kali ada push ke repository.

```
Developer push code → GitHub Actions otomatis:
  1. Build Docker image
  2. Push image ke registry (ghcr.io)
  3. SSH ke server → pull image baru → restart container
```

#### Nginx sebagai Reverse Proxy
Nginx menerima semua request dari internet lalu meneruskannya ke service yang tepat berdasarkan path URL.

```
Internet → Nginx :443
              ├── /        → frontend :3000
              └── /api/    → backend  :8080
```

Keuntungan: hanya 1 port (443) yang expose ke internet, semua service internal tidak bisa diakses langsung dari luar.

---

### 2. Strategi Multi-Environment

Satu codebase, tiga environment berbeda dengan konfigurasi masing-masing:

```
┌─────────────┬──────────────┬─────────────────────┬──────────────────────┐
│ Environment │ Branch Git   │ Database            │ Frontend             │
├─────────────┼──────────────┼─────────────────────┼──────────────────────┤
│ Development │ lokal        │ Docker PostgreSQL   │ npm run dev          │
│ Staging     │ develop      │ Supabase (managed)  │ Vercel (auto-deploy) │
│ Production  │ main         │ Self-hosted Docker  │ Self-hosted Docker   │
└─────────────┴──────────────┴─────────────────────┴──────────────────────┘
```

**Kenapa Supabase untuk staging?**
Staging tidak butuh performa tinggi. Supabase gratis, tidak perlu setup, dan koneksinya cukup ganti connection string di `.env`.

**Kenapa self-hosted untuk production?**
Kontrol penuh, tidak bergantung pihak ketiga, biaya lebih prediktif.

**GitHub Environments** — fitur GitHub untuk menyimpan secrets per-environment. Satu secret `SERVER_HOST` bisa punya nilai berbeda di `staging` vs `production`.

---

### 3. Arsitektur File Deployment

```
backend-pretest-ai/
├── Dockerfile                        # Build image backend Go
├── .dockerignore                     # File yang dikecualikan dari build context
├── docker-compose.yml                # Production: semua service (nginx+fe+be+genkit+pg)
├── docker-compose.dev.yml            # Development: postgres lokal saja
├── nginx/
│   └── conf.d/
│       └── pretest.conf              # Reverse proxy + rate limiting config
├── genkit/
│   └── Dockerfile                    # Build image genkit Go
└── .github/
    └── workflows/
        └── deploy.yml                # CI/CD pipeline

frontend-pretest-ai/
├── Dockerfile                        # Build image Next.js (standalone mode)
├── .dockerignore
└── .github/
    └── workflows/
        └── deploy.yml                # CI/CD pipeline (production only)
```

---

### 4. Next.js Standalone Mode

Tambahkan `output: 'standalone'` di `next.config.ts` agar Docker image lebih kecil.

Tanpa standalone: image ~800MB–1GB (harus include `node_modules` penuh)
Dengan standalone: image ~150–200MB (hanya file yang benar-benar dipakai)

```typescript
const nextConfig: NextConfig = {
  output: 'standalone',
  // ...
};
```

Penting: `NEXT_PUBLIC_*` env vars di-**bake** saat build time, bukan runtime. Harus dipass sebagai `--build-arg` saat `docker build`.

---

### 5. GitHub Container Registry (ghcr.io)

Registry gratis dari GitHub untuk menyimpan Docker images. Terintegrasi langsung dengan GitHub Actions menggunakan `GITHUB_TOKEN` — tidak perlu buat credentials tambahan.

Konvensi penamaan image:
```
ghcr.io/<github-username>/<nama-image>:<tag>

Contoh:
ghcr.io/ariska/pretest-backend:latest    ← production
ghcr.io/ariska/pretest-backend:develop   ← staging
ghcr.io/ariska/pretest-backend:abc1234   ← commit SHA (untuk rollback)
```

Untuk pull image private dari server:
```bash
# Buat Personal Access Token di GitHub
# Settings → Developer Settings → Personal Access Tokens → scope: read:packages
echo "TOKEN" | docker login ghcr.io -u USERNAME --password-stdin
```

---

### 6. Nginx Rate Limiting

Rate limiting melindungi server dari abuse dan brute force. Nginx menggunakan sistem "leaky bucket":

```nginx
# Deklarasi zone (nama, ukuran memory, batas request)
limit_req_zone $binary_remote_addr zone=auth:10m rate=5r/m;
#              └── per IP address   └── nama    └── 5 request per menit

# Pakai di location block
location /api/v1/auth/ {
    limit_req zone=auth burst=3 nodelay;
    #                   └── boleh burst 3 request sebelum kena limit
}
```

Strategi rate limit project ini:
| Endpoint | Limit | Alasan |
|---|---|---|
| `/api/v1/auth/` | 5/menit | Cegah brute force password |
| `/api/v1/modules` | 3/menit | Upload PDF mahal, batasi spam |
| `/api/` | 20/detik | API umum, cukup longgar |
| `/api/v1/webhook/` | Tidak dibatasi | Sudah ada secret validation |

---

### 7. Memilih VPS

Pertimbangan utama saat memilih VPS:

**Spesifikasi minimum** untuk project dengan 5 Docker container:
- RAM: **4 GB** (2 GB bisa jalan tapi mepet)
- vCPU: **2 core**
- Storage: **40 GB SSD** minimum
- OS: **Ubuntu 22.04 LTS** (paling banyak support)

**Lokasi server** — pilih yang paling dekat dengan target pengguna. Untuk Indonesia, pilih datacenter Jakarta atau Singapura untuk latency rendah.

**Jangan ambil** control panel (cPanel, Plesk) — tidak dibutuhkan untuk Docker, hanya buang RAM.

**Virtualisasi KVM** lebih baik dari OpenVZ karena KVM memberikan resource dedicated, bukan shared.

---

### 8. Setup VPS dari Nol (Checklist)

Jalankan perintah ini berurutan setelah beli VPS baru:

```bash
# 1. Login pertama kali
ssh root@IP_SERVER
# Ketik "yes" saat ditanya authenticity — normal, hanya muncul sekali

# 2. Update sistem
apt update && apt upgrade -y

# 3. Install Docker
curl -fsSL https://get.docker.com | sh

# 4. Install Docker Compose plugin
apt install docker-compose-plugin -y

# 5. Verifikasi
docker --version
docker compose version

# 6. Buat folder project
mkdir -p /opt/pretest

# 7. Buat .env production
nano /opt/pretest/.env
# Isi semua variabel yang dibutuhkan

# 8. Login ke ghcr.io
echo "TOKEN" | docker login ghcr.io -u USERNAME --password-stdin

# 9. Install Certbot untuk SSL
apt install certbot -y
mkdir -p /var/www/certbot

# 10. Dapatkan SSL certificate
certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com

# 11. Update domain di nginx config
sed -i 's/YOUR_DOMAIN/yourdomain.com/g' /opt/pretest/nginx/conf.d/pretest.conf

# 12. Jalankan semua service
cd /opt/pretest
docker compose up -d

# 13. Setup auto-renew SSL
echo "0 3 * * * certbot renew --quiet && docker compose -f /opt/pretest/docker-compose.yml restart nginx" | crontab -
```

---

### 9. Isi `.env` Production

File ini **tidak boleh di-commit** ke git. Buat manual di server.

```env
# App
APP_PORT=8080
APP_ENV=production

# Docker Images (diupdate otomatis oleh CI/CD)
BACKEND_IMAGE=ghcr.io/<owner>/pretest-backend:latest
GENKIT_IMAGE=ghcr.io/<owner>/pretest-genkit:latest
FRONTEND_IMAGE=ghcr.io/<owner>/pretest-frontend:latest

# PostgreSQL — DB_HOST wajib "postgres" (nama service docker-compose)
DB_HOST=postgres
DB_PORT=5432
DB_USER=pretestai
DB_PASSWORD=ganti_dengan_password_kuat
DB_NAME=pretestai
DB_SSLMODE=disable

# Genkit — gunakan nama service docker-compose, bukan localhost
GENKIT_URL=http://genkit:3400
GENKIT_PORT=3400
GENKIT_TIMEOUT_SECONDS=60
GROQ_API_KEY=your_groq_api_key

# Frontend
NEXT_PUBLIC_API_URL=https://yourdomain.com

# JWT
JWT_SECRET=ganti_dengan_random_string_panjang
JWT_EXPIRE_HOURS=24

# Cloudflare R2
R2_ACCOUNT_ID=
R2_ACCESS_KEY=
R2_SECRET_KEY=
R2_BUCKET_NAME=
R2_PUBLIC_URL=

# Mailer
MAIL_HOST=smtp.gmail.com
MAIL_PORT=587
MAIL_USERNAME=
MAIL_PASSWORD=
MAIL_FROM=
```

---

### 10. GitHub Secrets — Setup Lengkap

Buat di **Settings → Environments** (bukan Settings → Secrets biasa) agar secrets terisolasi per environment.

**Environment: `staging`** (backend repo)
```
SERVER_HOST        = IP VPS staging
SERVER_USER        = root (atau ubuntu)
SERVER_SSH_KEY     = isi dengan: cat ~/.ssh/id_rsa
GROQ_API_KEY       = API key Groq untuk genkit
```

**Environment: `production`** (backend repo)
```
SERVER_HOST        = IP VPS production
SERVER_USER        = root
SERVER_SSH_KEY     = isi dengan: cat ~/.ssh/id_rsa
GROQ_API_KEY       = API key Groq untuk genkit
```

**Environment: `production`** (frontend repo)
```
SERVER_HOST        = IP VPS production (sama dengan backend)
SERVER_USER        = root
SERVER_SSH_KEY     = isi dengan: cat ~/.ssh/id_rsa
NEXT_PUBLIC_API_URL = https://yourdomain.com
```

Cara generate SSH key pair (jika belum punya):
```bash
ssh-keygen -t rsa -b 4096 -C "deploy-key"
cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys  # jalankan di server
cat ~/.ssh/id_rsa                                 # copy isi ini ke SERVER_SSH_KEY di GitHub
```

---

### 11. Alur Deploy Lengkap

```
┌─────────────────────────────────────────────────────────────────┐
│  DEVELOPMENT                                                     │
│  docker compose -f docker-compose.dev.yml up -d  (postgres)     │
│  make run-genkit + make run  (native, hot reload)                │
└─────────────────────────────────────────────────────────────────┘
                        ↓ push ke develop
┌─────────────────────────────────────────────────────────────────┐
│  STAGING                                                         │
│  GitHub Actions build → image:develop → push ghcr.io            │
│  SSH → docker pull → docker run (Supabase DB)                    │
│  Frontend → Vercel auto-deploy                                   │
└─────────────────────────────────────────────────────────────────┘
                        ↓ merge develop → main
┌─────────────────────────────────────────────────────────────────┐
│  PRODUCTION                                                      │
│  GitHub Actions build → image:latest → push ghcr.io             │
│  SCP docker-compose.yml + nginx/ → server                        │
│  SSH → docker compose pull → docker compose up -d               │
│  (nginx + frontend + backend + genkit + postgres dalam 1 VPS)   │
└─────────────────────────────────────────────────────────────────┘
```

---

### 12. Troubleshooting Umum

**Container tidak mau start**
```bash
docker compose logs <nama-service>   # lihat error log
docker compose ps                    # cek status semua container
```

**RAM habis / server lambat**
```bash
free -h                  # cek penggunaan RAM
docker stats             # RAM usage per container realtime
docker compose stop genkit   # matikan service yang tidak dipakai sementara
```

**Nginx error 502 Bad Gateway**
Berarti nginx tidak bisa reach backend/frontend. Cek apakah containernya jalan:
```bash
docker compose ps
docker compose restart backend
```

**SSL certificate expired**
```bash
certbot renew --quiet
docker compose restart nginx
```

**Rollback ke versi sebelumnya**
Setiap deploy di-tag dengan commit SHA. Untuk rollback:
```bash
# Di server
docker pull ghcr.io/<owner>/pretest-backend:SHA_LAMA
sed -i "s|BACKEND_IMAGE=.*|BACKEND_IMAGE=ghcr.io/<owner>/pretest-backend:SHA_LAMA|" /opt/pretest/.env
docker compose up -d backend
```

**Upload .env dari lokal ke VPS**
Jangan edit .env langsung di server kalau tidak nyaman dengan nano/vim. Edit di lokal lalu upload:
```powershell
# Di PowerShell lokal (Windows)
scp "c:\path\ke\project\.env" root@IP_VPS:/opt/pretest/.env
```

**GEMINI_API_KEY tidak dipakai lagi**
`.env.example` masih mencantumkan `GEMINI_API_KEY` tapi genkit sudah full migrasi ke Groq. Key Gemini tidak perlu diisi. Hanya `GROQ_API_KEY` yang dibutuhkan.

**Hati-hati highlight credentials di editor**
Claude Code VS Code extension mengirim teks yang di-highlight sebagai konteks. Jangan highlight baris yang berisi API key, password, atau token saat chat aktif.

**VPS 2GB RAM — tips hemat memori**
Untuk VPS dengan RAM terbatas (2GB), matikan service yang tidak aktif dipakai:
```bash
docker compose stop genkit    # matikan saat tidak ada request AI
docker compose start genkit   # nyalakan kembali saat dibutuhkan
```
Monitor RAM secara berkala:
```bash
free -h
docker stats --no-stream
```
