package controllers

import (
	"buildANDfly/pkg/dtos"
	"buildANDfly/pkg/middleware"
	"buildANDfly/pkg/service"
	"context"
	"fmt"
	. "github.com/KoNekoD/swaglay/pkg/adapters/swaglay_fiber"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
)

type AdminController struct {
	authService   *service.AuthService
	courseService *service.CourseService
	storage       service.StorageService
	pool          *pgxpool.Pool
}

func NewAdminController(auth *service.AuthService, cs *service.CourseService, storage service.StorageService, pool *pgxpool.Pool) *AdminController {
	return &AdminController{authService: auth, courseService: cs, storage: storage, pool: pool}
}

func (c *AdminController) RegisterRoutes(app *fiber.App) {
	const api = "admin"
	// регистрируем в своей группе
	Fiber = app.Group("/api/admin")
	// подключаем middleware аутентификации к admin группе
	Fiber.Use(func(ctx fiber.Ctx) error {
		// используем ваш middleware напрямую, он требует authService
		return middleware.AuthMiddleware(c.authService)(ctx)
	})

	PostI(api, "/roles/assign", c.AssignRoleHandler, "assign role")
	PostI(api, "/courses", c.CreateCourseHandler, "create course (admin)")
	PutI(api, "/courses/:id", c.UpdateCourseHandler, "update course (admin)")
	PostI(api, "/uploads", c.UploadHandler, "upload file to storage (admin)")
}

// assign role: body {"user_id":"...", "role":"admin"}
type assignRoleReq struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func (c *AdminController) AssignRoleHandler(i *assignRoleReq, ctx fiber.Ctx) error {
	if err := ctx.Bind().Body(&i); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	uid, err := uuid.Parse(i.UserID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	if err := c.authService.SetUserRole(ctx.Context(), uid, i.Role); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot set role"})
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

func (c *AdminController) CreateCourseHandler(i *dtos.CreateCourseDTO, ctx fiber.Ctx) error {
	if err := ctx.Bind().Body(&i); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	// проверка роли (middleware уже ставит userRole, но дополнительная проверка не помешает)
	if ctx.Locals("userRole") != "admin" {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	course, err := c.courseService.CreateCourse(ctx.Context(), *i)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusCreated).JSON(course)
}

func (c *AdminController) UpdateCourseHandler(i *dtos.UpdateCourseDTO, ctx fiber.Ctx) error {
	if err := ctx.Bind().Body(&i); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if ctx.Locals("userRole") != "admin" {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	course, err := c.courseService.UpdateCourse(ctx.Context(), id, *i)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(fiber.StatusOK).JSON(course)
}

// UploadHandler: multipart/form-data. Используем swaglay signature: (_ *AnyDTO, ctx fiber.Ctx) error
func (c *AdminController) UploadHandler(_ *dtos.CreateCourseDTO, ctx fiber.Ctx) error {
	// проверим роль
	if ctx.Locals("userRole") != "admin" {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file required"})
	}

	f, err := fileHeader.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot open file"})
	}
	defer f.Close()

	mediaType := ctx.FormValue("media_type", "file")
	courseIDStr := ctx.FormValue("course_id", "")
	isCoverStr := ctx.FormValue("is_cover", "false")
	isCover := false
	if b, _ := strconv.ParseBool(isCoverStr); b {
		isCover = true
	}

	mediaID := uuid.New()
	key := fmt.Sprintf("media/%s_%s", mediaID.String(), fileHeader.Filename)

	// Upload to storage
	cdnURL, err := c.storage.Upload(ctx.Context(), key, f, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "upload failed", "detail": err.Error()})
	}

	var courseID *uuid.UUID
	if courseIDStr != "" {
		if parsed, err := uuid.Parse(courseIDStr); err == nil {
			courseID = &parsed
		}
	}

	size := int64(0)
	if fileHeader.Size > 0 {
		size = fileHeader.Size
	}

	_, err = c.pool.Exec(context.Background(), `INSERT INTO media (id, course_id, media_type, storage_key, cdn_url, mime_type, size_bytes, is_cover, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now(), now())`,
		mediaID, courseID, mediaType, key, cdnURL, fileHeader.Header.Get("Content-Type"), size, isCover)
	if err != nil {
		_ = c.storage.Delete(context.Background(), key)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db insert failed", "detail": err.Error()})
	}

	if isCover && courseID != nil {
		_, _ = c.pool.Exec(context.Background(), `UPDATE courses SET cover_media_id = $1, updated_at = now() WHERE id = $2`, mediaID, *courseID)
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         mediaID,
		"cdn_url":    cdnURL,
		"storageKey": key,
		"media_type": mediaType,
		"size_bytes": size,
	})
}
