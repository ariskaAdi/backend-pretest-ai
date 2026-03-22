package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
	"backend-pretest-ai/pkg/response"
)

type ModuleHandler struct {
	moduleService service.ModuleServiceContract
	validate      *validator.Validate
}

func NewModuleHandler(moduleService service.ModuleServiceContract) *ModuleHandler {
	return &ModuleHandler{
		moduleService: moduleService,
		validate:      validator.New(),
	}
}

// POST /api/v1/modules
func (h *ModuleHandler) Upload(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	// Parse form data
	var req dto.UploadModuleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "format request tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	// Ambil file dari multipart
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "file PDF wajib disertakan (field: file)")
	}

	result, err := h.moduleService.Upload(c.Context(), userID, fileHeader, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidFileType) ||
			errors.Is(err, service.ErrFileTooLarge) ||
			errors.Is(err, service.ErrPDFNoText) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "gagal mengupload modul")
	}

	return response.Created(c, "modul berhasil diupload, proses ringkasan sedang berjalan", result)
}

// GET /api/v1/modules
func (h *ModuleHandler) GetAll(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	modules, err := h.moduleService.GetAll(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "gagal mengambil daftar modul")
	}

	return response.OK(c, "berhasil", modules)
}

// GET /api/v1/modules/:id
func (h *ModuleHandler) GetByID(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	moduleID := c.Params("id")

	module, err := h.moduleService.GetByID(c.Context(), userID, moduleID)
	if err != nil {
		if errors.Is(err, service.ErrModuleNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotModuleOwner) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "gagal mengambil modul")
	}

	return response.OK(c, "berhasil", module)
}

// DELETE /api/v1/modules/:id
func (h *ModuleHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	moduleID := c.Params("id")

	if err := h.moduleService.Delete(c.Context(), userID, moduleID); err != nil {
		if errors.Is(err, service.ErrModuleNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotModuleOwner) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "gagal menghapus modul")
	}

	return response.OK(c, "modul berhasil dihapus", nil)
}
