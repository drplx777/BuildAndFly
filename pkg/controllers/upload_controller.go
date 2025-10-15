package controllers

import (
	"context"
	"strconv"

	"buildANDfly/pkg/dtos"
	"buildANDfly/pkg/middleware"
	"buildANDfly/pkg/service"

	. "github.com/KoNekoD/swaglay/pkg/adapters/swaglay_fiber"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// UploadController — контроллер только для HTTP-слоя.
type UploadController struct {
	authService  *service.AuthService
	mediaService *service.MediaService
}

// NewUploadController создает контроллер.
func NewUploadController(auth *service.AuthService, mediaSvc *service.MediaService) *UploadController {
	return &UploadController{authService: auth, mediaService: mediaSvc}
}

// RegisterRoutes — регистрирует маршруты в swaglay-стиле; контроллер сам создаёт группу /api/admin и навешивает auth middleware.
func (u *UploadController) RegisterRoutes(app *fiber.App) {
	const api = "admin"
	Fiber = app.Group("/api/admin")
	Fiber.Use(func(ctx fiber.Ctx) error {
		return middleware.AuthMiddleware(u.authService)(ctx)
	})
	PostI(api, "/uploads", u.UploadHandler, "upload file to storage (admin)")
}

func (u *UploadController) UploadHandler(_ *dtos.CreateCourseDTO, ctx fiber.Ctx) error {
	// проверка роли
	roleRaw := ctx.Locals("userRole")
	if roleRaw == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "not authorized"})
	}
	roleStr, _ := roleRaw.(string)
	if roleStr != "admin" {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file required", "detail": err.Error()})
	}

	fileReader, err := fileHeader.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot open file", "detail": err.Error()})
	}

	mediaType := ctx.FormValue("media_type", "file")
	courseIDStr := ctx.FormValue("course_id", "")
	isCoverStr := ctx.FormValue("is_cover", "false")
	isCover := false
	if b, _ := strconv.ParseBool(isCoverStr); b {
		isCover = true
	}

	var courseID *uuid.UUID
	if courseIDStr != "" {
		if parsed, err := uuid.Parse(courseIDStr); err == nil {
			courseID = &parsed
		}
	}

	mediaID, cdnURL, storageKey, size, err := u.mediaService.UploadMedia(context.Background(), fileReader, fileHeader.Filename, fileHeader.Header.Get("Content-Type"), fileHeader.Size, courseID, mediaType, isCover)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "upload failed", "detail": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         mediaID,
		"cdn_url":    cdnURL,
		"storageKey": storageKey,
		"media_type": mediaType,
		"size_bytes": size,
	})
}
