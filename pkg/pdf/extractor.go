package pdf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"strings"

	"github.com/ledongthuc/pdf"
	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// VisionExtractor dipanggil sebagai fallback saat teks tidak bisa diekstrak secara langsung.
var VisionExtractor func(imageBase64 string, mimeType string) (string, error)

// ExtractText mencoba berbagai strategi untuk mengekstrak teks dari PDF.
func ExtractText(filePath string) (string, error) {
	if text := extractWithLib(filePath); text != "" {
		return text, nil
	}

	if text, err := extractWithPdftotext(filePath); err == nil && text != "" {
		return text, nil
	}

	if VisionExtractor != nil {
		if text, err := extractWithVision(filePath); err == nil && text != "" {
			return text, nil
		}
	}

	return "", fmt.Errorf("PDF tidak mengandung teks yang bisa diekstrak (mungkin PDF scan/image)")
}

func extractWithLib(filePath string) string {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var sb strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

func extractWithPdftotext(filePath string) (string, error) {
	out, err := exec.Command("pdftotext", "-layout", filePath, "-").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type pdfImage struct {
	data     []byte
	mimeType string
}

func extractWithVision(filePath string) (string, error) {
	images, err := extractImagesFromPDF(filePath)
	if err != nil || len(images) == 0 {
		return "", fmt.Errorf("tidak ada gambar ditemukan di PDF: %w", err)
	}

	var sb strings.Builder
	maxPages := 8
	if len(images) < maxPages {
		maxPages = len(images)
	}

	for i := 0; i < maxPages; i++ {
		b64, mime, err := imageToBase64(images[i])
		if err != nil {
			continue
		}
		text, err := VisionExtractor(b64, mime)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String()), nil
}

func extractImagesFromPDF(filePath string) ([]pdfImage, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	var images []pdfImage

	callback := func(img model.Image, singleImgPerPage bool, pageNr int) error {
		if len(images) >= 8 {
			return nil // sudah cukup
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(img); err != nil {
			return nil
		}
		images = append(images, pdfImage{
			data:     buf.Bytes(),
			mimeType: detectMIME(img.FileType),
		})
		return nil
	}

	if err := pdfcpuapi.ExtractImages(f, nil, callback, conf); err != nil {
		return nil, err
	}

	return images, nil
}

func imageToBase64(img pdfImage) (string, string, error) {
	var buf bytes.Buffer
	mime := img.mimeType

	switch mime {
	case "image/png":
		decoded, err := png.Decode(bytes.NewReader(img.data))
		if err != nil {
			return "", "", err
		}
		if err := png.Encode(&buf, decoded); err != nil {
			return "", "", err
		}
	default:
		decoded, err := jpeg.Decode(bytes.NewReader(img.data))
		if err != nil {
			buf.Write(img.data)
		} else {
			if err := jpeg.Encode(&buf, decoded, &jpeg.Options{Quality: 85}); err != nil {
				return "", "", err
			}
		}
		mime = "image/jpeg"
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), mime, nil
}

func detectMIME(fileType string) string {
	switch strings.ToLower(fileType) {
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
