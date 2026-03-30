package middleware

import (
	jwtpkg "backend-pretest-ai/pkg/jwt"
	"backend-pretest-ai/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// Auth — middleware validasi JWT dari header Authorization
func Auth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			return response.Unauthorized(c, "missing authorization token")
		}

		tokenStr := authHeader[7:]
		claims, err := jwtpkg.Parse(tokenStr)
		if err != nil {
			return response.Unauthorized(c, "invalid or expired token")
		}

		// Simpan claims ke locals agar bisa diakses handler
		c.Locals("userID", claims.UserID)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

// RequireRole — middleware cek role spesifik
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok {
			return response.Unauthorized(c, "role not found")
		}

		for _, r := range roles {
			if role == r {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "access denied: insufficient role",
		})
	}
}
