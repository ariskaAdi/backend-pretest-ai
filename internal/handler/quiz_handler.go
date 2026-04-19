package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
	"backend-pretest-ai/pkg/response"
)

type QuizHandler struct {
	quizService service.QuizServiceContract
	validate    *validator.Validate
}

func NewQuizHandler(quizService service.QuizServiceContract) *QuizHandler {
	return &QuizHandler{
		quizService: quizService,
		validate:    validator.New(),
	}
}

// POST /api/v1/quiz
func (h *QuizHandler) Generate(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req dto.GenerateQuizRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "format request tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	result, err := h.quizService.Generate(c.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrModuleNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotModuleOwner) {
			return response.Unauthorized(c, err.Error())
		}
		if errors.Is(err, service.ErrModuleNotSummarized) {
			return response.BadRequest(c, err.Error())
		}
		if errors.Is(err, service.ErrInsufficientQuizQuota) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "gagal generate quiz")
	}

	return response.Created(c, "quiz berhasil dibuat", result)
}

// POST /api/v1/quiz/:id/submit
func (h *QuizHandler) Submit(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	quizID := c.Params("id")

	var req dto.SubmitAnswerRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "format request tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	result, err := h.quizService.Submit(c.Context(), userID, quizID, req)
	if err != nil {
		if errors.Is(err, service.ErrQuizNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotQuizOwner) {
			return response.Unauthorized(c, err.Error())
		}
		if errors.Is(err, service.ErrQuizAlreadyDone) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "gagal submit jawaban")
	}

	return response.OK(c, "quiz berhasil dikumpulkan", result)
}

// GET /api/v1/quiz/history
func (h *QuizHandler) GetHistory(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	history, err := h.quizService.GetHistory(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "gagal mengambil riwayat quiz")
	}

	return response.OK(c, "berhasil", history)
}

// GET /api/v1/quiz/history/module/:moduleId
func (h *QuizHandler) GetHistoryByModule(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	moduleID := c.Params("moduleId")

	history, err := h.quizService.GetHistoryByModule(c.Context(), userID, moduleID)
	if err != nil {
		return response.InternalError(c, "gagal mengambil riwayat quiz")
	}

	return response.OK(c, "berhasil", history)
}

// GET /api/v1/quiz/:id/result
func (h *QuizHandler) GetResult(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	quizID := c.Params("id")

	result, err := h.quizService.GetResult(c.Context(), userID, quizID)
	if err != nil {
		if errors.Is(err, service.ErrQuizNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotQuizOwner) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "gagal mengambil hasil quiz")
	}

	return response.OK(c, "berhasil", result)
}

// DELETE /api/v1/quiz/:id
func (h *QuizHandler) Cancel(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	quizID := c.Params("id")

	err := h.quizService.Cancel(c.Context(), userID, quizID)
	if err != nil {
		if errors.Is(err, service.ErrQuizNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotQuizOwner) {
			return response.Unauthorized(c, err.Error())
		}
		if errors.Is(err, service.ErrQuizCannotBeCancelled) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "gagal membatalkan quiz")
	}

	return response.OK(c, "quiz berhasil dibatalkan dan kuota dikembalikan", nil)
}

// POST /api/v1/quiz/:id/explain
func (h *QuizHandler) Explain(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	quizID := c.Params("id")

	result, err := h.quizService.Explain(c.Context(), userID, quizID)
	if err != nil {
		if errors.Is(err, service.ErrQuizNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotQuizOwner) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "gagal membuat penjelasan")
	}

	return response.OK(c, "penjelasan berhasil dibuat", result)
}

// POST /api/v1/quiz/:id/retry
func (h *QuizHandler) Retry(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	quizID := c.Params("id")

	result, err := h.quizService.Retry(c.Context(), userID, quizID)
	if err != nil {
		if errors.Is(err, service.ErrQuizNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotQuizOwner) {
			return response.Unauthorized(c, err.Error())
		}
		if errors.Is(err, service.ErrModuleNotSummarized) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "gagal membuat ulang quiz")
	}

	return response.Created(c, "quiz baru berhasil dibuat", result)
}
