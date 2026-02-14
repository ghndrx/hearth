package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// FileHash represents the SHA256 checksum of a Svelte component file.
type FileHash string

// ComponentMetadata defines the data returned for a component context.
type ComponentMetadata struct {
	Hash     FileHash
	Path     string
	Version  string // e.g., "4.2.0"
	FileName string // e.g., "Dashboard.svelte"
}

// IComponentRepository defines the interface for managing component data.
// This abstraction allows us to mock the storage layer (DB, FS, etc.) during testing.
type IComponentRepository interface {
	GetByHash(hash FileHash) (*ComponentMetadata, error)
	Save(hash FileHash, meta *ComponentMetadata) error
}

// IComponentParser defines the interface for reading raw file content.
// In a real scenario, this might involve AST parsing to validate component syntax.
type IComponentParser interface {
	ReadHash(hash FileHash) ([]byte, error)
	ParseNames(content []byte) (string, error)
}

// SvelteService handles the high-level logic for component retrieval.
type SvelteService struct {
	repo     IComponentRepository
	parser   IComponentParser
}

// NewSvelteService creates a new instance of the service.
func NewSvelteService(repo IComponentRepository, parser IComponentParser) *SvelteService {
	return &SvelteService{
		repo:     repo,
		parser:   parser,
	}
}

// RegisterComponent is the entry point for persisting new components.
func (s *SvelteService) RegisterComponent(rawHash FileHash, path string, version string) error {
	// 1. Pre-validate logic (e.g., check if the file exists/is parsable)
	// Note: IComponentParser is used to verify existence.
	content, err := s.parser.ReadHash(rawHash)
	if err != nil {
		return errors.New("failed to access component source file")
	}

	// We could extract the file name here or estimate it based on the content/name
	info := &ComponentMetadata{
		Hash:    rawHash,
		Path:    path,
		Version: version,
	}

	// 2. Business Logic: Ensure the file is a valid Svelte component
	// (This is a simplistic check, normally we would use a proper Svelte parser library)
	if _, err := s.parser.ParseNames(content); err != nil {
		return errors.New("failed to parse component; not a valid Svelte file")
	}

	// 3. Persist
	return s.repo.Save(rawHash, info)
}

// GetComponent retrieves a component by its hash, preferring cache/storage over re-calculating.
func (s *SvelteService) GetComponent(hash FileHash) (*ComponentMetadata, error) {
	// 1. Check Cache/Storage
	component, err := s.repo.GetByHash(hash)
	if err != nil {
		return nil, err
	}

	// 2. Cache Hit
	if component != nil {
		return component, nil
	}

	// 3. Cache Miss: Re-calculate or fetch from source (placeholder logic)
	// In a real scenario, this might look up a remote CDN by hash.
	
	// For this example, since GetByHash returned nil (no error), we construct a placeholder.
	// A production service would handle this differently.
	component = &ComponentMetadata{
		Hash:    hash,
		FileName: "Unknown.svelte",
		Version: "dev",
		Path:    "/unknown",
	}

	// Update Storage/Cache (assume it was lost)
	_ = s.repo.Save(hash, component)

	return component, nil
}

// GenerateHash calculates the SHA256 hash of a string using standard Svelte asset naming rules.
// Note: In SvelteKit, this is usually handled internally, but we implement it for clarity.
func GenerateHash(content []byte) FileHash {
	hash := sha256.Sum256(content)
	return FileHash(hex.EncodeToString(hash[:]))
}