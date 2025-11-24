package dtos

import "github.com/google/uuid"

type CreateLessonDTO struct {
	Title           string `json:"title"`
	ContentURL      string `json:"content_url,omitempty"`
	ContentType     string `json:"content_type,omitempty"`
	DurationSeconds int    `json:"duration_seconds,omitempty"`
	Order           int    `json:"order,omitempty"`
}

type CreateModuleDTO struct {
	Title   string            `json:"title"`
	Order   int               `json:"order,omitempty"`
	Lessons []CreateLessonDTO `json:"lessons,omitempty"`
}

type CreateCourseDTO struct {
	Title       string            `json:"title" validate:"required"`
	Description string            `json:"description,omitempty"`
	ImageURL    string            `json:"image_url,omitempty"`
	Modules     []CreateModuleDTO `json:"modules,omitempty"`
}

type UpdateCourseDTO struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}

type MarkLessonCompleteDTO struct {
	CourseID uuid.UUID `json:"course_id"`
	LessonID uuid.UUID `json:"lesson_id"`
}
type GetUserDtos struct {
	jwt string `json:"jwt"`
}

type UserResponse struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Email   string `json:"email"`
	Login   string `json:"login"`
	RoleID  int    `json:"role_id"`
}

type JWTInput struct {
	token string `json:"token"`
}
