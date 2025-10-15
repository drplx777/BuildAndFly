-- enable extension for uuid generation
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- roles
CREATE TABLE IF NOT EXISTS roles (
                                     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role TEXT NOT NULL UNIQUE
    );

-- users
CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT,
    surname TEXT,
    email TEXT UNIQUE,
    login TEXT UNIQUE,
    password TEXT NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );

-- courses (cover_media_id добавим ниже после создания media)
CREATE TABLE IF NOT EXISTS courses (
                                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );

-- media
CREATE TABLE IF NOT EXISTS media (
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
    parent_id UUID REFERENCES media(id) ON DELETE SET NULL,
    variant_type TEXT,
    meta_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );

-- associative table for ordering / many-to-many
CREATE TABLE IF NOT EXISTS course_media (
                                            course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    order_index INT DEFAULT 0,
    PRIMARY KEY (course_id, media_id)
    );

-- add cover reference on courses (fast access to cover)
ALTER TABLE courses
    ADD COLUMN IF NOT EXISTS cover_media_id UUID;

ALTER TABLE courses
    ADD CONSTRAINT IF NOT EXISTS fk_courses_cover_media FOREIGN KEY (cover_media_id) REFERENCES media(id) ON DELETE SET NULL;

-- indexes
CREATE INDEX IF NOT EXISTS idx_media_course ON media (course_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role_id);

-- optional: ensure a course has at most one cover by having
-- a partial unique index on courses.cover_media_id (since it's one column,
-- uniqueness of the value is guaranteed by the FK itself).
-- If you want to ensure every cover_media_id is unique across courses use:
CREATE UNIQUE INDEX IF NOT EXISTS ux_courses_cover_media ON courses (cover_media_id) WHERE cover_media_id IS NOT NULL;
