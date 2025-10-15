package service

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client  *minio.Client
	bucket  string
	baseURL string
}

func NewMinioStorage(endpoint, accessKey, secretKey string, useSSL bool, bucket string, baseURL string) (*MinioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	return &MinioStorage{client: client, bucket: bucket, baseURL: baseURL}, nil
}

func (s *MinioStorage) Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	// -1 сообщает minio, что размер неизвестен (streaming)
	info, err := s.client.PutObject(ctx, s.bucket, key, body, -1, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}

	if s.baseURL != "" {
		// baseURL + bucket + key
		return fmt.Sprintf("%s/%s/%s", s.baseURL, s.bucket, url.PathEscape(key)), nil
	}
	ep := s.client.EndpointURL()
	if ep == nil {
		return info.Location, nil
	}
	return fmt.Sprintf("%s/%s/%s", ep.String(), s.bucket, url.PathEscape(key)), nil
}

func (s *MinioStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
