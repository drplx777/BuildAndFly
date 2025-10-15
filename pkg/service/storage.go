package service

import (
	"context"
	"io"
)

type StorageService interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
}
