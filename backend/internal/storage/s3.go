package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// S3Backend implements StorageBackend for S3-compatible storage
// This is a stub - implement with AWS SDK when Go 1.23+ is available
type S3Backend struct {
	bucket    string
	endpoint  string
	publicURL string
}

// S3Config holds S3 configuration
type S3Config struct {
	Endpoint       string
	Bucket         string
	Region         string
	AccessKey      string
	SecretKey      string
	PublicURL      string // Optional CDN URL
	ForcePathStyle bool
}

var ErrS3NotImplemented = errors.New("S3 storage requires Go 1.23+, use local storage for now")

// NewS3Backend creates a new S3 storage backend (stub)
func NewS3Backend(cfg S3Config) (*S3Backend, error) {
	// Return a stub that will error on use
	// Full implementation requires Go 1.23+ for AWS SDK v2
	return &S3Backend{
		bucket:    cfg.Bucket,
		endpoint:  cfg.Endpoint,
		publicURL: cfg.PublicURL,
	}, nil
}

// Upload uploads a file to S3
func (b *S3Backend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (string, error) {
	return "", ErrS3NotImplemented
}

// Download retrieves a file from S3
func (b *S3Backend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, ErrS3NotImplemented
}

// Delete removes a file from S3
func (b *S3Backend) Delete(ctx context.Context, path string) error {
	return ErrS3NotImplemented
}

// GetURL returns a public URL for a file
func (b *S3Backend) GetURL(path string) string {
	return b.publicURL + "/" + path
}

// GetSignedURL returns a presigned URL for temporary access
func (b *S3Backend) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return "", ErrS3NotImplemented
}
