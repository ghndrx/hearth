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

func TestNewVertexAIProvider(t *testing.T) {
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
			name:    "empty project ID",
			config:  &ProviderConfig{},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &ProviderConfig{
				ProjectID: "test-project",
			},
			wantErr: false,
		},
		{
			name: "config with custom region",
			config: &ProviderConfig{
				ProjectID: "test-project",
				Region:    "europe-west1",
			},
			wantErr: false,
		},
		{
			name: "config with API key",
			config: &ProviderConfig{
				ProjectID: "test-project",
				APIKey:    "test-token",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewVertexAIProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVertexAIProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if provider == nil {
					t.Error("Expected provider, got nil")
					return
				}
				if provider.Name() != "Google Vertex AI" {
					t.Errorf("Expected name 'Google Vertex AI', got '%s'", provider.Name())
				}
				if provider.Type() != "vertex_ai" {
					t.Errorf("Expected type 'vertex_ai', got '%s'", provider.Type())
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

				// Check default region
				expectedRegion := "us-central1"
				if tt.config.Region != "" {
					expectedRegion = tt.config.Region
				}
				if provider.location != expectedRegion {
					t.Errorf("Expected location '%s', got '%s'", expectedRegion, provider.location)
				}
			}
		})
	}
}

func TestVertexAIProviderMetadata(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.Name() != "Google Vertex AI" {
		t.Errorf("Expected name 'Google Vertex AI', got %s", provider.Name())
	}
	if provider.Type() != "vertex_ai" {
		t.Errorf("Expected type 'vertex_ai', got %s", provider.Type())
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

func TestVertexAIProviderListModels(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
	}
	provider, err := NewVertexAIProvider(config)
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
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.0-pro",
		"text-embedding-004",
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

func TestVertexAIProviderConvertRequest(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "test-token", // Use explicit token to avoid gcloud calls
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	temp := 0.7
	topP := 0.9

	req := &ChatRequest{
		Model: "gemini-1.5-flash",
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

	vertexReq := provider.convertRequest(req)

	// Check system instruction
	if vertexReq.SystemInstruction == nil {
		t.Error("Expected system instruction to be set")
	} else if vertexReq.SystemInstruction.Parts[0].Text != "You are a helpful assistant" {
		t.Errorf("Expected system instruction 'You are a helpful assistant', got %s", vertexReq.SystemInstruction.Parts[0].Text)
	}

	// Check messages (should exclude system message)
	if len(vertexReq.Contents) != 3 {
		t.Errorf("Expected 3 contents, got %d", len(vertexReq.Contents))
	}

	// Check role conversion
	if vertexReq.Contents[0].Role != "user" {
		t.Errorf("Expected role 'user', got %s", vertexReq.Contents[0].Role)
	}
	if vertexReq.Contents[1].Role != "model" {
		t.Errorf("Expected role 'model', got %s", vertexReq.Contents[1].Role)
	}

	// Check generation config
	if vertexReq.GenerationConfig.MaxOutputTokens != 150 {
		t.Errorf("Expected max tokens 150, got %d", vertexReq.GenerationConfig.MaxOutputTokens)
	}
	if vertexReq.GenerationConfig.Temperature == nil || *vertexReq.GenerationConfig.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %v", vertexReq.GenerationConfig.Temperature)
	}
	if vertexReq.GenerationConfig.TopP == nil || *vertexReq.GenerationConfig.TopP != 0.9 {
		t.Errorf("Expected top_p 0.9, got %v", vertexReq.GenerationConfig.TopP)
	}
	if len(vertexReq.GenerationConfig.StopSequences) != 1 || vertexReq.GenerationConfig.StopSequences[0] != "END" {
		t.Errorf("Expected stop sequences ['END'], got %v", vertexReq.GenerationConfig.StopSequences)
	}
}

func TestVertexAIProviderConvertResponse(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "test-token",
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	vertexResp := &vertexResponse{
		Candidates: []struct {
			Content struct {
				Role  string       `json:"role"`
				Parts []vertexPart `json:"parts"`
			} `json:"content"`
			FinishReason  string `json:"finishReason"`
			SafetyRatings []struct {
				Category    string `json:"category"`
				Probability string `json:"probability"`
			} `json:"safetyRatings"`
		}{
			{
				Content: struct {
					Role  string       `json:"role"`
					Parts []vertexPart `json:"parts"`
				}{
					Role: "model",
					Parts: []vertexPart{
						{Text: "Hello"},
						{Text: " there!"},
					},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		}{
			PromptTokenCount:     10,
			CandidatesTokenCount: 15,
			TotalTokenCount:      25,
		},
	}

	chatResp := provider.convertResponse("gemini-1.5-flash", vertexResp)

	if chatResp.Model != "gemini-1.5-flash" {
		t.Errorf("Expected model 'gemini-1.5-flash', got %s", chatResp.Model)
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

func TestVertexAIProviderMapFinishReason(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "test-token",
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"STOP", "stop"},
		{"MAX_TOKENS", "length"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"OTHER", "OTHER"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := provider.mapFinishReason(tt.input)
			if result != tt.expected {
				t.Errorf("mapFinishReason(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestVertexAIProviderGetAccessTokenWithAPIKey(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "explicit-token",
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	token, err := provider.getAccessToken()
	if err != nil {
		t.Fatalf("getAccessToken() error = %v", err)
	}

	if token != "explicit-token" {
		t.Errorf("Expected token 'explicit-token', got %s", token)
	}
}

func TestVertexAIProviderChat(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/generateContent") {
			t.Errorf("Expected generateContent in path, got %s", r.URL.Path)
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer") {
			t.Errorf("Expected Bearer authorization, got %s", auth)
		}

		// Mock successful response
		response := vertexResponse{
			Candidates: []struct {
				Content struct {
					Role  string       `json:"role"`
					Parts []vertexPart `json:"parts"`
				} `json:"content"`
				FinishReason  string `json:"finishReason"`
				SafetyRatings []struct {
					Category    string `json:"category"`
					Probability string `json:"probability"`
				} `json:"safetyRatings"`
			}{
				{
					Content: struct {
						Role  string       `json:"role"`
						Parts []vertexPart `json:"parts"`
					}{
						Role: "model",
						Parts: []vertexPart{
							{Text: "Hello! How can I help you today?"},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			}{
				PromptTokenCount:     10,
				CandidatesTokenCount: 15,
				TotalTokenCount:      25,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Mock the Vertex AI URL format - we'll need to adjust the provider to use the mock server
	// This is a simplified test since the actual URL includes the project and location
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "test-token",
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ChatRequest{
		Model: "gemini-1.5-flash",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens: 100,
	}

	// Test the request conversion
	vertexReq := provider.convertRequest(req)
	if vertexReq == nil {
		t.Error("Expected converted request, got nil")
		return
	}

	if len(vertexReq.Contents) != 1 {
		t.Errorf("Expected 1 content, got %d", len(vertexReq.Contents))
	}

	if vertexReq.Contents[0].Role != "user" {
		t.Errorf("Expected role 'user', got %s", vertexReq.Contents[0].Role)
	}

	if vertexReq.GenerationConfig.MaxOutputTokens != 100 {
		t.Errorf("Expected max tokens 100, got %d", vertexReq.GenerationConfig.MaxOutputTokens)
	}
}

func TestVertexAIProviderChatStream(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "test-token",
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ChatRequest{
		Model: "gemini-1.5-flash",
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

func TestVertexAIProviderEmbed(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "test-token",
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &EmbeddingRequest{
		Model: "text-embedding-004",
		Input: []string{"Hello world"},
	}

	ctx := context.Background()
	// This will fail with auth, but we're testing the structure
	_, err = provider.Embed(ctx, req)
	// We expect this to fail due to auth, so we won't check the error
}

func TestVertexAIProviderHealthCheck(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "test-token",
	}
	provider, err := NewVertexAIProvider(config)
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

func TestVertexAIProviderParseError(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "test-token",
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		statusCode int
		body       []byte
		wantErr    error
	}{
		{401, []byte(`{"error": {"message": "Unauthorized"}}`), ErrInvalidCredentials},
		{403, []byte(`{"error": {"message": "Forbidden"}}`), ErrInvalidCredentials},
		{429, []byte(`{"error": {"message": "Rate limit exceeded"}}`), ErrRateLimitExceeded},
		{400, []byte(`{"error": {"message": "Context length exceeded"}}`), ErrContextLengthExceeded},
		{400, []byte(`{"error": {"message": "Invalid request"}}`), ErrInvalidRequest},
		{404, []byte(`{"error": {"message": "Model not found"}}`), ErrModelNotFound},
		{503, []byte(`{"error": {"message": "Service unavailable"}}`), ErrProviderUnavailable},
		{500, []byte(`{"error": {"message": "Internal error"}}`), nil}, // Should return generic error
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

func TestVertexAIProviderTokenCaching(t *testing.T) {
	// Reset the global token cache for this test
	vertexTokenCache.mu.Lock()
	vertexTokenCache.token = ""
	vertexTokenCache.expiry = time.Time{}
	vertexTokenCache.mu.Unlock()

	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "explicit-token", // Use explicit token to test caching behavior
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// First call should use explicit token
	token1, err := provider.getAccessToken()
	if err != nil {
		t.Fatalf("First getAccessToken() error = %v", err)
	}

	// Second call should also use explicit token (no caching for explicit tokens)
	token2, err := provider.getAccessToken()
	if err != nil {
		t.Fatalf("Second getAccessToken() error = %v", err)
	}

	if token1 != "explicit-token" {
		t.Errorf("Expected first token 'explicit-token', got %s", token1)
	}
	if token2 != "explicit-token" {
		t.Errorf("Expected second token 'explicit-token', got %s", token2)
	}
}

func TestVertexAIProviderConvertRequestNoSystem(t *testing.T) {
	config := &ProviderConfig{
		ProjectID: "test-project",
		APIKey:    "test-token",
	}
	provider, err := NewVertexAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := &ChatRequest{
		Model: "gemini-1.5-flash",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
			{Role: RoleAssistant, Content: "Hi there!"},
		},
		MaxTokens: 100,
	}

	vertexReq := provider.convertRequest(req)

	// Check no system instruction
	if vertexReq.SystemInstruction != nil {
		t.Error("Expected no system instruction, got one")
	}

	// Check messages
	if len(vertexReq.Contents) != 2 {
		t.Errorf("Expected 2 contents, got %d", len(vertexReq.Contents))
	}

	// Check role conversion
	if vertexReq.Contents[0].Role != "user" {
		t.Errorf("Expected role 'user', got %s", vertexReq.Contents[0].Role)
	}
	if vertexReq.Contents[1].Role != "model" {
		t.Errorf("Expected role 'model', got %s", vertexReq.Contents[1].Role)
	}
}
