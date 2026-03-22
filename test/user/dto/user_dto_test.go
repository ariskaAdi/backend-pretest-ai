package dto_test

import (
	"testing"
	"backend-pretest-ai/internal/dto"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRequest_Validation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name    string
		request dto.RegisterRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			request: dto.RegisterRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "Invalid email",
			request: dto.RegisterRequest{
				Name:     "John Doe",
				Email:    "invalid-email",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "Password too short",
			request: dto.RegisterRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "short",
			},
			wantErr: true,
		},
		{
			name: "Missing name",
			request: dto.RegisterRequest{
				Name:     "",
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoginRequest_Validation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name    string
		request dto.LoginRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			request: dto.LoginRequest{
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "Missing email",
			request: dto.LoginRequest{
				Email:    "",
				Password: "password123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
