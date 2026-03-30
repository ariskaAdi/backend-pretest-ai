package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"

	"golang.org/x/crypto/bcrypt"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/repository"
	jwtpkg "backend-pretest-ai/pkg/jwt"
	"backend-pretest-ai/pkg/mailer"
)

var (
	ErrEmailAlreadyExists   = errors.New("email already registered")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidOTP           = errors.New("invalid OTP")
	ErrEmailNotVerified     = errors.New("email not verified, please check your inbox")
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrEmailSameAsCurrent   = errors.New("new email must be different from current email")
	ErrNewEmailAlreadyInUse = errors.New("new email is already in use")
	ErrAlreadyVerified      = errors.New("email is already verified")
)

type UserService interface {
	Register(ctx context.Context, req dto.RegisterRequest) error
	VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest) error
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	RequestUpdateEmail(ctx context.Context, userID string, req dto.UpdateEmailRequest) error
	VerifyUpdateEmail(ctx context.Context, userID string, req dto.VerifyUpdateEmailRequest) error
	GetMe(ctx context.Context, userID string) (*dto.UserResponse, error)
	ResendOTP(ctx context.Context, req dto.ResendOTPRequest) error
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

// Register — simpan user baru (belum verified), kirim OTP via goroutine
func (s *userService) Register(ctx context.Context, req dto.RegisterRequest) error {
	// Validasi: email sudah ada?
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrEmailAlreadyExists
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate OTP
	otp, err := generateOTP()
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	user := &domain.User{
		Name:       req.Name,
		Email:      req.Email,
		Password:   string(hashed),
		Role:       domain.RoleGuest,
		OTP:        otp,
		IsVerified: false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	// Kirim OTP lewat email — goroutine, tidak block response
	go func() {
		if err := mailer.Client.SendOTP(user.Email, otp); err != nil {
			log.Printf("[user_service] gagal kirim OTP ke %s: %v", user.Email, err)
		}
	}()

	return nil
}

// VerifyOTP — verifikasi OTP saat register
func (s *userService) VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest) error {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.OTP == "" || user.OTP != req.OTP {
		return ErrInvalidOTP
	}

	return s.userRepo.VerifyUser(ctx, user.ID)
}

// Login — validasi credential, return JWT
func (s *userService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Cek apakah email sudah diverifikasi
	if !user.IsVerified {
		return nil, ErrEmailNotVerified
	}

	// Cek password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate JWT
	token, err := jwtpkg.Generate(user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:         user.ID,
			Name:       user.Name,
			Email:      user.Email,
			Role:           string(user.Role),
			QuizQuota:      user.QuizQuota,
			SummarizeQuota: user.SummarizeQuota,
			IsVerified:     user.IsVerified,
		},
	}, nil
}

// RequestUpdateEmail — kirim OTP ke email baru untuk verifikasi
func (s *userService) RequestUpdateEmail(ctx context.Context, userID string, req dto.UpdateEmailRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Validasi: email baru tidak sama dengan yang lama
	if user.Email == req.NewEmail {
		return ErrEmailSameAsCurrent
	}

	// Validasi: email baru belum dipakai akun lain
	existing, err := s.userRepo.FindByEmail(ctx, req.NewEmail)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrNewEmailAlreadyInUse
	}

	// Generate OTP baru
	otp, err := generateOTP()
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Simpan OTP ke user yang sedang login
	if err := s.userRepo.UpdateOTP(ctx, userID, otp); err != nil {
		return err
	}

	// Kirim OTP ke email BARU — goroutine
	go func() {
		if err := mailer.Client.SendOTP(req.NewEmail, otp); err != nil {
			log.Printf("[user_service] gagal kirim OTP update email ke %s: %v", req.NewEmail, err)
		}
	}()

	return nil
}

// VerifyUpdateEmail — konfirmasi OTP lalu update email
func (s *userService) VerifyUpdateEmail(ctx context.Context, userID string, req dto.VerifyUpdateEmailRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.OTP == "" || user.OTP != req.OTP {
		return ErrInvalidOTP
	}

	// Cek lagi email baru belum dipakai (edge case: ada user lain daftar di antara request & verify)
	existing, err := s.userRepo.FindByEmail(ctx, req.NewEmail)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrNewEmailAlreadyInUse
	}

	return s.userRepo.UpdateEmail(ctx, userID, req.NewEmail)
}

// GetMe — ambil profil user yang sedang login
func (s *userService) GetMe(ctx context.Context, userID string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return &dto.UserResponse{
		ID:         user.ID,
		Name:       user.Name,
		Email:      user.Email,
		Role:           string(user.Role),
		QuizQuota:      user.QuizQuota,
		SummarizeQuota: user.SummarizeQuota,
		IsVerified:     user.IsVerified,
	}, nil
}

// ResendOTP — kirim ulang OTP untuk user yang belum verified
func (s *userService) ResendOTP(ctx context.Context, req dto.ResendOTPRequest) error {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	if user.IsVerified {
		return ErrAlreadyVerified
	}

	otp, err := generateOTP()
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	if err := s.userRepo.UpdateOTP(ctx, user.ID, otp); err != nil {
		return err
	}

	// Kirim OTP baru lewat email — goroutine
	go func() {
		if err := mailer.Client.SendOTP(user.Email, otp); err != nil {
			log.Printf("[user_service] gagal kirim ulang OTP ke %s: %v", user.Email, err)
		}
	}()

	return nil
}

// generateOTP — buat 6 digit angka acak yang aman secara kriptografi
func generateOTP() (string, error) {
	const digits = "0123456789"
	otp := make([]byte, 6)
	for i := range otp {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[n.Int64()]
	}
	return string(otp), nil
}
