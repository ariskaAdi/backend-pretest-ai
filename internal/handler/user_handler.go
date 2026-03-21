package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/service"
	"backend-pretest-ai/pkg/response"
)

type UserHandler struct {
	userService *service.UserService
	validate    *validator.Validate
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		validate:    validator.New(),
	}
}

// POST /api/v1/auth/register
func (h *UserHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "format request tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	if err := h.userService.Register(c.Context(), req); err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "gagal melakukan registrasi")
	}

	return response.Created(c, "registrasi berhasil, cek email kamu untuk verifikasi OTP", nil)
}

// POST /api/v1/auth/verify-otp
func (h *UserHandler) VerifyOTP(c *fiber.Ctx) error {
	var req dto.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "format request tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	if err := h.userService.VerifyOTP(c.Context(), req); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrInvalidOTP) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "gagal verifikasi OTP")
	}

	return response.OK(c, "email berhasil diverifikasi, silakan login", nil)
}

// POST /api/v1/auth/login
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "format request tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	result, err := h.userService.Login(c.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return response.Unauthorized(c, err.Error())
		}
		if errors.Is(err, service.ErrEmailNotVerified) {
			return response.Unauthorized(c, err.Error())
		}
		return response.InternalError(c, "gagal login")
	}

	return response.OK(c, "login berhasil", result)
}

// POST /api/v1/auth/logout
// Stateless JWT — logout cukup di client dengan hapus token
// Kalau nanti butuh blacklist, bisa tambah redis di sini
func (h *UserHandler) Logout(c *fiber.Ctx) error {
	return response.OK(c, "logout berhasil", nil)
}

// POST /api/v1/user/email/request-update
func (h *UserHandler) RequestUpdateEmail(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req dto.UpdateEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "format request tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	if err := h.userService.RequestUpdateEmail(c.Context(), userID, req); err != nil {
		if errors.Is(err, service.ErrEmailSameAsCurrent) || errors.Is(err, service.ErrNewEmailAlreadyInUse) {
			return response.BadRequest(c, err.Error())
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalError(c, "gagal request update email")
	}

	return response.OK(c, "OTP telah dikirim ke email baru kamu", nil)
}

// POST /api/v1/user/email/verify-update
func (h *UserHandler) VerifyUpdateEmail(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req dto.VerifyUpdateEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "format request tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	if err := h.userService.VerifyUpdateEmail(c.Context(), userID, req); err != nil {
		if errors.Is(err, service.ErrInvalidOTP) || errors.Is(err, service.ErrNewEmailAlreadyInUse) {
			return response.BadRequest(c, err.Error())
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalError(c, "gagal update email")
	}

	return response.OK(c, "email berhasil diperbarui", nil)
}

// formatValidationError — ambil pesan error pertama dari validator
func formatValidationError(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		field := ve[0]
		switch field.Tag() {
		case "required":
			return field.Field() + " wajib diisi"
		case "email":
			return field.Field() + " format email tidak valid"
		case "min":
			return field.Field() + " minimal " + field.Param() + " karakter"
		case "max":
			return field.Field() + " maksimal " + field.Param() + " karakter"
		case "len":
			return field.Field() + " harus " + field.Param() + " karakter"
		}
	}
	return "validasi gagal"
}
