package middleware

import (
	"buildANDfly/pkg/service"
	"github.com/gofiber/fiber/v3"
	"log/slog"
)

func AuthMiddleware(authService *service.AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Path() == "/api/auth/login" || c.Path() == "/api/auth/reg" || c.Path() == "/health" {
			return c.Next()
		}

		token := c.Cookies("api_token")
		if token == "" {
			slog.Warn("Token not found in cookies")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token not found"})
		}

		claims, err := authService.ValidateToken(token)
		if err != nil {
			slog.Warn("Invalid token", "error", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		userIDStr := ""
		if v, ok := claims["sub"].(string); ok {
			userIDStr = v
		}

		roleStr := ""
		if r, ok := claims["role"].(string); ok {
			roleStr = r
		}

		if userIDStr == "" {
			slog.Warn("sub claim empty")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
		}

		c.Locals("userID", userIDStr)
		c.Locals("userRole", roleStr)

		return c.Next()
	}
}
