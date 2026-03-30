package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

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
// Upload godoc
// @Summary      Upload a new module PDF
// @Description  Upload a PDF file to be parsed and summarized asynchronously
// @Tags         modules
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        title  formData  string  true  "Module title"
// @Param        file   formData  file    true  "PDF file"
// @Success      201    {object}  response.APIResponse{data=dto.ModuleResponse}
// @Failure      400    {object}  response.APIResponse
// @Failure      500    {object}  response.APIResponse
// @Router       /modules [post]
func (h *ModuleHandler) Upload(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	// Parse form data
	var req dto.UploadModuleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request format")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	// Ambil file dari multipart
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "PDF file is required (field: file)")
	}

	result, err := h.moduleService.Upload(c.Context(), userID, fileHeader, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidFileType) ||
			errors.Is(err, service.ErrFileTooLarge) ||
			errors.Is(err, service.ErrPDFNoText) ||
			errors.Is(err, service.ErrInsufficientSummarizeQuota) {
			return response.BadRequest(c, err.Error())
		}
		// Untuk error lain (500), kembalikan ke Fiber agar dicatat oleh LoggerMiddleware
		return err
	}

	return response.Created(c, "module uploaded successfully, summarization in progress", result)
}

// GET /api/v1/modules
// GetAll godoc
// @Summary      Get all modules
// @Description  Get all modules belonging to the authenticated user
// @Tags         modules
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse{data=[]dto.ModuleResponse}
// @Failure      500  {object}  response.APIResponse
// @Router       /modules [get]
func (h *ModuleHandler) GetAll(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	modules, err := h.moduleService.GetAll(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "failed to retrieve module list")
	}

	return response.OK(c, "success", modules)
}

// GET /api/v1/modules/:id
// GetByID godoc
// @Summary      Get module details
// @Description  Get specific module details including its AI summary
// @Tags         modules
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Module ID"
// @Success      200  {object}  response.APIResponse{data=dto.ModuleDetailResponse}
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Router       /modules/{id} [get]
func (h *ModuleHandler) GetByID(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	moduleID := c.Params("id")

	if _, err := uuid.Parse(moduleID); err != nil {
		return response.NotFound(c, "module not found")
	}

	module, err := h.moduleService.GetByID(c.Context(), userID, moduleID)
	if err != nil {
		if errors.Is(err, service.ErrModuleNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotModuleOwner) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "failed to retrieve module")
	}

	return response.OK(c, "success", module)
}

// DELETE /api/v1/modules/:id
// Delete godoc
// @Summary      Delete a module
// @Description  Delete a specific module and its history
// @Tags         modules
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Module ID"
// @Success      200  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Router       /modules/{id} [delete]
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
		return response.InternalError(c, "failed to delete module")
	}

	return response.OK(c, "module deleted successfully", nil)
}

// POST /api/v1/modules/:id/retry-summarize
func (h *ModuleHandler) RetrySummarize(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	moduleID := c.Params("id")

	if err := h.moduleService.RetrySummarize(c.Context(), userID, moduleID); err != nil {
		if errors.Is(err, service.ErrModuleNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotModuleOwner) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "failed to restart summarization")
	}

	return response.OK(c, "summarization restarted successfully", nil)
}

