# Dokumentasi Genkit AI Service

Genkit adalah service AI terpisah yang bertugas memproses teks menggunakan Gemini API (Google AI SDK). Service ini didefinisikan dalam direktori `genkit/` sebagai Go module mandiri.

## Arsitektur

Genkit berjalan sebagai server HTTP di port `3400` (default) dan menyediakan endpoint untuk setiap "Flow".

- **summarizeModule**: Menerima teks PDF dan mengembalikan ringkasan.
- **generateQuiz**: Menerima ringkasan materi dan mengembalikan daftar soal pilihan ganda dalam format JSON.

## Instalasi & Menjalankan

1. Pastikan `GEMINI_API_KEY` sudah ada di file `.env` di root project.
2. Masuk ke direktori genkit: `cd genkit`
3. Download dependencies: `go mod tidy`
4. Jalankan service: `go run main.go`

## Pengembangan Flow

Setiap flow didaftarkan di `genkit/main.go` dan logika implementasinya ada di folder `genkit/flows/`.

- `genkit/flows/summarize.go`: Logika ringkasan modul.
- `genkit/flows/generate_quiz.go`: Logika pembuatan soal quiz.

## Prompt Management

Template prompt untuk setiap flow disimpan secara terpisah dalam folder `genkit/prompts/` untuk memudahkan eksperimen tanpa harus mengubah kode Go secara langsung (meskipun saat ini kode masih menggunakan prompt yang ter-hardcode sebagai cadangan).
