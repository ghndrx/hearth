package services

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
)

// SvelteService defines the interface for interacting with the frontend frontend.
// It abstracts the source code location and the Svelte compiler/build process.
type SvelteService interface {
	// Build builds the Svelte static assets and returns the underlying filesystem
	// ready for serving.
	Build(ctx context.Context) (fs.FS, error)

	// RouteHandler returns an HTTP handler function specific for this Svelte app.
	RouteHandler(sourceFs fs.FS) http.HandlerFunc
}

// ServiceConfig holds the configuration needed to initialize the service.
type ServiceConfig struct {
	// SourcePath is the local filesystem path relative to the working directory
	// where the Svelte source code resides.
	SourcePath string

	// BuildPath is the path where the built static assets will be placed during the build process.
	BuildPath string
}

// svelteServiceImpl implements SvelteService.
type svelteServiceImpl struct {
	config ServiceConfig
}

// NewSvelteService creates a new instance of the Svelte service.
func NewSvelteService(cfg ServiceConfig) SvelteService {
	return &svelteServiceImpl{
		config: cfg,
	}
}

// Build simulates the compilation of the Svelte application. 
// In a real-world scenario, this would shell out to `npm run build` 
// or use a go-based Svelte compiler like ramirago/svelte-c().
func (s *svelteServiceImpl) Build(ctx context.Context) (fs.FS, error) {
	// Simulate build process delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// In production, unpack build artifacts here using archive/zip or similar.
		// For this example, we assume BuildPath contains the pre-compiled static assets.
		return http.FS(fs.Sub(http.Dir(s.config.BuildPath)), "build"), nil
	}
}

// RouteHandler constructs an HTTP handler that serves the built Svelte application.
// It proxies paths like '/' to 'index.html' so the Sapper/Svelte router works.
func (s *svelteServiceImpl) RouteHandler(sourceFs fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check would likely happen here before serving HTML.
		// See AuthMiddleware in core.

		// Redoc/Debug route (example)
		if r.URL.Path == "/_auth" {
			w.Write([]byte("Auth service active"))
			return
		}

		// Default to index.html to handle client-side routing (e.g., /channels/123)
		fileToServe := "build/index.html"

		// Attempt to serve the requested file statically
		f, err := sourceFs.Open(fileToServe)
		if err != nil {
			// Fallback: If we can't open index.html, 404 the user
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		// Serve the file
		http.ServeContent(w, r, fileToServe, int64(0), f)
	}
}

// GetBuildPath returns the configured build path for external processes.
func (s *svelteServiceImpl) GetBuildPath() string {
	return s.config.BuildPath
}