package storage

import (
	"context"
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
	client    *s3.Client
	bucket    string
	endpoint  string
	publicURL string
}

// S3Config holds S3 configuration
type S3Config struct {
	Endpoint    string
	Bucket      string
	Region      string
	AccessKey   string
	SecretKey   string
	PublicURL   string // Optional CDN URL
	ForcePathStyle bool
}

// NewS3Backend creates a new S3 storage backend
func NewS3Backend(cfg S3Config) (*S3Backend, error) {
	// Build custom resolver for MinIO/custom endpoints
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if cfg.Endpoint != "" {
				return aws.Endpoint{
					URL:               cfg.Endpoint,
					HostnameImmutable: true,
				}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		},
	)

	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForcePathStyle || cfg.Endpoint != ""
	})

	publicURL := cfg.PublicURL
	if publicURL == "" {
		if cfg.Endpoint != "" {
			publicURL = cfg.Endpoint + "/" + cfg.Bucket
		} else {
			publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
		}
	}

	return &S3Backend{
		client:    client,
		bucket:    cfg.Bucket,
		endpoint:  cfg.Endpoint,
		publicURL: publicURL,
	}, nil
}

// Upload uploads a file to S3
func (b *S3Backend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(b.bucket),
		Key:           aws.String(path),
		Body:          file,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}

	_, err := b.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return b.GetURL(path), nil
}

// Download retrieves a file from S3
func (b *S3Backend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	output, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return output.Body, nil
}

// Delete removes a file from S3
func (b *S3Backend) Delete(ctx context.Context, path string) error {
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// GetURL returns a public URL for a file
func (b *S3Backend) GetURL(path string) string {
	return b.publicURL + "/" + path
}

// GetSignedURL returns a presigned URL for temporary access
func (b *S3Backend) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(b.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(path),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}
