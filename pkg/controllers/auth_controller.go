package controllers

import (
	"buildANDfly/pkg/dtos"
	"buildANDfly/pkg/service"
	. "github.com/KoNekoD/swaglay/pkg/adapters/swaglay_fiber"
	"github.com/gofiber/fiber/v3"
)

type AuthController struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthController {
	return &AuthController{service: service}
}

func (c *AuthController) RegisterRoutes(app *fiber.App) {
	const api = "auth"
	Fiber = app.Group("/api/auth")
	PostI(api, "/reg", c.RegisterController, "register user")
	PostI(api, "/login", c.LoginController, "login")

}

func (c *AuthController) RegisterController(i *dtos.Registerdtos, ctx fiber.Ctx) error {
	if err := ctx.Bind().Body(&i); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	user, err := c.service.Register(ctx, *i)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Registration failed"})
	}

	return ctx.Status(fiber.StatusCreated).JSON(user)
}
func (c *AuthController) LoginController(i *dtos.LoginRequest, ctx fiber.Ctx) error {
	if err := ctx.Bind().Body(&i); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	token, user, exp, err := c.service.Login(ctx, i.Login, i.Password)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	ctx.Cookie(&fiber.Cookie{
		Expires:  exp,
		Name:     "api_token",
		Value:    token,
		Path:     "/",
		Domain:   "localhost",
		SameSite: fiber.CookieSameSiteLaxMode,
		Secure:   false,
		HTTPOnly: true,
	})

	return ctx.Status(fiber.StatusOK).JSON(user)
}
