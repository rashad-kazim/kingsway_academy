package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinIO(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
}

type ObjectStore struct {
	client *minio.Client
}

func NewObjectStore(client *minio.Client) *ObjectStore {
	return &ObjectStore{client: client}
}

func (s *ObjectStore) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

func (s *ObjectStore) Put(ctx context.Context, bucket string, key string, content io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, key, content, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	return err
}

func (s *ObjectStore) PresignedGet(ctx context.Context, bucket string, key string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}

	return url.String(), nil
}

func (s *ObjectStore) Delete(ctx context.Context, bucket string, key string) error {
	return s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}
