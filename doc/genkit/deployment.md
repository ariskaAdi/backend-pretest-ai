# Strategi Deployment Genkit Service

Genkit dirancang untuk dijalankan sebagai service independen. Berikut adalah beberapa opsi deployment yang direkomendasikan untuk tim:

## Opsi 1: Google Cloud Run (Direkomendasikan)

Karena Genkit Go SDK sangat ringan dan berbasis HTTP, Cloud Run adalah pilihan terbaik.

- **Kelebihan**: Auto-scaling ke nol saat tidak digunakan (hemat biaya), integrasi mudah dengan Secret Manager untuk `GEMINI_API_KEY`.
- **Konfigurasi**:
  - Gunakan `Dockerfile` di folder `genkit/`.
  - Set `GENKIT_URL` di backend utama agar menunjuk ke URL Cloud Run.

## Opsi 2: Sidecar Container (Kubernetes)

Jika backend utama di-deploy di Kubernetes, Genkit bisa berjalan sebagai sidecar dalam pod yang sama atau sebagai service terpisah di namespace yang sama.

- **Kelebihan**: Latensi rendah, komunikasi via `localhost:3400`.

## Opsi 3: Berjalan Bersamaan (Monolith-like)

Menjalankan `cmd/server/main.go` dan `genkit/main.go` di VM yang sama menggunakan process manager seperti `systemd` atau `pm2`.

- **Kelebihan**: Paling simpel untuk tahap awal.
- **Kekurangan**: Sulit mengukur utilisasi resource masing-masing service.

---

## Variabel Lingkungan (Env Vars)

Ketiga opsi di atas memerlukan:
- `GEMINI_API_KEY`: API Key dari Google AI Studio.
- `GENKIT_PORT`: Port HTTP (default 3400).
- `APP_ENV`: development / production.
