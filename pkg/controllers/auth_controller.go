package controllers

import (
	"buildANDfly/pkg/dtos"
	"buildANDfly/pkg/models"
	"buildANDfly/pkg/service"
	. "github.com/KoNekoD/swaglay/pkg/adapters/swaglay_fiber"
	"github.com/gofiber/fiber/v3"
	"strconv"
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
	GetO(api, "/user-by_jwt", c.GetUserHandler, "user-by-jwt")

}

func (c *AuthController) RegisterController(i *dtos.Registerdtos, ctx fiber.Ctx) error {
	if err := ctx.Bind().Body(&i); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	user, err := c.service.Register(ctx.Context(), *i)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(user)
}

func (c *AuthController) LoginController(i *dtos.LoginRequest, ctx fiber.Ctx) error {
	if err := ctx.Bind().Body(&i); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	token, user, exp, err := c.service.Login(ctx.Context(), i.Login, i.Password)
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

func (c *AuthController) GetUserHandler(ctx fiber.Ctx) (*models.User, error) {
	userIDVal := ctx.Locals("userID")
	var userID int

	switch v := userIDVal.(type) {
	case int:
		userID = v
	case int64:
		userID = int(v)
	case float64:
		userID = int(v)
	case string:
		// If it's a string, try to convert it
		if parsed, err := strconv.Atoi(v); err == nil {
			userID = parsed
		} else {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "User not authenticated")
		}
	default:
		return nil, fiber.NewError(fiber.StatusUnauthorized, "User not authenticated")
	}

	user, err := c.service.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	return user, nil
}
