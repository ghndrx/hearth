package providers

import (
	"testing"
)

func TestProviderFactoryCreateAll(t *testing.T) {
	factory := NewProviderFactory()

	// Test creating each provider type (those that don't need API keys)
	localProviders := []struct {
		providerType string
		expectedName string
	}{
		{"ollama", "Ollama"},
		{"lm_studio", "LM Studio"},
		{"vllm", "vLLM"},
		{"local_ai", "LocalAI"},
		{"llama_cpp", "llama.cpp"},
	}

	for _, tt := range localProviders {
		t.Run(tt.providerType, func(t *testing.T) {
			config := &ProviderConfig{
				BaseURL: "http://localhost:8080",
			}
			provider, err := factory.Create(tt.providerType, config)
			if err != nil {
				t.Fatalf("Create(%s) error: %v", tt.providerType, err)
			}
			if provider.Name() != tt.expectedName {
				t.Errorf("Name() = %s, want %s", provider.Name(), tt.expectedName)
			}
		})
	}
}

func TestProviderFactoryCreateWithKey(t *testing.T) {
	factory := NewProviderFactory()

	cloudProviders := []struct {
		providerType string
		expectedName string
	}{
		{"openai", "OpenAI"},
		{"anthropic", "Anthropic"},
		{"openrouter", "OpenRouter"},
	}

	for _, tt := range cloudProviders {
		t.Run(tt.providerType, func(t *testing.T) {
			config := &ProviderConfig{
				APIKey: "test-key",
			}
			provider, err := factory.Create(tt.providerType, config)
			if err != nil {
				t.Fatalf("Create(%s) error: %v", tt.providerType, err)
			}
			if provider.Name() != tt.expectedName {
				t.Errorf("Name() = %s, want %s", provider.Name(), tt.expectedName)
			}
		})
	}
}

func TestProviderFactoryCreateFromConfig(t *testing.T) {
	factory := NewProviderFactory()

	baseURL := "http://custom:11434"
	provider, err := factory.CreateFromConfig("ollama", &baseURL, nil)
	if err != nil {
		t.Fatalf("CreateFromConfig() error: %v", err)
	}

	if provider.Type() != "ollama" {
		t.Errorf("Type() = %s, want ollama", provider.Type())
	}
}

func TestProviderFactoryCreateFromConfigNilBaseURL(t *testing.T) {
	factory := NewProviderFactory()

	provider, err := factory.CreateFromConfig("ollama", nil, nil)
	if err != nil {
		t.Fatalf("CreateFromConfig() error: %v", err)
	}

	if provider.Type() != "ollama" {
		t.Errorf("Type() = %s, want ollama", provider.Type())
	}
}

func TestProviderFactoryCreateFromConfigWithCredentials(t *testing.T) {
	factory := NewProviderFactory()

	creds := &ProviderConfig{
		APIKey: "test-key",
	}
	provider, err := factory.CreateFromConfig("openai", nil, creds)
	if err != nil {
		t.Fatalf("CreateFromConfig() error: %v", err)
	}

	if provider.Type() != "openai" {
		t.Errorf("Type() = %s, want openai", provider.Type())
	}
}

func TestProviderFactoryRequiresAPIKey(t *testing.T) {
	factory := NewProviderFactory()

	tests := []struct {
		providerType string
		expected     bool
	}{
		{"openai", true},
		{"anthropic", true},
		{"openrouter", true},
		{"bedrock", true},
		{"vertex_ai", true},
		{"ollama", false},
		{"lm_studio", false},
		{"vllm", false},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			result := factory.RequiresAPIKey(tt.providerType)
			if result != tt.expected {
				t.Errorf("RequiresAPIKey(%s) = %v, want %v", tt.providerType, result, tt.expected)
			}
		})
	}
}

func TestProviderFactoryGetProviderInfoUnknown(t *testing.T) {
	factory := NewProviderFactory()

	info := factory.GetProviderInfo("unknown")
	if info.Name != "unknown" {
		t.Errorf("Name = %s, want unknown", info.Name)
	}
	if info.Description != "Unknown provider" {
		t.Errorf("Description = %s, want 'Unknown provider'", info.Description)
	}
}

func TestProviderFactoryAllProviderInfo(t *testing.T) {
	factory := NewProviderFactory()

	info := factory.GetAllProviderInfo()
	supported := factory.SupportedProviders()

	if len(info) != len(supported) {
		t.Errorf("Info count = %d, supported count = %d", len(info), len(supported))
	}

	// Verify all have required fields
	for _, i := range info {
		if i.Type == "" {
			t.Error("Provider type should not be empty")
		}
		if i.Name == "" {
			t.Error("Provider name should not be empty")
		}
		if i.Description == "" {
			t.Error("Provider description should not be empty")
		}
	}
}

func TestProviderFactoryDefaultBaseURLs(t *testing.T) {
	factory := NewProviderFactory()

	tests := []struct {
		providerType string
		expectEmpty  bool
	}{
		{"openai", false},
		{"anthropic", false},
		{"ollama", false},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			url := factory.DefaultBaseURL(tt.providerType)
			isEmpty := url == ""
			if isEmpty != tt.expectEmpty {
				t.Errorf("DefaultBaseURL(%s) empty = %v, want %v", tt.providerType, isEmpty, tt.expectEmpty)
			}
		})
	}
}

func TestProviderInfoCapabilities(t *testing.T) {
	factory := NewProviderFactory()

	// Check OpenAI capabilities
	openaiInfo := factory.GetProviderInfo("openai")
	if !openaiInfo.Capabilities.Chat {
		t.Error("OpenAI should support chat")
	}
	if !openaiInfo.Capabilities.Streaming {
		t.Error("OpenAI should support streaming")
	}
	if !openaiInfo.Capabilities.Embeddings {
		t.Error("OpenAI should support embeddings")
	}
	if !openaiInfo.Capabilities.FunctionCalling {
		t.Error("OpenAI should support function calling")
	}

	// Check Anthropic (no embeddings)
	anthropicInfo := factory.GetProviderInfo("anthropic")
	if anthropicInfo.Capabilities.Embeddings {
		t.Error("Anthropic should not support embeddings")
	}

	// Check Ollama (no function calling)
	ollamaInfo := factory.GetProviderInfo("ollama")
	if ollamaInfo.Capabilities.FunctionCalling {
		t.Error("Ollama should not support function calling")
	}
}

func TestProviderFactoryCreateUnknownType(t *testing.T) {
	factory := NewProviderFactory()

	_, err := factory.Create("unknown_provider", nil)
	if err == nil {
		t.Error("Create with unknown provider type should return error")
	}

	expectedMsg := "unknown provider type: unknown_provider"
	if err.Error() != expectedMsg {
		t.Errorf("Error message = %s, want %s", err.Error(), expectedMsg)
	}
}

func TestProviderFactorySupportedProviders(t *testing.T) {
	factory := NewProviderFactory()

	supported := factory.SupportedProviders()
	expected := []string{
		"openai",
		"anthropic",
		"openrouter",
		"bedrock",
		"vertex_ai",
		"ollama",
		"lm_studio",
		"vllm",
		"local_ai",
		"llama_cpp",
	}

	if len(supported) != len(expected) {
		t.Errorf("SupportedProviders count = %d, want %d", len(supported), len(expected))
	}

	for i, provider := range expected {
		if i >= len(supported) || supported[i] != provider {
			t.Errorf("Expected provider %s at index %d, got %s", provider, i, supported[i])
		}
	}
}

func TestProviderFactoryCloudProviders(t *testing.T) {
	factory := NewProviderFactory()

	cloud := factory.CloudProviders()
	expected := []string{
		"openai",
		"anthropic",
		"openrouter",
		"bedrock",
		"vertex_ai",
	}

	if len(cloud) != len(expected) {
		t.Errorf("CloudProviders count = %d, want %d", len(cloud), len(expected))
	}

	for i, provider := range expected {
		if i >= len(cloud) || cloud[i] != provider {
			t.Errorf("Expected cloud provider %s at index %d, got %s", provider, i, cloud[i])
		}
	}
}

func TestProviderFactoryLocalProviders(t *testing.T) {
	factory := NewProviderFactory()

	local := factory.LocalProviders()
	expected := []string{
		"ollama",
		"lm_studio",
		"vllm",
		"local_ai",
		"llama_cpp",
	}

	if len(local) != len(expected) {
		t.Errorf("LocalProviders count = %d, want %d", len(local), len(expected))
	}

	for i, provider := range expected {
		if i >= len(local) || local[i] != provider {
			t.Errorf("Expected local provider %s at index %d, got %s", provider, i, local[i])
		}
	}
}

func TestProviderFactoryIsCloudProvider(t *testing.T) {
	factory := NewProviderFactory()

	cloudTests := []struct {
		provider string
		expected bool
	}{
		{"openai", true},
		{"anthropic", true},
		{"bedrock", true},
		{"ollama", false},
		{"lm_studio", false},
		{"unknown", false},
	}

	for _, tt := range cloudTests {
		t.Run(tt.provider, func(t *testing.T) {
			result := factory.IsCloudProvider(tt.provider)
			if result != tt.expected {
				t.Errorf("IsCloudProvider(%s) = %v, want %v", tt.provider, result, tt.expected)
			}
		})
	}
}

func TestProviderFactoryIsLocalProvider(t *testing.T) {
	factory := NewProviderFactory()

	localTests := []struct {
		provider string
		expected bool
	}{
		{"ollama", true},
		{"lm_studio", true},
		{"vllm", true},
		{"openai", false},
		{"anthropic", false},
		{"unknown", false},
	}

	for _, tt := range localTests {
		t.Run(tt.provider, func(t *testing.T) {
			result := factory.IsLocalProvider(tt.provider)
			if result != tt.expected {
				t.Errorf("IsLocalProvider(%s) = %v, want %v", tt.provider, result, tt.expected)
			}
		})
	}
}

func TestProviderFactoryCreateFromConfigEmptyBaseURL(t *testing.T) {
	factory := NewProviderFactory()

	emptyURL := ""
	provider, err := factory.CreateFromConfig("ollama", &emptyURL, nil)
	if err != nil {
		t.Fatalf("CreateFromConfig() with empty URL error: %v", err)
	}

	if provider.Type() != "ollama" {
		t.Errorf("Type() = %s, want ollama", provider.Type())
	}
}
