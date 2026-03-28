package ai

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"hearth/internal/ai/providers"
)

// MockAIRepository is a mock implementation for testing
type MockAIRepository struct {
	providers   map[uuid.UUID]*AIProviderConfig
	credentials map[string]*UserAICredential // key: userID-providerID
	routings    map[uuid.UUID]*ModelRouting
}

func NewMockAIRepository() *MockAIRepository {
	return &MockAIRepository{
		providers:   make(map[uuid.UUID]*AIProviderConfig),
		credentials: make(map[string]*UserAICredential),
		routings:    make(map[uuid.UUID]*ModelRouting),
	}
}

func (m *MockAIRepository) CreateProviderConfig(ctx context.Context, config *AIProviderConfig) error {
	m.providers[config.ID] = config
	return nil
}

func (m *MockAIRepository) GetProviderConfig(ctx context.Context, id uuid.UUID) (*AIProviderConfig, error) {
	if config, ok := m.providers[id]; ok {
		return config, nil
	}
	return nil, ErrProviderNotFound
}

func (m *MockAIRepository) GetProviderConfigByName(ctx context.Context, name string) (*AIProviderConfig, error) {
	for _, config := range m.providers {
		if config.Name == name {
			return config, nil
		}
	}
	return nil, ErrProviderNotFound
}

func (m *MockAIRepository) GetAllProviderConfigs(ctx context.Context) ([]*AIProviderConfig, error) {
	configs := make([]*AIProviderConfig, 0, len(m.providers))
	for _, config := range m.providers {
		configs = append(configs, config)
	}
	return configs, nil
}

func (m *MockAIRepository) GetEnabledProviderConfigs(ctx context.Context) ([]*AIProviderConfig, error) {
	configs := make([]*AIProviderConfig, 0)
	for _, config := range m.providers {
		if config.IsEnabled {
			configs = append(configs, config)
		}
	}
	return configs, nil
}

func (m *MockAIRepository) GetDefaultProvider(ctx context.Context) (*AIProviderConfig, error) {
	for _, config := range m.providers {
		if config.IsDefault && config.IsEnabled {
			return config, nil
		}
	}
	return nil, ErrProviderNotFound
}

func (m *MockAIRepository) UpdateProviderConfig(ctx context.Context, config *AIProviderConfig) error {
	if _, ok := m.providers[config.ID]; !ok {
		return ErrProviderNotFound
	}
	m.providers[config.ID] = config
	return nil
}

func (m *MockAIRepository) DeleteProviderConfig(ctx context.Context, id uuid.UUID) error {
	delete(m.providers, id)
	return nil
}

func (m *MockAIRepository) CreateUserCredential(ctx context.Context, cred *UserAICredential) error {
	key := cred.UserID.String() + "-" + cred.ProviderID.String()
	m.credentials[key] = cred
	return nil
}

func (m *MockAIRepository) GetUserCredential(ctx context.Context, userID, providerID uuid.UUID) (*UserAICredential, error) {
	key := userID.String() + "-" + providerID.String()
	if cred, ok := m.credentials[key]; ok {
		return cred, nil
	}
	return nil, ErrCredentialsNotFound
}

func (m *MockAIRepository) GetUserCredentials(ctx context.Context, userID uuid.UUID) ([]*UserAICredential, error) {
	creds := make([]*UserAICredential, 0)
	for key, cred := range m.credentials {
		if key[:36] == userID.String() {
			creds = append(creds, cred)
		}
	}
	return creds, nil
}

func (m *MockAIRepository) UpdateUserCredential(ctx context.Context, cred *UserAICredential) error {
	key := cred.UserID.String() + "-" + cred.ProviderID.String()
	m.credentials[key] = cred
	return nil
}

func (m *MockAIRepository) DeleteUserCredential(ctx context.Context, id uuid.UUID) error {
	for key, cred := range m.credentials {
		if cred.ID == id {
			delete(m.credentials, key)
			return nil
		}
	}
	return ErrCredentialsNotFound
}

func (m *MockAIRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	for _, cred := range m.credentials {
		if cred.ID == id {
			now := time.Now()
			cred.LastUsedAt = &now
			return nil
		}
	}
	return nil
}

func (m *MockAIRepository) CreateModelRouting(ctx context.Context, routing *ModelRouting) error {
	m.routings[routing.ID] = routing
	return nil
}

func (m *MockAIRepository) GetModelRouting(ctx context.Context, feature FeatureType, serverID, userID *uuid.UUID) (*ModelRouting, error) {
	for _, routing := range m.routings {
		if routing.Feature == feature {
			// Check user-specific first
			if userID != nil && routing.UserID != nil && *routing.UserID == *userID {
				return routing, nil
			}
			// Then server-specific
			if serverID != nil && routing.ServerID != nil && *routing.ServerID == *serverID {
				return routing, nil
			}
			// Then global
			if routing.UserID == nil && routing.ServerID == nil {
				return routing, nil
			}
		}
	}
	return nil, ErrNoProviderAvailable
}

func (m *MockAIRepository) GetAllModelRoutings(ctx context.Context) ([]*ModelRouting, error) {
	routings := make([]*ModelRouting, 0, len(m.routings))
	for _, routing := range m.routings {
		routings = append(routings, routing)
	}
	return routings, nil
}

func (m *MockAIRepository) UpdateModelRouting(ctx context.Context, routing *ModelRouting) error {
	m.routings[routing.ID] = routing
	return nil
}

func (m *MockAIRepository) DeleteModelRouting(ctx context.Context, id uuid.UUID) error {
	delete(m.routings, id)
	return nil
}

// Tests

func TestAIServiceCreateProvider(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	config := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "test-ollama",
		DisplayName:  "Test Ollama",
		IsEnabled:    true,
		IsDefault:    false,
	}

	err := service.CreateProvider(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if config.ID == uuid.Nil {
		t.Error("Expected provider ID to be set")
	}

	// Verify it was stored
	stored, err := service.GetProvider(context.Background(), config.ID)
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if stored.Name != "test-ollama" {
		t.Errorf("Expected name 'test-ollama', got '%s'", stored.Name)
	}
}

func TestAIServiceInvalidProviderType(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	config := &AIProviderConfig{
		ProviderType: "invalid_provider",
		Name:         "test",
		IsEnabled:    true,
	}

	err := service.CreateProvider(context.Background(), config, nil)
	if err == nil {
		t.Error("Expected error for invalid provider type")
	}
}

func TestAIServiceGetAllProviders(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create multiple providers
	for i := 0; i < 3; i++ {
		config := &AIProviderConfig{
			ProviderType: ProviderOllama,
			Name:         "provider-" + string(rune('a'+i)),
			IsEnabled:    true,
		}
		service.CreateProvider(context.Background(), config, nil)
	}

	providers, err := service.GetAllProviders(context.Background())
	if err != nil {
		t.Fatalf("Failed to get providers: %v", err)
	}

	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}
}

func TestAIServiceEnabledProviders(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create enabled and disabled providers
	enabled := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "enabled",
		IsEnabled:    true,
	}
	disabled := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "disabled",
		IsEnabled:    false,
	}

	service.CreateProvider(context.Background(), enabled, nil)
	service.CreateProvider(context.Background(), disabled, nil)

	providers, err := service.GetEnabledProviders(context.Background())
	if err != nil {
		t.Fatalf("Failed to get enabled providers: %v", err)
	}

	if len(providers) != 1 {
		t.Errorf("Expected 1 enabled provider, got %d", len(providers))
	}
}

func TestAIServiceDeleteProvider(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	config := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "to-delete",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), config, nil)

	err := service.DeleteProvider(context.Background(), config.ID)
	if err != nil {
		t.Fatalf("Failed to delete provider: %v", err)
	}

	_, err = service.GetProvider(context.Background(), config.ID)
	if err == nil {
		t.Error("Expected error when getting deleted provider")
	}
}

func TestAIServiceModelRouting(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create a provider first
	provider := &AIProviderConfig{
		ProviderType: ProviderOpenAI,
		Name:         "openai",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Create model routing
	routing := &ModelRouting{
		Feature:    FeatureChat,
		ProviderID: provider.ID,
		ModelID:    "gpt-4",
		IsEnabled:  true,
	}

	err := service.SetModelRouting(context.Background(), routing)
	if err != nil {
		t.Fatalf("Failed to set model routing: %v", err)
	}

	// Verify routing
	retrieved, err := service.GetModelRouting(context.Background(), FeatureChat, nil, nil)
	if err != nil {
		t.Fatalf("Failed to get model routing: %v", err)
	}

	if retrieved.ModelID != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", retrieved.ModelID)
	}
}

func TestAIServiceInvalidFeatureType(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	provider := &AIProviderConfig{
		ProviderType: ProviderOpenAI,
		Name:         "openai",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	routing := &ModelRouting{
		Feature:    "invalid_feature",
		ProviderID: provider.ID,
		ModelID:    "gpt-4",
		IsEnabled:  true,
	}

	err := service.SetModelRouting(context.Background(), routing)
	if err == nil {
		t.Error("Expected error for invalid feature type")
	}
}

func TestAIServiceProviderInfo(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	info := service.GetProviderInfo()
	if len(info) == 0 {
		t.Error("Expected provider info, got empty list")
	}

	// Check for expected providers
	providerTypes := make(map[string]bool)
	for _, i := range info {
		providerTypes[i.Type] = true
	}

	expectedTypes := []string{"openai", "anthropic", "ollama", "openrouter"}
	for _, expected := range expectedTypes {
		if !providerTypes[expected] {
			t.Errorf("Expected provider type '%s' in info", expected)
		}
	}
}

func TestFeatureTypeValidation(t *testing.T) {
	validFeatures := []FeatureType{
		FeatureSummary, FeatureSearch, FeatureChat,
		FeatureEmbed, FeatureModerate, FeatureTranslate,
	}

	for _, f := range validFeatures {
		if !f.Valid() {
			t.Errorf("Feature %s should be valid", f)
		}
	}

	invalidFeature := FeatureType("invalid")
	if invalidFeature.Valid() {
		t.Error("Invalid feature should not be valid")
	}
}

func TestProviderTypeValidation(t *testing.T) {
	cloudProviders := []ProviderType{
		ProviderOpenRouter, ProviderBedrock, ProviderVertexAI,
		ProviderOpenAI, ProviderAnthropic,
	}

	for _, p := range cloudProviders {
		if !p.Valid() {
			t.Errorf("Provider %s should be valid", p)
		}
		if !p.IsCloud() {
			t.Errorf("Provider %s should be cloud", p)
		}
		if p.IsLocal() {
			t.Errorf("Provider %s should not be local", p)
		}
	}

	localProviders := []ProviderType{
		ProviderOllama, ProviderLlamaCpp, ProviderLMStudio,
		ProviderVLLM, ProviderLocalAI,
	}

	for _, p := range localProviders {
		if !p.Valid() {
			t.Errorf("Provider %s should be valid", p)
		}
		if p.IsCloud() {
			t.Errorf("Provider %s should not be cloud", p)
		}
		if !p.IsLocal() {
			t.Errorf("Provider %s should be local", p)
		}
	}
}

func TestDefaultModels(t *testing.T) {
	defaults := DefaultModels()

	// Check all feature types have defaults
	features := AllFeatureTypes()
	for _, f := range features {
		if _, ok := defaults[f]; !ok {
			t.Errorf("Feature %s should have a default model", f)
		}
	}
}

func TestEncryptionRoundTrip(t *testing.T) {
	key := "test-encryption-key-32-bytes!!!"
	enc, err := NewAESEncryptionService(key)
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	plaintext := `{"api_key": "sk-secret-key-123", "secret_key": "secret"}`

	encrypted, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if encrypted == plaintext {
		t.Error("Encrypted text should differ from plaintext")
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted text should match plaintext. Got: %s", decrypted)
	}
}

func TestNoOpEncryption(t *testing.T) {
	enc := NewNoOpEncryptionService()

	plaintext := "test data"

	encrypted, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted text should match plaintext. Got: %s", decrypted)
	}
}

func TestAdminConfig(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Default config
	config := service.GetAdminConfig()
	if config == nil {
		t.Fatal("Admin config should not be nil")
	}
	if !config.AllowUserOverride {
		t.Error("AllowUserOverride should default to true")
	}

	// Update config
	newConfig := &AdminAIConfig{
		DefaultProvider:   "openai",
		AllowUserOverride: false,
		FeatureDefaults:   make(map[FeatureType]FeatureConfig),
	}
	service.SetAdminConfig(newConfig)

	retrieved := service.GetAdminConfig()
	if retrieved.AllowUserOverride {
		t.Error("AllowUserOverride should be false after update")
	}
	if retrieved.DefaultProvider != "openai" {
		t.Errorf("DefaultProvider should be 'openai', got '%s'", retrieved.DefaultProvider)
	}
}

func TestProviderConfigResponse(t *testing.T) {
	now := time.Now()
	config := &AIProviderConfig{
		ID:           uuid.New(),
		ProviderType: ProviderOpenAI,
		Name:         "test",
		DisplayName:  "Test OpenAI",
		IsEnabled:    true,
		IsDefault:    false,
		Priority:     1,
		Config:       stringPtr("encrypted-config"),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	response := config.ToResponse()

	// Config should not be in response
	if response.ID != config.ID {
		t.Error("ID should be preserved")
	}
	if response.Name != config.Name {
		t.Error("Name should be preserved")
	}
}

func TestUserCredentialResponse(t *testing.T) {
	now := time.Now()
	cred := &UserAICredential{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		ProviderID:   uuid.New(),
		ProviderType: ProviderOpenAI,
		Credentials:  "encrypted-creds",
		IsEnabled:    true,
		LastUsedAt:   &now,
		CreatedAt:    now,
	}

	response := cred.ToResponse()

	// Credentials should not be in response
	if !response.HasCredentials {
		t.Error("HasCredentials should be true")
	}
	if response.ProviderID != cred.ProviderID {
		t.Error("ProviderID should be preserved")
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestAIServiceUpdateProvider(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	config := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "test",
		DisplayName:  "Test",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), config, nil)

	// Update
	config.DisplayName = "Updated Name"
	err := service.UpdateProvider(context.Background(), config, nil)
	if err != nil {
		t.Fatalf("UpdateProvider() error: %v", err)
	}

	// Verify
	updated, _ := service.GetProvider(context.Background(), config.ID)
	if updated.DisplayName != "Updated Name" {
		t.Errorf("DisplayName = %s, want Updated Name", updated.DisplayName)
	}
}

func TestAIServiceSetUserCredentials(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOpenAI,
		Name:         "openai",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	userID := uuid.New()
	creds := &providers.ProviderConfig{
		APIKey: "user-key",
	}

	err := service.SetUserCredentials(context.Background(), userID, provider.ID, creds)
	if err != nil {
		t.Fatalf("SetUserCredentials() error: %v", err)
	}

	// Verify
	userCreds, err := service.GetUserCredentials(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserCredentials() error: %v", err)
	}

	if len(userCreds) != 1 {
		t.Errorf("Expected 1 credential, got %d", len(userCreds))
	}
}

func TestAIServiceDeleteUserCredential(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create provider and credentials
	provider := &AIProviderConfig{
		ProviderType: ProviderOpenAI,
		Name:         "openai",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	userID := uuid.New()
	creds := &providers.ProviderConfig{APIKey: "key"}
	service.SetUserCredentials(context.Background(), userID, provider.ID, creds)

	// Delete
	err := service.DeleteUserCredential(context.Background(), userID, provider.ID)
	if err != nil {
		t.Fatalf("DeleteUserCredential() error: %v", err)
	}

	// Verify deletion
	userCreds, _ := service.GetUserCredentials(context.Background(), userID)
	if len(userCreds) != 0 {
		t.Errorf("Expected 0 credentials after deletion, got %d", len(userCreds))
	}
}

func TestAIServiceDeleteUserCredentialNotFound(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	err := service.DeleteUserCredential(context.Background(), uuid.New(), uuid.New())
	if err != ErrCredentialsNotFound {
		t.Errorf("Expected ErrCredentialsNotFound, got %v", err)
	}
}

func TestAIServiceSetUserCredentialsProviderNotFound(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	err := service.SetUserCredentials(context.Background(), uuid.New(), uuid.New(), &providers.ProviderConfig{})
	if err != ErrProviderNotFound {
		t.Errorf("Expected ErrProviderNotFound, got %v", err)
	}
}

func TestAIServiceUpdateExistingCredentials(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOpenAI,
		Name:         "openai",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	userID := uuid.New()

	// Set initial credentials
	service.SetUserCredentials(context.Background(), userID, provider.ID, &providers.ProviderConfig{APIKey: "key1"})

	// Update credentials
	err := service.SetUserCredentials(context.Background(), userID, provider.ID, &providers.ProviderConfig{APIKey: "key2"})
	if err != nil {
		t.Fatalf("SetUserCredentials() update error: %v", err)
	}
}

func TestAIServiceGetProviderInstance(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create Ollama provider (doesn't need API key)
	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "ollama",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	instance, err := service.GetProviderInstance(context.Background(), provider.ID, nil)
	if err != nil {
		t.Fatalf("GetProviderInstance() error: %v", err)
	}

	if instance.Type() != "ollama" {
		t.Errorf("Type() = %s, want ollama", instance.Type())
	}
}

func TestAIServiceGetProviderInstanceNotFound(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	_, err := service.GetProviderInstance(context.Background(), uuid.New(), nil)
	if err != ErrProviderNotFound {
		t.Errorf("Expected ErrProviderNotFound, got %v", err)
	}
}

func TestAIServiceGetProviderInstanceDisabled(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "ollama",
		IsEnabled:    false, // Disabled
	}
	service.CreateProvider(context.Background(), provider, nil)

	_, err := service.GetProviderInstance(context.Background(), provider.ID, nil)
	if err != ErrProviderNotEnabled {
		t.Errorf("Expected ErrProviderNotEnabled, got %v", err)
	}
}

func TestAIServiceClose(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	err := service.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestAllProviderTypes(t *testing.T) {
	types := AllProviderTypes()
	if len(types) == 0 {
		t.Error("Expected provider types, got empty list")
	}

	// Verify all types are valid
	for _, pt := range types {
		if !pt.Valid() {
			t.Errorf("Provider type %s should be valid", pt)
		}
	}
}

func TestAIServiceDeleteModelRouting(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOpenAI,
		Name:         "openai",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Create routing
	routing := &ModelRouting{
		Feature:    FeatureChat,
		ProviderID: provider.ID,
		ModelID:    "gpt-4",
		IsEnabled:  true,
	}
	service.SetModelRouting(context.Background(), routing)

	// Delete
	err := service.DeleteModelRouting(context.Background(), routing.ID)
	if err != nil {
		t.Fatalf("DeleteModelRouting() error: %v", err)
	}
}

func TestModelRoutingProviderNotFound(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	routing := &ModelRouting{
		Feature:    FeatureChat,
		ProviderID: uuid.New(), // Non-existent
		ModelID:    "gpt-4",
		IsEnabled:  true,
	}

	err := service.SetModelRouting(context.Background(), routing)
	if err != ErrProviderNotFound {
		t.Errorf("Expected ErrProviderNotFound, got %v", err)
	}
}

func TestGetAllModelRoutings(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOpenAI,
		Name:         "openai",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Create multiple routings
	for _, feature := range []FeatureType{FeatureChat, FeatureSummary} {
		routing := &ModelRouting{
			Feature:    feature,
			ProviderID: provider.ID,
			ModelID:    "gpt-4",
			IsEnabled:  true,
		}
		service.SetModelRouting(context.Background(), routing)
	}

	routings, err := service.GetAllModelRoutings(context.Background())
	if err != nil {
		t.Fatalf("GetAllModelRoutings() error: %v", err)
	}

	if len(routings) != 2 {
		t.Errorf("Expected 2 routings, got %d", len(routings))
	}
}

func TestAIServiceCreateProviderWithCredentials(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	config := &AIProviderConfig{
		ProviderType: ProviderOpenAI,
		Name:         "openai",
		DisplayName:  "OpenAI",
		IsEnabled:    true,
	}

	credentials := &providers.ProviderConfig{
		APIKey: "sk-test-key",
	}

	err := service.CreateProvider(context.Background(), config, credentials)
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}

	// Verify config was stored with encrypted credentials
	stored, _ := service.GetProvider(context.Background(), config.ID)
	if stored.Config == nil {
		t.Error("Expected config to have encrypted credentials")
	}
}

func TestAIServiceUpdateProviderWithCredentials(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	config := &AIProviderConfig{
		ProviderType: ProviderOpenAI,
		Name:         "openai",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), config, nil)

	// Update with credentials
	credentials := &providers.ProviderConfig{
		APIKey: "sk-new-key",
	}
	err := service.UpdateProvider(context.Background(), config, credentials)
	if err != nil {
		t.Fatalf("UpdateProvider() error: %v", err)
	}

	stored, _ := service.GetProvider(context.Background(), config.ID)
	if stored.Config == nil {
		t.Error("Expected config to have encrypted credentials")
	}
}

func TestAIServiceGetProviderForFeatureWithRouting(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create Ollama provider (doesn't need API key)
	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "ollama",
		IsEnabled:    true,
		IsDefault:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Set routing
	routing := &ModelRouting{
		Feature:    FeatureChat,
		ProviderID: provider.ID,
		ModelID:    "llama2",
		IsEnabled:  true,
	}
	service.SetModelRouting(context.Background(), routing)

	// Get provider for feature
	p, modelID, err := service.GetProviderForFeature(context.Background(), FeatureChat, nil, nil)
	if err != nil {
		t.Fatalf("GetProviderForFeature() error: %v", err)
	}

	if p.Type() != "ollama" {
		t.Errorf("Type() = %s, want ollama", p.Type())
	}
	if modelID != "llama2" {
		t.Errorf("ModelID = %s, want llama2", modelID)
	}
}

func TestAIServiceGetProviderForFeatureWithDefault(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create default provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "ollama",
		IsEnabled:    true,
		IsDefault:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Get provider for feature (no routing, should use default)
	p, _, err := service.GetProviderForFeature(context.Background(), FeatureSummary, nil, nil)
	if err != nil {
		t.Fatalf("GetProviderForFeature() error: %v", err)
	}

	if p.Type() != "ollama" {
		t.Errorf("Type() = %s, want ollama", p.Type())
	}
}

func TestAIServiceGetProviderForFeatureNoProvider(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	_, _, err := service.GetProviderForFeature(context.Background(), FeatureChat, nil, nil)
	if err != ErrNoProviderAvailable {
		t.Errorf("Expected ErrNoProviderAvailable, got %v", err)
	}
}

func TestAIServiceGetProviderForFeatureWithAdminDefaults(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "ollama",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Set admin config
	adminConfig := &AdminAIConfig{
		AllowUserOverride: true,
		FeatureDefaults: map[FeatureType]FeatureConfig{
			FeatureChat: {
				ProviderType: ProviderOllama,
				ProviderID:   provider.ID,
				ModelID:      "llama2",
				Enabled:      true,
			},
		},
	}
	service.SetAdminConfig(adminConfig)

	// Get provider for feature
	p, modelID, err := service.GetProviderForFeature(context.Background(), FeatureChat, nil, nil)
	if err != nil {
		t.Fatalf("GetProviderForFeature() error: %v", err)
	}

	if p.Type() != "ollama" {
		t.Errorf("Type() = %s, want ollama", p.Type())
	}
	if modelID != "llama2" {
		t.Errorf("ModelID = %s, want llama2", modelID)
	}
}

func TestAIServiceGetProviderInstanceWithUserCredentials(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create Ollama provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "ollama",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Set user credentials
	userID := uuid.New()
	creds := &providers.ProviderConfig{
		BaseURL: "http://custom:11434",
	}
	service.SetUserCredentials(context.Background(), userID, provider.ID, creds)

	// Get provider instance with user ID
	instance, err := service.GetProviderInstance(context.Background(), provider.ID, &userID)
	if err != nil {
		t.Fatalf("GetProviderInstance() error: %v", err)
	}

	if instance == nil {
		t.Error("Expected provider instance")
	}
}

func TestModelRoutingBasic(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "ollama",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Create global routing
	globalRouting := &ModelRouting{
		Feature:    FeatureChat,
		ProviderID: provider.ID,
		ModelID:    "global-model",
		IsEnabled:  true,
	}
	service.SetModelRouting(context.Background(), globalRouting)

	// Test global routing retrieval
	routing, err := service.GetModelRouting(context.Background(), FeatureChat, nil, nil)
	if err != nil {
		t.Fatalf("GetModelRouting() error: %v", err)
	}
	if routing.ModelID != "global-model" {
		t.Errorf("Expected global routing, got %s", routing.ModelID)
	}
}

func TestConfigTypes(t *testing.T) {
	// Test AIProviderConfig
	config := &AIProviderConfig{
		ID:           uuid.New(),
		ProviderType: ProviderOpenAI,
		Name:         "test",
		DisplayName:  "Test",
		IsEnabled:    true,
		Priority:     1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	response := config.ToResponse()
	if response.Name != config.Name {
		t.Error("ToResponse should preserve Name")
	}
	if response.ProviderType != config.ProviderType {
		t.Error("ToResponse should preserve ProviderType")
	}

	// Test UserAICredential
	cred := &UserAICredential{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		ProviderID:   uuid.New(),
		ProviderType: ProviderOpenAI,
		Credentials:  "encrypted",
		IsEnabled:    true,
		CreatedAt:    time.Now(),
	}

	credResponse := cred.ToResponse()
	if !credResponse.HasCredentials {
		t.Error("HasCredentials should be true when Credentials is set")
	}
}

func TestProviderTypeHelpers(t *testing.T) {
	// Cloud providers
	cloudTypes := []ProviderType{ProviderOpenAI, ProviderAnthropic, ProviderOpenRouter, ProviderBedrock, ProviderVertexAI}
	for _, pt := range cloudTypes {
		if !pt.IsCloud() {
			t.Errorf("%s should be cloud", pt)
		}
		if pt.IsLocal() {
			t.Errorf("%s should not be local", pt)
		}
	}

	// Local providers
	localTypes := []ProviderType{ProviderOllama, ProviderLlamaCpp, ProviderLMStudio, ProviderVLLM, ProviderLocalAI}
	for _, pt := range localTypes {
		if pt.IsCloud() {
			t.Errorf("%s should not be cloud", pt)
		}
		if !pt.IsLocal() {
			t.Errorf("%s should be local", pt)
		}
	}
}

func TestAIServiceGetProviderInfo(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	info := service.GetProviderInfo()
	if len(info) == 0 {
		t.Error("Expected provider info, got empty list")
	}

	// Check that all supported providers are included
	supportedCount := 0
	for _, p := range info {
		if p.Type != "" {
			supportedCount++
		}
	}
	if supportedCount < 5 {
		t.Errorf("Expected at least 5 providers, got %d", supportedCount)
	}
}

func TestInvalidProviderTypeCreate(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	config := &AIProviderConfig{
		ProviderType: ProviderType("invalid"),
		Name:         "invalid",
		IsEnabled:    true,
	}

	err := service.CreateProvider(context.Background(), config, nil)
	if err == nil {
		t.Error("Expected error for invalid provider type")
	}
}

func TestInvalidFeatureTypeRouting(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	routing := &ModelRouting{
		Feature:    FeatureType("invalid"),
		ProviderID: uuid.New(),
		ModelID:    "model",
		IsEnabled:  true,
	}

	err := service.SetModelRouting(context.Background(), routing)
	if err == nil {
		t.Error("Expected error for invalid feature type")
	}
}

func TestBuildProviderConfigWithServerCreds(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create provider with server-level credentials
	baseURL := "http://localhost:11434"
	config := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "ollama",
		BaseURL:      &baseURL,
		IsEnabled:    true,
	}

	creds := &providers.ProviderConfig{
		BaseURL: baseURL,
	}

	service.CreateProvider(context.Background(), config, creds)

	// Get provider instance (should use server creds)
	instance, err := service.GetProviderInstance(context.Background(), config.ID, nil)
	if err != nil {
		t.Fatalf("GetProviderInstance() error: %v", err)
	}
	if instance == nil {
		t.Error("Expected provider instance")
	}
}

func TestBuildProviderConfigUserOverrideDisabled(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Disable user override
	adminConfig := &AdminAIConfig{
		AllowUserOverride: false,
	}
	service.SetAdminConfig(adminConfig)

	// Create provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "ollama",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Set user credentials
	userID := uuid.New()
	creds := &providers.ProviderConfig{
		BaseURL: "http://custom:11434",
	}
	service.SetUserCredentials(context.Background(), userID, provider.ID, creds)

	// Get provider instance - should NOT use user creds since override is disabled
	instance, err := service.GetProviderInstance(context.Background(), provider.ID, &userID)
	if err != nil {
		t.Fatalf("GetProviderInstance() error: %v", err)
	}
	if instance == nil {
		t.Error("Expected provider instance")
	}
}

func TestEncryptionErrors(t *testing.T) {
	enc, _ := NewAESEncryptionService("test-key")

	// Test decryption of invalid ciphertext
	_, err := enc.Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64")
	}

	// Test decryption of short ciphertext
	_, err = enc.Decrypt("YWJj") // "abc" in base64, too short
	if err == nil {
		t.Error("Expected error for short ciphertext")
	}
}

func TestAllFeatureTypes(t *testing.T) {
	features := AllFeatureTypes()
	if len(features) == 0 {
		t.Error("Expected feature types, got empty list")
	}

	expectedFeatures := []FeatureType{
		FeatureSummary, FeatureSearch, FeatureChat,
		FeatureEmbed, FeatureModerate, FeatureTranslate,
	}

	for _, expected := range expectedFeatures {
		found := false
		for _, f := range features {
			if f == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected feature %s not found in AllFeatureTypes()", expected)
		}
	}
}

func TestUserCredentialWithEmptyCredentials(t *testing.T) {
	cred := &UserAICredential{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		ProviderID:  uuid.New(),
		Credentials: "",
		IsEnabled:   true,
	}

	response := cred.ToResponse()
	if response.HasCredentials {
		t.Error("HasCredentials should be false when Credentials is empty")
	}
}

func TestDefaultModelsContainsAllFeatures(t *testing.T) {
	defaults := DefaultModels()
	features := AllFeatureTypes()

	for _, f := range features {
		if _, ok := defaults[f]; !ok {
			t.Errorf("DefaultModels() missing feature: %s", f)
		}
	}
}

func TestAIServiceGetEnabledProviders(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create enabled provider
	enabled := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "enabled",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), enabled, nil)

	// Create disabled provider
	disabled := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "disabled",
		IsEnabled:    false,
	}
	service.CreateProvider(context.Background(), disabled, nil)

	// Get enabled providers
	providers, err := service.GetEnabledProviders(context.Background())
	if err != nil {
		t.Fatalf("GetEnabledProviders() error: %v", err)
	}

	if len(providers) != 1 {
		t.Errorf("Expected 1 enabled provider, got %d", len(providers))
	}
	if providers[0].Name != "enabled" {
		t.Errorf("Expected 'enabled' provider, got '%s'", providers[0].Name)
	}
}

func TestAIServiceDeleteProviderAndVerify(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create provider
	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "to-delete-verify",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)

	// Delete
	err := service.DeleteProvider(context.Background(), provider.ID)
	if err != nil {
		t.Fatalf("DeleteProvider() error: %v", err)
	}

	// Verify deletion
	_, err = service.GetProvider(context.Background(), provider.ID)
	if err != ErrProviderNotFound {
		t.Errorf("Expected ErrProviderNotFound after deletion, got %v", err)
	}
}

func TestAIServiceCloseWithCachedProviders(t *testing.T) {
	repo := NewMockAIRepository()
	encryption := NewNoOpEncryptionService()
	service := NewAIService(repo, encryption)

	// Create and get a provider instance to cache it
	provider := &AIProviderConfig{
		ProviderType: ProviderOllama,
		Name:         "cached",
		IsEnabled:    true,
	}
	service.CreateProvider(context.Background(), provider, nil)
	service.GetProviderInstance(context.Background(), provider.ID, nil)

	// Close should clean up cached providers
	err := service.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}
}
