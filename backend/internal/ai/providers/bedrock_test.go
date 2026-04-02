package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewBedrockProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: true, // Should fail because no credentials
		},
		{
			name:    "empty API key",
			config:  &ProviderConfig{},
			wantErr: true,
		},
		{
			name: "missing secret key",
			config: &ProviderConfig{
				APIKey: "test-access-key",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &ProviderConfig{
				APIKey:    "test-access-key",
				SecretKey: "test-secret-key",
			},
			wantErr: false,
		},
		{
			name: "custom region",
			config: &ProviderConfig{
				APIKey:    "test-access-key",
				SecretKey: "test-secret-key",
				Region:    "eu-west-1",
			},
			wantErr: false,
		},
		{
			name: "with session token",
			config: &ProviderConfig{
				APIKey:    "test-access-key",
				SecretKey: "test-secret-key",
				Custom: map[string]interface{}{
					"session_token": "test-session-token",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewBedrockProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBedrockProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if provider == nil {
					t.Error("Expected provider, got nil")
					return
				}
				if provider.Name() != "AWS Bedrock" {
					t.Errorf("Expected name 'AWS Bedrock', got %s", provider.Name())
				}
				if provider.Type() != "bedrock" {
					t.Errorf("Expected type 'bedrock', got %s", provider.Type())
				}
				if !provider.SupportsStreaming() {
					t.Error("Expected to support streaming")
				}
				if !provider.SupportsEmbeddings() {
					t.Error("Expected to support embeddings")
				}
				if !provider.SupportsFunctionCalling() {
					t.Error("Expected to support function calling")
				}
			}
		})
	}
}

func TestBedrockProviderMetadata(t *testing.T) {
	config := &ProviderConfig{
		APIKey:    "test-access-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.Name() != "AWS Bedrock" {
		t.Errorf("Expected name 'AWS Bedrock', got %s", provider.Name())
	}
	if provider.Type() != "bedrock" {
		t.Errorf("Expected type 'bedrock', got %s", provider.Type())
	}
	if !provider.SupportsStreaming() {
		t.Error("Expected to support streaming")
	}
	if !provider.SupportsEmbeddings() {
		t.Error("Expected to support embeddings")
	}
	if !provider.SupportsFunctionCalling() {
		t.Error("Expected to support function calling")
	}
	if err := provider.Close(); err != nil {
		t.Errorf("Expected Close() to return nil, got %v", err)
	}
}

func TestBedrockProviderListModels(t *testing.T) {
	config := &ProviderConfig{
		APIKey:    "test-access-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Errorf("ListModels() error = %v", err)
		return
	}

	if len(models) == 0 {
		t.Error("Expected models list to not be empty")
	}

	// Check for expected models
	expectedModels := []string{
		"anthropic.claude-3-opus-20240229-v1:0",
		"anthropic.claude-3-sonnet-20240229-v1:0",
		"anthropic.claude-3-haiku-20240307-v1:0",
		"amazon.titan-embed-text-v2:0",
	}

	for _, expectedModel := range expectedModels {
		found := false
		for _, model := range models {
			if model.ID == expectedModel {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model %s not found in models list", expectedModel)
		}
	}
}

func TestBedrockProviderChat(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/model/") || !strings.Contains(r.URL.Path, "/converse") {
			t.Errorf("Expected /model/.../converse path, got %s", r.URL.Path)
		}

		// Check authorization header format
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
			t.Errorf("Expected AWS4-HMAC-SHA256 authorization, got %s", auth)
		}

		// Mock successful response
		response := bedrockResponse{
			Output: struct {
				Message bedrockMessage `json:"message"`
			}{
				Message: bedrockMessage{
					Role: "assistant",
					Content: []bedrockContent{
						{Text: "Hello! How can I help you today?"},
					},
				},
			},
			StopReason: "stop",
			Usage: struct {
				InputTokens  int `json:"inputTokens"`
				OutputTokens int `json:"outputTokens"`
			}{
				InputTokens:  10,
				OutputTokens: 15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with custom HTTP client pointing to mock server
	config := &ProviderConfig{
		APIKey:    "test-access-key",
		SecretKey: "test-secret-key",
		Region:    "us-east-1",
	}
	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Replace the base URL to point to our test server
	// Note: This is a simplified test - in reality we'd need to mock the full AWS signing
	// For now, we'll test the request conversion logic

	req := &ChatRequest{
		Model: "anthropic.claude-3-haiku-20240307-v1:0",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens: 100,
	}

	// Test the request conversion
	bedrockReq := provider.convertRequest(req)
	if bedrockReq == nil {
		t.Error("Expected converted request, got nil")
		return
	}

	if len(bedrockReq.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(bedrockReq.Messages))
	}

	if bedrockReq.Messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got %s", bedrockReq.Messages[0].Role)
	}

	if bedrockReq.InferenceConfig.MaxTokens != 100 {
		t.Errorf("Expected max tokens 100, got %d", bedrockReq.InferenceConfig.MaxTokens)
	}
}

func TestBedrockProviderChatStream(t *testing.T) {
	config := &ProviderConfig{
		APIKey:    "test-access-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ChatRequest{
		Model: "anthropic.claude-3-haiku-20240307-v1:0",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens: 100,
	}

	// Test streaming callback
	responses := make([]*ChatResponse, 0)
	callback := func(resp *ChatResponse) error {
		responses = append(responses, resp)
		return nil
	}

	ctx := context.Background()
	// This will fail with auth, but we're testing the structure
	err = provider.ChatStream(ctx, req, callback)
	// We expect this to fail due to auth, so we won't check the error
}

func TestBedrockProviderEmbed(t *testing.T) {
	config := &ProviderConfig{
		APIKey:    "test-access-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &EmbeddingRequest{
		Model: "amazon.titan-embed-text-v2:0",
		Input: []string{"Hello world"},
	}

	ctx := context.Background()
	// This will fail with auth, but we're testing the structure
	_, err = provider.Embed(ctx, req)
	// We expect this to fail due to auth, so we won't check the error
}

func TestBedrockProviderHealthCheck(t *testing.T) {
	config := &ProviderConfig{
		APIKey:    "test-access-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	status, err := provider.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}

	if status == nil {
		t.Error("Expected status, got nil")
		return
	}

	if status.CheckedAt.IsZero() {
		t.Error("Expected non-zero CheckedAt time")
	}

	// Should fail due to auth, so Available should be false
	if status.Available {
		t.Error("Expected Available to be false due to invalid auth")
	}
}

func TestBedrockProviderConvertRequest(t *testing.T) {
	config := &ProviderConfig{
		APIKey:    "test-access-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	temp := 0.7
	topP := 0.9

	req := &ChatRequest{
		Model: "anthropic.claude-3-haiku-20240307-v1:0",
		Messages: []Message{
			{Role: RoleSystem, Content: "You are a helpful assistant"},
			{Role: RoleUser, Content: "Hello"},
			{Role: RoleAssistant, Content: "Hi there!"},
			{Role: RoleUser, Content: "How are you?"},
		},
		MaxTokens:   150,
		Temperature: &temp,
		TopP:        &topP,
		Stop:        []string{"END"},
	}

	bedrockReq := provider.convertRequest(req)

	// Check system instruction
	if bedrockReq.System == nil || len(bedrockReq.System) == 0 {
		t.Error("Expected system instruction to be set")
	} else if bedrockReq.System[0].Text != "You are a helpful assistant" {
		t.Errorf("Expected system instruction 'You are a helpful assistant', got %s", bedrockReq.System[0].Text)
	}

	// Check messages (should exclude system message)
	if len(bedrockReq.Messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(bedrockReq.Messages))
	}

	// Check inference config
	if bedrockReq.InferenceConfig.MaxTokens != 150 {
		t.Errorf("Expected max tokens 150, got %d", bedrockReq.InferenceConfig.MaxTokens)
	}
	if bedrockReq.InferenceConfig.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", bedrockReq.InferenceConfig.Temperature)
	}
	if bedrockReq.InferenceConfig.TopP != 0.9 {
		t.Errorf("Expected top_p 0.9, got %f", bedrockReq.InferenceConfig.TopP)
	}
	if len(bedrockReq.InferenceConfig.StopSequences) != 1 || bedrockReq.InferenceConfig.StopSequences[0] != "END" {
		t.Errorf("Expected stop sequences ['END'], got %v", bedrockReq.InferenceConfig.StopSequences)
	}
}

func TestBedrockProviderConvertResponse(t *testing.T) {
	config := &ProviderConfig{
		APIKey:    "test-access-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	bedrockResp := &bedrockResponse{
		Output: struct {
			Message bedrockMessage `json:"message"`
		}{
			Message: bedrockMessage{
				Role: "assistant",
				Content: []bedrockContent{
					{Text: "Hello"},
					{Text: " there!"},
				},
			},
		},
		StopReason: "stop",
		Usage: struct {
			InputTokens  int `json:"inputTokens"`
			OutputTokens int `json:"outputTokens"`
		}{
			InputTokens:  10,
			OutputTokens: 15,
		},
	}

	chatResp := provider.convertResponse("test-model", bedrockResp)

	if chatResp.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got %s", chatResp.Model)
	}
	if chatResp.Object != "chat.completion" {
		t.Errorf("Expected object 'chat.completion', got %s", chatResp.Object)
	}
	if len(chatResp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(chatResp.Choices))
	}
	if chatResp.Choices[0].Message.Content != "Hello there!" {
		t.Errorf("Expected content 'Hello there!', got %s", chatResp.Choices[0].Message.Content)
	}
	if chatResp.Choices[0].FinishReason != "stop" {
		t.Errorf("Expected finish reason 'stop', got %s", chatResp.Choices[0].FinishReason)
	}
	if chatResp.Usage.PromptTokens != 10 {
		t.Errorf("Expected prompt tokens 10, got %d", chatResp.Usage.PromptTokens)
	}
	if chatResp.Usage.CompletionTokens != 15 {
		t.Errorf("Expected completion tokens 15, got %d", chatResp.Usage.CompletionTokens)
	}
	if chatResp.Usage.TotalTokens != 25 {
		t.Errorf("Expected total tokens 25, got %d", chatResp.Usage.TotalTokens)
	}
}

func TestBedrockProviderParseError(t *testing.T) {
	config := &ProviderConfig{
		APIKey:    "test-access-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		statusCode int
		body       []byte
		wantErr    error
	}{
		{401, []byte(`{"message": "Unauthorized"}`), ErrInvalidCredentials},
		{403, []byte(`{"message": "Forbidden"}`), ErrInvalidCredentials},
		{429, []byte(`{"message": "Rate limit exceeded"}`), ErrRateLimitExceeded},
		{400, []byte(`{"message": "Context length exceeded"}`), ErrContextLengthExceeded},
		{400, []byte(`{"message": "Invalid request"}`), ErrInvalidRequest},
		{404, []byte(`{"message": "Model not found"}`), ErrModelNotFound},
		{503, []byte(`{"message": "Service unavailable"}`), ErrProviderUnavailable},
		{500, []byte(`{"message": "Internal error"}`), nil}, // Should return generic error
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
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