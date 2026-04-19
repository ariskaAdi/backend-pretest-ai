package flows

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

type SummarizeInput struct {
	PdfText string `json:"pdf_text"`
}

type SummarizeOutput struct {
	Summary string `json:"summary"`
}

func RegisterSummarizeFlow(g *genkit.Genkit) *core.Flow[*SummarizeInput, *SummarizeOutput, struct{}] {
	return genkit.DefineFlow(g, "summarizeModule",
		func(ctx context.Context, input *SummarizeInput) (*SummarizeOutput, error) {
			if input.PdfText == "" {
				return nil, fmt.Errorf("pdf_text tidak boleh kosong")
			}

			// Potong teks kalau terlalu panjang — Groq batas ~6000 token ≈ 24000 karakter
			text := input.PdfText
			if len(text) > 24000 {
				text = text[:24000]
			}

			resp, err := generateWithGroq(buildSummarizePrompt(text))
			if err != nil {
				return nil, fmt.Errorf("gagal generate summary: %w", err)
			}

			return &SummarizeOutput{Summary: resp}, nil
		},
	)
}

func buildSummarizePrompt(text string) string {
	return fmt.Sprintf(`Kamu adalah asisten akademik ahli yang bertugas membuat ringkasan komprehensif modul kuliah untuk mahasiswa.

TUGAS UTAMA:
Buat ringkasan LENGKAP dan MENYELURUH dari seluruh isi modul berikut. Ringkasan harus mencakup SEMUA topik, bab, konsep, definisi, algoritma, dan metode yang disebutkan dalam materi — tidak boleh ada yang terlewat.

STRUKTUR RINGKASAN YANG WAJIB DIIKUTI:

1. **Gambaran Umum Mata Kuliah**
   - Capaian pembelajaran yang ingin dicapai
   - Topik-topik yang dibahas secara keseluruhan

2. **Ringkasan Per Topik/Bab** (bahas SETIAP topik secara terpisah)
   Untuk setiap topik, sertakan:
   - Definisi dan pengertian konsep utama
   - Komponen, tahapan, atau jenis-jenisnya
   - Cara kerja atau proses (jika ada)
   - Kelebihan dan kekurangan (jika disebutkan)
   - Contoh penerapan atau algoritma terkait

3. **Konsep dan Istilah Penting**
   - Daftar semua istilah teknis beserta definisinya
   - Rumus, metrik, atau ukuran statistik yang disebutkan
   - Perbedaan antara konsep yang sering dibandingkan

4. **Algoritma dan Metode**
   - Sebutkan semua algoritma/metode yang dibahas beserta kegunaannya
   - Kapan menggunakan metode tertentu vs metode lainnya

ATURAN PENULISAN:
- Bahasa Indonesia yang jelas, padat, dan akademis
- Gunakan heading dan sub-poin agar mudah dipindai
- Tulis SEMUA detail penting — lebih lengkap lebih baik
- Panjang: tidak dibatasi, utamakan kelengkapan
- Jangan meringkas terlalu singkat — mahasiswa butuh detail untuk belajar

Isi modul:
%s

Tulis ringkasan komprehensif sekarang:`, text)
}
