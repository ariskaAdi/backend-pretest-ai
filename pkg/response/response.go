package response

import "github.com/gofiber/fiber/v2"

type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func OK(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Created(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func BadRequest(c *fiber.Ctx, err string) error {
	return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
		Success: false,
		Error:   err,
	})
}

func Unauthorized(c *fiber.Ctx, err string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(APIResponse{
		Success: false,
		Error:   err,
	})
}

func NotFound(c *fiber.Ctx, err string) error {
	return c.Status(fiber.StatusNotFound).JSON(APIResponse{
		Success: false,
		Error:   err,
	})
}

func InternalError(c *fiber.Ctx, err string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(APIResponse{
		Success: false,
		Error:   err,
	})
}
