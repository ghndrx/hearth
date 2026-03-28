package providers

import (
	"fmt"
)

// ProviderFactory creates provider instances
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// Create creates a provider instance based on type
func (f *ProviderFactory) Create(providerType string, config *ProviderConfig) (Provider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch providerType {
	case "openai":
		return NewOpenAIProvider(config)
	case "anthropic":
		return NewAnthropicProvider(config)
	case "openrouter":
		return NewOpenRouterProvider(config)
	case "bedrock":
		return NewBedrockProvider(config)
	case "vertex_ai":
		return NewVertexAIProvider(config)
	case "ollama":
		return NewOllamaProvider(config)
	case "lm_studio":
		return NewLMStudioProvider(config)
	case "vllm":
		return NewVLLMProvider(config)
	case "local_ai":
		return NewLocalAIProvider(config)
	case "llama_cpp":
		return NewLlamaCppProvider(config)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// CreateFromConfig creates a provider from AIProviderConfig
func (f *ProviderFactory) CreateFromConfig(providerType string, baseURL *string, credentials *ProviderConfig) (Provider, error) {
	config := credentials
	if config == nil {
		config = DefaultConfig()
	}

	if baseURL != nil && *baseURL != "" {
		config.BaseURL = *baseURL
	}

	return f.Create(providerType, config)
}

// SupportedProviders returns all supported provider types
func (f *ProviderFactory) SupportedProviders() []string {
	return []string{
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
}

// CloudProviders returns cloud provider types
func (f *ProviderFactory) CloudProviders() []string {
	return []string{
		"openai",
		"anthropic",
		"openrouter",
		"bedrock",
		"vertex_ai",
	}
}

// LocalProviders returns local provider types
func (f *ProviderFactory) LocalProviders() []string {
	return []string{
		"ollama",
		"lm_studio",
		"vllm",
		"local_ai",
		"llama_cpp",
	}
}

// IsCloudProvider returns true if the provider type is a cloud provider
func (f *ProviderFactory) IsCloudProvider(providerType string) bool {
	for _, p := range f.CloudProviders() {
		if p == providerType {
			return true
		}
	}
	return false
}

// IsLocalProvider returns true if the provider type is a local provider
func (f *ProviderFactory) IsLocalProvider(providerType string) bool {
	for _, p := range f.LocalProviders() {
		if p == providerType {
			return true
		}
	}
	return false
}

// RequiresAPIKey returns true if the provider type requires an API key
func (f *ProviderFactory) RequiresAPIKey(providerType string) bool {
	switch providerType {
	case "openai", "anthropic", "openrouter":
		return true
	case "bedrock", "vertex_ai":
		return true // Or service account/IAM
	default:
		return false // Local providers typically don't need auth
	}
}

// DefaultBaseURL returns the default base URL for a provider type
func (f *ProviderFactory) DefaultBaseURL(providerType string) string {
	switch providerType {
	case "openai":
		return "https://api.openai.com/v1"
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "ollama":
		return "http://localhost:11434"
	case "lm_studio":
		return "http://localhost:1234/v1"
	case "vllm":
		return "http://localhost:8000/v1"
	case "local_ai":
		return "http://localhost:8080/v1"
	case "llama_cpp":
		return "http://localhost:8080/v1"
	default:
		return ""
	}
}

// ProviderInfo contains metadata about a provider type
type ProviderInfo struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	IsCloud        bool   `json:"is_cloud"`
	RequiresAPIKey bool   `json:"requires_api_key"`
	DefaultBaseURL string `json:"default_base_url,omitempty"`
	Capabilities   struct {
		Chat            bool `json:"chat"`
		Streaming       bool `json:"streaming"`
		Embeddings      bool `json:"embeddings"`
		FunctionCalling bool `json:"function_calling"`
	} `json:"capabilities"`
}

// GetProviderInfo returns metadata about a provider type
func (f *ProviderFactory) GetProviderInfo(providerType string) *ProviderInfo {
	info := &ProviderInfo{
		Type:           providerType,
		IsCloud:        f.IsCloudProvider(providerType),
		RequiresAPIKey: f.RequiresAPIKey(providerType),
		DefaultBaseURL: f.DefaultBaseURL(providerType),
	}

	switch providerType {
	case "openai":
		info.Name = "OpenAI"
		info.Description = "OpenAI GPT models (GPT-4, GPT-3.5, etc.)"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = true
		info.Capabilities.FunctionCalling = true
	case "anthropic":
		info.Name = "Anthropic"
		info.Description = "Anthropic Claude models (Claude 3, etc.)"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = false
		info.Capabilities.FunctionCalling = true
	case "openrouter":
		info.Name = "OpenRouter"
		info.Description = "OpenRouter unified API for multiple providers"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = true
		info.Capabilities.FunctionCalling = true
	case "bedrock":
		info.Name = "AWS Bedrock"
		info.Description = "AWS Bedrock managed AI service"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = true
		info.Capabilities.FunctionCalling = true
	case "vertex_ai":
		info.Name = "Google Vertex AI"
		info.Description = "Google Cloud Vertex AI (Gemini, PaLM, etc.)"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = true
		info.Capabilities.FunctionCalling = true
	case "ollama":
		info.Name = "Ollama"
		info.Description = "Local Ollama server for running open models"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = true
		info.Capabilities.FunctionCalling = false
	case "lm_studio":
		info.Name = "LM Studio"
		info.Description = "LM Studio local inference server"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = true
		info.Capabilities.FunctionCalling = false
	case "vllm":
		info.Name = "vLLM"
		info.Description = "vLLM high-throughput inference server"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = false
		info.Capabilities.FunctionCalling = false
	case "local_ai":
		info.Name = "LocalAI"
		info.Description = "LocalAI self-hosted OpenAI alternative"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = true
		info.Capabilities.FunctionCalling = false
	case "llama_cpp":
		info.Name = "llama.cpp"
		info.Description = "llama.cpp server for efficient local inference"
		info.Capabilities.Chat = true
		info.Capabilities.Streaming = true
		info.Capabilities.Embeddings = true
		info.Capabilities.FunctionCalling = false
	default:
		info.Name = providerType
		info.Description = "Unknown provider"
	}

	return info
}

// GetAllProviderInfo returns metadata about all providers
func (f *ProviderFactory) GetAllProviderInfo() []*ProviderInfo {
	providers := f.SupportedProviders()
	info := make([]*ProviderInfo, len(providers))
	for i, p := range providers {
		info[i] = f.GetProviderInfo(p)
	}
	return info
}
