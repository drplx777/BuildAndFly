package dtos

import "github.com/google/uuid"

// Create course + nested modules + lessons
type CreateLessonDTO struct {
	Title           string `json:"title"`
	ContentURL      string `json:"content_url,omitempty"` // can be uploaded separately and set
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

// Update DTOs (partial)
type UpdateCourseDTO struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}

// For marking lesson completed:
type MarkLessonCompleteDTO struct {
	CourseID uuid.UUID `json:"course_id"`
	LessonID uuid.UUID `json:"lesson_id"`
}
