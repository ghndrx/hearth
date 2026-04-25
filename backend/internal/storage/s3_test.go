package storage

import (
	"testing"
)

func TestNewS3Backend_EmptyBucket(t *testing.T) {
	_, err := NewS3Backend(S3Config{
		Bucket: "",
		Region: "us-east-1",
	})
	if err == nil {
		t.Error("expected error for empty bucket")
	}
}

func TestNewS3Backend_WithCredentials(t *testing.T) {
	backend, err := NewS3Backend(S3Config{
		Bucket:          "test-bucket",
		Region:          "us-west-2",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		PublicURL:       "https://cdn.example.com",
	})
	if err != nil {
		t.Fatalf("NewS3Backend failed: %v", err)
	}
	if backend.bucket != "test-bucket" {
		t.Errorf("bucket = %s, want test-bucket", backend.bucket)
	}
	if backend.publicURL != "https://cdn.example.com" {
		t.Errorf("publicURL = %s, want https://cdn.example.com", backend.publicURL)
	}
}

func TestNewS3Backend_WithEndpoint(t *testing.T) {
	backend, err := NewS3Backend(S3Config{
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		Endpoint:        "http://localhost:9000",
		ForcePathStyle:  true,
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
	})
	if err != nil {
		t.Fatalf("NewS3Backend failed: %v", err)
	}
	if !backend.forcePathStyle {
		t.Error("expected forcePathStyle to be true")
	}
}

func TestNewS3Backend_ForcePathStyleWithoutEndpoint(t *testing.T) {
	backend, err := NewS3Backend(S3Config{
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		ForcePathStyle:  true,
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
	})
	if err != nil {
		t.Fatalf("NewS3Backend failed: %v", err)
	}
	if !backend.forcePathStyle {
		t.Error("expected forcePathStyle to be true")
	}
}

func TestNewS3Backend_MinimalConfig(t *testing.T) {
	backend, err := NewS3Backend(S3Config{
		Bucket:          "my-bucket",
		Region:          "eu-west-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
	})
	if err != nil {
		t.Fatalf("NewS3Backend failed: %v", err)
	}
	if backend.client == nil {
		t.Error("expected non-nil client")
	}
	if backend.presigner == nil {
		t.Error("expected non-nil presigner")
	}
}

func TestS3Backend_GetURL(t *testing.T) {
	tests := []struct {
		name      string
		publicURL string
		bucket    string
		path      string
		want      string
	}{
		{
			name:      "with public URL",
			publicURL: "https://cdn.example.com",
			bucket:    "test-bucket",
			path:      "attachments/file.txt",
			want:      "https://cdn.example.com/attachments/file.txt",
		},
		{
			name:      "without public URL falls back to S3 URL",
			publicURL: "",
			bucket:    "test-bucket",
			path:      "attachments/file.txt",
			want:      "https://test-bucket.s3.amazonaws.com/attachments/file.txt",
		},
		{
			name:      "nested path with public URL",
			publicURL: "https://static.example.com",
			bucket:    "my-bucket",
			path:      "users/12/avatar.png",
			want:      "https://static.example.com/users/12/avatar.png",
		},
		{
			name:      "empty path with public URL",
			publicURL: "https://cdn.example.com",
			bucket:    "bucket",
			path:      "",
			want:      "https://cdn.example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &S3Backend{
				bucket:    tt.bucket,
				publicURL: tt.publicURL,
			}
			got := backend.GetURL(tt.path)
			if got != tt.want {
				t.Errorf("GetURL() = %s, want %s", got, tt.want)
			}
		})
	}
}
