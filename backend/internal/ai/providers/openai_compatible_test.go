package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOpenAICompatibleProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *OpenAICompatibleConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty base URL",
			config: &OpenAICompatibleConfig{
				ProviderName: "Test",
				ProviderType: "test",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &OpenAICompatibleConfig{
				ProviderName: "Test Provider",
				ProviderType: "test",
				BaseURL:      "http://localhost:8080",
				APIKey:       "test-key",
			},
			wantErr: false,
		},
		{
			name: "config with custom timeout",
			config: &OpenAICompatibleConfig{
				ProviderName: "Test Provider",
				ProviderType: "test",
				BaseURL:      "http://localhost:8080",
				Timeout:      30 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenAICompatibleProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAICompatibleProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if provider == nil {
					t.Error("Expected provider, got nil")
					return
				}
				if provider.Name() != tt.config.ProviderName {
					t.Errorf("Expected name '%s', got '%s'", tt.config.ProviderName, provider.Name())
				}
				if provider.Type() != tt.config.ProviderType {
					t.Errorf("Expected type '%s', got '%s'", tt.config.ProviderType, provider.Type())
				}
				if !provider.SupportsStreaming() {
					t.Error("Expected to support streaming")
				}
				if !provider.SupportsEmbeddings() {
					t.Error("Expected to support embeddings")
				}
				if provider.SupportsFunctionCalling() {
					t.Error("Expected to not support function calling")
				}
			}
		})
	}
}

func TestNewLMStudioProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantURL string
	}{
		{
			name:    "default config",
			config:  &ProviderConfig{},
			wantURL: "http://localhost:1234",
		},
		{
			name: "custom base URL",
			config: &ProviderConfig{
				BaseURL: "http://custom:1234/v1",
			},
			wantURL: "http://custom:1234",
		},
		{
			name: "with API key",
			config: &ProviderConfig{
				APIKey: "test-key",
			},
			wantURL: "http://localhost:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewLMStudioProvider(tt.config)
			if err != nil {
				t.Fatalf("NewLMStudioProvider() error = %v", err)
			}
			if provider.Name() != "LM Studio" {
				t.Errorf("Expected name 'LM Studio', got '%s'", provider.Name())
			}
			if provider.Type() != "lm_studio" {
				t.Errorf("Expected type 'lm_studio', got '%s'", provider.Type())
			}
		})
	}
}

func TestNewVLLMProvider(t *testing.T) {
	config := &ProviderConfig{}
	provider, err := NewVLLMProvider(config)
	if err != nil {
		t.Fatalf("NewVLLMProvider() error = %v", err)
	}
	if provider.Name() != "vLLM" {
		t.Errorf("Expected name 'vLLM', got '%s'", provider.Name())
	}
	if provider.Type() != "vllm" {
		t.Errorf("Expected type 'vllm', got '%s'", provider.Type())
	}
}

func TestNewLocalAIProvider(t *testing.T) {
	config := &ProviderConfig{}
	provider, err := NewLocalAIProvider(config)
	if err != nil {
		t.Fatalf("NewLocalAIProvider() error = %v", err)
	}
	if provider.Name() != "LocalAI" {
		t.Errorf("Expected name 'LocalAI', got '%s'", provider.Name())
	}
	if provider.Type() != "local_ai" {
		t.Errorf("Expected type 'local_ai', got '%s'", provider.Type())
	}
}

func TestNewLlamaCppProvider(t *testing.T) {
	config := &ProviderConfig{}
	provider, err := NewLlamaCppProvider(config)
	if err != nil {
		t.Fatalf("NewLlamaCppProvider() error = %v", err)
	}
	if provider.Name() != "llama.cpp" {
		t.Errorf("Expected name 'llama.cpp', got '%s'", provider.Name())
	}
	if provider.Type() != "llama_cpp" {
		t.Errorf("Expected type 'llama_cpp', got '%s'", provider.Type())
	}
}

func TestOpenAICompatibleProviderChat(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions path, got %s", r.URL.Path)
		}

		// Check content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Mock response
		response := ChatResponse{
			Model:   "test-model",
			Object:  "chat.completion",
			Created: 1234567890,
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    RoleAssistant,
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: &Usage{
				PromptTokens:     10,
				CompletionTokens: 15,
				TotalTokens:      25,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &OpenAICompatibleConfig{
		ProviderName: "Test Provider",
		ProviderType: "test",
		BaseURL:      server.URL,
	}
	provider, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ChatRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens: 100,
	}

	ctx := context.Background()
	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got %s", resp.Model)
	}
	if len(resp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello! How can I help you today?" {
		t.Errorf("Expected specific content, got %s", resp.Choices[0].Message.Content)
	}
}

func TestOpenAICompatibleProviderChatStream(t *testing.T) {
	// Create a mock HTTP server that returns SSE data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Write SSE data
		chunks := []string{
			`data: {"choices": [{"delta": {"role": "assistant", "content": "Hello"}}]}`,
			`data: {"choices": [{"delta": {"content": " world"}}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n"))
		}
	}))
	defer server.Close()

	config := &OpenAICompatibleConfig{
		ProviderName: "Test Provider",
		ProviderType: "test",
		BaseURL:      server.URL,
	}
	provider, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ChatRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens: 100,
	}

	responses := make([]*ChatResponse, 0)
	callback := func(resp *ChatResponse) error {
		responses = append(responses, resp)
		return nil
	}

	ctx := context.Background()
	err = provider.ChatStream(ctx, req, callback)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	if len(responses) == 0 {
		t.Error("Expected at least one response from stream")
	}
}

func TestOpenAICompatibleProviderEmbed(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/embeddings" {
			t.Errorf("Expected /embeddings path, got %s", r.URL.Path)
		}

		// Mock response
		response := EmbeddingResponse{
			Object: "list",
			Model:  "test-embedding-model",
			Data: []Embedding{
				{
					Object:    "embedding",
					Index:     0,
					Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &OpenAICompatibleConfig{
		ProviderName: "Test Provider",
		ProviderType: "test",
		BaseURL:      server.URL,
	}
	provider, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &EmbeddingRequest{
		Model: "test-embedding-model",
		Input: []string{"Hello world"},
	}

	ctx := context.Background()
	resp, err := provider.Embed(ctx, req)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if resp.Model != "test-embedding-model" {
		t.Errorf("Expected model 'test-embedding-model', got %s", resp.Model)
	}
	if len(resp.Data) != 1 {
		t.Errorf("Expected 1 embedding, got %d", len(resp.Data))
	}
	if len(resp.Data[0].Embedding) != 5 {
		t.Errorf("Expected embedding length 5, got %d", len(resp.Data[0].Embedding))
	}
}

func TestOpenAICompatibleProviderListModels(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/models" {
			t.Errorf("Expected /models path, got %s", r.URL.Path)
		}

		// Mock response
		response := struct {
			Data []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				OwnedBy string `json:"owned_by"`
			} `json:"data"`
		}{
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				OwnedBy string `json:"owned_by"`
			}{
				{ID: "model1", Object: "model", OwnedBy: "test"},
				{ID: "model2", Object: "model", OwnedBy: "test"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &OpenAICompatibleConfig{
		ProviderName: "Test Provider",
		ProviderType: "test",
		BaseURL:      server.URL,
	}
	provider, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}
	if models[0].ID != "model1" {
		t.Errorf("Expected first model ID 'model1', got %s", models[0].ID)
	}
}

func TestOpenAICompatibleProviderHealthCheck(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data": []}`))
		} else if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := &OpenAICompatibleConfig{
		ProviderName: "Test Provider",
		ProviderType: "test",
		BaseURL:      server.URL,
	}
	provider, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	status, err := provider.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	if !status.Available {
		t.Error("Expected provider to be available")
	}
	if status.CheckedAt.IsZero() {
		t.Error("Expected non-zero CheckedAt time")
	}
}

func TestOpenAICompatibleProviderParseError(t *testing.T) {
	config := &OpenAICompatibleConfig{
		ProviderName: "Test Provider",
		ProviderType: "test",
		BaseURL:      "http://localhost:8080",
	}
	provider, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name       string
		statusCode int
		body       []byte
		wantErr    error
	}{
		{
			name:       "unauthorized",
			statusCode: 401,
			body:       []byte(`{"error": {"message": "Unauthorized"}}`),
			wantErr:    ErrInvalidCredentials,
		},
		{
			name:       "rate limit",
			statusCode: 429,
			body:       []byte(`{"error": {"message": "Rate limit exceeded"}}`),
			wantErr:    ErrRateLimitExceeded,
		},
		{
			name:       "context length",
			statusCode: 400,
			body:       []byte(`{"error": {"message": "Context length exceeded"}}`),
			wantErr:    ErrContextLengthExceeded,
		},
		{
			name:       "model not found",
			statusCode: 404,
			body:       []byte(`{"error": {"message": "Model not found"}}`),
			wantErr:    ErrModelNotFound,
		},
		{
			name:       "service unavailable",
			statusCode: 503,
			body:       []byte(`{"error": {"message": "Service unavailable"}}`),
			wantErr:    ErrProviderUnavailable,
		},
		{
			name:       "invalid request",
			statusCode: 400,
			body:       []byte(`{"error": {"message": "Invalid request"}}`),
			wantErr:    ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.parseError(tt.statusCode, tt.body)
			if tt.wantErr != nil {
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("Expected error containing %v, got %v", tt.wantErr, err)
				}
			} else {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			}
		})
	}
}

func TestOpenAICompatibleProviderWithAuth(t *testing.T) {
	// Create a mock HTTP server that checks for authorization header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header 'Bearer test-api-key', got '%s'", auth)
		}

		response := ChatResponse{
			Model:   "test-model",
			Object:  "chat.completion",
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: "Hello"}}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &OpenAICompatibleConfig{
		ProviderName: "Test Provider",
		ProviderType: "test",
		BaseURL:      server.URL,
		APIKey:       "test-api-key",
	}
	provider, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	ctx := context.Background()
	_, err = provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
}

func TestOpenAICompatibleProviderExtraHeaders(t *testing.T) {
	// Create a mock HTTP server that checks for extra headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customHeader := r.Header.Get("X-Custom-Header")
		if customHeader != "custom-value" {
			t.Errorf("Expected X-Custom-Header 'custom-value', got '%s'", customHeader)
		}

		response := ChatResponse{
			Model:   "test-model",
			Object:  "chat.completion",
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: "Hello"}}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &OpenAICompatibleConfig{
		ProviderName: "Test Provider",
		ProviderType: "test",
		BaseURL:      server.URL,
		ExtraHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}
	provider, err := NewOpenAICompatibleProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	ctx := context.Background()
	_, err = provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
}
