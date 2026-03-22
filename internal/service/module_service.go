package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/repository"
	pkgai "backend-pretest-ai/pkg/ai"
	pdfpkg "backend-pretest-ai/pkg/pdf"
	"backend-pretest-ai/pkg/storage"
)

var (
	ErrModuleNotFound     = errors.New("modul tidak ditemukan")
	ErrNotModuleOwner     = errors.New("kamu tidak memiliki akses ke modul ini")
	ErrInvalidFileType    = errors.New("file harus berformat PDF")
	ErrFileTooLarge       = errors.New("ukuran file maksimal 20MB")
	ErrPDFNoText          = errors.New("PDF tidak mengandung teks yang bisa diekstrak")
)

const maxFileSizeBytes = 20 * 1024 * 1024 // 20MB

type ModuleServiceContract interface {
	Upload(ctx context.Context, userID string, fileHeader *multipart.FileHeader, req dto.UploadModuleRequest) (*dto.ModuleResponse, error)
	GetAll(ctx context.Context, userID string) ([]dto.ModuleResponse, error)
	GetByID(ctx context.Context, userID string, moduleID string) (*dto.ModuleDetailResponse, error)
	Delete(ctx context.Context, userID string, moduleID string) error
}

type R2Uploader interface {
	UploadFile(ctx context.Context, file multipart.File, filename string, contentType string) (string, error)
}

type AISummarizer interface {
	Summarize(pdfText string) (*pkgai.SummarizeOutput, error)
}

var (
	r2Client   R2Uploader   = storage.R2
	aiClient   AISummarizer = pkgai.Client
)

type ModuleService struct {
	moduleRepo repository.ModuleRepositoryContract
}

func NewModuleService(moduleRepo repository.ModuleRepositoryContract) ModuleServiceContract {
	return &ModuleService{moduleRepo: moduleRepo}
}

// Upload — terima PDF, upload ke R2, extract text, simpan ke DB, trigger summarize async
func (s *ModuleService) Upload(ctx context.Context, userID string, fileHeader *multipart.FileHeader, req dto.UploadModuleRequest) (*dto.ModuleResponse, error) {
	// Validasi tipe file
	if filepath.Ext(fileHeader.Filename) != ".pdf" {
		return nil, ErrInvalidFileType
	}

	// Validasi ukuran file
	if fileHeader.Size > maxFileSizeBytes {
		return nil, ErrFileTooLarge
	}

	// Buka file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("gagal membuka file: %w", err)
	}
	defer file.Close()

	// Simpan sementara ke /tmp untuk ekstraksi teks
	tmpPath := fmt.Sprintf("/tmp/%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat file sementara: %w", err)
	}
	defer os.Remove(tmpPath)

	if _, err := tmpFile.ReadFrom(file); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("gagal menulis file sementara: %w", err)
	}
	tmpFile.Close()

	// Extract teks dari PDF
	rawText, err := pdfpkg.ExtractText(tmpPath)
	if err != nil {
		return nil, ErrPDFNoText
	}

	// Reset reader ke awal sebelum upload ke R2
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("gagal reset file reader: %w", err)
	}

	// Upload ke Cloudflare R2
	filename := fmt.Sprintf("modules/%s_%s", userID[:8], filepath.Base(fileHeader.Filename))
	fileURL, err := r2Client.UploadFile(ctx, file, filename, "application/pdf")
	if err != nil {
		return nil, fmt.Errorf("gagal upload file: %w", err)
	}

	// Simpan ke database
	module := &domain.Module{
		UserID:       userID,
		Title:        req.Title,
		FileURL:      fileURL,
		RawText:      rawText,
		IsSummarized: false,
	}
	if err := s.moduleRepo.Create(ctx, module); err != nil {
		return nil, fmt.Errorf("gagal menyimpan modul: %w", err)
	}

	// Trigger summarize ke Genkit — async, tidak block response
	go func() {
		result, err := aiClient.Summarize(rawText)
		if err != nil {
			log.Printf("[module_service] gagal summarize modul %s: %v", module.ID, err)
			return
		}
		if err := s.moduleRepo.UpdateSummary(context.Background(), module.ID, result.Summary); err != nil {
			log.Printf("[module_service] gagal simpan summary modul %s: %v", module.ID, err)
		}
		log.Printf("[module_service] summary modul %s selesai", module.ID)
	}()

	return &dto.ModuleResponse{
		ID:           module.ID,
		Title:        module.Title,
		FileURL:      module.FileURL,
		IsSummarized: module.IsSummarized,
		CreatedAt:    module.CreatedAt.Format(time.RFC3339),
	}, nil
}

// GetAll — ambil semua modul milik user
func (s *ModuleService) GetAll(ctx context.Context, userID string) ([]dto.ModuleResponse, error) {
	modules, err := s.moduleRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var result []dto.ModuleResponse
	for _, m := range modules {
		result = append(result, dto.ModuleResponse{
			ID:           m.ID,
			Title:        m.Title,
			FileURL:      m.FileURL,
			IsSummarized: m.IsSummarized,
			CreatedAt:    m.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// GetByID — ambil detail modul termasuk summary, validasi ownership
func (s *ModuleService) GetByID(ctx context.Context, userID string, moduleID string) (*dto.ModuleDetailResponse, error) {
	module, err := s.moduleRepo.FindByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, ErrModuleNotFound
	}
	if module.UserID != userID {
		return nil, ErrNotModuleOwner
	}

	return &dto.ModuleDetailResponse{
		ID:           module.ID,
		Title:        module.Title,
		FileURL:      module.FileURL,
		Summary:      module.Summary,
		IsSummarized: module.IsSummarized,
		CreatedAt:    module.CreatedAt.Format(time.RFC3339),
	}, nil
}

// Delete — hapus modul, validasi ownership
func (s *ModuleService) Delete(ctx context.Context, userID string, moduleID string) error {
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

	return s.moduleRepo.Delete(ctx, moduleID)
}
