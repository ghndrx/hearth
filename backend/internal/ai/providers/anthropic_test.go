package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAnthropicProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "empty API key",
			config:  &ProviderConfig{},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &ProviderConfig{
				APIKey: "sk-ant-test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAnthropicProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAnthropicProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && provider == nil {
				t.Error("Expected provider, got nil")
			}
		})
	}
}

func TestAnthropicProviderMetadata(t *testing.T) {
	provider, _ := NewAnthropicProvider(&ProviderConfig{APIKey: "test-key"})

	if provider.Name() != "Anthropic" {
		t.Errorf("Name() = %s, want Anthropic", provider.Name())
	}
	if provider.Type() != "anthropic" {
		t.Errorf("Type() = %s, want anthropic", provider.Type())
	}
	if !provider.SupportsStreaming() {
		t.Error("SupportsStreaming() should be true")
	}
	if provider.SupportsEmbeddings() {
		t.Error("SupportsEmbeddings() should be false")
	}
	if !provider.SupportsFunctionCalling() {
		t.Error("SupportsFunctionCalling() should be true")
	}
}

func TestAnthropicChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("Missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Error("Missing or invalid anthropic-version header")
		}

		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{"type": "text", "text": "Hello! I'm Claude."},
			},
			"model":       "claude-3-sonnet-20240229",
			"stop_reason": "end_turn",
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &ChatRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens: 1000,
	}

	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if resp.ID != "msg_123" {
		t.Errorf("ID = %s, want msg_123", resp.ID)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(resp.Choices))
	}
}

func TestAnthropicChatWithSystemPrompt(t *testing.T) {
	var receivedReq map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := map[string]interface{}{
			"id":      "msg_123",
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]interface{}{{"type": "text", "text": "Hi"}},
			"model":   "claude-3-sonnet-20240229",
			"usage":   map[string]int{"input_tokens": 10, "output_tokens": 2},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &ChatRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []Message{
			{Role: RoleSystem, Content: "You are helpful"},
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens: 1000,
	}

	_, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	// Verify system prompt was extracted
	if receivedReq["system"] != "You are helpful" {
		t.Error("System prompt should be extracted to top-level field")
	}
}

func TestAnthropicChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		events := []string{
			`{"type":"content_block_start","index":0}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"!"}}`,
			`{"type":"message_stop"}`,
		}

		for _, event := range events {
			_, _ = w.Write([]byte("data: " + event + "\n\n"))
		}
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &ChatRequest{
		Model:     "claude-3-sonnet-20240229",
		Messages:  []Message{{Role: RoleUser, Content: "Hi"}},
		MaxTokens: 1000,
	}

	var accumulated string
	err := provider.ChatStream(context.Background(), req, func(resp *ChatResponse) error {
		if resp.Delta != nil {
			accumulated += resp.Delta.Content
		}
		return nil
	})

	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}

	if accumulated != "Hello!" {
		t.Errorf("Accumulated = %s, want Hello!", accumulated)
	}
}

func TestAnthropicEmbed(t *testing.T) {
	provider, _ := NewAnthropicProvider(&ProviderConfig{APIKey: "test-key"})

	_, err := provider.Embed(context.Background(), &EmbeddingRequest{})
	if err == nil {
		t.Error("Expected error for unsupported embeddings")
	}
}

func TestAnthropicListModels(t *testing.T) {
	provider, _ := NewAnthropicProvider(&ProviderConfig{APIKey: "test-key"})

	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected models, got empty list")
	}

	// Check for known models
	foundOpus := false
	foundSonnet := false
	for _, m := range models {
		if strings.Contains(m.ID, "opus") {
			foundOpus = true
		}
		if strings.Contains(m.ID, "sonnet") {
			foundSonnet = true
		}
	}

	if !foundOpus || !foundSonnet {
		t.Error("Expected to find opus and sonnet models")
	}
}

func TestAnthropicErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{
			name:       "401 unauthorized",
			statusCode: 401,
			body:       `{"error":{"type":"authentication_error","message":"Invalid API key"}}`,
		},
		{
			name:       "429 rate limit",
			statusCode: 429,
			body:       `{"error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`,
		},
		{
			name:       "529 overloaded",
			statusCode: 529,
			body:       `{"error":{"type":"overloaded_error","message":"Service overloaded"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				//nolint:errcheck
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			provider, _ := NewAnthropicProvider(&ProviderConfig{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			_, err := provider.Chat(context.Background(), &ChatRequest{
				Model:     "claude-3-sonnet-20240229",
				Messages:  []Message{{Role: RoleUser, Content: "Hi"}},
				MaxTokens: 100,
			})

			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestAnthropicToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type":        "tool_use",
					"tool_use_id": "toolu_123",
					"name":        "get_weather",
					"input":       map[string]string{"location": "San Francisco"},
				},
			},
			"model":       "claude-3-sonnet-20240229",
			"stop_reason": "tool_use",
			"usage":       map[string]int{"input_tokens": 10, "output_tokens": 20},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider(&ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &ChatRequest{
		Model:    "claude-3-sonnet-20240229",
		Messages: []Message{{Role: RoleUser, Content: "What's the weather?"}},
		Tools: []Tool{{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_weather",
				Description: "Get the weather",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		}},
		MaxTokens: 1000,
	}

	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %s, want tool_calls", resp.FinishReason)
	}
}

func TestAnthropicClose(t *testing.T) {
	provider, _ := NewAnthropicProvider(&ProviderConfig{APIKey: "test-key"})
	if err := provider.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}
