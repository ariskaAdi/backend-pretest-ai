package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
	"backend-pretest-ai/pkg/response"
)

type ReviewHandler struct {
	reviewService service.ReviewService
	validate      *validator.Validate
}

func NewReviewHandler(reviewService service.ReviewService) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
		validate:    validator.New(),
	}
}

// POST /api/v1/reviews
func (h *ReviewHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req dto.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request format")
	}

	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	result, err := h.reviewService.Create(c.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFoundReview) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrReviewAlreadyExist) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "failed to submit review")
	}

	return response.Created(c, "review submitted successfully", result)
}

// GET /api/v1/reviews
func (h *ReviewHandler) GetAll(c *fiber.Ctx) error {
	reviews, err := h.reviewService.GetAll(c.Context())
	if err != nil {
		return response.InternalError(c, "failed to retrieve reviews")
	}

	return response.OK(c, "success", reviews)
}

// PUT /api/v1/reviews/:id
func (h *ReviewHandler) Update(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	reviewID := c.Params("id")

	var req dto.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request format")
	}

	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	result, err := h.reviewService.Update(c.Context(), userID, reviewID, req)
	if err != nil {
		if errors.Is(err, service.ErrReviewNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotReviewOwner) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "failed to update review")
	}

	return response.OK(c, "review updated successfully", result)
}

// DELETE /api/v1/reviews/:id
func (h *ReviewHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	reviewID := c.Params("id")

	err := h.reviewService.Delete(c.Context(), userID, reviewID)
	if err != nil {
		if errors.Is(err, service.ErrReviewNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrNotReviewOwner) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "failed to delete review")
	}

	return response.OK(c, "review deleted successfully", nil)
}
