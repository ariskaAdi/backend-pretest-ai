package domain_test

import (
	"testing"
	"backend-pretest-ai/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestUserStruct(t *testing.T) {
	user := domain.User{
		ID:    "123",
		Name:  "Test User",
		Email: "test@example.com",
		Role:  domain.RoleMember,
	}

	assert.Equal(t, "123", user.ID)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, domain.RoleMember, user.Role)
}
