package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
	"backend-pretest-ai/pkg/response"
)

type SummaryHandler struct {
	summaryService service.SummaryServiceContract
	validate       *validator.Validate
}

func NewSummaryHandler(summaryService service.SummaryServiceContract) *SummaryHandler {
	return &SummaryHandler{
		summaryService: summaryService,
		validate:       validator.New(),
	}
}

// GET /api/v1/summary/:moduleId
func (h *SummaryHandler) GetByModuleID(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	moduleID := c.Params("moduleId")

	result, err := h.summaryService.GetByModuleID(c.Context(), userID, moduleID)
	if err != nil {
		if errors.Is(err, service.ErrModuleNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotModuleOwner) {
			return response.Unauthorized(c, err.Error())
		}
		if errors.Is(err, service.ErrSummaryNotReady) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "gagal mengambil summary")
	}

	return response.OK(c, "berhasil", result)
}

// PUT /api/v1/summary/:moduleId
func (h *SummaryHandler) UpdateManual(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	moduleID := c.Params("moduleId")

	var req dto.UpdateSummaryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "format request tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	result, err := h.summaryService.UpdateManual(c.Context(), userID, moduleID, req)
	if err != nil {
		if errors.Is(err, service.ErrModuleNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotModuleOwner) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "gagal update summary")
	}

	return response.OK(c, "summary berhasil diperbarui", result)
}
