package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.HealthCheckPeriod = 1 * time.Minute
	config.MaxConnLifetime = 30 * time.Minute

	var pool *pgxpool.Pool
	const maxAttempts = 10
	for i := 0; i < maxAttempts; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				break
			}
		}
		if i < maxAttempts-1 {
			delay := time.Second * time.Duration(i*2)
			log.Printf("Database connection failed (attempt %d/%d), retrying in %v: %v", i+1, maxAttempts, delay, err)
			time.Sleep(delay)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxAttempts, err)
	}

	if err := createTables(ctx, pool); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	log.Printf("Database connection successfully created")
	return pool, nil
}

func createTables(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto;`); err != nil {
		return fmt.Errorf("create extension pgcrypto: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	stmts := []string{
		// roles
		`CREATE TABLE IF NOT EXISTS roles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			role TEXT NOT NULL UNIQUE
		);`,

		// users
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT,
			surname TEXT,
			email TEXT UNIQUE,
			login TEXT UNIQUE,
			password TEXT NOT NULL,
			role_id UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`,

		// courses
		`CREATE TABLE IF NOT EXISTS courses (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title TEXT NOT NULL,
			description TEXT,
			cover_media_id UUID,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`,

		// media
		`CREATE TABLE IF NOT EXISTS media (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			course_id UUID REFERENCES courses(id) ON DELETE CASCADE,
			media_type TEXT NOT NULL,
			storage_key TEXT NOT NULL,
			cdn_url TEXT,
			mime_type TEXT,
			size_bytes BIGINT,
			width INT,
			height INT,
			duration INT,
			is_cover BOOLEAN DEFAULT false,
			parent_id UUID REFERENCES media(id) ON DELETE SET NULL,
			variant_type TEXT,
			meta_data JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`,

		// course_media (ordering / many-to-many)
		`CREATE TABLE IF NOT EXISTS course_media (
			course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
			media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
			order_index INT DEFAULT 0,
			PRIMARY KEY (course_id, media_id)
		);`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_media_course ON media (course_id);`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users (role_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_courses_cover_media ON courses (cover_media_id) WHERE cover_media_id IS NOT NULL;`,
	}

	for _, s := range stmts {
		if _, err := tx.Exec(ctx, s); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Printf("rollback error after exec fail: %v", rbErr)
			}
			return fmt.Errorf("exec stmt failed: %w; stmt: %s", err, s)
		}
	}

	// Добавляем foreign key constraint отдельно с проверкой существования
	fkCheckQuery := `
	DO $$ 
	BEGIN 
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.table_constraints 
			WHERE constraint_name = 'fk_courses_cover_media' 
			AND table_name = 'courses'
		) THEN
			ALTER TABLE courses 
			ADD CONSTRAINT fk_courses_cover_media 
			FOREIGN KEY (cover_media_id) REFERENCES media(id) ON DELETE SET NULL;
		END IF;
	END $$;
	`

	if _, err := tx.Exec(ctx, fkCheckQuery); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			log.Printf("rollback error after exec fail: %v", rbErr)
		}
		return fmt.Errorf("exec fk constraint failed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
