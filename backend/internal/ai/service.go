package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"hearth/internal/ai/providers"
)

var (
	ErrProviderNotFound     = errors.New("provider not found")
	ErrProviderNotEnabled   = errors.New("provider not enabled")
	ErrCredentialsNotFound  = errors.New("credentials not found")
	ErrNoProviderAvailable  = errors.New("no provider available for this feature")
	ErrInvalidConfiguration = errors.New("invalid configuration")
)

// AIRepository defines the interface for AI data persistence
type AIRepository interface {
	// Provider configurations
	CreateProviderConfig(ctx context.Context, config *AIProviderConfig) error
	GetProviderConfig(ctx context.Context, id uuid.UUID) (*AIProviderConfig, error)
	GetProviderConfigByName(ctx context.Context, name string) (*AIProviderConfig, error)
	GetAllProviderConfigs(ctx context.Context) ([]*AIProviderConfig, error)
	GetEnabledProviderConfigs(ctx context.Context) ([]*AIProviderConfig, error)
	GetDefaultProvider(ctx context.Context) (*AIProviderConfig, error)
	UpdateProviderConfig(ctx context.Context, config *AIProviderConfig) error
	DeleteProviderConfig(ctx context.Context, id uuid.UUID) error

	// User credentials
	CreateUserCredential(ctx context.Context, cred *UserAICredential) error
	GetUserCredential(ctx context.Context, userID, providerID uuid.UUID) (*UserAICredential, error)
	GetUserCredentials(ctx context.Context, userID uuid.UUID) ([]*UserAICredential, error)
	UpdateUserCredential(ctx context.Context, cred *UserAICredential) error
	DeleteUserCredential(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error

	// Model routing
	CreateModelRouting(ctx context.Context, routing *ModelRouting) error
	GetModelRouting(ctx context.Context, feature FeatureType, serverID, userID *uuid.UUID) (*ModelRouting, error)
	GetAllModelRoutings(ctx context.Context) ([]*ModelRouting, error)
	UpdateModelRouting(ctx context.Context, routing *ModelRouting) error
	DeleteModelRouting(ctx context.Context, id uuid.UUID) error
}

// EncryptionService defines the interface for credential encryption
type EncryptionService interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// AIService manages AI providers and model routing
type AIService struct {
	repo       AIRepository
	encryption EncryptionService
	factory    *providers.ProviderFactory

	// Provider cache
	providersMu sync.RWMutex
	providers   map[string]providers.Provider

	// Admin default config
	adminConfig *AdminAIConfig
}

// AdminAIConfig holds server-level admin defaults
type AdminAIConfig struct {
	DefaultProvider   string                        `json:"default_provider"`
	FeatureDefaults   map[FeatureType]FeatureConfig `json:"feature_defaults"`
	AllowUserOverride bool                          `json:"allow_user_override"`
}

// FeatureConfig holds configuration for a specific AI feature
type FeatureConfig struct {
	ProviderType ProviderType `json:"provider_type"`
	ProviderID   uuid.UUID    `json:"provider_id"`
	ModelID      string       `json:"model_id"`
	Enabled      bool         `json:"enabled"`
}

// NewAIService creates a new AI service
func NewAIService(repo AIRepository, encryption EncryptionService) *AIService {
	return &AIService{
		repo:       repo,
		encryption: encryption,
		factory:    providers.NewProviderFactory(),
		providers:  make(map[string]providers.Provider),
		adminConfig: &AdminAIConfig{
			AllowUserOverride: true,
			FeatureDefaults:   make(map[FeatureType]FeatureConfig),
		},
	}
}

// SetAdminConfig updates the admin configuration
func (s *AIService) SetAdminConfig(config *AdminAIConfig) {
	s.adminConfig = config
}

// GetAdminConfig returns the admin configuration
func (s *AIService) GetAdminConfig() *AdminAIConfig {
	return s.adminConfig
}

// --- Provider Configuration ---

// CreateProvider creates a new provider configuration
func (s *AIService) CreateProvider(ctx context.Context, config *AIProviderConfig, credentials *providers.ProviderConfig) error {
	// Validate provider type
	if !config.ProviderType.Valid() {
		return fmt.Errorf("%w: invalid provider type: %s", ErrInvalidConfiguration, config.ProviderType)
	}

	// Encrypt and store credentials if provided
	if credentials != nil {
		credJSON, err := json.Marshal(credentials)
		if err != nil {
			return fmt.Errorf("failed to marshal credentials: %w", err)
		}
		encrypted, err := s.encryption.Encrypt(string(credJSON))
		if err != nil {
			return fmt.Errorf("failed to encrypt credentials: %w", err)
		}
		config.Config = &encrypted
	}

	config.ID = uuid.New()
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	return s.repo.CreateProviderConfig(ctx, config)
}

// GetProvider returns a provider configuration by ID
func (s *AIService) GetProvider(ctx context.Context, id uuid.UUID) (*AIProviderConfig, error) {
	return s.repo.GetProviderConfig(ctx, id)
}

// GetAllProviders returns all provider configurations
func (s *AIService) GetAllProviders(ctx context.Context) ([]*AIProviderConfig, error) {
	return s.repo.GetAllProviderConfigs(ctx)
}

// GetEnabledProviders returns all enabled provider configurations
func (s *AIService) GetEnabledProviders(ctx context.Context) ([]*AIProviderConfig, error) {
	return s.repo.GetEnabledProviderConfigs(ctx)
}

// UpdateProvider updates a provider configuration
func (s *AIService) UpdateProvider(ctx context.Context, config *AIProviderConfig, credentials *providers.ProviderConfig) error {
	if credentials != nil {
		credJSON, err := json.Marshal(credentials)
		if err != nil {
			return fmt.Errorf("failed to marshal credentials: %w", err)
		}
		encrypted, err := s.encryption.Encrypt(string(credJSON))
		if err != nil {
			return fmt.Errorf("failed to encrypt credentials: %w", err)
		}
		config.Config = &encrypted
	}

	config.UpdatedAt = time.Now()

	// Invalidate cached provider
	s.providersMu.Lock()
	delete(s.providers, config.ID.String())
	s.providersMu.Unlock()

	return s.repo.UpdateProviderConfig(ctx, config)
}

// DeleteProvider deletes a provider configuration
func (s *AIService) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	// Invalidate cached provider
	s.providersMu.Lock()
	delete(s.providers, id.String())
	s.providersMu.Unlock()

	return s.repo.DeleteProviderConfig(ctx, id)
}

// --- User Credentials ---

// SetUserCredentials sets user-specific credentials for a provider
func (s *AIService) SetUserCredentials(ctx context.Context, userID, providerID uuid.UUID, credentials *providers.ProviderConfig) error {
	// Get provider config to verify it exists and get type
	providerConfig, err := s.repo.GetProviderConfig(ctx, providerID)
	if err != nil {
		return ErrProviderNotFound
	}

	// Encrypt credentials
	credJSON, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}
	encrypted, err := s.encryption.Encrypt(string(credJSON))
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Check if user already has credentials for this provider
	existing, err := s.repo.GetUserCredential(ctx, userID, providerID)
	if err == nil && existing != nil {
		// Update existing
		existing.Credentials = encrypted
		existing.IsEnabled = true
		existing.UpdatedAt = time.Now()
		return s.repo.UpdateUserCredential(ctx, existing)
	}

	// Create new
	cred := &UserAICredential{
		ID:           uuid.New(),
		UserID:       userID,
		ProviderID:   providerID,
		ProviderType: providerConfig.ProviderType,
		Credentials:  encrypted,
		IsEnabled:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return s.repo.CreateUserCredential(ctx, cred)
}

// GetUserCredentials returns all credentials for a user
func (s *AIService) GetUserCredentials(ctx context.Context, userID uuid.UUID) ([]*UserAICredentialResponse, error) {
	creds, err := s.repo.GetUserCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*UserAICredentialResponse, len(creds))
	for i, cred := range creds {
		resp := cred.ToResponse()
		responses[i] = &resp
	}

	return responses, nil
}

// DeleteUserCredential deletes user credentials for a provider
func (s *AIService) DeleteUserCredential(ctx context.Context, userID, providerID uuid.UUID) error {
	cred, err := s.repo.GetUserCredential(ctx, userID, providerID)
	if err != nil {
		return ErrCredentialsNotFound
	}
	return s.repo.DeleteUserCredential(ctx, cred.ID)
}

// --- Provider Instance Management ---

// GetProviderInstance returns a provider instance for the given configuration and optional user
func (s *AIService) GetProviderInstance(ctx context.Context, providerID uuid.UUID, userID *uuid.UUID) (providers.Provider, error) {
	config, err := s.repo.GetProviderConfig(ctx, providerID)
	if err != nil {
		return nil, ErrProviderNotFound
	}

	if !config.IsEnabled {
		return nil, ErrProviderNotEnabled
	}

	// Build provider config
	providerConfig, err := s.buildProviderConfig(ctx, config, userID)
	if err != nil {
		return nil, err
	}

	// Create provider instance
	return s.factory.Create(string(config.ProviderType), providerConfig)
}

// GetProviderForFeature returns the appropriate provider for a feature
func (s *AIService) GetProviderForFeature(ctx context.Context, feature FeatureType, serverID, userID *uuid.UUID) (providers.Provider, string, error) {
	// Try to get routing configuration
	routing, err := s.repo.GetModelRouting(ctx, feature, serverID, userID)
	if err == nil && routing != nil && routing.IsEnabled {
		provider, err := s.GetProviderInstance(ctx, routing.ProviderID, userID)
		if err == nil {
			return provider, routing.ModelID, nil
		}
	}

	// Fall back to admin defaults
	if featureConfig, ok := s.adminConfig.FeatureDefaults[feature]; ok && featureConfig.Enabled {
		provider, err := s.GetProviderInstance(ctx, featureConfig.ProviderID, userID)
		if err == nil {
			return provider, featureConfig.ModelID, nil
		}
	}

	// Fall back to default provider
	defaultProvider, err := s.repo.GetDefaultProvider(ctx)
	if err != nil {
		return nil, "", ErrNoProviderAvailable
	}

	provider, err := s.GetProviderInstance(ctx, defaultProvider.ID, userID)
	if err != nil {
		return nil, "", err
	}

	// Use default model for feature
	defaultModels := DefaultModels()
	modelID := defaultModels[feature]

	return provider, modelID, nil
}

func (s *AIService) buildProviderConfig(ctx context.Context, config *AIProviderConfig, userID *uuid.UUID) (*providers.ProviderConfig, error) {
	providerConfig := providers.DefaultConfig()

	if config.BaseURL != nil {
		providerConfig.BaseURL = *config.BaseURL
	}

	// Try user credentials first if user is specified and override is allowed
	if userID != nil && s.adminConfig.AllowUserOverride {
		userCred, err := s.repo.GetUserCredential(ctx, *userID, config.ID)
		if err == nil && userCred != nil && userCred.IsEnabled && userCred.Credentials != "" {
			decrypted, err := s.encryption.Decrypt(userCred.Credentials)
			if err == nil {
				var userConfig providers.ProviderConfig
				if err := json.Unmarshal([]byte(decrypted), &userConfig); err == nil {
					// Merge user config
					if userConfig.APIKey != "" {
						providerConfig.APIKey = userConfig.APIKey
					}
					if userConfig.SecretKey != "" {
						providerConfig.SecretKey = userConfig.SecretKey
					}
					if userConfig.Region != "" {
						providerConfig.Region = userConfig.Region
					}
					if userConfig.ProjectID != "" {
						providerConfig.ProjectID = userConfig.ProjectID
					}
					if userConfig.ServiceAccount != "" {
						providerConfig.ServiceAccount = userConfig.ServiceAccount
					}

					// Update last used
					s.repo.UpdateLastUsed(ctx, userCred.ID)

					return providerConfig, nil
				}
			}
		}
	}

	// Use server-level credentials
	if config.Config != nil && *config.Config != "" {
		decrypted, err := s.encryption.Decrypt(*config.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
		}

		if err := json.Unmarshal([]byte(decrypted), providerConfig); err != nil {
			return nil, fmt.Errorf("failed to parse credentials: %w", err)
		}
	}

	return providerConfig, nil
}

// --- Model Routing ---

// SetModelRouting sets the model routing for a feature
func (s *AIService) SetModelRouting(ctx context.Context, routing *ModelRouting) error {
	if !routing.Feature.Valid() {
		return fmt.Errorf("%w: invalid feature type: %s", ErrInvalidConfiguration, routing.Feature)
	}

	// Verify provider exists
	_, err := s.repo.GetProviderConfig(ctx, routing.ProviderID)
	if err != nil {
		return ErrProviderNotFound
	}

	routing.ID = uuid.New()
	routing.CreatedAt = time.Now()
	routing.UpdatedAt = time.Now()

	return s.repo.CreateModelRouting(ctx, routing)
}

// GetModelRouting returns the model routing for a feature
func (s *AIService) GetModelRouting(ctx context.Context, feature FeatureType, serverID, userID *uuid.UUID) (*ModelRouting, error) {
	return s.repo.GetModelRouting(ctx, feature, serverID, userID)
}

// GetAllModelRoutings returns all model routings
func (s *AIService) GetAllModelRoutings(ctx context.Context) ([]*ModelRouting, error) {
	return s.repo.GetAllModelRoutings(ctx)
}

// DeleteModelRouting deletes a model routing
func (s *AIService) DeleteModelRouting(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteModelRouting(ctx, id)
}

// --- High-Level AI Operations ---

// Chat performs a chat completion using the appropriate provider
func (s *AIService) Chat(ctx context.Context, feature FeatureType, request *providers.ChatRequest, serverID, userID *uuid.UUID) (*providers.ChatResponse, error) {
	provider, modelID, err := s.GetProviderForFeature(ctx, feature, serverID, userID)
	if err != nil {
		return nil, err
	}

	// Use default model if not specified
	if request.Model == "" {
		request.Model = modelID
	}

	return provider.Chat(ctx, request)
}

// ChatStream performs a streaming chat completion
func (s *AIService) ChatStream(ctx context.Context, feature FeatureType, request *providers.ChatRequest, callback providers.StreamCallback, serverID, userID *uuid.UUID) error {
	provider, modelID, err := s.GetProviderForFeature(ctx, feature, serverID, userID)
	if err != nil {
		return err
	}

	if request.Model == "" {
		request.Model = modelID
	}

	return provider.ChatStream(ctx, request, callback)
}

// Embed generates embeddings
func (s *AIService) Embed(ctx context.Context, request *providers.EmbeddingRequest, serverID, userID *uuid.UUID) (*providers.EmbeddingResponse, error) {
	provider, modelID, err := s.GetProviderForFeature(ctx, FeatureEmbed, serverID, userID)
	if err != nil {
		return nil, err
	}

	if request.Model == "" {
		request.Model = modelID
	}

	return provider.Embed(ctx, request)
}

// ListAvailableModels lists models from all enabled providers
func (s *AIService) ListAvailableModels(ctx context.Context, userID *uuid.UUID) ([]ModelInfo, error) {
	configs, err := s.repo.GetEnabledProviderConfigs(ctx)
	if err != nil {
		return nil, err
	}

	var allModels []ModelInfo
	for _, config := range configs {
		provider, err := s.GetProviderInstance(ctx, config.ID, userID)
		if err != nil {
			continue
		}

		models, err := provider.ListModels(ctx)
		if err != nil {
			continue
		}

		for _, m := range models {
			allModels = append(allModels, ModelInfo{
				ID:            m.ID,
				Name:          m.Name,
				ProviderType:  config.ProviderType,
				ProviderID:    config.ID,
				Description:   m.Description,
				ContextWindow: m.ContextWindow,
				MaxTokens:     m.MaxTokens,
				Capabilities:  m.Capabilities,
			})
		}
	}

	return allModels, nil
}

// HealthCheck checks the health of all enabled providers
func (s *AIService) HealthCheck(ctx context.Context) (map[string]*providers.HealthStatus, error) {
	configs, err := s.repo.GetEnabledProviderConfigs(ctx)
	if err != nil {
		return nil, err
	}

	results := make(map[string]*providers.HealthStatus)
	var wg sync.WaitGroup

	for _, config := range configs {
		wg.Add(1)
		go func(cfg *AIProviderConfig) {
			defer wg.Done()

			provider, err := s.GetProviderInstance(ctx, cfg.ID, nil)
			if err != nil {
				results[cfg.Name] = &providers.HealthStatus{
					Available: false,
					Error:     err.Error(),
					CheckedAt: time.Now(),
				}
				return
			}

			status, _ := provider.HealthCheck(ctx)
			results[cfg.Name] = status
		}(config)
	}

	wg.Wait()
	return results, nil
}

// GetProviderInfo returns info about available provider types
func (s *AIService) GetProviderInfo() []*providers.ProviderInfo {
	return s.factory.GetAllProviderInfo()
}

// Close closes all cached provider connections
func (s *AIService) Close() error {
	s.providersMu.Lock()
	defer s.providersMu.Unlock()

	for _, provider := range s.providers {
		provider.Close()
	}
	s.providers = make(map[string]providers.Provider)

	return nil
}
