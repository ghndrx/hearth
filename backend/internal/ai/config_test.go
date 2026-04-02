package ai

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProviderType_IsCloud(t *testing.T) {
	tests := []struct {
		provider ProviderType
		expected bool
	}{
		{ProviderOpenRouter, true},
		{ProviderBedrock, true},
		{ProviderVertexAI, true},
		{ProviderOpenAI, true},
		{ProviderAnthropic, true},
		{ProviderOllama, false},
		{ProviderLlamaCpp, false},
		{ProviderLMStudio, false},
		{ProviderVLLM, false},
		{ProviderLocalAI, false},
		{ProviderType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			result := tt.provider.IsCloud()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderType_IsLocal(t *testing.T) {
	tests := []struct {
		provider ProviderType
		expected bool
	}{
		{ProviderOpenRouter, false},
		{ProviderBedrock, false},
		{ProviderVertexAI, false},
		{ProviderOpenAI, false},
		{ProviderAnthropic, false},
		{ProviderOllama, true},
		{ProviderLlamaCpp, true},
		{ProviderLMStudio, true},
		{ProviderVLLM, true},
		{ProviderLocalAI, true},
		{ProviderType("invalid"), true}, // !IsCloud() = true for invalid
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			result := tt.provider.IsLocal()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderType_Valid(t *testing.T) {
	tests := []struct {
		provider ProviderType
		expected bool
	}{
		{ProviderOpenRouter, true},
		{ProviderBedrock, true},
		{ProviderVertexAI, true},
		{ProviderOpenAI, true},
		{ProviderAnthropic, true},
		{ProviderOllama, true},
		{ProviderLlamaCpp, true},
		{ProviderLMStudio, true},
		{ProviderVLLM, true},
		{ProviderLocalAI, true},
		{ProviderType("invalid"), false},
		{ProviderType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			result := tt.provider.Valid()
			assert.Equal(t, tt.expected, result)
		})
	}
}


func TestFeatureType_Valid(t *testing.T) {
	tests := []struct {
		feature  FeatureType
		expected bool
	}{
		{FeatureSummary, true},
		{FeatureSearch, true},
		{FeatureChat, true},
		{FeatureEmbed, true},
		{FeatureModerate, true},
		{FeatureTranslate, true},
		{FeatureType("invalid"), false},
		{FeatureType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			result := tt.feature.Valid()
			assert.Equal(t, tt.expected, result)
		})
	}
}


func TestAIProviderConfig_ToResponse(t *testing.T) {
	id := uuid.New()
	baseURL := "https://api.example.com"
	config := "encrypted-config"
	createdAt := time.Now()
	updatedAt := time.Now().Add(time.Hour)

	providerConfig := &AIProviderConfig{
		ID:           id,
		ProviderType: ProviderOpenAI,
		Name:         "my-openai",
		DisplayName:  "My OpenAI Provider",
		BaseURL:      &baseURL,
		IsEnabled:    true,
		IsDefault:    false,
		Priority:     1,
		Config:       &config,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}

	response := providerConfig.ToResponse()

	// Should copy public fields
	assert.Equal(t, id, response.ID)
	assert.Equal(t, ProviderOpenAI, response.ProviderType)
	assert.Equal(t, "my-openai", response.Name)
	assert.Equal(t, "My OpenAI Provider", response.DisplayName)
	assert.Equal(t, &baseURL, response.BaseURL)
	assert.Equal(t, true, response.IsEnabled)
	assert.Equal(t, false, response.IsDefault)
	assert.Equal(t, 1, response.Priority)
	assert.Equal(t, createdAt, response.CreatedAt)
	assert.Equal(t, updatedAt, response.UpdatedAt)
}

func TestAIProviderConfig_ToResponseWithNilBaseURL(t *testing.T) {
	id := uuid.New()

	providerConfig := &AIProviderConfig{
		ID:           id,
		ProviderType: ProviderOllama,
		Name:         "local-ollama",
		DisplayName:  "Local Ollama",
		BaseURL:      nil,
		IsEnabled:    true,
		IsDefault:    true,
		Priority:     0,
		Config:       nil,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	response := providerConfig.ToResponse()

	assert.Equal(t, id, response.ID)
	assert.Equal(t, ProviderOllama, response.ProviderType)
	assert.Nil(t, response.BaseURL)
	assert.True(t, response.IsDefault)
}

func TestUserAICredential_ToResponse(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	providerID := uuid.New()
	lastUsedAt := time.Now()
	createdAt := time.Now().Add(-time.Hour)

	credential := &UserAICredential{
		ID:           id,
		UserID:       userID,
		ProviderID:   providerID,
		ProviderType: ProviderBedrock,
		Credentials:  "encrypted-credentials",
		IsEnabled:    true,
		LastUsedAt:   &lastUsedAt,
		CreatedAt:    createdAt,
		UpdatedAt:    time.Now(),
	}

	response := credential.ToResponse()

	assert.Equal(t, id, response.ID)
	assert.Equal(t, providerID, response.ProviderID)
	assert.Equal(t, ProviderBedrock, response.ProviderType)
	assert.Equal(t, true, response.IsEnabled)
	assert.Equal(t, true, response.HasCredentials)
	assert.Equal(t, &lastUsedAt, response.LastUsedAt)
	assert.Equal(t, createdAt, response.CreatedAt)
}

func TestUserAICredential_ToResponseWithoutCredentials(t *testing.T) {
	id := uuid.New()
	providerID := uuid.New()

	credential := &UserAICredential{
		ID:           id,
		ProviderID:   providerID,
		ProviderType: ProviderOpenRouter,
		Credentials:  "", // Empty credentials
		IsEnabled:    false,
		LastUsedAt:   nil,
		CreatedAt:    time.Now(),
	}

	response := credential.ToResponse()

	assert.Equal(t, id, response.ID)
	assert.Equal(t, providerID, response.ProviderID)
	assert.Equal(t, ProviderOpenRouter, response.ProviderType)
	assert.Equal(t, false, response.IsEnabled)
	assert.Equal(t, false, response.HasCredentials)
	assert.Nil(t, response.LastUsedAt)
}


func TestProviderTypeConstants(t *testing.T) {
	// Verify that all provider type constants are strings
	assert.Equal(t, "openrouter", string(ProviderOpenRouter))
	assert.Equal(t, "bedrock", string(ProviderBedrock))
	assert.Equal(t, "vertex_ai", string(ProviderVertexAI))
	assert.Equal(t, "openai", string(ProviderOpenAI))
	assert.Equal(t, "anthropic", string(ProviderAnthropic))
	assert.Equal(t, "ollama", string(ProviderOllama))
	assert.Equal(t, "llama_cpp", string(ProviderLlamaCpp))
	assert.Equal(t, "lm_studio", string(ProviderLMStudio))
	assert.Equal(t, "vllm", string(ProviderVLLM))
	assert.Equal(t, "local_ai", string(ProviderLocalAI))
}

func TestFeatureTypeConstants(t *testing.T) {
	// Verify that all feature type constants are strings
	assert.Equal(t, "summary", string(FeatureSummary))
	assert.Equal(t, "search", string(FeatureSearch))
	assert.Equal(t, "chat", string(FeatureChat))
	assert.Equal(t, "embed", string(FeatureEmbed))
	assert.Equal(t, "moderate", string(FeatureModerate))
	assert.Equal(t, "translate", string(FeatureTranslate))
}

func TestProviderTypeCloudLocalConsistency(t *testing.T) {
	// Test that IsLocal() is always the opposite of IsCloud()
	allProviders := AllProviderTypes()

	for _, provider := range allProviders {
		isCloud := provider.IsCloud()
		isLocal := provider.IsLocal()
		assert.NotEqual(t, isCloud, isLocal, "Provider %s should be either cloud or local, not both", provider)
		assert.True(t, isCloud || isLocal, "Provider %s should be either cloud or local", provider)
	}

	// Test with invalid provider
	invalidProvider := ProviderType("invalid")
	assert.False(t, invalidProvider.IsCloud())
	assert.True(t, invalidProvider.IsLocal()) // IsLocal() = !IsCloud()
}