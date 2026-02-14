package services

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Component represents a frontend component or entity.
// In this context, it maps to a .svelte file or a generated build artifact.
type Component struct {
	ID          string
	Name        string
	Version     string
	SourcePath  string // Path to the .svelte source file
	BuildPath   string // Path to the compiled .css/.js output
	DocumentDir string // Directory where the build artifacts are served
}

// BuildConfig holds necessary options for building the component.
type BuildConfig struct {
	OutputDir string
 PurgeCache bool // For testing or hot-reloading scenarios
}

// SvelteService defines the contract for managing Svelte components.
// This interface allows us to mock the complex Svelte compilation logic in tests.
type SvelteService interface {
	// Build sources a Svelte file, compiles it, and places outputs in the filesystem.
	BuildComponent(comp Component) error

	// CopyToDocumentDir copies the finalized build artifact to the public serving directory.
	CopyToDocumentDir(comp Component) error

	// ExecuteCompiler runs the Svelte compiler executable with specific arguments.
	// This is an abstraction used internally by the service to keep the logic testable.
	ExecuteCompiler(source, outputDir string) error
}

// svelteService is the concrete implementation.
type svelteService struct {
	compilerPath string
	logger       ServiceLogger
	fileSystem   FileSystem
}

// SvelteServiceConfig holds dependencies for the service.
type SvelteServiceConfig struct {
	CompilerPath string
	Logger       ServiceLogger
	FS           FileSystem
}

// NewSvelteService initializes the service with dependencies.
func NewSvelteService(config SvelteServiceConfig) SvelteService {
	return &svelteService{
		compilerPath: config.CompilerPath,
		logger:       config.Logger,
		fileSystem:   config.FS,
	}
}

// BuildComponent orchestrates the build process.
func (s *svelteService) BuildComponent(comp Component) error {
	s.logger.Info(fmt.Sprintf("Starting Svelte build for: %s (%s)", comp.Name, comp.ID))

	// 1. Ensure output directory exists
	if err := s.fileSystem.MkdirAll(comp.BuildPath, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// 2. Execute the compiler
	if err := s.ExecuteCompiler(comp.SourcePath, comp.BuildPath); err != nil {
		return fmt.Errorf("compilation failed for %s: %w", comp.Name, err)
	}

	s.logger.Info(fmt.Sprintf("Successfully built component: %s", comp.Name))
	return nil
}

// CopyToDocumentDir moves the artifact to the web root (Discord clone public files).
func (s *svelteService) CopyToDocumentDir(comp Component) error {
	s.logger.Info(fmt.Sprintf("Copying %s to %s", comp.BuildPath, comp.DocumentDir))

	// Walk through the build output directory
	return s.fileSystem.CopyDir(comp.BuildPath, comp.DocumentDir)
}

// ExecuteCompiler invokes the Svelte compiler binary.
// In a real production environment, passing arguments securely is critical.
func (s *svelteService) ExecuteCompiler(source, outputDir string) error {
	s.logger.Info(fmt.Sprintf("Running compiler: %s --input %s --output %s", s.compilerPath, source, outputDir))
	
	// Example: using exec.Command logic here. 
	// For simplicity, we just log the command that would be run.
	return nil
}

// --- Interfaces for Dependency Injection / Testing ---

// FileSystem abstracts os/fs operations to allow mocking in tests.
type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	CopyDir(src, dst string) error
	RemoveAll(path string) error
	Exists(path string) bool
}

// ServiceLogger represents a logging abstraction.
type ServiceLogger interface {
	Info(msg string)
	Error(msg string)
}

// RealFileSystem is the standard implementation of FileSystem.
type RealFileSystem struct{}

func (r *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *RealFileSystem) CopyDir(src, dst string) error {
	return copyDir(src, dst)
}

func (r *RealFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (r *RealFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyDir(src string, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = destination.ReadFrom(source)
	return err
}