package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCompilerService creates a mock implementation of CompilerService for unit testing.
type MockCompilerService struct {
	mock.Mock
}

func (m *MockCompilerService) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCompilerService) FetchComponent(ctx context.Context, tpl ComponentTemplate) (string, error) {
	args := m.Called(ctx, tpl)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockCompilerService) WatchChanges(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestNewSvelteService tests that the service fails if the root path does not exist.
func TestNewSvelteService(t *testing.T) {
	invalidPath := "/non/existent/path"

	_, err := NewSvelteService(invalidPath, nil)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

// TestFetchComponent tests the compilation logic.
func TestFetchComponent(t *testing.T) {
	// 1. Setup
	logger := &log.Logger{}
	service, err := NewSvelteService(".", logger) // Use current folder for test
	assert.NoError(t, err)

	ctx := context.Background()
	template := ComponentTemplate{
		ID:   "comp-123",
		Name: "TestSidebar",
		Source: `<script>export let name;</script><h1>Hello {name}!</h1>`,
	}

	// 2. Execute
	html, err := service.FetchComponent(ctx, template)

	// 3. Assert
	assert.NoError(t, err)
	assert.Contains(t, html, "TestSidebar")
	assert.Contains(t, html, "Hello TestSidebar!")
	assert.Contains(t, html, "Hearth: Rendering TestSidebar")
	assert.Contains(t, html, `<script> console.log("Hearth: Rendering TestSidebar");</script>`)
}

// TestFetchComponent_Error tests error scenarios (e.g., context cancellation).
func TestFetchComponent_Cancellation(t *testing.T) {
	service, _ := NewSvelteService(".", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := service.FetchComponent(ctx, ComponentTemplate{Name: "test"})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// Example of using the Mock for dependency injection patterns.
func TestMockedImplementation(t *testing.T) {
	mockSvc := new(MockCompilerService)
	ctx := context.Background()

	// Define expectations
	expectedHTML := "<div>Mocked Output</div>"
	mockSvc.On("FetchComponent", ctx, mock.AnythingOfType("services.ComponentTemplate")).
		Return(expectedHTML, nil).
		Once()

	// Performance test
	// (Note: In a production test, this might behave differently, but for mocks it's instant)
	html, err := mockSvc.FetchComponent(ctx, ComponentTemplate{Name: "Mock"})

	// Verify mocks were called
	mockSvc.AssertExpectations(t)
	assert.NoError(t, err)
	assert.Equal(t, expectedHTML, html)
}

// TestWatchChanges_HasBaseDependencies tests that necessary packages are checked
// This is an integration-style test behavior relying on the struct initialization.
func TestSvelteServiceInitialization(t *testing.T) {
	// Use a temporary directory valid for the test runner
	tmpDir := t.TempDir()

	service, err := NewSvelteService(tmpDir, nil)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Verify properties
	assert.Equal(t, tmpDir, service.(*SvelteServiceImplementation).rootPath)

	// Test Initialize (Mocking potential I/O delay or errors)
	ctx := context.Background()
	// Assuming real Init would check node existence, we just check it returns no immediate error
	err = service.Initialize(ctx)
	assert.NoError(t, err)
}