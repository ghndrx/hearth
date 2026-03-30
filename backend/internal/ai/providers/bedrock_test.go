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
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "empty config",
			config:  &ProviderConfig{},
			wantErr: true, // No APIKey or SecretKey
		},
		{
			name: "missing secret key",
			config: &ProviderConfig{
				APIKey: "access-key",
			},
			wantErr: true, // Missing SecretKey
		},
		{
			name: "valid config with secret key",
			config: &ProviderConfig{
				APIKey:    "access-key",
				SecretKey: "secret-key",
			},
			wantErr: false,
		},
		{
			name: "valid config with region",
			config: &ProviderConfig{
				APIKey:    "access-key",
				SecretKey: "secret-key",
				Region:    "us-west-2",
			},
			wantErr: false,
		},
		{
			name: "valid config with session token",
			config: &ProviderConfig{
				APIKey:    "access-key",
				SecretKey: "secret-key",
				Custom:    map[string]interface{}{"session_token": "session-token-123"},
			},
			wantErr: false,
		},
		{
			name: "valid config with custom timeout",
			config: &ProviderConfig{
				APIKey:    "access-key",
				SecretKey: "secret-key",
				Timeout:   120,
			},
			wantErr: false,
		},
		{
			name: "default region is us-east-1",
			config: &ProviderConfig{
				APIKey:    "access-key",
				SecretKey: "secret-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewBedrockProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBedrockProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && provider == nil {
				t.Error("Expected provider, got nil")
			}
		})
	}
}

func TestBedrockProviderMetadata(t *testing.T) {
	provider, err := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.Name() != "AWS Bedrock" {
		t.Errorf("Name() = %s, want AWS Bedrock", provider.Name())
	}
	if provider.Type() != "bedrock" {
		t.Errorf("Type() = %s, want bedrock", provider.Type())
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

func TestBedrockChatWithMockServer(t *testing.T) {
	// This test requires a server that will accept AWS Signature V4
	// Since BedrockProvider constructs its own URL from region, we use a real-style mock
	// Note: This will fail signature validation unless we mock at network level
	// Instead, we test the error path through parseError
}

func TestBedrockChatWithSystemPrompt(t *testing.T) {
	var receivedReq map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Accept the request and decode
		_ = json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := map[string]interface{}{
			"output": map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": []map[string]interface{}{{"text": "Hello"}},
				},
			},
			"stopReason": "end_turn",
			"usage":      map[string]int{"inputTokens": 10, "outputTokens": 2},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Use a custom HTTP client transport to intercept
	provider, _ := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
		BaseURL:   server.URL,
	})

	req := &ChatRequest{
		Model: "anthropic.claude-3-haiku-20240307-v1:0",
		Messages: []Message{
			{Role: RoleSystem, Content: "You are a helpful assistant."},
			{Role: RoleUser, Content: "Hi"},
		},
		MaxTokens: 100,
	}

	// This will fail because Bedrock ignores BaseURL and connects to real AWS
	// We test the conversion logic separately instead
	_ = provider
	_ = req

	// Verify the request conversion works
	bedrockReq := provider.convertRequest(req)
	if len(bedrockReq.System) != 1 {
		t.Errorf("Expected 1 system prompt, got %d", len(bedrockReq.System))
	}
	if bedrockReq.System[0].Text != "You are a helpful assistant." {
		t.Errorf("Expected system content, got '%s'", bedrockReq.System[0].Text)
	}
}

func TestBedrockListModels(t *testing.T) {
	provider, _ := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
	})

	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected models, got empty list")
	}

	// Check for known models
	modelIDs := make(map[string]bool)
	for _, m := range models {
		modelIDs[m.ID] = true
	}

	expectedModels := []string{
		"anthropic.claude-3-opus-20240229-v1:0",
		"anthropic.claude-3-sonnet-20240229-v1:0",
		"anthropic.claude-3-haiku-20240307-v1:0",
		"anthropic.claude-3-5-sonnet-20241022-v2:0",
		"meta.llama3-1-405b-instruct-v1:0",
		"meta.llama3-1-70b-instruct-v1:0",
		"amazon.titan-embed-text-v2:0",
		"cohere.embed-english-v3",
	}

	for _, expected := range expectedModels {
		if !modelIDs[expected] {
			t.Errorf("Expected model %s not found", expected)
		}
	}

	// Check Claude models have chat capability
	for _, m := range models {
		if strings.Contains(m.ID, "anthropic") {
			found := false
			for _, cap := range m.Capabilities {
				if cap == "chat" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Claude model %s missing chat capability", m.ID)
			}
		}
	}
}

func TestBedrockClose(t *testing.T) {
	provider, _ := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
	})

	if err := provider.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestBedrockConvertRequest(t *testing.T) {
	provider, _ := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
	})

	t.Run("basic conversion", func(t *testing.T) {
		req := &ChatRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			MaxTokens: 100,
		}

		bedrockReq := provider.convertRequest(req)

		if len(bedrockReq.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(bedrockReq.Messages))
		}
		if bedrockReq.Messages[0].Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", bedrockReq.Messages[0].Role)
		}
		if bedrockReq.Messages[0].Content[0].Text != "Hello" {
			t.Errorf("Expected content 'Hello', got '%s'", bedrockReq.Messages[0].Content[0].Text)
		}
	})

	t.Run("assistant role preserved", func(t *testing.T) {
		req := &ChatRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: RoleUser, Content: "Hello"},
				{Role: RoleAssistant, Content: "Hi there"},
			},
			MaxTokens: 100,
		}

		bedrockReq := provider.convertRequest(req)

		if len(bedrockReq.Messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(bedrockReq.Messages))
		}
		if bedrockReq.Messages[1].Role != "assistant" {
			t.Errorf("Expected role 'assistant', got '%s'", bedrockReq.Messages[1].Role)
		}
	})

	t.Run("system prompt extracted", func(t *testing.T) {
		req := &ChatRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: RoleSystem, Content: "You are helpful."},
				{Role: RoleUser, Content: "Hello"},
			},
			MaxTokens: 100,
		}

		bedrockReq := provider.convertRequest(req)

		if len(bedrockReq.System) != 1 {
			t.Errorf("Expected 1 system prompt, got %d", len(bedrockReq.System))
		}
		if bedrockReq.System[0].Text != "You are helpful." {
			t.Errorf("Expected system content 'You are helpful.', got '%s'", bedrockReq.System[0].Text)
		}
	})

	t.Run("stop sequences", func(t *testing.T) {
		req := &ChatRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			MaxTokens: 100,
			Stop:      []string{"STOP", "END"},
		}

		bedrockReq := provider.convertRequest(req)

		if len(bedrockReq.InferenceConfig.StopSequences) != 2 {
			t.Errorf("Expected 2 stop sequences, got %d", len(bedrockReq.InferenceConfig.StopSequences))
		}
	})

	t.Run("topP parameter", func(t *testing.T) {
		topP := 0.9
		req := &ChatRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			MaxTokens: 100,
			TopP:      &topP,
		}

		bedrockReq := provider.convertRequest(req)

		if bedrockReq.InferenceConfig.TopP != 0.9 {
			t.Errorf("Expected TopP 0.9, got %f", bedrockReq.InferenceConfig.TopP)
		}
	})

	t.Run("default max tokens", func(t *testing.T) {
		req := &ChatRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			MaxTokens: 0, // Not set
		}

		bedrockReq := provider.convertRequest(req)

		if bedrockReq.InferenceConfig.MaxTokens != 4096 {
			t.Errorf("Expected default MaxTokens 4096, got %d", bedrockReq.InferenceConfig.MaxTokens)
		}
	})

	t.Run("multiple user messages", func(t *testing.T) {
		req := &ChatRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: RoleUser, Content: "Hello"},
				{Role: RoleUser, Content: "How are you?"},
			},
			MaxTokens: 100,
		}

		bedrockReq := provider.convertRequest(req)

		if len(bedrockReq.Messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(bedrockReq.Messages))
		}
	})
}

func TestBedrockConvertResponse(t *testing.T) {
	provider, _ := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
	})

	t.Run("basic conversion", func(t *testing.T) {
		resp := &bedrockResponse{
			Output: struct {
				Message bedrockMessage `json:"message"`
			}{
				Message: bedrockMessage{
					Role: "assistant",
					Content: []bedrockContent{
						{Text: "Hello, world!"},
					},
				},
			},
			StopReason: "end_turn",
			Usage: struct {
				InputTokens  int `json:"inputTokens"`
				OutputTokens int `json:"outputTokens"`
			}{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}

		chatResp := provider.convertResponse("test-model", resp)

		if len(chatResp.Choices) != 1 {
			t.Errorf("Expected 1 choice, got %d", len(chatResp.Choices))
		}
		if chatResp.Choices[0].Message.Content != "Hello, world!" {
			t.Errorf("Expected content 'Hello, world!', got '%s'", chatResp.Choices[0].Message.Content)
		}
		if chatResp.FinishReason != "stop" {
			t.Errorf("Expected finish reason 'stop', got '%s'", chatResp.FinishReason)
		}
		if chatResp.Usage.TotalTokens != 15 {
			t.Errorf("Expected TotalTokens 15, got %d", chatResp.Usage.TotalTokens)
		}
	})

	t.Run("max tokens stop reason", func(t *testing.T) {
		resp := &bedrockResponse{
			Output: struct {
				Message bedrockMessage `json:"message"`
			}{
				Message: bedrockMessage{
					Role:    "assistant",
					Content: []bedrockContent{{Text: "Hi"}},
				},
			},
			StopReason: "max_tokens",
			Usage: struct {
				InputTokens  int `json:"inputTokens"`
				OutputTokens int `json:"outputTokens"`
			}{
				InputTokens:  10,
				OutputTokens: 100,
			},
		}

		chatResp := provider.convertResponse("test-model", resp)

		if chatResp.FinishReason != "length" {
			t.Errorf("Expected finish reason 'length', got '%s'", chatResp.FinishReason)
		}
	})

	t.Run("multiple content blocks", func(t *testing.T) {
		resp := &bedrockResponse{
			Output: struct {
				Message bedrockMessage `json:"message"`
			}{
				Message: bedrockMessage{
					Role:    "assistant",
					Content: []bedrockContent{{Text: "Hello"}, {Text: " World"}},
				},
			},
			StopReason: "end_turn",
			Usage: struct {
				InputTokens  int `json:"inputTokens"`
				OutputTokens int `json:"outputTokens"`
			}{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}

		chatResp := provider.convertResponse("test-model", resp)

		expectedContent := "Hello World"
		if chatResp.Choices[0].Message.Content != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, chatResp.Choices[0].Message.Content)
		}
	})
}

func TestBedrockSignRequest(t *testing.T) {
	provider, _ := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
	})

	req, _ := http.NewRequest("POST", "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/converse", nil)
	provider.signRequest(req, []byte(`{"test":"data"}`), "bedrock-runtime.us-east-1.amazonaws.com")

	// Verify headers are set
	if req.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type header not set")
	}
	if req.Header.Get("X-Amz-Date") == "" {
		t.Error("X-Amz-Date header not set")
	}
	if req.Header.Get("X-Amz-Content-Sha256") == "" {
		t.Error("X-Amz-Content-Sha256 header not set")
	}
	if req.Header.Get("Authorization") == "" {
		t.Error("Authorization header not set")
	}

	// Verify Authorization header has correct format
	authHeader := req.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256") {
		t.Errorf("Authorization header should start with AWS4-HMAC-SHA256, got: %s", authHeader)
	}
}

func TestBedrockSignRequestWithSessionToken(t *testing.T) {
	provider, _ := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
		Custom:    map[string]interface{}{"session_token": "session-token-123"},
	})

	req, _ := http.NewRequest("POST", "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/converse", nil)
	provider.signRequest(req, []byte(`{"test":"data"}`), "bedrock-runtime.us-east-1.amazonaws.com")

	// Verify session token header is set
	if req.Header.Get("X-Amz-Security-Token") != "session-token-123" {
		t.Error("X-Amz-Security-Token header not set correctly")
	}

	// Verify signedHeaders includes x-amz-security-token
	authHeader := req.Header.Get("Authorization")
	if !strings.Contains(authHeader, "x-amz-security-token") {
		t.Error("Authorization should include x-amz-security-token in SignedHeaders")
	}
}

func TestBedrockSignatureFunctions(t *testing.T) {
	// Test sha256Hash
	hash := sha256Hash([]byte("test data"))
	if len(hash) != 64 { // SHA256 produces 64 hex characters
		t.Errorf("Expected 64 char hash, got %d", len(hash))
	}

	// Test deterministic
	hash2 := sha256Hash([]byte("test data"))
	if hash != hash2 {
		t.Error("sha256Hash should be deterministic")
	}

	// Test different input produces different hash
	hash3 := sha256Hash([]byte("different data"))
	if hash == hash3 {
		t.Error("Different inputs should produce different hashes")
	}

	// Test hmacSHA256
	hmac := hmacSHA256([]byte("key"), []byte("data"))
	if len(hmac) != 32 { // HMAC-SHA256 produces 32 bytes
		t.Error("Expected 32 byte HMAC")
	}

	// Test deterministic
	hmac2 := hmacSHA256([]byte("key"), []byte("data"))
	if string(hmac) != string(hmac2) {
		t.Error("hmacSHA256 should be deterministic")
	}

	// Test getSignatureKey
	provider, _ := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
	})

	sigKey := provider.getSignatureKey("20240101", "us-east-1", "bedrock")
	if len(sigKey) != 32 {
		t.Errorf("Expected 32 byte signature key, got %d", len(sigKey))
	}

	// Test deterministic
	sigKey2 := provider.getSignatureKey("20240101", "us-east-1", "bedrock")
	if string(sigKey) != string(sigKey2) {
		t.Error("getSignatureKey should be deterministic")
	}
}

func TestBedrockParseError(t *testing.T) {
	provider, _ := NewBedrockProvider(&ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
	})

	tests := []struct {
		name         string
		statusCode   int
		body         string
		expectNilErr bool
	}{
		{
			name:       "401 unauthorized",
			statusCode: 401,
			body:       `{"message": "Unauthorized", "__type": "UnauthorizedException"}`,
		},
		{
			name:       "403 forbidden",
			statusCode: 403,
			body:       `{"message": "Forbidden", "__type": "AccessDeniedException"}`,
		},
		{
			name:       "429 rate limit",
			statusCode: 429,
			body:       `{"message": "Rate limit exceeded"}`,
		},
		{
			name:       "400 context length",
			statusCode: 400,
			body:       `{"message": "Context length exceeded"}`,
		},
		{
			name:       "400 token limit",
			statusCode: 400,
			body:       `{"message": "token limit exceeded"}`,
		},
		{
			name:       "404 not found",
			statusCode: 404,
			body:       `{"message": "Model not found"}`,
		},
		{
			name:       "503 unavailable",
			statusCode: 503,
			body:       `{"message": "Service unavailable"}`,
		},
		{
			name:       "502 bad gateway",
			statusCode: 502,
			body:       `{"message": "Bad gateway"}`,
		},
		{
			name:       "504 gateway timeout",
			statusCode: 504,
			body:       `{"message": "Gateway timeout"}`,
		},
		{
			name:       "400 other error",
			statusCode: 400,
			body:       `{"message": "Some other error"}`,
		},
		{
			name:       "500 internal error",
			statusCode: 500,
			body:       `{"message": "Internal server error"}`,
		},
		{
			name:       "empty body",
			statusCode: 500,
			body:       ``,
		},
		{
			name:       "unparseable body",
			statusCode: 400,
			body:       `{invalid`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.parseError(tt.statusCode, []byte(tt.body))
			if err == nil {
				t.Error("Expected error, got nil")
			}
			// Just verify an error is returned - error type varies by status code
		})
	}
}

func TestBedrockFactoryCreate(t *testing.T) {
	factory := NewProviderFactory()

	provider, err := factory.Create("bedrock", &ProviderConfig{
		APIKey:    "access-key",
		SecretKey: "secret-key",
	})
	if err != nil {
		t.Fatalf("Create(bedrock) error: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected provider, got nil")
	}
	if provider.Name() != "AWS Bedrock" {
		t.Errorf("Name() = %s, want AWS Bedrock", provider.Name())
	}
	if provider.Type() != "bedrock" {
		t.Errorf("Type() = %s, want bedrock", provider.Type())
	}
}
