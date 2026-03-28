package dto

// --- Request ---

type RegisterRequest struct {
	Name     string `json:"name"     validate:"required,min=2,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type VerifyOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp"   validate:"required,len=6"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UpdateEmailRequest struct {
	NewEmail string `json:"new_email" validate:"required,email"`
}

type VerifyUpdateEmailRequest struct {
	NewEmail string `json:"new_email" validate:"required,email"`
	OTP      string `json:"otp"       validate:"required,len=6"`
}

type ResendOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// --- Response ---

type UserResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	QuizQuota      int    `json:"quiz_quota"`
	SummarizeQuota int    `json:"summarize_quota"`
	IsVerified     bool   `json:"is_verified"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
