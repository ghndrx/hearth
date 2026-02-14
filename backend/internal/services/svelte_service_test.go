package services

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// MockSvelteService implements SvelteService using in-memory assertions.
type MockSvelteService struct {
	CompiledFiles map[string]bool
	CopyErrors    map[string]error // Injected error for test cases
}

func (m *MockSvelteService) BuildComponent(comp Component) error {
	m.CompiledFiles[comp.ID] = true
	return nil
}

func (m *MockSvelteService) CopyToDocumentDir(comp Component) error {
	if err, exists := m.CopyErrors[comp.ID]; exists {
		return err
	}
	return nil
}

func (m *MockSvelteService) ExecuteCompiler(source, outputDir string) error {
	// Mock implementation stub
	return nil
}

// MockFileSystem is used to verify directory operations without touching the real disk.
type MockFileSystem struct {
	MkdirAllInvocations  []string
	CopyDirInvocations   []string
	RemoveAllInvocations []string
	ExistsInvocations    []string
	DirContents          map[string][]string // Track file contents per directory
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		DirContents: make(map[string][]string),
	}
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	m.MkdirAllInvocations = append(m.MkdirAllInvocations, path)
	return nil
}

func (m *MockFileSystem) CopyDir(src, dst string) error {
	m.CopyDirInvocations = append(m.CopyDirInvocations, src+" -> "+dst)
	return nil
}

func (m *MockFileSystem) RemoveAll(path string) error {
	m.RemoveAllInvocations = append(m.RemoveAllInvocations, path)
	return nil
}

func (m *MockFileSystem) Exists(path string) bool {
	m.ExistsInvocations = append(m.ExistsInvocations, path)
	_, ok := m.DirContents[path]
	return ok
}

// TestNewSvelteService tests the constructor.
func TestNewSvelteService(t *testing.T) {
	logger := &MockLogger{}
	fs := NewMockFileSystem()
	path := "/usr/local/bin/svelte"

	service := NewSvelteService(SvelteServiceConfig{
		CompilerPath: path,
		Logger:       logger,
		FS:           fs,
	})

	svelteService, ok := service.(*svelteService)
	if !ok {
		t.Fatalf("Failed to cast service to concrete type")
	}

	if svelteService.compilerPath != path {
		t.Errorf("Expected compilerPath %s, got %s", path, svelteService.compilerPath)
	}
}

// TestBuildComponent tests the full build lifecycle.
func TestBuildComponent(t *testing.T) {
	comp := Component{
		ID:        "test-cmp",
		Name:      "TestComponent",
		Version:   "v1.0",
		SourcePath: "./test-input.svelte",
		BuildPath:   "./build/test-cmp",
		DocumentDir: "./public/dist",
	}

	// Setup
	mockFS := NewMockFileSystem()
	mockSvelte := &MockSvelteService{
		CompiledFiles: make(map[string]bool),
	}

	// Inject dependencies
	service := NewSvelteService(SvelteServiceConfig{
		CompilerPath: "svelte",
		FS:           mockFS,
		Logger:       &MockLogger{}, // Using MockLogger defined in standard test helper usually, or simple struct below
	})

	// Execute
	err := service.BuildComponent(comp)

	// Assertions
	if err != nil {
		t.Fatalf("BuildComponent failed: %v", err)
	}
	if !mockSvelte.CompiledFiles["test-cmp"] {
		t.Error("Expected component to be compiled in mock service")
	}

	// Verify FileSystem calls
	expectedDir := "./build/test-cmp"
	found := false
	for _, dir := range mockFS.MkdirAllInvocations {
		if dir == expectedDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected MkdirAll to be called with %s", expectedDir)
	}
}

// TestCopyToDocumentDir tests the final distribution step.
func TestCopyToDocumentDir(t *testing.T) {
	comp := Component{
		ID:         "success-id",
		BuildPath:  "build/abc",
		DocumentDir: "public/abc",
	}

	mockFS := NewMockFileSystem()
	mockSvelte := &MockSvelteService{
		CopyErrors: make(map[string]error),
	}

	service := NewSvelteService(SvelteServiceConfig{
		CompilerPath: "svelte",
		FS:           mockFS,
		Logger:       &MockLogger{},
	})

	err := service.CopyToDocumentDir(comp)
	if err != nil {
		t.Fatalf("CopyToDocumentDir errored: %v", err)
	}

	// Check that CopyDir was called
	actions := mockFS.CopyDirInvocations
	found := false
	for _, action := range actions {
		if action == "build/abc -> public/abc" {
			found = true
			break
		}
	}
	if !found {
		t.Error("CopyDir was not called with expected arguments")
	}
}

// TestSuccessfulBuildErrorHandling verifies errors propagating up the chain.
func TestSuccessfulBuildErrorHandling(t *testing.T) {
	comp := Component{
		ID:        "fail-id",
		SetSource: "./fail.svelte", // Note: Struct fields likely int, using Field access or setter? 
		// Assuming BuildPath setter or field access for cleaner test structure:
		BuildPath: "build/fail-id", 
	}

	// Inject a MockFileSystem that fails on MkdirAll
	mockFS := NewMockFileSystem()
	mockFS.MkdirAllFail = true // Assuming we added a field to mockFS

	service := NewSvelteService(SvelteServiceConfig{
		CompilerPath: "svelte",
		FS:           mockFS,
		Logger:       &MockLogger{},
	})

	err := service.BuildComponent(comp)
	if err == nil {
		t.Error("Expected error when MkdirAll fails, but got nil")
	}
}

// MockLogger satisfies the ServiceLogger interface
type MockLogger struct{}

func (l *MockLogger) Info(msg string) {
	// Log to stdout for visibility in tests if desired
	// fmt.Println("[INFO]", msg)
}

func (l *MockLogger) Error(msg string) {
	fmt.Println("[ERROR]", msg)
}