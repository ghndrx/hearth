package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOllamaProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name:    "nil config (uses defaults)",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "empty config (uses defaults)",
			config:  &ProviderConfig{},
			wantErr: false,
		},
		{
			name: "custom base URL",
			config: &ProviderConfig{
				BaseURL: "http://custom:11434",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOllamaProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOllamaProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && provider == nil {
				t.Error("Expected provider, got nil")
			}
		})
	}
}

func TestOllamaProviderMetadata(t *testing.T) {
	provider, _ := NewOllamaProvider(nil)

	if provider.Name() != "Ollama" {
		t.Errorf("Name() = %s, want Ollama", provider.Name())
	}
	if provider.Type() != "ollama" {
		t.Errorf("Type() = %s, want ollama", provider.Type())
	}
	if !provider.SupportsStreaming() {
		t.Error("SupportsStreaming() should be true")
	}
	if !provider.SupportsEmbeddings() {
		t.Error("SupportsEmbeddings() should be true")
	}
	if provider.SupportsFunctionCalling() {
		t.Error("SupportsFunctionCalling() should be false")
	}
}

func TestOllamaChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := map[string]interface{}{
			"model":      "llama2",
			"created_at": "2024-01-01T00:00:00Z",
			"message": map[string]string{
				"role":    "assistant",
				"content": "Hello! I'm Llama.",
			},
			"done":              true,
			"total_duration":    1000000000,
			"prompt_eval_count": 10,
			"eval_count":        8,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(&ProviderConfig{BaseURL: server.URL})

	req := &ChatRequest{
		Model: "llama2",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if resp.Model != "llama2" {
		t.Errorf("Model = %s, want llama2", resp.Model)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello! I'm Llama." {
		t.Errorf("Unexpected content: %s", resp.Choices[0].Message.Content)
	}
}

func TestOllamaChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)

		chunks := []map[string]interface{}{
			{
				"model":      "llama2",
				"created_at": "2024-01-01T00:00:00Z",
				"message":    map[string]string{"role": "assistant", "content": "Hello"},
				"done":       false,
			},
			{
				"model":      "llama2",
				"created_at": "2024-01-01T00:00:00Z",
				"message":    map[string]string{"role": "assistant", "content": "!"},
				"done":       false,
			},
			{
				"model":             "llama2",
				"created_at":        "2024-01-01T00:00:00Z",
				"message":           map[string]string{"role": "assistant", "content": ""},
				"done":              true,
				"prompt_eval_count": 10,
				"eval_count":        8,
			},
		}

		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			w.Write(data)
			w.Write([]byte("\n"))
		}
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(&ProviderConfig{BaseURL: server.URL})

	req := &ChatRequest{
		Model:    "llama2",
		Messages: []Message{{Role: RoleUser, Content: "Hi"}},
	}

	var accumulated string
	var gotDone bool
	err := provider.ChatStream(context.Background(), req, func(resp *ChatResponse) error {
		if resp.Delta != nil {
			accumulated += resp.Delta.Content
		}
		if resp.FinishReason == "stop" {
			gotDone = true
		}
		return nil
	})

	if err != nil {
		t.Fatalf("ChatStream() error: %v", err)
	}

	if accumulated != "Hello!" {
		t.Errorf("Accumulated = %s, want Hello!", accumulated)
	}
	if !gotDone {
		t.Error("Expected done signal")
	}
}

func TestOllamaEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := map[string]interface{}{
			"embedding": []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(&ProviderConfig{BaseURL: server.URL})

	req := &EmbeddingRequest{
		Model: "nomic-embed-text",
		Input: []string{"Hello world"},
	}

	resp, err := provider.Embed(context.Background(), req)
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("Expected 1 embedding, got %d", len(resp.Data))
	}
	if len(resp.Data[0].Embedding) != 5 {
		t.Errorf("Expected 5 dimensions, got %d", len(resp.Data[0].Embedding))
	}
}

func TestOllamaListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := map[string]interface{}{
			"models": []map[string]interface{}{
				{
					"name":        "llama2:latest",
					"modified_at": "2024-01-01T00:00:00Z",
					"size":        3800000000,
					"details": map[string]interface{}{
						"family":         "llama",
						"parameter_size": "7B",
					},
				},
				{
					"name":        "nomic-embed-text:latest",
					"modified_at": "2024-01-01T00:00:00Z",
					"size":        274000000,
					"details": map[string]interface{}{
						"family":         "nomic",
						"parameter_size": "137M",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(&ProviderConfig{BaseURL: server.URL})

	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	// Check embed model is properly identified
	for _, m := range models {
		if m.ID == "nomic-embed-text:latest" {
			found := false
			for _, cap := range m.Capabilities {
				if cap == "embed" {
					found = true
				}
			}
			if !found {
				t.Error("Embedding model should have 'embed' capability")
			}
		}
	}
}

func TestOllamaHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"models": []map[string]interface{}{
				{"name": "llama2:latest"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(&ProviderConfig{BaseURL: server.URL})

	status, err := provider.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck() error: %v", err)
	}

	if !status.Available {
		t.Error("Expected Available to be true")
	}
}

func TestOllamaHealthCheckUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(&ProviderConfig{BaseURL: server.URL})

	status, err := provider.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck() should not error: %v", err)
	}

	if status.Available {
		t.Error("Expected Available to be false")
	}
}

func TestOllamaErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{
			name:       "404 model not found",
			statusCode: 404,
			body:       `{"error":"model 'nonexistent' not found"}`,
		},
		{
			name:       "400 bad request",
			statusCode: 400,
			body:       `{"error":"invalid request"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			provider, _ := NewOllamaProvider(&ProviderConfig{BaseURL: server.URL})

			_, err := provider.Chat(context.Background(), &ChatRequest{
				Model:    "nonexistent",
				Messages: []Message{{Role: RoleUser, Content: "Hi"}},
			})

			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestOllamaWithOptions(t *testing.T) {
	var receivedReq map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := map[string]interface{}{
			"model":   "llama2",
			"message": map[string]string{"role": "assistant", "content": "Hi"},
			"done":    true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(&ProviderConfig{BaseURL: server.URL})

	temp := 0.7
	topP := 0.9
	req := &ChatRequest{
		Model:       "llama2",
		Messages:    []Message{{Role: RoleUser, Content: "Hi"}},
		Temperature: &temp,
		TopP:        &topP,
		MaxTokens:   100,
		Stop:        []string{"\n"},
	}

	_, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	// Verify options were passed
	options, ok := receivedReq["options"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected options in request")
	}

	if options["temperature"] != 0.7 {
		t.Error("Temperature not set correctly")
	}
	if options["top_p"] != 0.9 {
		t.Error("TopP not set correctly")
	}
	if options["num_predict"] != float64(100) {
		t.Error("MaxTokens (num_predict) not set correctly")
	}
}

func TestOllamaClose(t *testing.T) {
	provider, _ := NewOllamaProvider(nil)
	if err := provider.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}
