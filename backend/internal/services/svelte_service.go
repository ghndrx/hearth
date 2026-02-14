package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// Service defines the interface for engine services.
type Service interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
}

// SvelteService implements a static file server with Serve-Middleware capabilities for a Discord voice clone.
type SvelteService struct {
	logger     *zap.Logger
	address    string
	buildPath  string
	staticPath string
	httpServer *http.Server
	server     *httprouter.Router
	mu         sync.RWMutex
}

// NewSvelteService creates a new SvelteService instance.
func NewSvelteService(logger *zap.Logger, addr, buildPath, staticPath string) *SvelteService {
	return &SvelteService{
		logger:     logger,
		address:    addr,
		buildPath:  buildPath,
		staticPath: staticPath,
		server:     httprouter.New(),
	}
}

// Name returns the identifier of the service.
func (s *SvelteService) Name() string {
	return "svelte_frontend"
}

// Run starts the HTTP server. 
// It decompresses or sets up static file serving based on the buildPath.
func (s *SvelteService) Run(ctx context.Context) error {
	s.logger.Info("Initializing Svelte Service", 
		zap.String("address", s.address),
		zap.String("build_directory", s.buildPath),
	)

	// Middleware: Add CORS headers for WebSocket/WebRTC connection origins
	s.server.Use(func(h httprouter.Handler) httprouter.Handler {
		return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			h.ServeHTTP(w, r, ps)
		}
	})

	// Middleware: Serve static assets (React Router fallback)
	s.server.GET("/*any", s.handleReroute)

	s.httpServer = &http.Server{
		Addr:         s.address,
		Handler:      s.server,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Svelte Service stopped unexpectedly", zap.Error(err))
		}
	}()

	s.logger.Info("Svelte Service started")
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (s *SvelteService) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Svelte Service")
	
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
	}
	return nil
}

// handleReroute handles basic static file serving and catch-all for client side routing.
// In a production scenario, this would check if `file` exists. If not, serve index.html.
func (s *SvelteService) handleReroute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Construct the requested path
	file := ps.ByName("any")
	
	// Security: Prevent directory traversal
	// Ensure the requested file/folder is inside the static path
	reqPath := filepath.Join(s.buildPath, file)
	
	absPath, err := filepath.Abs(reqPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	absBuildPath, err := filepath.Abs(s.buildPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(absPath, absBuildPath) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if file exists
	info, err := os.Stat(reqPath)
	if err != nil {
		// If file not found, handle React/Svelte Router fallback by serving index.html
		if os.IsNotExist(err) {
			// Note: gsap (Gateway-Serving) intentions: Static files usually need file watcher.
			// For this scenario, we assume "build_path" contains the built assets.
			// We serve index.html for all non-existent files to handle client routing.
			w.Header().Set("Content-Type", "text/html")
			http.ServeFile(w, r, filepath.Join(s.buildPath, "index.html"))
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Serve static
	http.ServeFile(w, r, reqPath)
}

// Route dynamically registers a handler for a specific Svelte route path (e.g., `/channels/1234`)
// Note: This is a hook for the backend to inject dynamic data into the front end context.
func (s *SvelteService) Route(route string, handler func(http.ResponseWriter, *http.Request, httprouter.Params)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.server.GET(route, handler)
}