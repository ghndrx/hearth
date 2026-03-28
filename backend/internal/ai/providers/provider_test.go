package providers

import (
	"context"
	"testing"
	"time"
)

func TestProviderFactory(t *testing.T) {
	factory := NewProviderFactory()

	// Test supported providers list
	supported := factory.SupportedProviders()
	if len(supported) == 0 {
		t.Error("Expected supported providers, got empty list")
	}

	expectedProviders := []string{
		"openai", "anthropic", "openrouter", "bedrock", "vertex_ai",
		"ollama", "lm_studio", "vllm", "local_ai", "llama_cpp",
	}

	for _, expected := range expectedProviders {
		found := false
		for _, p := range supported {
			if p == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected provider %s in supported list", expected)
		}
	}
}

func TestProviderFactoryCloudLocal(t *testing.T) {
	factory := NewProviderFactory()

	cloudProviders := factory.CloudProviders()
	localProviders := factory.LocalProviders()

	// Verify cloud providers
	for _, p := range cloudProviders {
		if !factory.IsCloudProvider(p) {
			t.Errorf("Provider %s should be cloud", p)
		}
		if factory.IsLocalProvider(p) {
			t.Errorf("Provider %s should not be local", p)
		}
	}

	// Verify local providers
	for _, p := range localProviders {
		if !factory.IsLocalProvider(p) {
			t.Errorf("Provider %s should be local", p)
		}
		if factory.IsCloudProvider(p) {
			t.Errorf("Provider %s should not be cloud", p)
		}
	}
}

func TestProviderFactoryInfo(t *testing.T) {
	factory := NewProviderFactory()

	allInfo := factory.GetAllProviderInfo()
	if len(allInfo) == 0 {
		t.Error("Expected provider info, got empty list")
	}

	// Verify each provider has required fields
	for _, info := range allInfo {
		if info.Type == "" {
			t.Error("Provider info missing type")
		}
		if info.Name == "" {
			t.Errorf("Provider %s missing name", info.Type)
		}
	}

	// Test specific provider info
	openaiInfo := factory.GetProviderInfo("openai")
	if openaiInfo == nil {
		t.Fatal("OpenAI provider info is nil")
	}
	if openaiInfo.Name != "OpenAI" {
		t.Errorf("Expected OpenAI name, got %s", openaiInfo.Name)
	}
	if !openaiInfo.Capabilities.Chat {
		t.Error("OpenAI should support chat")
	}
	if !openaiInfo.Capabilities.Streaming {
		t.Error("OpenAI should support streaming")
	}
}

func TestProviderFactoryDefaultURLs(t *testing.T) {
	factory := NewProviderFactory()

	tests := []struct {
		providerType string
		expectedURL  string
	}{
		{"openai", "https://api.openai.com/v1"},
		{"anthropic", "https://api.anthropic.com/v1"},
		{"openrouter", "https://openrouter.ai/api/v1"},
		{"ollama", "http://localhost:11434"},
		{"lm_studio", "http://localhost:1234/v1"},
	}

	for _, tt := range tests {
		url := factory.DefaultBaseURL(tt.providerType)
		if url != tt.expectedURL {
			t.Errorf("DefaultBaseURL(%s) = %s, want %s", tt.providerType, url, tt.expectedURL)
		}
	}
}

func TestCreateOllamaProvider(t *testing.T) {
	factory := NewProviderFactory()

	config := &ProviderConfig{
		BaseURL: "http://localhost:11434",
		Timeout: 30 * time.Second,
	}

	provider, err := factory.Create("ollama", config)
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}

	if provider.Name() != "Ollama" {
		t.Errorf("Expected name Ollama, got %s", provider.Name())
	}
	if provider.Type() != "ollama" {
		t.Errorf("Expected type ollama, got %s", provider.Type())
	}
	if !provider.SupportsStreaming() {
		t.Error("Ollama should support streaming")
	}
	if !provider.SupportsEmbeddings() {
		t.Error("Ollama should support embeddings")
	}
}

func TestCreateOpenAIProviderRequiresKey(t *testing.T) {
	factory := NewProviderFactory()

	// Without API key
	_, err := factory.Create("openai", &ProviderConfig{})
	if err == nil {
		t.Error("Expected error when creating OpenAI provider without API key")
	}

	// With API key
	_, err = factory.Create("openai", &ProviderConfig{APIKey: "test-key"})
	if err != nil {
		t.Errorf("Failed to create OpenAI provider with API key: %v", err)
	}
}

func TestMessageRoles(t *testing.T) {
	roles := []Role{RoleSystem, RoleUser, RoleAssistant, RoleTool}
	expectedStrings := []string{"system", "user", "assistant", "tool"}

	for i, role := range roles {
		if string(role) != expectedStrings[i] {
			t.Errorf("Role %d = %s, want %s", i, string(role), expectedStrings[i])
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want 30s", config.Timeout)
	}
	if config.MaxRetries != 3 {
		t.Errorf("Default max retries = %d, want 3", config.MaxRetries)
	}
}

func TestUnknownProviderType(t *testing.T) {
	factory := NewProviderFactory()

	_, err := factory.Create("unknown_provider", &ProviderConfig{})
	if err == nil {
		t.Error("Expected error for unknown provider type")
	}
}

// Integration test (skipped by default, requires running Ollama)
func TestOllamaHealthCheckIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := &ProviderConfig{
		BaseURL: "http://localhost:11434",
		Timeout: 5 * time.Second,
	}

	provider, err := NewOllamaProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := provider.HealthCheck(ctx)
	if err != nil {
		t.Logf("Health check error (Ollama may not be running): %v", err)
	} else {
		t.Logf("Ollama health check: available=%v, latency=%v", status.Available, status.Latency)
	}
}

func TestCreateAnthropicProviderRequiresKey(t *testing.T) {
	factory := NewProviderFactory()

	// Without API key
	_, err := factory.Create("anthropic", &ProviderConfig{})
	if err == nil {
		t.Error("Expected error when creating Anthropic provider without API key")
	}

	// With API key
	provider, err := factory.Create("anthropic", &ProviderConfig{APIKey: "test-key"})
	if err != nil {
		t.Errorf("Failed to create Anthropic provider with API key: %v", err)
	}

	if provider.Name() != "Anthropic" {
		t.Errorf("Expected name Anthropic, got %s", provider.Name())
	}
	if !provider.SupportsStreaming() {
		t.Error("Anthropic should support streaming")
	}
	if provider.SupportsEmbeddings() {
		t.Error("Anthropic should not support embeddings")
	}
	if !provider.SupportsFunctionCalling() {
		t.Error("Anthropic should support function calling")
	}
}

func TestCreateOpenRouterProvider(t *testing.T) {
	factory := NewProviderFactory()

	// Without API key
	_, err := factory.Create("openrouter", &ProviderConfig{})
	if err == nil {
		t.Error("Expected error when creating OpenRouter provider without API key")
	}

	// With API key
	provider, err := factory.Create("openrouter", &ProviderConfig{APIKey: "test-key"})
	if err != nil {
		t.Errorf("Failed to create OpenRouter provider with API key: %v", err)
	}

	if provider.Name() != "OpenRouter" {
		t.Errorf("Expected name OpenRouter, got %s", provider.Name())
	}
}

func TestCreateBedrockProviderRequiresCredentials(t *testing.T) {
	factory := NewProviderFactory()

	// Without credentials
	_, err := factory.Create("bedrock", &ProviderConfig{})
	if err == nil {
		t.Error("Expected error when creating Bedrock provider without credentials")
	}

	// With credentials
	provider, err := factory.Create("bedrock", &ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
		Region:    "us-east-1",
	})
	if err != nil {
		t.Errorf("Failed to create Bedrock provider with credentials: %v", err)
	}

	if provider.Name() != "AWS Bedrock" {
		t.Errorf("Expected name AWS Bedrock, got %s", provider.Name())
	}
}

func TestCreateLocalProviders(t *testing.T) {
	factory := NewProviderFactory()

	localProviders := []struct {
		providerType string
		expectedName string
	}{
		{"lm_studio", "LM Studio"},
		{"vllm", "vLLM"},
		{"local_ai", "LocalAI"},
		{"llama_cpp", "llama.cpp"},
	}

	for _, tc := range localProviders {
		t.Run(tc.providerType, func(t *testing.T) {
			provider, err := factory.Create(tc.providerType, &ProviderConfig{
				BaseURL: "http://localhost:8080/v1",
			})
			if err != nil {
				t.Fatalf("Failed to create %s provider: %v", tc.providerType, err)
			}

			if provider.Name() != tc.expectedName {
				t.Errorf("Expected name %s, got %s", tc.expectedName, provider.Name())
			}
			if provider.Type() != tc.providerType {
				t.Errorf("Expected type %s, got %s", tc.providerType, provider.Type())
			}
			if !provider.SupportsStreaming() {
				t.Errorf("%s should support streaming", tc.providerType)
			}
		})
	}
}

func TestRequiresAPIKeyByType(t *testing.T) {
	factory := NewProviderFactory()

	// Providers that require API keys
	requiresKey := []string{"openai", "anthropic", "openrouter"}
	for _, pt := range requiresKey {
		if !factory.RequiresAPIKey(pt) {
			t.Errorf("Provider %s should require API key", pt)
		}
	}

	// Local providers that don't require API keys
	noKey := []string{"ollama", "lm_studio", "vllm", "local_ai", "llama_cpp"}
	for _, pt := range noKey {
		if factory.RequiresAPIKey(pt) {
			t.Errorf("Provider %s should not require API key", pt)
		}
	}
}

func TestProviderCapabilitiesDetail(t *testing.T) {
	factory := NewProviderFactory()

	tests := []struct {
		providerType string
		chat         bool
		streaming    bool
		embeddings   bool
		funcCalling  bool
	}{
		{"openai", true, true, true, true},
		{"anthropic", true, true, false, true},
		{"ollama", true, true, true, false},
		{"vllm", true, true, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.providerType, func(t *testing.T) {
			info := factory.GetProviderInfo(tc.providerType)
			if info.Capabilities.Chat != tc.chat {
				t.Errorf("Chat capability mismatch: got %v, want %v", info.Capabilities.Chat, tc.chat)
			}
			if info.Capabilities.Streaming != tc.streaming {
				t.Errorf("Streaming capability mismatch: got %v, want %v", info.Capabilities.Streaming, tc.streaming)
			}
			if info.Capabilities.Embeddings != tc.embeddings {
				t.Errorf("Embeddings capability mismatch: got %v, want %v", info.Capabilities.Embeddings, tc.embeddings)
			}
			if info.Capabilities.FunctionCalling != tc.funcCalling {
				t.Errorf("FunctionCalling capability mismatch: got %v, want %v", info.Capabilities.FunctionCalling, tc.funcCalling)
			}
		})
	}
}

func TestCreateFromConfig(t *testing.T) {
	factory := NewProviderFactory()

	baseURL := "http://custom:11434"
	credentials := &ProviderConfig{
		Timeout: 60 * time.Second,
	}

	provider, err := factory.CreateFromConfig("ollama", &baseURL, credentials)
	if err != nil {
		t.Fatalf("CreateFromConfig failed: %v", err)
	}

	if provider.Type() != "ollama" {
		t.Errorf("Expected type ollama, got %s", provider.Type())
	}
}

func TestCreateFromConfigNilCredentials(t *testing.T) {
	factory := NewProviderFactory()

	provider, err := factory.CreateFromConfig("ollama", nil, nil)
	if err != nil {
		t.Fatalf("CreateFromConfig with nil credentials failed: %v", err)
	}

	if provider.Type() != "ollama" {
		t.Errorf("Expected type ollama, got %s", provider.Type())
	}
}

func TestUnknownProviderInfo(t *testing.T) {
	factory := NewProviderFactory()

	info := factory.GetProviderInfo("unknown_type")
	if info == nil {
		t.Fatal("GetProviderInfo should not return nil even for unknown types")
	}
	if info.Name != "unknown_type" {
		t.Errorf("Unknown provider name should be the type, got %s", info.Name)
	}
	if info.Description != "Unknown provider" {
		t.Errorf("Unknown provider description should be 'Unknown provider', got %s", info.Description)
	}
}

func TestOllamaProviderClose(t *testing.T) {
	provider, err := NewOllamaProvider(&ProviderConfig{
		BaseURL: "http://localhost:11434",
	})
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}

	err = provider.Close()
	if err != nil {
		t.Errorf("Close() should not return error: %v", err)
	}
}

func TestOllamaProviderCapabilities(t *testing.T) {
	provider, err := NewOllamaProvider(&ProviderConfig{
		BaseURL: "http://localhost:11434",
	})
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}

	if !provider.SupportsStreaming() {
		t.Error("Ollama should support streaming")
	}
	if !provider.SupportsEmbeddings() {
		t.Error("Ollama should support embeddings")
	}
	if provider.SupportsFunctionCalling() {
		t.Error("Ollama should not support function calling")
	}
}

func TestOpenAIProviderCapabilities(t *testing.T) {
	provider, err := NewOpenAIProvider(&ProviderConfig{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	if !provider.SupportsStreaming() {
		t.Error("OpenAI should support streaming")
	}
	if !provider.SupportsEmbeddings() {
		t.Error("OpenAI should support embeddings")
	}
	if !provider.SupportsFunctionCalling() {
		t.Error("OpenAI should support function calling")
	}
}
