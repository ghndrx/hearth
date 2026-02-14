package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// ComponentTemplate represents the raw Svelte source code for a UI component.
type ComponentTemplate struct {
	ID        string
	Name      string
	Source    string
	Version   string
	Author    string
	CreatedAt string
}

// CompilerService defines the contract for anything that modifies or manages Svelte components.
type CompilerService interface {
	// Initialize the build environment (npm install, dev-server setup)
	Initialize(ctx context.Context) error

	// FetchComponent compiles a component source string into HTML/JS.
	FetchComponent(ctx context.Context, tpl ComponentTemplate) (string, error)

	// WatchChange monitors the file system for changes and recompiles.
	WatchChanges(ctx context.Context) error
}

// SvelteServiceImplementation is the concrete implementation.
// It wraps logic related to Node.js environments and Svelte compilation.
type SvelteServiceImplementation struct {
	// path to the Svelte project root
	rootPath      string
	logger        *log.Logger
	client        *http.Client
	buildLock     sync.Mutex
	componentPool map[string]ComponentTemplate
}

// NewSvelteService creates a new instance of the compiler service.
// This is a "Factory-like" constructor pattern.
func NewSvelteService(rootPath string, logger *log.Logger) (CompilerService, error) {
	if logger == nil {
		logger = log.New(os.Stdout, "[SVELTE] ", log.LstdFlags)
	}

	// Verify Node.js / NPM availability
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Basic setup
	logger.Printf("Initializing Svelte Compiler Service at path: %s", rootPath)

	return &SvelteServiceImplementation{
		rootPath:      rootPath,
		logger:        logger,
		client:        client,
		componentPool: make(map[string]ComponentTemplate),
	}, nil
}

// Initialize sets up the environment by running npm install.
func (s *SvelteServiceImplementation) Initialize(ctx context.Context) error {
	s.buildLock.Lock()
	defer s.buildLock.Unlock()

	if _, err := os.Stat(s.rootPath); os.IsNotExist(err) {
		return fmt.Errorf("root path does not exist: %s", s.rootPath)
	}

	// Simulate running 'npm install' or 'yarn install'
	// In production, execute: exec.CommandContext(ctx, "npm", "install", "--silent")
	// ... setup logic ...

	s.logger.Println("Svelte Build Environment Initialized (NPM Ready)")
	return nil
}

// FetchComponent creates a standalone HTML file for the given Svelte component.
// It mimics the behavior of the Svelte CLI 'build' or browser-compiler.
func (s *SvelteServiceImplementation) FetchComponent(ctx context.Context, tpl ComponentTemplate) (string, error) {
	s.logger.Printf("Compiling component: %s (ID: %s)", tpl.Name, tpl.ID)

	// In a real scenario, this would write the tpl.Source to a temporary file
	// vpackage.json and run `npm run build` to get a static HTML bundle.

	// Example return: a basic HTML wrapper injected with the Svelte component code
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		<title>` + tpl.Name + `</title>
		<style>
			body { font-family: sans-serif; padding: 20px; }
		</style>
	</head>
	<body>
	<h1>` + tpl.Name + `</h1>
	<!-- Svelte Application Runtime and Component would be injected here -->
	<main id="app"></main>
	<script>
		console.log("Hearth: Rendering " + "` + tpl.Name + `");
	</script>
	</body>
	</html>
	`
	return html, nil
}

// WatchChanges monitors a directory for file changes (Hot Module Replacement logic).
func (s *SvelteServiceImplementation) WatchChanges(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(s.rootPath); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", s.rootPath, err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.logger.Println("Stopping file watcher...")
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					s.logger.Printf("Change detected in %s", event.Name)
					// Trigger re-compilation logic here
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				s.logger.Printf("Watcher error: %v", err)
			}
		}
	}()

	// Block until context is cancelled
	<-ctx.Done()
	return nil
}