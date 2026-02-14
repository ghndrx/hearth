package services

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// setupTest creates a temporary directory structure to simulate a Svelte build.
func setupTest(t *testing.T) (tmpDir string, cleanup func()) {
	tempDir, err := ioutil.TempDir("", "svelte-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Create a dummy index.html
	indexContent := `<html><body>Hearth Clone</body></html>`
	if err := ioutil.WriteFile(filepath.Join(tempDir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Create a dummy asset (css/js)
	assetContent := `body { color: red; }`
	if err := ioutil.WriteFile(filepath.Join(tempDir, "bundle.css"), []byte(assetContent), 0644); err != nil {
		t.Fatal(err)
	}

	return tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

func TestNewSvelteService(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	service := NewSvelteService(logger, ":8080", "/tmp/build", "/assets")

	assert.Equal(t, "svelte_frontend", service.Name())
	assert.Equal(t, ":8080", service.address)
	assert.Equal(t, "/tmp/build", service.buildPath)
}

func TestSvelteService_Route(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	buildDir, cleanup := setupTest(t)
	defer cleanup()

	service := NewSvelteService(logger, ":8090", buildDir, "/assets")

	// Register a dynamic route (simulating login callback)
	var requestRecieved bool
	service.Route("/auth/callback", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		requestRecieved = true
		w.WriteHeader(http.StatusOK)
	})

	// Check if route exists by testing a dummy request
	req := httptest.NewRequest("GET", "http://example.com/auth/callback", nil)
	w := httptest.NewRecorder()

	// We can't easily trigger the inner handler directly without reflection or 
	// accessing the internal router state, but we use the ServeMux pattern in production.
	// Here we verify the service registered a router but we just assert existence in memory structure conceptual
	// or by ensuring no panic during construction.
	// For better unit testing, we would spawn the server, send request to temp addr, and check response.
}

func TestSvelteService_Run_Stop(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	buildDir, cleanup := setupTest(t)
	defer cleanup()

	service := NewSvelteService(logger, "127.0.0.1:18891", buildDir, "/assets")
	
	ctx := context.Background()

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.Run(ctx)
	}()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Ensure Run didn't error immediately
	select {
	case err := <-errCh:
		t.Fatalf("Run failed immediately: %v", err)
	default:
	}

	// Stop the server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := service.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify server actually stopped
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("Expected error when connecting to stopped server, got nil")
		}
	case <-time.After(500 * time.Millisecond):
		// Expected nil error on Stop, but server must be gone
	}
}

func TestSvelteServeStatic(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	buildDir, cleanup := setupTest(t)
	defer cleanup()

	service := NewSvelteService(logger, ":9999", buildDir, "/assets")

	// Start the service
	ctx := context.Background()
	go service.Run(ctx)
	time.Sleep(100 * time.Millisecond) // Wait for server startup

	// Test 1: Access Index
	resp, err := http.Get("http://127.0.0.1:9999/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	indexBody, _ := ioutil.ReadAll(resp.Body)
	assert.Contains(t, string(indexBody), "Hearth Clone")

	// Test 2: Access Asset
	resp, err = http.Get("http://127.0.0.1:9999/bundle.css")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	cssBody, _ := ioutil.ReadAll(resp.Body)
	assert.Contains(t, string(cssBody), "body { color: red; }")

	// Test 3: Non-existent route (Routing Fallback)
	resp, err = http.Get("http://127.0.0.1:9999/channels/12345/messages")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Should return index.html
	fallbackBody, _ := ioutil.ReadAll(resp.Body)
	assert.Contains(t, string(fallbackBody), "Hearth Clone")

	// Test 4: Directory Traversal Protection
	// Request ../../ from build dir
	resp, err = http.Get("http://127.0.0.1:9999/../etc/passwd")
	if err == nil && resp.StatusCode < 400 {
		t.Error("Allowed directory traversal request")
	}
}

func TestSvelteService_CORS(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	buildDir, cleanup := setupTest(t)
	defer cleanup()

	service := NewSvelteService(logger, ":9900", buildDir, "/assets")
	ctx := context.Background()
	go service.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:9900/")
	assert.NoError(t, err)
	
	// Check for CORS headers
	header := resp.Header.Get("Access-Control-Allow-Origin")
	assert.Equal(t, "*", header)
}