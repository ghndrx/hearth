package services

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewSvelteService(t *testing.T) {
	// Test that the service initializes without panicking
	s := NewSvelteService(8080)
	if s == nil {
		t.Fatal("NewSvelteService returned nil")
	}
}

func TestSvelteService_StartServer(t *testing.T) {
	service := NewSvelteService(0) // Use 0 to let OS assign a port

	// Use a timeout to prevent the test from hanging forever
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start the service in a goroutine
	errCh := make(chan error, 1)
	go func() { errCh <- service.Start() }()

	// Wait for a short moment for the server to boot
	time.Sleep(500 * time.Millisecond)

	// Verify the service is running
	select {
	case err := <-errCh:
		t.Fatalf("Service error during start: %v", err)
	case <-ctx.Done():
		// Good, it started in time
	default:
		// Continue
	}

	// Test the /status endpoint to ensure it returns HTML
	req, _ := http.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()

	// Access the internal HTTP handler for testing purposes
	service.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Hearth Core") {
		t.Error("Response body does not contain expected title")
	}
}

func TestSvelteService_HealthCheck(t *testing.T) {
	service := NewSvelteService(0)
	service.Start() // Start the server

	// Give the server a moment to listen
	time.Sleep(200 * time.Millisecond)

	health, err := service.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	if health != "healthy" {
		t.Errorf("Health status = %s, want 'healthy'", health)
	}

	service.Stop()
}

func TestSvelteService_Stop(t *testing.T) {
	service := NewSvelteService(0)
	service.Start()

	// Ensure the server is reachable
	time.Sleep(500 * time.Millisecond)
	
	onlineConn, err := net.DialTimeout("tcp", "127.0.0.1:0", 100*time.Millisecond)
	if err != nil {
		t.Skip("Server not listening, cannot test shutdown properly")
	}
	onlineConn.Close()

	// Stop the service
	err = service.Stop()
	if err != nil {
		t.Errorf("Stop returned error: %v", err)
	}

	// Verify the server is actually down
	disconnected, err := net.DialTimeout("tcp", "127.0.0.1:0", 100*time.Millisecond)
	if err == nil {
		disconnected.Close()
		t.Error("Service shutdown failed: Server is still accepting connections")
	}
}