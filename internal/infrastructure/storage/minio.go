package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"go-api/internal/infrastructure/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type MinIOStorage struct {
	client     *s3.Client
	presignCli *s3.PresignClient
	bucket     string
}

func NewMinIOStorage(cfg *config.Config) (*MinIOStorage, error) {
	internalEndpoint := cfg.StorageInternalEndpoint
	if internalEndpoint == "" {
		internalEndpoint = cfg.StorageEndpoint
	}

	internalClient, err := newS3Client(cfg, internalEndpoint)
	if err != nil {
		return nil, err
	}

	publicClient, err := newS3Client(cfg, cfg.StorageEndpoint)
	if err != nil {
		return nil, err
	}

	return &MinIOStorage{
		client:     internalClient,
		presignCli: s3.NewPresignClient(publicClient),
		bucket:     cfg.StorageBucket,
	}, nil
}

func newS3Client(cfg *config.Config, endpoint string) (*s3.Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.StorageRegion),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.StorageAccessKey, cfg.StorageSecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load storage config: %w", err)
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = cfg.StorageUsePathStyle
	}), nil
}

func (s *MinIOStorage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
	return nil
}

func (s *MinIOStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("object not found: %w", err)
	}
	return out.Body, nil
}

func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (s *MinIOStorage) PresignedPutURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	req, err := s.presignCli.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate upload url: %w", err)
	}
	return req.URL, nil
}

func (s *MinIOStorage) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	req, err := s.presignCli.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate download url: %w", err)
	}
	return req.URL, nil
}
