package services

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func init() {
	// Ensure the module has a go.sum stable environment for the tests
	// (Mocking inputs usually avoids this, but good for sanity)
}

func setUpTestBuildPath(t *testing.T) string {
	tmpDir := t.TempDir()
	source := filepath.Join(tmpDir, "src")
	indexContent := "<!DOCTYPE html><script type='module'>console.log('test');</script></html>"
	
	build := filepath.Join(tmpDir, "build")
	
	// Create a fake directory structure
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(build, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a fake build artifact
	indexPath := filepath.Join(build, "index.html")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

func TestNewSvelteService(t *testing.T) {
 cfg := ServiceConfig{
  SourcePath: "./src",
  BuildPath: "./dist",
 }

 svc := NewSvelteService(cfg)
 
 if svc == nil {
  t.Fatal("Expected non-nil service")
 }
 
 svelteSvc, ok := svc.(SvelteService)
 if !ok {
  t.Fatal("Service does not implement SvelteService interface")
 }
}

func TestBuild(t *testing.T) {
 tmpDir := setUpTestBuildPath(t)

 cfg := ServiceConfig{
  SourcePath: "unused",
  BuildPath:  filepath.Join(tmpDir, "build"),
 }

 svc := NewSvelteService(cfg)

 // Test successful build
 fs, err := svc.Build(context.Background())
 if err != nil {
  t.Fatalf("Build failed unexpectedly: %v", err)
 }

 if fs == nil {
  t.Fatal("Expected fs.FS to be returned")
 }

 // Verify the FS looks like an overlay
 sub, ok := fs.(fs.FS)
 if !ok {
  t.Fatal("Returned fs is not a valid FS")
 }
 
 // Opening the index file
 f, err := sub.Open("index.html")
 if err != nil {
  t.Fatalf("Failed to open index.html from returned FS: %v", err)
 }
 defer f.Close()
 
 // Test context cancellation
 ctx, cancel := context.WithCancel(context.Background())
 cancel()
 _, err = svc.Build(ctx)
 if err != context.Canceled {
  t.Errorf("Expected context.Canceled, got %v", err)
 }
}

func TestRouteHandler(t *testing.T) {
 tmpDir := setUpTestBuildPath(t)

 cfg := ServiceConfig{
  SourcePath: "unused",
  BuildPath:  filepath.Join(tmpDir, "build"),
 }

 svc := NewSvelteService(cfg)
 fsMount, err := svc.Build(context.Background())
 if err != nil {
  t.Fatal(err)
 }

 handler := svc.RouteHandler(fsMount)
 server := httptest.NewServer(handler)
 defer server.Close()

 tests := []struct {
  name           string
  url            string
  expectedStatus int
  expectedBody   string
  contentType    string
 }{
  {
   name:           "Root path forwards to index.html",
   url:            "/",
   expectedStatus: http.StatusOK,
   expectedBody:   "<!DOCTYPE html>",
   contentType:    "text/html",
  },
  {
   name:           "Client side route forwards to index.html",
   url:            "/channels/123/esterling",
   expectedStatus: http.StatusOK,
   expectedBody:   "<!DOCTYPE html>",
   contentType:    "text/html",
  },
 }

 for _, tt := range tests {
  t.Run(tt.name, func(t *testing.T) {
   resp, err := http.Get(server.URL + tt.url)
   if err != nil {
    t.Fatalf("Request failed: %v", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != tt.expectedStatus {
    t.Errorf("Status code = %v, want %v", resp.StatusCode, tt.expectedStatus)
   }

   // Basic body check to ensure we are serving the HTML
   buf := new(strings.Builder)
   buf.ReadFrom(resp.Body)
   body := buf.String()
   
   if !strings.Contains(body, tt.expectedBody) {
    t.Errorf("Response body does not contain expected content. Got: %s", body)
   }
   
   if resp.Header.Get("Content-Type") != tt.contentType {
    t.Errorf("Content-Type mismatch, got %s", resp.Header.Get("Content-Type"))
   }
  })
 }
}

func TestRouteHandler_NotFound(t *testing.T) {
 tmpDir := setUpTestBuildPath(t)

 cfg := ServiceConfig{
  SourcePath: "unused",
  BuildPath:  filepath.Join(tmpDir, "build"),
 }

 svc := NewSvelteService(cfg)
 fsMount, err := svc.Build(context.Background())
 if err != nil {
  t.Fatal(err)
 }

 handler := svc.RouteHandler(fsMount)
 
 req := httptest.NewRequest("GET", "/nonexistent/file.js", nil)
 w := httptest.NewRecorder()
 
 handler(w, req)

 if w.Code != http.StatusNotFound {
  t.Errorf("Expected 404, got %d", w.Code)
 }
}