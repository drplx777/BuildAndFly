package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
)

// MediaService содержит логику загрузки и сохранения метаданных media.
type MediaService struct {
	db      *pgxpool.Pool
	storage StorageService
}

// NewMediaService создает MediaService.
func NewMediaService(db *pgxpool.Pool, storage StorageService) *MediaService {
	return &MediaService{db: db, storage: storage}
}

// UploadMedia загружает файл в storage и сохраняет запись в таблице media.
// r будет закрыт внутри функции.
// Возвращает mediaID, public CDN URL и error.
func (s *MediaService) UploadMedia(ctx context.Context, r io.ReadCloser, filename, contentType string, size int64, courseID *uuid.UUID, mediaType string, isCover bool) (uuid.UUID, string, string, int64, error) {
	// ensure stream is closed
	defer r.Close()

	mediaID := uuid.New()
	// ключ можно дополнительно хешировать при желании
	key := fmt.Sprintf("media/%s_%s", mediaID.String(), filename)

	// Upload to storage (streaming)
	cdnURL, err := s.storage.Upload(ctx, key, r, contentType)
	if err != nil {
		return uuid.Nil, "", "", 0, err
	}

	// сохранение в БД
	_, err = s.db.Exec(ctx, `
		INSERT INTO media (id, course_id, media_type, storage_key, cdn_url, mime_type, size_bytes, is_cover, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now(), now())
	`, mediaID, courseID, mediaType, key, cdnURL, contentType, size, isCover)
	if err != nil {
		// При ошибке БД нужно удалить уже загруженный объект из storage
		_ = s.storage.Delete(ctx, key)
		return uuid.Nil, "", "", 0, err
	}

	// Если это обложка и привязан к курсу — обновляем поле cover_media_id
	if isCover && courseID != nil {
		_, _ = s.db.Exec(ctx, `UPDATE courses SET cover_media_id = $1, updated_at = now() WHERE id = $2`, mediaID, *courseID)
	}

	// если storage вернул неявный url (например с path escape), вернём именно cdnURL.
	// Также можно вернуть storage key.
	return mediaID, cdnURL, key, size, nil
}
