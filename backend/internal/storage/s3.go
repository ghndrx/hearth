package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Backend implements StorageBackend for S3-compatible storage
type S3Backend struct {
	client         *s3.Client
	presigner      *s3.PresignClient
	bucket         string
	publicURL      string
	forcePathStyle bool // For MinIO or other S3-compatible services
}

// S3Config holds S3 configuration
type S3Config struct {
	Endpoint        string // For MinIO or other S3-compatible services
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	PublicURL       string // Optional CDN URL prefix
	ForcePathStyle  bool   // Required for MinIO and some S3-compatible services
}

// NewS3Backend creates a new S3 storage backend
func NewS3Backend(cfg S3Config) (*S3Backend, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("S3 bucket is required")
	}

	var awsCfg aws.Config
	var err error

	// If explicit credentials are provided, build a minimal config to avoid
	// loading shared config files that may contain invalid profiles.
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		awsCfg = aws.Config{
			Region: cfg.Region,
			Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		}
	} else {
		// Build options for AWS SDK v2
		var opts []func(*config.LoadOptions) error
		opts = append(opts, config.WithRegion(cfg.Region))
		// Disable loading shared config files to avoid failures from local
		// profiles that may not be valid in the runtime environment.
		opts = append(opts, config.WithSharedConfigFiles([]string{}))

		// Load AWS configuration
		awsCfg, err = config.LoadDefaultConfig(context.Background(), opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	}

	// Create S3 client options
	var s3Opts []func(*s3.Options)

	// Custom endpoint for MinIO or other S3-compatible services
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = cfg.ForcePathStyle
		})
	} else if cfg.ForcePathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, s3Opts...)
	presigner := s3.NewPresignClient(client)

	return &S3Backend{
		client:         client,
		presigner:      presigner,
		bucket:         cfg.Bucket,
		publicURL:      cfg.PublicURL,
		forcePathStyle: cfg.ForcePathStyle,
	}, nil
}

// Upload uploads a file to S3
func (b *S3Backend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (string, error) {
	// Read all data into memory (for simplicity; for large files, use multipart upload)
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Upload to S3
	input := &s3.PutObjectInput{
		Bucket:        aws.String(b.bucket),
		Key:           aws.String(path),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}

	_, err = b.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return b.GetURL(path), nil
}

// Download retrieves a file from S3
func (b *S3Backend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(path),
	}

	result, err := b.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return result.Body, nil
}

// Delete removes a file from S3
func (b *S3Backend) Delete(ctx context.Context, path string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(path),
	}

	_, err := b.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// GetURL returns a public URL for a file
func (b *S3Backend) GetURL(path string) string {
	if b.publicURL != "" {
		return b.publicURL + "/" + path
	}
	// Fallback to S3 URL format
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", b.bucket, path)
}

// GetSignedURL returns a presigned URL for temporary access
func (b *S3Backend) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(path),
	}

	presignedReq, err := b.presigner.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedReq.URL, nil
}

// Exists checks if a file exists in S3
func (b *S3Backend) Exists(ctx context.Context, path string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(path),
	}

	_, err := b.client.HeadObject(ctx, input)
	if err != nil {
		// Check for not found using error code from the API error
		// The S3 API returns "NoSuchKey" for missing objects
		var ae interface{ ErrorCode() string }
		if errors.As(err, &ae) {
			if ae.ErrorCode() == "NoSuchKey" {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}
