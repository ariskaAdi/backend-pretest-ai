package service

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
	pkgai "backend-pretest-ai/pkg/ai"
)

// --- Mocks ---

type MockModuleRepository struct {
	mock.Mock
}

func (m *MockModuleRepository) Create(ctx context.Context, module *domain.Module) error {
	args := m.Called(ctx, module)
	return args.Error(0)
}
func (m *MockModuleRepository) FindByID(ctx context.Context, id string) (*domain.Module, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Module), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockModuleRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Module, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]domain.Module), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockModuleRepository) UpdateSummary(ctx context.Context, moduleID string, summary string) error {
	args := m.Called(ctx, moduleID, summary)
	return args.Error(0)
}
func (m *MockModuleRepository) UpdateSummaryManual(ctx context.Context, moduleID string, summary string) error {
	args := m.Called(ctx, moduleID, summary)
	return args.Error(0)
}
func (m *MockModuleRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockR2Uploader struct {
	mock.Mock
}

func (m *MockR2Uploader) UploadFile(ctx context.Context, file multipart.File, filename string, contentType string) (string, error) {
	args := m.Called(ctx, file, filename, contentType)
	return args.String(0), args.Error(1)
}

type MockAISummarizer struct {
	mock.Mock
}

func (m *MockAISummarizer) Summarize(pdfText string) (*pkgai.SummarizeOutput, error) {
	args := m.Called(pdfText)
	if args.Get(0) != nil {
		return args.Get(0).(*pkgai.SummarizeOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Helpers ---

var minimalPDF = []byte(
	"%PDF-1.1\n%\xc2\xa5\xc2\xb1\xc3\xab\n1 0 obj\n  << /Type /Catalog\n     /Pages 2 0 R\n  >>\nendobj\n" +
		"2 0 obj\n  << /Type /Pages\n     /Kids [3 0 R]\n     /Count 1\n     /MediaBox [0 0 300 144]\n  >>\nendobj\n" +
		"3 0 obj\n  <<  /Type /Page\n      /Parent 2 0 R\n      /Resources\n       << /Font\n           << /F1\n" +
		"               << /Type /Font\n                  /Subtype /Type1\n                  /BaseFont /Times-Roman\n" +
		"               >>\n           >>\n       >>\n      /Contents 4 0 R\n  >>\nendobj\n" +
		"4 0 obj\n  << /Length 55 >>\nstream\n  BT\n    /F1 18 Tf\n    0 0 Td\n    (Hello World) Tj\n  ET\n" +
		"endstream\nendobj\nxref\n0 5\n0000000000 65535 f \n0000000018 00000 n \n0000000077 00000 n \n" +
		"0000000178 00000 n \n0000000457 00000 n \ntrailer\n  <<  /Root 1 0 R\n      /Size 5\n  >>\n" +
		"startxref\n565\n%%EOF\n",
)

func createMultipartFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	err = req.ParseMultipartForm(32 << 20)
	require.NoError(t, err)

	_, header, err := req.FormFile("file")
	require.NoError(t, err)
	return header
}

func setupModuleServiceTest(t *testing.T) (service.ModuleServiceContract, *MockModuleRepository, *MockR2Uploader, *MockAISummarizer) {
	mockRepo := new(MockModuleRepository)
	mockR2 := new(MockR2Uploader)
	mockAI := new(MockAISummarizer)

	srv := service.NewModuleService(mockRepo, mockR2, mockAI)

	return srv, mockRepo, mockR2, mockAI
}

// --- Tests ---

func TestModuleService_Upload(t *testing.T) {
	t.Run("File bukan .pdf", func(t *testing.T) {
		srv, _, _, _ := setupModuleServiceTest(t)
		header := createMultipartFileHeader(t, "test.txt", []byte("hello"))

		resp, err := srv.Upload(context.Background(), "user1", header, dto.UploadModuleRequest{})

		assert.Error(t, err)
		assert.Equal(t, service.ErrInvalidFileType, err)
		assert.Nil(t, resp)
	})

	t.Run("File > 20MB", func(t *testing.T) {
		srv, _, _, _ := setupModuleServiceTest(t)
		header := createMultipartFileHeader(t, "test.pdf", []byte("pdf"))
		header.Size = 25 * 1024 * 1024 // Fake size

		resp, err := srv.Upload(context.Background(), "user1", header, dto.UploadModuleRequest{})

		assert.Error(t, err)
		assert.Equal(t, service.ErrFileTooLarge, err)
		assert.Nil(t, resp)
	})

	t.Run("PDF tidak mengandung teks", func(t *testing.T) {
		srv, _, _, _ := setupModuleServiceTest(t)
		header := createMultipartFileHeader(t, "test.pdf", []byte("not a valid pdf content"))

		resp, err := srv.Upload(context.Background(), "user1", header, dto.UploadModuleRequest{})

		assert.Error(t, err)
		assert.Equal(t, service.ErrPDFNoText, err)
		assert.Nil(t, resp)
	})

	t.Run("R2 upload gagal", func(t *testing.T) {
		srv, _, mockR2, _ := setupModuleServiceTest(t)
		header := createMultipartFileHeader(t, "test.pdf", minimalPDF)

		mockR2.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, "application/pdf").
			Return("", assert.AnError)

		resp, err := srv.Upload(context.Background(), "user1", header, dto.UploadModuleRequest{Title: "Test"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gagal upload file")
		assert.Nil(t, resp)
	})

	t.Run("DB create gagal", func(t *testing.T) {
		srv, mockRepo, mockR2, _ := setupModuleServiceTest(t)
		header := createMultipartFileHeader(t, "test.pdf", minimalPDF)

		mockR2.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, "application/pdf").
			Return("https://r2/file.pdf", nil)
		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Module")).
			Return(assert.AnError)

		resp, err := srv.Upload(context.Background(), "user1", header, dto.UploadModuleRequest{Title: "Test"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gagal menyimpan modul")
		assert.Nil(t, resp)
	})

	t.Run("Sukses upload", func(t *testing.T) {
		srv, mockRepo, mockR2, mockAI := setupModuleServiceTest(t)
		header := createMultipartFileHeader(t, "test.pdf", minimalPDF)

		req := dto.UploadModuleRequest{Title: "My Module"}

		mockR2.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, "application/pdf").
			Return("https://r2/file.pdf", nil)
		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Module")).
			Return(nil).
			Run(func(args mock.Arguments) {
				m := args.Get(1).(*domain.Module)
				m.ID = "mod-1"
			})

		// Menunggu goroutine selesai
		var wg sync.WaitGroup
		wg.Add(1)

		mockAI.On("Summarize", mock.Anything).
			Return(&pkgai.SummarizeOutput{Summary: "This is a summary"}, nil)
		mockRepo.On("UpdateSummary", mock.Anything, "mod-1", "This is a summary").
			Return(nil).
			Run(func(args mock.Arguments) {
				wg.Done()
			})

		resp, err := srv.Upload(context.Background(), "user1", header, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, req.Title, resp.Title)
		assert.Equal(t, "https://r2/file.pdf", resp.FileURL)
		assert.False(t, resp.IsSummarized) // Waktu response awal masih false, async summarize baru jalan

		// Tunggu goroutine (maks 2 detik agar test tak nyangkut)
		c := make(chan struct{})
		go func() {
			wg.Wait()
			close(c)
		}()
		select {
		case <-c:
			// Berhasil, UpdateSummary dipanggil
		case <-time.After(2 * time.Second):
			t.Fatal("Goroutine summarize tidak selesai tepat waktu")
		}

		mockRepo.AssertExpectations(t)
		mockR2.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})
}

func TestModuleService_GetAll(t *testing.T) {
	t.Run("User tidak punya modul", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		mockRepo.On("FindByUserID", mock.Anything, "user1").Return([]domain.Module{}, nil)

		resp, err := srv.GetAll(context.Background(), "user1")

		assert.NoError(t, err)
		assert.Empty(t, resp)
	})

	t.Run("User punya beberapa modul", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		modules := []domain.Module{
			{Title: "M1", FileURL: "url1"},
			{Title: "M2", FileURL: "url2"},
		}
		mockRepo.On("FindByUserID", mock.Anything, "user1").Return(modules, nil)

		resp, err := srv.GetAll(context.Background(), "user1")

		assert.NoError(t, err)
		assert.Len(t, resp, 2)
		assert.Equal(t, "M1", resp[0].Title)
	})

	t.Run("DB error", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		mockRepo.On("FindByUserID", mock.Anything, "user1").Return(nil, assert.AnError)

		resp, err := srv.GetAll(context.Background(), "user1")

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestModuleService_GetByID(t *testing.T) {
	t.Run("Modul tidak ditemukan", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		mockRepo.On("FindByID", mock.Anything, "mod1").Return(nil, nil)

		resp, err := srv.GetByID(context.Background(), "user1", "mod1")

		assert.Error(t, err)
		assert.Equal(t, service.ErrModuleNotFound, err)
		assert.Nil(t, resp)
	})

	t.Run("Modul milik user lain", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		mockRepo.On("FindByID", mock.Anything, "mod1").Return(&domain.Module{UserID: "other-user"}, nil)

		resp, err := srv.GetByID(context.Background(), "user1", "mod1")

		assert.Error(t, err)
		assert.Equal(t, service.ErrNotModuleOwner, err)
		assert.Nil(t, resp)
	})

	t.Run("Sukses, summary belum ada", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		mod := &domain.Module{UserID: "user1", IsSummarized: false, Title: "T1"}
		mockRepo.On("FindByID", mock.Anything, "mod1").Return(mod, nil)

		resp, err := srv.GetByID(context.Background(), "user1", "mod1")

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.IsSummarized)
		assert.Equal(t, "T1", resp.Title)
	})

	t.Run("Sukses, summary sudah ada", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		mod := &domain.Module{UserID: "user1", IsSummarized: true, Summary: "Summ", Title: "T1"}
		mockRepo.On("FindByID", mock.Anything, "mod1").Return(mod, nil)

		resp, err := srv.GetByID(context.Background(), "user1", "mod1")

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.IsSummarized)
		assert.Equal(t, "Summ", resp.Summary)
	})
}

func TestModuleService_Delete(t *testing.T) {
	t.Run("Modul tidak ditemukan", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		mockRepo.On("FindByID", mock.Anything, "mod1").Return(nil, nil)

		err := srv.Delete(context.Background(), "user1", "mod1")

		assert.Error(t, err)
		assert.Equal(t, service.ErrModuleNotFound, err)
	})

	t.Run("Modul milik user lain", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		mockRepo.On("FindByID", mock.Anything, "mod1").Return(&domain.Module{UserID: "other-user"}, nil)

		err := srv.Delete(context.Background(), "user1", "mod1")

		assert.Error(t, err)
		assert.Equal(t, service.ErrNotModuleOwner, err)
	})

	t.Run("Sukses delete", func(t *testing.T) {
		srv, mockRepo, _, _ := setupModuleServiceTest(t)

		mockRepo.On("FindByID", mock.Anything, "mod1").Return(&domain.Module{UserID: "user1"}, nil)
		mockRepo.On("Delete", mock.Anything, "mod1").Return(nil)

		err := srv.Delete(context.Background(), "user1", "mod1")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
