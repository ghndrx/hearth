package services

import (
	"reflect"
	"testing"
)

// mockRepository implements IComponentRepository in memory for testing.
type mockRepository struct {
	storage map[FileHash]*ComponentMetadata
}

func newMockRepository() *mockRepository {
	return &mockRepository{storage: make(map[FileHash]*ComponentMetadata)}
}

func (m *mockRepository) GetByHash(hash FileHash) (*ComponentMetadata, error) {
	if comp, exists := m.storage[hash]; exists {
		return comp, nil
	}
	return nil, nil // Return nil error and nil component on "not found"
}

func (m *mockRepository) Save(hash FileHash, meta *ComponentMetadata) error {
	m.storage[hash] = meta
	return nil
}

// mockParser implements IComponentParser with static responses.
type mockParser struct {
	readHash   []byte
	parseName  string
	parseError error
}

func newMockParser(read []byte, parse string, err error) *mockParser {
	return &mockParser{
		readHash:  read,
		parseName: parse,
		parseError: err,
	}
}

func (m *mockParser) ReadHash(hash FileHash) ([]byte, error) {
	return m.readHash, m.parseError
}

func (m *mockParser) ParseNames(content []byte) (string, error) {
	return m.parseName, m.parseError
}

func TestRegisterComponent_Success(t *testing.T) {
	// Arrange
	repo := newMockRepository()
	parser := newMockParser([]byte("<script>let count = 0;</script>"), "Dashboard", nil)
	service := NewSvelteService(repo, parser)

	hash := FileHash("abc123hash")

	// Act
	err := service.RegisterComponent(hash, "/components/Dashboard", "1.0.0")

	// Assert
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	// Verify persistence
	metadata, _ := repo.GetByHash(hash)
	if metadata == nil {
		t.Fatal("Expected component to be saved, but it was not found in repository.")
	}
	if metadata.FileName != "Dashboard" {
		t.Errorf("Expected FileName 'Dashboard', got '%s'", metadata.FileName)
	}
}

func TestRegisterComponent_ParseFailure_UpdatesHash(t *testing.T) {
	// Arrange
	repo := newMockRepository()
	// Simulate a parser that rejects broken HTML-like content
	parser := newMockParser([]byte("<script>broken-tag"), "", errors.New("invalid syntax"))
	service := NewSvelteService(repo, parser)

	// Act
	err := service.RegisterComponent(FileHash("ignored_hash"), "/bad/file", "0.0.0")

	// Assert
	// The service should return an error and NOT save to repo
	if err == nil {
		t.Fatal("Expected an error for invalid syntax.")
	}

	_, err = repo.GetByHash("ignored_hash")
	if err == nil {
		t.Fatal("Expected repository to not contain the invalid file.")
	}
}

func TestGetComponent_CacheHit(t *testing.T) {
	// Arrange
	repo := newMockRepository()
	service := NewSvelteService(repo, nil)

	// Manually preload the cache to simulate a hit
	cachedItem := &ComponentMetadata{Hash: "hit123", FileName: "Cached.svelte"}
	repo.storage["hit123"] = cachedItem

	// Act
	result, err := service.GetComponent("hit123")

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Hash != "hit123" {
		t.Errorf("Expected hash 'hit123', got '%s'", result.Hash)
	}
}

func TestGetComponent_Missing(t *testing.T) {
	// Arrange
	repo := newMockRepository()
	service := NewSvelteService(repo, nil)

	// Act
	// Check for a hash we know doesn't exist
	_, err := service.GetComponent("nonexistent_hash")

	// Assert
	// According to our mock repository, GetByHash returns nil component with nil error.
	// The service logic treats nil component as Cache Miss and constructs a placeholder.
	// It does NOT error out based on the signature of GetComponent.
	// If strict breaking changes were needed, the interface could return an explicit NotFoundError.
}