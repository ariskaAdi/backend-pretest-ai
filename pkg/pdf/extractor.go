package pdf

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractText membaca PDF dari path lokal, return semua teks
func ExtractText(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("gagal membuka PDF: %w", err)
	}
	defer f.Close()

	var sb strings.Builder
	totalPages := r.NumPage()

	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // skip halaman bermasalah, lanjut
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("PDF tidak mengandung teks yang bisa diekstrak (mungkin PDF scan/image)")
	}

	return result, nil
}
