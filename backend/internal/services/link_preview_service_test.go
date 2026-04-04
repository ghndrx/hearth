package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

func TestLinkPreviewService_InvalidURL(t *testing.T) {
	svc := NewLinkPreviewService()
	ctx := context.Background()

	tests := []struct {
		name string
		url  string
	}{
		{"empty string", ""},
		{"no scheme", "example.com"},
		{"just path", "/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetPreview(ctx, tt.url)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.url)
			}
		})
	}
}

func TestLinkPreviewService_CacheHit(t *testing.T) {
	svc := NewLinkPreviewService()
	ctx := context.Background()

	// Create a test server that returns consistent OG tags
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Test Site</title>
<meta property="og:title" content="OG Title">
<meta property="og:description" content="OG Description">
<meta property="og:image" content="https://example.com/image.png">
<meta property="og:type" content="website">
</head>
<body></body>
</html>`))
	}))
	defer server.Close()

	// First call should hit the server
	preview1, err := svc.GetPreview(ctx, server.URL)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	if preview1.Title == nil || *preview1.Title != "OG Title" {
		t.Errorf("expected title 'OG Title', got %v", preview1.Title)
	}

	// Second call should use cache (same preview ID)
	preview2, err := svc.GetPreview(ctx, server.URL)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if preview1.ID != preview2.ID {
		t.Errorf("expected same preview ID from cache, got different")
	}
}

func TestLinkPreviewService_ExtractOGTags(t *testing.T) {
	svc := NewLinkPreviewService()

	tests := []struct {
		name     string
		html     string
		checkFn  func(*testing.T, *testPreview)
	}{
		{
			name: "basic og tags",
			html: `<!DOCTYPE html><html><head>
<meta property="og:title" content="Test Title">
<meta property="og:description" content="Test Description">
<meta property="og:image" content="https://img.example.com/test.png">
<meta property="og:type" content="article">
</head></html>`,
			checkFn: func(t *testing.T, p *testPreview) {
				if p.Title == nil || *p.Title != "Test Title" {
					t.Errorf("expected title 'Test Title', got %v", p.Title)
				}
				if p.Description == nil || *p.Description != "Test Description" {
					t.Errorf("expected description 'Test Description', got %v", p.Description)
				}
				if p.Type != "article" {
					t.Errorf("expected type 'article', got %s", p.Type)
				}
			},
		},
		{
			name: "html entities decoded",
			html: `<!DOCTYPE html><html><head><title>Tom &amp; Jerry</title></head></html>`,
			checkFn: func(t *testing.T, p *testPreview) {
				if p.Title == nil || *p.Title != "Tom & Jerry" {
					t.Errorf("expected title 'Tom & Jerry', got %v", p.Title)
				}
			},
		},
		{
			name: "no og tags falls back to title tag",
			html: `<!DOCTYPE html><html><head><title>Fallback Title</title></head></html>`,
			checkFn: func(t *testing.T, p *testPreview) {
				if p.Title == nil || *p.Title != "Fallback Title" {
					t.Errorf("expected title 'Fallback Title', got %v", p.Title)
				}
			},
		},
		{
			name: "video content type",
			html: `<!DOCTYPE html><html><head></head></html>`,
			checkFn: func(t *testing.T, p *testPreview) {
				// default type should be website
				if p.Type != "website" {
					t.Errorf("expected default type 'website', got %s", p.Type)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(tt.html))
			}))
			defer server.Close()

			preview, err := svc.extractPreview(context.Background(), server.URL)
			if err != nil {
				t.Fatalf("extractPreview failed: %v", err)
			}

			tt.checkFn(t, &testPreview{
				Title:       preview.Title,
				Description: preview.Description,
				ImageURL:    preview.ImageURL,
				Type:        preview.Type,
			})
		})
	}
}

type testPreview struct {
	Title       *string
	Description *string
	ImageURL    *string
	Type        string
}

func TestLinkPreviewService_ClearCache(t *testing.T) {
	svc := NewLinkPreviewService()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test</title></head></html>`))
	}))
	defer server.Close()

	ctx := context.Background()

	preview1, _ := svc.GetPreview(ctx, server.URL)
	id1 := preview1.ID

	// Modify the server to return different content
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Different</title></head></html>`))
	})

	// Without clear, should still get cached version
	preview2, _ := svc.GetPreview(ctx, server.URL)
	if preview1.ID != preview2.ID {
		t.Errorf("expected cache hit, got different ID")
	}

	// Clear cache
	svc.ClearCache()

	// After clear, should get new version
	preview3, _ := svc.GetPreview(ctx, server.URL)
	if id1 == preview3.ID {
		t.Errorf("expected new preview after cache clear, got same ID")
	}
}

func TestLinkPreviewService_PurgeExpired(t *testing.T) {
	svc := NewLinkPreviewService()
	// Manually add an expired entry
	svc.mu.Lock()
	svc.cache["http://expired.example.com"] = &cachedPreview{
		preview: &models.LinkPreview{ID: uuid.New()},
		expiry:  time.Now().Add(-1 * time.Hour),
	}
	svc.mu.Unlock()

	svc.PurgeExpired()

	svc.mu.RLock()
	if _, ok := svc.cache["http://expired.example.com"]; ok {
		t.Errorf("expected expired entry to be purged")
	}
	svc.mu.RUnlock()
}
