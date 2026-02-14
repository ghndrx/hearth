package services

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Service defines the interface for the Svelte component management system.
// It ensures the main application can interact with the service without knowing
// the implementation details of the HTTP server or the heartbeat logic.
type Service interface {
	// Start initializes the service, including the heartbeat and HTTP server.
	Start() error

	// Stop gracefully shuts down the service and its resources.
	Stop() error

	// GetHealth returns the current health status of the Svelte frontend.
	GetHealth() (string, error)
}

// SvelteService implements the Service interface.
type SvelteService struct {
	port       int
	listener   net.Listener
	httpServer *http.Server
}

// NewSvelteService creates a new instance of the SvelteService.
// Port 0 allows Go to assign an available ephemeral port, making the unit testing easier.
func NewSvelteService(port int) *SvelteService {
	return &SvelteService{
		port: port,
	}
}

// Start begins the listen loop and the background heartbeat ticker.
func (s *SvelteService) Start() error {
	// Bind to the socket (loopback only for local services)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: s.port}
	
	var err error
	s.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	// Initialize the HTTP server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Handler: mux,
		// Add a sensible read/write timeout to prevent hijacking
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	// Start the background heartbeat (Heart of Hearth)
	go s.startHeartbeat()

	// Start the HTTP server in a goroutine
	go func() {
		log.Printf("Svelte Frontend Service starting on interface 127.0.0.1:%d", s.port)
		if err := s.httpServer.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Error serving HTTP: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the service.
func (s *SvelteService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Svelte Service shutting down...")
	
	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}

// startHeartbeat simulates keeping the Svelte connection (WebSocket) alive.
func (s *SvelteService) startHeartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Simulate network activity for the Discord clone frontend
			log.Println("💓 Svelte Frontend Heartbeat: Connection Active")
		}
	}
}

// handleStatus renders a mock view of the Svelte application status.
func (s *SvelteService) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Hearth - Svelte Status</title>
		<style>
			body { font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; background-color: #36393f; color: #fff; }
			.card { background: #23272a; padding: 40px; border-radius: 10px; box-shadow: 0 8px 24px rgba(0,0,0,0.2); text-align: center; max-width: 300px; }
			h1 { color: #7289da; margin-bottom: 10px; }
			.status-badge { background: #43b581; color: white; padding: 8px 16px; border-radius: 20px; font-size: 0.8em; display: inline-block; margin-top: 15px; }
			.logger { margin-top: 20px; font-family: monospace; font-size: 12px; color: #b9bbbe; height: 100px; overflow-y: auto; border: 1px solid #202225; padding: 10px; }
		</style>
	</head>
	<body>
		<div class="card">
			<h1>Hearth Core</h1>
			<p>Svelte Frontend Component</p>
			<div id="status" class="status-badge">Online</div>
			<div class="logger">System initialized.<br>WebSocket adapter loaded.<br>Waiting for events...</div>
		</div>
		<script>
			// Logic to refresh the logger based on Heartbeat rate
			setInterval(() => {
				const logger = document.querySelector('.logger');
				const timestamp = new Date().toLocaleTimeString();
				logger.innerHTML += ` + "`<br>[${timestamp}] Heart received.`" + `
				logger.scrollTop = logger.scrollHeight;
			}, 1500);
		</script>
	</body>
	</html>
	`

	w.Write([]byte(html))
}

// GetHealth allows the parent framework to query the service.
func (s *SvelteService) GetHealth() (string, error) {
	if s.httpServer != nil && s.httpServer.Addr != "" {
		// Try to verify the server is actually reachable via TCP
		addr, _ := url.Parse("http://" + s.httpServer.Addr)
		port := addr.Port()
		conn, err := net.DialTimeout("tcp", addr.Hostname()+":"+port, 1*time.Second)
		if err != nil {
			return "unhealthy", err
		}
		conn.Close()
		return "healthy", nil
	}
	return "unknown", nil
}