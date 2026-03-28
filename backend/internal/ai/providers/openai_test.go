package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOpenAIProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true, // No API key
		},
		{
			name:    "empty API key",
			config:  &ProviderConfig{},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &ProviderConfig{
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "custom base URL",
			config: &ProviderConfig{
				APIKey:  "test-key",
				BaseURL: "https://custom.api.com/v1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenAIProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAIProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && provider == nil {
				t.Error("Expected provider, got nil")
			}
		})
	}
}

func TestOpenAIProviderMetadata(t *testing.T) {
	provider, err := NewOpenAIProvider(&ProviderConfig{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.Name() != "OpenAI" {
		t.Errorf("Name() = %s, want OpenAI", provider.Name())
	}
	if provider.Type() != "openai" {
		t.Errorf("Type() = %s, want openai", provider.Type())
	}
	if !provider.SupportsStreaming() {
		t.Error("SupportsStreaming() should be true")
	}
	if !provider.SupportsEmbeddings() {
		t.Error("SupportsEmbeddings() should be true")
	}
	if !provider.SupportsFunctionCalling() {
		t.Error("SupportsFunctionCalling() should be true")
	}
}

func TestOpenAIChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("Missing or invalid Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Missing Content-Type header")
		}

		// Return mock response
		resp := ChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    RoleAssistant,
						Content: "Hello! How can I help?",
					},
					FinishReason: "stop",
				},
			},
			Usage: &Usage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("ID = %s, want chatcmpl-123", resp.ID)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello! How can I help?" {
		t.Errorf("Unexpected content: %s", resp.Choices[0].Message.Content)
	}
}

func TestOpenAIChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body has stream: true
		body, _ := io.ReadAll(r.Body)
		var req ChatRequest
		json.Unmarshal(body, &req)
		if !req.Stream {
			t.Error("Expected stream: true in request")
		}

		// Return SSE stream
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`{"id":"chatcmpl-123","choices":[{"delta":{"role":"assistant"},"index":0}]}`,
			`{"id":"chatcmpl-123","choices":[{"delta":{"content":"Hello"},"index":0}]}`,
			`{"id":"chatcmpl-123","choices":[{"delta":{"content":"!"},"index":0}]}`,
			`{"id":"chatcmpl-123","choices":[{"delta":{},"index":0,"finish_reason":"stop"}]}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte("data: " + chunk + "\n\n"))
		}
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "Hi"},
		},
	}

	callbackCount := 0
	err := provider.ChatStream(context.Background(), req, func(resp *ChatResponse) error {
		callbackCount++
		return nil
	})

	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}

	// Verify we received multiple callbacks
	if callbackCount == 0 {
		t.Error("Expected stream callbacks, got none")
	}
}

func TestOpenAIEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/embeddings") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := EmbeddingResponse{
			Object: "list",
			Model:  "text-embedding-3-small",
			Data: []Embedding{
				{
					Object:    "embedding",
					Index:     0,
					Embedding: []float32{0.1, 0.2, 0.3},
				},
			},
			Usage: &Usage{
				PromptTokens: 5,
				TotalTokens:  5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: []string{"Hello world"},
	}

	resp, err := provider.Embed(context.Background(), req)
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("Expected 1 embedding, got %d", len(resp.Data))
	}
	if len(resp.Data[0].Embedding) != 3 {
		t.Errorf("Expected 3 dimensions, got %d", len(resp.Data[0].Embedding))
	}
}

func TestOpenAIListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		resp := struct {
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
				{ID: "gpt-4", Object: "model", OwnedBy: "openai"},
				{ID: "gpt-3.5-turbo", Object: "model", OwnedBy: "openai"},
				{ID: "text-embedding-3-small", Object: "model", OwnedBy: "openai"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}

	if len(models) != 3 {
		t.Errorf("Expected 3 models, got %d", len(models))
	}
}

func TestOpenAIHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}{
			Data: []struct {
				ID string `json:"id"`
			}{
				{ID: "gpt-4"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	status, err := provider.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck() error: %v", err)
	}

	if !status.Available {
		t.Error("Expected Available to be true")
	}
	if status.Latency <= 0 {
		t.Error("Expected positive latency")
	}
}

func TestOpenAIErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
	}{
		{
			name:       "401 unauthorized",
			statusCode: 401,
			body:       `{"error":{"message":"Invalid API key","type":"invalid_api_key"}}`,
			wantErr:    ErrInvalidCredentials,
		},
		{
			name:       "429 rate limit",
			statusCode: 429,
			body:       `{"error":{"message":"Rate limit exceeded","type":"rate_limit"}}`,
			wantErr:    ErrRateLimitExceeded,
		},
		{
			name:       "400 context length",
			statusCode: 400,
			body:       `{"error":{"message":"context_length exceeded","type":"invalid_request"}}`,
			wantErr:    ErrContextLengthExceeded,
		},
		{
			name:       "503 unavailable",
			statusCode: 503,
			body:       `{"error":{"message":"Service unavailable","type":"server_error"}}`,
			wantErr:    ErrProviderUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			provider, _ := NewOpenAIProvider(&ProviderConfig{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			_, err := provider.Chat(context.Background(), &ChatRequest{
				Model:    "gpt-4",
				Messages: []Message{{Role: RoleUser, Content: "Hi"}},
			})

			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestOpenAIClose(t *testing.T) {
	provider, _ := NewOpenAIProvider(&ProviderConfig{APIKey: "test-key"})
	if err := provider.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}
