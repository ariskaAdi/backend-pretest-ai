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
)

var (
	ErrModuleNotFound              = errors.New("module not found")
	ErrNotModuleOwner              = errors.New("you do not have access to this module")
	ErrInvalidFileType             = errors.New("file must be in PDF format")
	ErrFileTooLarge                = errors.New("maximum file size is 20MB")
	ErrPDFNoText                   = errors.New("PDF does not contain extractable text")
	ErrInsufficientSummarizeQuota  = errors.New("kuota ringkas habis, silakan beli paket terlebih dahulu")
)

const maxFileSizeBytes = 20 * 1024 * 1024 // 20MB

type ModuleServiceContract interface {
	Upload(ctx context.Context, userID string, fileHeader *multipart.FileHeader, req dto.UploadModuleRequest) (*dto.ModuleResponse, error)
	GetAll(ctx context.Context, userID string) ([]dto.ModuleResponse, error)
	GetByID(ctx context.Context, userID string, moduleID string) (*dto.ModuleDetailResponse, error)
	Delete(ctx context.Context, userID string, moduleID string) error
	RetrySummarize(ctx context.Context, userID string, moduleID string) error
}

type R2Uploader interface {
	UploadFile(ctx context.Context, file multipart.File, filename string, contentType string) (string, error)
}

type AISummarizer interface {
	Summarize(pdfText string) (*pkgai.SummarizeOutput, error)
}

type ModuleService struct {
	moduleRepo repository.ModuleRepositoryContract
	userRepo   repository.UserRepository
	r2Client   R2Uploader
	aiClient   AISummarizer
}

func NewModuleService(moduleRepo repository.ModuleRepositoryContract, userRepo repository.UserRepository, r2Client R2Uploader, aiClient AISummarizer) ModuleServiceContract {
	return &ModuleService{
		moduleRepo: moduleRepo,
		userRepo:   userRepo,
		r2Client:   r2Client,
		aiClient:   aiClient,
	}
}

// Upload — terima PDF, upload ke R2, extract text, simpan ke DB, trigger summarize async
func (s *ModuleService) Upload(ctx context.Context, userID string, fileHeader *multipart.FileHeader, req dto.UploadModuleRequest) (*dto.ModuleResponse, error) {
	// Cek role user — admin bypass quota
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrModuleNotFound
	}

	// Deduct summarize quota (skip untuk admin)
	quotaDeducted := false
	if user.Role != domain.RoleAdmin {
		if err := s.userRepo.DeductSummarizeQuota(ctx, userID); err != nil {
			if errors.Is(err, repository.ErrQuotaInsufficient) {
				return nil, ErrInsufficientSummarizeQuota
			}
			return nil, err
		}
		quotaDeducted = true
	}

	// Jika ada error setelah quota dikurangi, kembalikan quota
	var moduleCreated bool
	defer func() {
		if quotaDeducted && !moduleCreated {
			if restoreErr := s.userRepo.RestoreSummarizeQuota(context.Background(), userID); restoreErr != nil {
				log.Printf("[module_service] gagal restore summarize quota untuk user %s: %v", userID, restoreErr)
			}
		}
	}()

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
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Simpan sementara ke temp folder untuk ekstraksi teks
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename)))
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpPath)

	if _, err := tmpFile.ReadFrom(file); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temporary file: %w", err)
	}
	tmpFile.Close()

	// Extract teks dari PDF
	rawText, err := pdfpkg.ExtractText(tmpPath)
	if err != nil {
		return nil, ErrPDFNoText
	}

	// Reset reader ke awal sebelum upload ke R2
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file reader: %w", err)
	}

	// Upload ke Cloudflare R2
	filename := fmt.Sprintf("modules/%s_%s", userID[:8], filepath.Base(fileHeader.Filename))
	fileURL, err := s.r2Client.UploadFile(ctx, file, filename, "application/pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
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
		return nil, fmt.Errorf("failed to save module: %w", err)
	}

	go func() {
		result, err := s.aiClient.Summarize(rawText)
		if err != nil {
			log.Printf("[module_service] failed to summarize module %s: %v", module.ID, err)
			// Simpan status gagal ke DB agar frontend bisa mendeteksi
			if dbErr := s.moduleRepo.MarkSummarizeFailed(context.Background(), module.ID); dbErr != nil {
				log.Printf("[module_service] failed to update status to failed for module %s: %v", module.ID, dbErr)
			}
			return
		}
		if err := s.moduleRepo.UpdateSummary(context.Background(), module.ID, result.Summary); err != nil {
			log.Printf("[module_service] failed to save summary for module %s: %v", module.ID, err)
		}
		log.Printf("[module_service] summary for module %s completed", module.ID)
	}()

	moduleCreated = true
	return &dto.ModuleResponse{
		ID:              module.ID,
		Title:           module.Title,
		FileURL:         module.FileURL,
		IsSummarized:    module.IsSummarized,
		SummarizeFailed: module.SummarizeFailed,
		CreatedAt:       module.CreatedAt.Format(time.RFC3339),
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
			ID:              m.ID,
			Title:           m.Title,
			FileURL:         m.FileURL,
			IsSummarized:    m.IsSummarized,
			SummarizeFailed: m.SummarizeFailed,
			CreatedAt:       m.CreatedAt.Format(time.RFC3339),
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
		ID:              module.ID,
		Title:           module.Title,
		FileURL:         module.FileURL,
		Summary:         module.Summary,
		IsSummarized:    module.IsSummarized,
		SummarizeFailed: module.SummarizeFailed,
		CreatedAt:       module.CreatedAt.Format(time.RFC3339),
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
func (s *ModuleService) RetrySummarize(ctx context.Context, userID string, moduleID string) error {
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
	if module.IsSummarized {
		return nil // Sudah selesai, tidak perlu retry
	}

	// Reset status gagal sebelum coba lagi
	if err := s.moduleRepo.UpdateSummarizeStatus(ctx, moduleID, false, false); err != nil {
		return fmt.Errorf("failed to reset summarize status: %w", err)
	}

	go func() {
		result, err := s.aiClient.Summarize(module.RawText)
		if err != nil {
			log.Printf("[module_service] retry summarize failed for module %s: %v", moduleID, err)
			if dbErr := s.moduleRepo.MarkSummarizeFailed(context.Background(), moduleID); dbErr != nil {
				log.Printf("[module_service] failed to update status to failed for module %s: %v", moduleID, dbErr)
			}
			return
		}
		if err := s.moduleRepo.UpdateSummary(context.Background(), moduleID, result.Summary); err != nil {
			log.Printf("[module_service] failed to save retry summary for module %s: %v", moduleID, err)
		}
	}()

	return nil
}
