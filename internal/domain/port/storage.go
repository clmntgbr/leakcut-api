package port

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, key string) error
	PresignedPutURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}
