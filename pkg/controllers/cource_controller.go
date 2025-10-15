package controllers

import (
	"buildANDfly/pkg/dtos"
	"buildANDfly/pkg/service"
	. "github.com/KoNekoD/swaglay/pkg/adapters/swaglay_fiber"
	"github.com/gofiber/fiber/v3"
	"strconv"

	"github.com/google/uuid"
)

type CourseController struct {
	service *service.CourseService
}

func NewCourseController(s *service.CourseService) *CourseController {
	return &CourseController{service: s}
}

func (c *CourseController) RegisterRoutes(app *fiber.App) {
	const api = "courses"
	Fiber = app.Group("/api/courses")
	GetI(api, "/", c.ListCoursesHandler, "list courses")
	GetI(api, "/:id", c.GetCourseHandler, "get course by id")
	PostI(api, "/:id/lessons/:lessonid/complete", c.MarkLessonCompleteHandler, "mark lesson complete")
	GetI(api, "/:id/progress", c.GetCourseProgressHandler, "get course progress(for current user)")
}

func (c *CourseController) ListCoursesHandler(_ *dtos.CreateCourseDTO, ctx fiber.Ctx) error {
	limitStr := ctx.Query("limit", "20")
	offsetStr := ctx.Query("offset", "0")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	courses, err := c.service.ListCourses(ctx.Context(), limit, offset)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot list courses"})
	}
	return ctx.Status(fiber.StatusOK).JSON(courses)
}

func (c *CourseController) GetCourseHandler(_ *dtos.CreateCourseDTO, ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	course, err := c.service.GetCourse(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "course not found"})
	}
	return ctx.Status(fiber.StatusOK).JSON(course)
}

func (c *CourseController) MarkLessonCompleteHandler(_ *dtos.MarkLessonCompleteDTO, ctx fiber.Ctx) error {
	// userID кладём в middleware как string
	userIDRaw := ctx.Locals("userID")
	userIDStr, ok := userIDRaw.(string)
	if !ok || userIDStr == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not authorized"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user id"})
	}

	courseID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid course id"})
	}
	lessonID, err := uuid.Parse(ctx.Params("lessonid"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lesson id"})
	}

	if err := c.service.MarkLessonComplete(ctx.Context(), userID, courseID, lessonID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot mark complete"})
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

func (c *CourseController) GetCourseProgressHandler(_ *dtos.CreateCourseDTO, ctx fiber.Ctx) error {
	userIDRaw := ctx.Locals("userID")
	userIDStr, ok := userIDRaw.(string)
	if !ok || userIDStr == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not authorized"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user id"})
	}

	courseID, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid course id"})
	}

	completed, total, err := c.service.GetCourseProgress(ctx.Context(), userID, courseID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot get progress"})
	}
	percent := 0
	if total > 0 {
		percent = (completed * 100) / total
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"completed": completed, "total": total, "percent": percent})
}
