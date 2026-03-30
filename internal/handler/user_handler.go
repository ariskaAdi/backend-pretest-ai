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
	userService service.UserService
	validate    *validator.Validate
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		validate:    validator.New(),
	}
}

// POST /api/v1/auth/register
// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account and send OTP to email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RegisterRequest  true  "Registration info"
// @Success      201      {object}  response.APIResponse
// @Failure      400      {object}  response.APIResponse
// @Failure      500      {object}  response.APIResponse
// @Router       /auth/register [post]
func (h *UserHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request format")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	if err := h.userService.Register(c.Context(), req); err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "registration failed")
	}

	return response.Created(c, "registration successful, check your email for OTP verification", nil)
}

// POST /api/v1/auth/verify-otp
// VerifyOTP godoc
// @Summary      Verify OTP for registration
// @Description  Verify the OTP sent to user email during registration
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.VerifyOTPRequest  true  "OTP verification info"
// @Success      200      {object}  response.APIResponse
// @Failure      400      {object}  response.APIResponse
// @Failure      404      {object}  response.APIResponse
// @Failure      500      {object}  response.APIResponse
// @Router       /auth/verify-otp [post]
func (h *UserHandler) VerifyOTP(c *fiber.Ctx) error {
	var req dto.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request format")
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
		return response.InternalError(c, "OTP verification failed")
	}

	return response.OK(c, "email verified successfully, you can now log in", nil)
}

// POST /api/v1/auth/login
// Login godoc
// @Summary      Login user
// @Description  Authenticate user and return JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequest  true  "Login credentials"
// @Success      200      {object}  response.APIResponse{data=dto.LoginResponse}
// @Failure      400      {object}  response.APIResponse
// @Failure      401      {object}  response.APIResponse
// @Failure      500      {object}  response.APIResponse
// @Router       /auth/login [post]
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request format")
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
		return response.InternalError(c, "login failed")
	}

	return response.OK(c, "login successful", result)
}

// POST /api/v1/auth/logout
// Stateless JWT — logout cukup di client dengan hapus token
// Kalau nanti butuh blacklist, bisa tambah redis di sini
func (h *UserHandler) Logout(c *fiber.Ctx) error {
	return response.OK(c, "logout successful", nil)
}

// GET /api/v1/user/me
// GetMe godoc
// @Summary      Get current user profile
// @Description  Get the profile data of the currently logged-in user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.APIResponse{data=dto.UserResponse}
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Router       /user/me [get]
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	user, err := h.userService.GetMe(c.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalError(c, "failed to retrieve user data")
	}

	return response.OK(c, "success", user)
}

// POST /api/v1/auth/resend-otp
// ResendOTP godoc
// @Summary      Resend OTP
// @Description  Resend verification OTP to user email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ResendOTPRequest  true  "Email for OTP resend"
// @Success      200      {object}  response.APIResponse
// @Failure      400      {object}  response.APIResponse
// @Failure      404      {object}  response.APIResponse
// @Failure      500      {object}  response.APIResponse
// @Router       /auth/resend-otp [post]
func (h *UserHandler) ResendOTP(c *fiber.Ctx) error {
	var req dto.ResendOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request format")
	}
	if err := h.validate.Struct(req); err != nil {
		return response.BadRequest(c, formatValidationError(err))
	}

	if err := h.userService.ResendOTP(c.Context(), req); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, service.ErrAlreadyVerified) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "failed to resend OTP")
	}

	return response.OK(c, "new OTP sent to your email", nil)
}

// POST /api/v1/user/email/request-update
// RequestUpdateEmail godoc
// @Summary      Request email update
// @Description  Send OTP to new email for verification
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.UpdateEmailRequest  true  "New email info"
// @Success      200      {object}  response.APIResponse
// @Failure      400      {object}  response.APIResponse
// @Failure      404      {object}  response.APIResponse
// @Failure      500      {object}  response.APIResponse
// @Router       /user/email/request-update [post]
func (h *UserHandler) RequestUpdateEmail(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req dto.UpdateEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request format")
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
		return response.InternalError(c, "failed to request email update")
	}

	return response.OK(c, "OTP sent to your new email", nil)
}

// POST /api/v1/user/email/verify-update
// VerifyUpdateEmail godoc
// @Summary      Verify email update
// @Description  Verify OTP and update user email
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.VerifyUpdateEmailRequest  true  "OTP verification for new email"
// @Success      200      {object}  response.APIResponse
// @Failure      400      {object}  response.APIResponse
// @Failure      404      {object}  response.APIResponse
// @Failure      500      {object}  response.APIResponse
// @Router       /user/email/verify-update [post]
func (h *UserHandler) VerifyUpdateEmail(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req dto.VerifyUpdateEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request format")
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
		return response.InternalError(c, "failed to update email")
	}

	return response.OK(c, "email updated successfully", nil)
}

// formatValidationError — ambil pesan error pertama dari validator
func formatValidationError(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		field := ve[0]
		switch field.Tag() {
		case "required":
			return field.Field() + " is required"
		case "email":
			return field.Field() + " must be a valid email"
		case "min":
			return field.Field() + " must be at least " + field.Param() + " characters"
		case "max":
			return field.Field() + " must be at most " + field.Param() + " characters"
		case "len":
			return field.Field() + " must be exactly " + field.Param() + " characters"
		default:
			return field.Field() + " is invalid"
		}
	}
	return "validation failed"
}
