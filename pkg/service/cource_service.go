package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"buildANDfly/pkg/dtos"
	"buildANDfly/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseService struct {
	db      *pgxpool.Pool
	storage StorageService
}

func NewCourseService(db *pgxpool.Pool, storage StorageService) *CourseService {
	return &CourseService{db: db, storage: storage}
}

var ErrNotFound = errors.New("not found")

func (s *CourseService) CreateCourse(ctx context.Context, in dtos.CreateCourseDTO) (*models.Course, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	cID := uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO courses (id, title, description, cover_media_id, created_at, updated_at) VALUES ($1,$2,$3,$4, now(), now())`,
		cID, in.Title, in.Description, nil)
	if err != nil {
		return nil, err
	}

	for _, m := range in.Modules {
		modID := uuid.New()
		_, err = tx.Exec(ctx, `INSERT INTO modules (id, course_id, title, "order", created_at, updated_at) VALUES ($1,$2,$3,$4, now(), now())`,
			modID, cID, m.Title, m.Order)
		if err != nil {
			return nil, err
		}
		for _, l := range m.Lessons {
			lessonID := uuid.New()
			_, err = tx.Exec(ctx, `INSERT INTO lessons (id, module_id, title, content_url, content_type, duration_seconds, "order", created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7, now(), now())`,
				lessonID, modID, l.Title, l.ContentURL, l.ContentType, l.DurationSeconds, l.Order)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetCourse(ctx, cID)
}

func (s *CourseService) GetCourse(ctx context.Context, courseID uuid.UUID) (*models.Course, error) {
	var c models.Course
	row := s.db.QueryRow(ctx, `SELECT id, title, description, cover_media_id, created_at, updated_at FROM courses WHERE id = $1`, courseID)
	var coverMediaID *uuid.UUID
	if err := row.Scan(&c.ID, &c.Title, &c.Description, &coverMediaID, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, ErrNotFound
	}

	// load modules and lessons (modules and lessons tables must exist)
	rows, err := s.db.Query(ctx, `SELECT id, title, "order", created_at, updated_at FROM modules WHERE course_id = $1 ORDER BY "order" ASC`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []models.Module
	for rows.Next() {
		var m models.Module
		if err := rows.Scan(&m.ID, &m.Title, &m.Order, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		lessRows, err := s.db.Query(ctx, `SELECT id, title, content_url, content_type, duration_seconds, "order", created_at, updated_at FROM lessons WHERE module_id = $1 ORDER BY "order" ASC`, m.ID)
		if err != nil {
			return nil, err
		}
		var lessons []models.Lesson
		for lessRows.Next() {
			var l models.Lesson
			if err := lessRows.Scan(&l.ID, &l.Title, &l.ContentURL, &l.ContentType, &l.DurationSeconds, &l.Order, &l.CreatedAt, &l.UpdatedAt); err != nil {
				lessRows.Close()
				return nil, err
			}
			l.ModuleID = m.ID
			lessons = append(lessons, l)
		}
		lessRows.Close()
		m.Lessons = lessons
		m.CourseID = c.ID
		modules = append(modules, m)
	}
	c.Modules = modules
	return &c, nil
}

func (s *CourseService) ListCourses(ctx context.Context, limit, offset int) ([]models.Course, error) {
	rows, err := s.db.Query(ctx, `SELECT id, title, description, cover_media_id, created_at, updated_at FROM courses ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Course, 0)
	for rows.Next() {
		var c models.Course
		// ignore cover_media_id here - front-end can request media separately
		var coverMediaID *uuid.UUID
		if err := rows.Scan(&c.ID, &c.Title, &c.Description, &coverMediaID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *CourseService) MarkLessonComplete(ctx context.Context, userID, courseID, lessonID uuid.UUID) error {
	now := time.Now()
	id := uuid.New()
	_, err := s.db.Exec(ctx, `
		INSERT INTO lesson_progress (id, user_id, course_id, lesson_id, completed, completed_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,true,$5, now(), now())
		ON CONFLICT (user_id, lesson_id) DO UPDATE SET completed = EXCLUDED.completed, completed_at = EXCLUDED.completed_at, updated_at = now()
	`, id, userID, courseID, lessonID, now)
	return err
}

func (s *CourseService) GetCourseProgress(ctx context.Context, userID, courseID uuid.UUID) (completed int, total int, err error) {
	if err = s.db.QueryRow(ctx, `SELECT count(1) FROM lessons l JOIN modules m ON l.module_id = m.id WHERE m.course_id = $1`, courseID).Scan(&total); err != nil {
		return
	}
	if err = s.db.QueryRow(ctx, `SELECT count(1) FROM lesson_progress WHERE user_id = $1 AND course_id = $2 AND completed = true`, userID, courseID).Scan(&completed); err != nil {
		return
	}
	return
}

func (s *CourseService) UpdateCourse(ctx context.Context, courseID uuid.UUID, upd dtos.UpdateCourseDTO) (*models.Course, error) {
	if upd.Title == nil && upd.Description == nil && upd.ImageURL == nil {
		return s.GetCourse(ctx, courseID)
	}

	set := []string{}
	args := []interface{}{}
	idx := 1
	if upd.Title != nil {
		set = append(set, fmt.Sprintf("title = $%d", idx))
		args = append(args, *upd.Title)
		idx++
	}
	if upd.Description != nil {
		set = append(set, fmt.Sprintf("description = $%d", idx))
		args = append(args, *upd.Description)
		idx++
	}
	if upd.ImageURL != nil {
		set = append(set, fmt.Sprintf("cover_media_id = $%d", idx))
		args = append(args, *upd.ImageURL)
		idx++
	}
	if len(set) == 0 {
		return s.GetCourse(ctx, courseID)
	}
	q := "UPDATE courses SET " + strings.Join(set, ", ") + fmt.Sprintf(", updated_at = now() WHERE id = $%d", idx)
	args = append(args, courseID)
	if _, err := s.db.Exec(ctx, q, args...); err != nil {
		return nil, err
	}
	return s.GetCourse(ctx, courseID)
}
