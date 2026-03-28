package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/ai"
	"hearth/internal/ai/providers"
)

// mockAIRepo implements ai.AIRepository for handler tests
type mockAIRepo struct {
	providers   map[uuid.UUID]*ai.AIProviderConfig
	credentials map[string]*ai.UserAICredential
	routings    map[uuid.UUID]*ai.ModelRouting
}

func newMockAIRepo() *mockAIRepo {
	return &mockAIRepo{
		providers:   make(map[uuid.UUID]*ai.AIProviderConfig),
		credentials: make(map[string]*ai.UserAICredential),
		routings:    make(map[uuid.UUID]*ai.ModelRouting),
	}
}

func (m *mockAIRepo) CreateProviderConfig(_ context.Context, config *ai.AIProviderConfig) error {
	m.providers[config.ID] = config
	return nil
}
func (m *mockAIRepo) GetProviderConfig(_ context.Context, id uuid.UUID) (*ai.AIProviderConfig, error) {
	if c, ok := m.providers[id]; ok {
		return c, nil
	}
	return nil, ai.ErrProviderNotFound
}
func (m *mockAIRepo) GetProviderConfigByName(_ context.Context, name string) (*ai.AIProviderConfig, error) {
	for _, c := range m.providers {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, ai.ErrProviderNotFound
}
func (m *mockAIRepo) GetAllProviderConfigs(_ context.Context) ([]*ai.AIProviderConfig, error) {
	out := make([]*ai.AIProviderConfig, 0, len(m.providers))
	for _, c := range m.providers {
		out = append(out, c)
	}
	return out, nil
}
func (m *mockAIRepo) GetEnabledProviderConfigs(_ context.Context) ([]*ai.AIProviderConfig, error) {
	out := make([]*ai.AIProviderConfig, 0)
	for _, c := range m.providers {
		if c.IsEnabled {
			out = append(out, c)
		}
	}
	return out, nil
}
func (m *mockAIRepo) GetDefaultProvider(_ context.Context) (*ai.AIProviderConfig, error) {
	for _, c := range m.providers {
		if c.IsDefault && c.IsEnabled {
			return c, nil
		}
	}
	return nil, ai.ErrProviderNotFound
}
func (m *mockAIRepo) UpdateProviderConfig(_ context.Context, config *ai.AIProviderConfig) error {
	m.providers[config.ID] = config
	return nil
}
func (m *mockAIRepo) DeleteProviderConfig(_ context.Context, id uuid.UUID) error {
	delete(m.providers, id)
	return nil
}
func (m *mockAIRepo) CreateUserCredential(_ context.Context, cred *ai.UserAICredential) error {
	key := cred.UserID.String() + "-" + cred.ProviderID.String()
	m.credentials[key] = cred
	return nil
}
func (m *mockAIRepo) GetUserCredential(_ context.Context, userID, providerID uuid.UUID) (*ai.UserAICredential, error) {
	key := userID.String() + "-" + providerID.String()
	if c, ok := m.credentials[key]; ok {
		return c, nil
	}
	return nil, ai.ErrCredentialsNotFound
}
func (m *mockAIRepo) GetUserCredentials(_ context.Context, userID uuid.UUID) ([]*ai.UserAICredential, error) {
	out := make([]*ai.UserAICredential, 0)
	prefix := userID.String() + "-"
	for key, c := range m.credentials {
		if strings.HasPrefix(key, prefix) {
			out = append(out, c)
		}
	}
	return out, nil
}
func (m *mockAIRepo) UpdateUserCredential(_ context.Context, cred *ai.UserAICredential) error {
	key := cred.UserID.String() + "-" + cred.ProviderID.String()
	m.credentials[key] = cred
	return nil
}
func (m *mockAIRepo) DeleteUserCredential(_ context.Context, id uuid.UUID) error {
	for key, c := range m.credentials {
		if c.ID == id {
			delete(m.credentials, key)
			return nil
		}
	}
	return ai.ErrCredentialsNotFound
}
func (m *mockAIRepo) UpdateLastUsed(_ context.Context, id uuid.UUID) error { return nil }
func (m *mockAIRepo) CreateModelRouting(_ context.Context, routing *ai.ModelRouting) error {
	m.routings[routing.ID] = routing
	return nil
}
func (m *mockAIRepo) GetModelRouting(_ context.Context, feature ai.FeatureType, serverID, userID *uuid.UUID) (*ai.ModelRouting, error) {
	for _, r := range m.routings {
		if r.Feature == feature {
			return r, nil
		}
	}
	return nil, ai.ErrNoProviderAvailable
}
func (m *mockAIRepo) GetAllModelRoutings(_ context.Context) ([]*ai.ModelRouting, error) {
	out := make([]*ai.ModelRouting, 0, len(m.routings))
	for _, r := range m.routings {
		out = append(out, r)
	}
	return out, nil
}
func (m *mockAIRepo) UpdateModelRouting(_ context.Context, routing *ai.ModelRouting) error {
	m.routings[routing.ID] = routing
	return nil
}
func (m *mockAIRepo) DeleteModelRouting(_ context.Context, id uuid.UUID) error {
	delete(m.routings, id)
	return nil
}

// setupAITestApp creates a fiber app with an AIHandler for testing
func setupAITestApp(t testing.TB, repo *mockAIRepo) (*fiber.App, *AIHandler) {
	encryption := ai.NewNoOpEncryptionService()
	service := ai.NewAIService(repo, encryption)
	handler := NewAIHandler(service)

	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-Test-User-ID")
		if userID != "" {
			uid, _ := uuid.Parse(userID)
			c.Locals("userID", uid)
		}
		return c.Next()
	})

	return app, handler
}

// addTestProvider adds a provider to the mock repo and returns its ID
func addTestProvider(repo *mockAIRepo) uuid.UUID {
	id := uuid.New()
	repo.providers[id] = &ai.AIProviderConfig{
		ID:           id,
		ProviderType: ai.ProviderOpenAI,
		Name:         "test-openai",
		DisplayName:  "Test OpenAI",
		IsEnabled:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return id
}

// readJSONResponse reads and parses JSON from an HTTP response
func readJSONResponse(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response JSON: %v (body: %s)", err, string(body))
	}
	return result
}

func TestAIHandlerCreation(t *testing.T) {
	encryption := ai.NewNoOpEncryptionService()
	// Create with nil repo for basic test
	service := ai.NewAIService(nil, encryption)

	handler := NewAIHandler(service)
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
	if handler.aiService != service {
		t.Error("Handler should have the provided service")
	}
}

func TestCreateProviderRequestParsing(t *testing.T) {
	tests := []struct {
		name    string
		body    CreateProviderRequest
		wantErr bool
	}{
		{
			name: "valid OpenAI request",
			body: CreateProviderRequest{
				ProviderType: "openai",
				Name:         "openai-main",
				DisplayName:  "OpenAI GPT-4",
				IsEnabled:    true,
				APIKey:       "sk-test",
			},
			wantErr: false,
		},
		{
			name: "valid Ollama request (no key)",
			body: CreateProviderRequest{
				ProviderType: "ollama",
				Name:         "ollama-local",
				DisplayName:  "Local Ollama",
				IsEnabled:    true,
			},
			wantErr: false,
		},
		{
			name: "valid Bedrock request",
			body: CreateProviderRequest{
				ProviderType: "bedrock",
				Name:         "aws-bedrock",
				DisplayName:  "AWS Bedrock",
				IsEnabled:    true,
				APIKey:       "AKIAIOSFODNN7EXAMPLE",
				SecretKey:    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				Region:       "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "valid Anthropic request",
			body: CreateProviderRequest{
				ProviderType: "anthropic",
				Name:         "anthropic-main",
				DisplayName:  "Anthropic Claude",
				IsEnabled:    true,
				APIKey:       "sk-ant-test",
			},
			wantErr: false,
		},
		{
			name: "valid LM Studio request",
			body: CreateProviderRequest{
				ProviderType: "lm_studio",
				Name:         "lm-studio-local",
				DisplayName:  "LM Studio",
				IsEnabled:    true,
			},
			wantErr: false,
		},
		{
			name: "valid llama.cpp request",
			body: CreateProviderRequest{
				ProviderType: "llama_cpp",
				Name:         "llamacpp-local",
				DisplayName:  "llama.cpp Server",
				IsEnabled:    true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			var parsed CreateProviderRequest
			err = json.Unmarshal(jsonBody, &parsed)
			if err != nil && !tt.wantErr {
				t.Errorf("Failed to parse request: %v", err)
			}

			if parsed.ProviderType != tt.body.ProviderType {
				t.Errorf("ProviderType mismatch: got %s, want %s", parsed.ProviderType, tt.body.ProviderType)
			}
		})
	}
}

func TestSetUserCredentialsRequestParsing(t *testing.T) {
	body := SetUserCredentialsRequest{
		ProviderID: "550e8400-e29b-41d4-a716-446655440000",
		APIKey:     "sk-user-key",
		Region:     "us-west-2",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed SetUserCredentialsRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.ProviderID != body.ProviderID {
		t.Errorf("ProviderID mismatch")
	}
	if parsed.APIKey != body.APIKey {
		t.Errorf("APIKey mismatch")
	}
}

func TestSetModelRoutingRequestParsing(t *testing.T) {
	serverID := "550e8400-e29b-41d4-a716-446655440001"
	body := SetModelRoutingRequest{
		ServerID:   &serverID,
		Feature:    "chat",
		ProviderID: "550e8400-e29b-41d4-a716-446655440000",
		ModelID:    "gpt-4",
		Priority:   1,
		IsEnabled:  true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed SetModelRoutingRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.Feature != body.Feature {
		t.Errorf("Feature mismatch")
	}
	if parsed.ModelID != body.ModelID {
		t.Errorf("ModelID mismatch")
	}
}

func TestSetModelRoutingRequestWithUserID(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440002"
	body := SetModelRoutingRequest{
		UserID:     &userID,
		Feature:    "summary",
		ProviderID: "550e8400-e29b-41d4-a716-446655440000",
		ModelID:    "gpt-3.5-turbo",
		Priority:   2,
		IsEnabled:  true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed SetModelRoutingRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.UserID == nil {
		t.Error("UserID should not be nil")
	}
	if *parsed.UserID != userID {
		t.Errorf("UserID mismatch: got %s, want %s", *parsed.UserID, userID)
	}
}

func TestChatCompletionRequestParsing(t *testing.T) {
	temp := 0.7
	body := ChatCompletionRequest{
		Model:   "gpt-4",
		Feature: "chat",
		Messages: []providers.Message{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello!"},
		},
		MaxTokens:   1000,
		Temperature: &temp,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed ChatCompletionRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.Model != body.Model {
		t.Errorf("Model mismatch")
	}
	if len(parsed.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(parsed.Messages))
	}
}

func TestChatCompletionRequestWithServerID(t *testing.T) {
	serverID := "550e8400-e29b-41d4-a716-446655440003"
	body := ChatCompletionRequest{
		Model:    "claude-3-sonnet-20240229",
		Feature:  "search",
		ServerID: &serverID,
		Messages: []providers.Message{
			{Role: "user", Content: "Search query"},
		},
		MaxTokens: 500,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed ChatCompletionRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.ServerID == nil {
		t.Error("ServerID should not be nil")
	}
	if *parsed.ServerID != serverID {
		t.Errorf("ServerID mismatch")
	}
}

func TestChatCompletionRequestStreaming(t *testing.T) {
	body := ChatCompletionRequest{
		Model: "gpt-4-turbo",
		Messages: []providers.Message{
			{Role: "user", Content: "Stream this"},
		},
		Stream: true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed ChatCompletionRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !parsed.Stream {
		t.Error("Stream should be true")
	}
}

func TestEmbeddingRequestParsing(t *testing.T) {
	body := EmbeddingRequestBody{
		Model: "text-embedding-3-small",
		Input: []string{"Hello world", "How are you?"},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed EmbeddingRequestBody
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(parsed.Input) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(parsed.Input))
	}
}

func TestEmbeddingRequestWithServerID(t *testing.T) {
	serverID := "550e8400-e29b-41d4-a716-446655440004"
	body := EmbeddingRequestBody{
		Model:    "nomic-embed-text",
		Input:    []string{"Text to embed"},
		ServerID: &serverID,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed EmbeddingRequestBody
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.ServerID == nil {
		t.Error("ServerID should not be nil")
	}
}

func TestMessageTypes(t *testing.T) {
	messages := []providers.Message{
		{Role: providers.RoleSystem, Content: "System message"},
		{Role: providers.RoleUser, Content: "User message"},
		{Role: providers.RoleAssistant, Content: "Assistant message"},
		{Role: providers.RoleTool, Content: "Tool result", ToolCallID: "call_123"},
	}

	for i, msg := range messages {
		if msg.Content == "" {
			t.Errorf("Message %d content should not be empty", i)
		}
	}

	// Check tool message has tool call ID
	if messages[3].ToolCallID == "" {
		t.Error("Tool message should have ToolCallID")
	}
}

func TestToolCallsParsing(t *testing.T) {
	msg := providers.Message{
		Role:    providers.RoleAssistant,
		Content: "",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_abc123",
				Type: "function",
				Function: providers.FunctionCall{
					Name:      "get_weather",
					Arguments: `{"location": "San Francisco"}`,
				},
			},
		},
	}

	jsonBody, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed providers.Message
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(parsed.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(parsed.ToolCalls))
	}
	if parsed.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("Function name mismatch")
	}
}

func TestProviderTypeValidation(t *testing.T) {
	validTypes := []string{
		"openai", "anthropic", "openrouter", "bedrock", "vertex_ai",
		"ollama", "llama_cpp", "lm_studio", "vllm", "local_ai",
	}

	for _, pt := range validTypes {
		providerType := ai.ProviderType(pt)
		if !providerType.Valid() {
			t.Errorf("Provider type %s should be valid", pt)
		}
	}

	invalidType := ai.ProviderType("invalid")
	if invalidType.Valid() {
		t.Error("Invalid provider type should not be valid")
	}
}

func TestFeatureTypeValidation(t *testing.T) {
	validFeatures := []string{
		"summary", "search", "chat", "embed", "moderate", "translate",
	}

	for _, ft := range validFeatures {
		featureType := ai.FeatureType(ft)
		if !featureType.Valid() {
			t.Errorf("Feature type %s should be valid", ft)
		}
	}

	invalidFeature := ai.FeatureType("invalid")
	if invalidFeature.Valid() {
		t.Error("Invalid feature type should not be valid")
	}
}

func TestAllFeatureTypes(t *testing.T) {
	features := ai.AllFeatureTypes()
	if len(features) == 0 {
		t.Error("Expected feature types, got empty list")
	}

	expected := map[ai.FeatureType]bool{
		ai.FeatureSummary:   true,
		ai.FeatureSearch:    true,
		ai.FeatureChat:      true,
		ai.FeatureEmbed:     true,
		ai.FeatureModerate:  true,
		ai.FeatureTranslate: true,
	}

	for _, f := range features {
		if !expected[f] {
			t.Errorf("Unexpected feature type: %s", f)
		}
	}
}

func TestAllProviderTypes(t *testing.T) {
	providers := ai.AllProviderTypes()
	if len(providers) == 0 {
		t.Error("Expected provider types, got empty list")
	}

	// Check cloud providers
	cloudCount := 0
	for _, p := range providers {
		if p.IsCloud() {
			cloudCount++
		}
	}

	if cloudCount == 0 {
		t.Error("Expected some cloud providers")
	}

	// Check local providers
	localCount := 0
	for _, p := range providers {
		if p.IsLocal() {
			localCount++
		}
	}

	if localCount == 0 {
		t.Error("Expected some local providers")
	}
}

// Helper to create test request
func createTestRequest(method, path string, body interface{}) *http.Request {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBody)
		req := httptest.NewRequest(method, path, bodyReader)
		req.Header.Set("Content-Type", "application/json")
		return req
	}
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestCreateTestRequest(t *testing.T) {
	body := map[string]string{"key": "value"}
	req := createTestRequest("POST", "/test", body)

	if req.Method != "POST" {
		t.Errorf("Method = %s, want POST", req.Method)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type header should be application/json")
	}
}

func TestCreateTestRequestNilBody(t *testing.T) {
	req := createTestRequest("GET", "/test", nil)

	if req.Method != "GET" {
		t.Errorf("Method = %s, want GET", req.Method)
	}
}

// --- New endpoint tests ---

func TestUserAISettingsResponseParsing(t *testing.T) {
	resp := UserAISettingsResponse{
		DefaultModel: "gpt-4",
		Credentials:  []*ai.UserAICredentialResponse{},
		Routings:     []*ai.ModelRouting{},
	}

	jsonBody, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed UserAISettingsResponse
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.DefaultModel != resp.DefaultModel {
		t.Errorf("DefaultModel mismatch: got %s, want %s", parsed.DefaultModel, resp.DefaultModel)
	}
}

func TestUpdateUserSettingsRequestParsing(t *testing.T) {
	defaultProvider := "550e8400-e29b-41d4-a716-446655440000"
	body := UpdateUserSettingsRequest{
		DefaultProvider: &defaultProvider,
		DefaultModel:    "gpt-4-turbo",
		ProviderCredentials: &SetUserCredentialsRequest{
			ProviderID: "550e8400-e29b-41d4-a716-446655440000",
			APIKey:     "sk-test-key",
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed UpdateUserSettingsRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.DefaultModel != body.DefaultModel {
		t.Errorf("DefaultModel mismatch")
	}
	if parsed.ProviderCredentials == nil {
		t.Error("ProviderCredentials should not be nil")
	}
}

func TestUpdateUserSettingsWithFeatureRouting(t *testing.T) {
	body := UpdateUserSettingsRequest{
		FeatureRouting: &SetModelRoutingRequest{
			Feature:    "chat",
			ProviderID: "550e8400-e29b-41d4-a716-446655440000",
			ModelID:    "gpt-4",
			Priority:   1,
			IsEnabled:  true,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed UpdateUserSettingsRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.FeatureRouting == nil {
		t.Fatal("FeatureRouting should not be nil")
	}
	if parsed.FeatureRouting.Feature != "chat" {
		t.Errorf("Feature mismatch: got %s, want chat", parsed.FeatureRouting.Feature)
	}
}

func TestAdminAIDefaultsRequestParsing(t *testing.T) {
	body := AdminAIDefaultsRequest{
		DefaultProvider:   "openai",
		AllowUserOverride: true,
		FeatureDefaults: map[string]FeatureDefaultConfig{
			"chat": {
				ProviderID: "550e8400-e29b-41d4-a716-446655440000",
				ModelID:    "gpt-4",
				Enabled:    true,
			},
			"summary": {
				ProviderID: "550e8400-e29b-41d4-a716-446655440000",
				ModelID:    "gpt-3.5-turbo",
				Enabled:    true,
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed AdminAIDefaultsRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.DefaultProvider != body.DefaultProvider {
		t.Errorf("DefaultProvider mismatch")
	}
	if !parsed.AllowUserOverride {
		t.Error("AllowUserOverride should be true")
	}
	if len(parsed.FeatureDefaults) != 2 {
		t.Errorf("Expected 2 feature defaults, got %d", len(parsed.FeatureDefaults))
	}
}

func TestFeatureDefaultConfigParsing(t *testing.T) {
	config := FeatureDefaultConfig{
		ProviderID: "550e8400-e29b-41d4-a716-446655440000",
		ModelID:    "claude-3-opus-20240229",
		Enabled:    true,
	}

	jsonBody, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed FeatureDefaultConfig
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.ProviderID != config.ProviderID {
		t.Errorf("ProviderID mismatch")
	}
	if parsed.ModelID != config.ModelID {
		t.Errorf("ModelID mismatch")
	}
	if !parsed.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestAdminConfigFeatureDefaults(t *testing.T) {
	adminConfig := &ai.AdminAIConfig{
		DefaultProvider:   "openai",
		AllowUserOverride: true,
		FeatureDefaults:   make(map[ai.FeatureType]ai.FeatureConfig),
	}

	// Add feature defaults for all valid features
	for _, feature := range ai.AllFeatureTypes() {
		adminConfig.FeatureDefaults[feature] = ai.FeatureConfig{
			ProviderType: ai.ProviderOpenAI,
			ModelID:      "gpt-4",
			Enabled:      true,
		}
	}

	// Verify all features have defaults
	for _, feature := range ai.AllFeatureTypes() {
		cfg, ok := adminConfig.FeatureDefaults[feature]
		if !ok {
			t.Errorf("Missing default for feature: %s", feature)
		}
		if !cfg.Enabled {
			t.Errorf("Feature %s should be enabled", feature)
		}
	}
}

func TestProviderTypesCloudVsLocal(t *testing.T) {
	cloudProviders := []ai.ProviderType{
		ai.ProviderOpenRouter,
		ai.ProviderBedrock,
		ai.ProviderVertexAI,
		ai.ProviderOpenAI,
		ai.ProviderAnthropic,
	}

	localProviders := []ai.ProviderType{
		ai.ProviderOllama,
		ai.ProviderLlamaCpp,
		ai.ProviderLMStudio,
		ai.ProviderVLLM,
		ai.ProviderLocalAI,
	}

	for _, p := range cloudProviders {
		if !p.IsCloud() {
			t.Errorf("%s should be a cloud provider", p)
		}
		if p.IsLocal() {
			t.Errorf("%s should not be a local provider", p)
		}
	}

	for _, p := range localProviders {
		if p.IsCloud() {
			t.Errorf("%s should not be a cloud provider", p)
		}
		if !p.IsLocal() {
			t.Errorf("%s should be a local provider", p)
		}
	}
}

func TestDefaultModelsForFeatures(t *testing.T) {
	defaults := ai.DefaultModels()

	// Check all features have defaults
	for _, feature := range ai.AllFeatureTypes() {
		model, ok := defaults[feature]
		if !ok {
			t.Errorf("Missing default model for feature: %s", feature)
		}
		if model == "" {
			t.Errorf("Default model for feature %s should not be empty", feature)
		}
	}
}

func TestMultipleFeatureRoutingConfig(t *testing.T) {
	// Test setting up different models for different features
	routingConfigs := map[ai.FeatureType]string{
		ai.FeatureSummary: "gpt-3.5-turbo", // Cheap for summaries
		ai.FeatureSearch:  "gpt-4",         // Smart for search
		ai.FeatureChat:    "gpt-4-turbo",   // Balanced for chat
		ai.FeatureEmbed:   "text-embedding-3-small",
	}

	for feature, model := range routingConfigs {
		if !feature.Valid() {
			t.Errorf("Feature %s should be valid", feature)
		}
		if model == "" {
			t.Errorf("Model for feature %s should not be empty", feature)
		}
	}
}

func TestUserCredentialResponseMasksCredentials(t *testing.T) {
	// Simulate creating a credential and converting to response
	cred := &ai.UserAICredential{
		Credentials: "encrypted-api-key-should-not-appear",
		IsEnabled:   true,
	}

	response := cred.ToResponse()

	// The response should indicate credentials exist but not expose them
	if response.HasCredentials != true {
		t.Error("HasCredentials should be true when credentials are set")
	}

	// Verify the raw Credentials field doesn't appear in JSON
	jsonBody, _ := json.Marshal(response)
	jsonStr := string(jsonBody)
	if strings.Contains(jsonStr, "encrypted-api-key") {
		t.Error("Response should not contain raw credential data")
	}
}

func TestProviderConfigResponseMasksConfig(t *testing.T) {
	secretConfig := "encrypted-config-data"
	config := &ai.AIProviderConfig{
		Name:        "test-provider",
		DisplayName: "Test Provider",
		Config:      &secretConfig,
		IsEnabled:   true,
	}

	response := config.ToResponse()

	// Verify the Config field doesn't appear in response
	jsonBody, _ := json.Marshal(response)
	jsonStr := string(jsonBody)
	if strings.Contains(jsonStr, "encrypted-config-data") {
		t.Error("Response should not contain encrypted config data")
	}
}

// --- Error Response Helper Tests ---

func TestRespondBadRequest(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)

	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.respondBadRequest(c, "test bad request")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	result := readJSONResponse(t, resp)
	if result["error"] != "invalid_request" {
		t.Errorf("Expected error 'invalid_request', got %v", result["error"])
	}
	if result["message"] != "test bad request" {
		t.Errorf("Expected message 'test bad request', got %v", result["message"])
	}
}

func TestRespondUnauthorized(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)

	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.respondUnauthorized(c, "not authenticated")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}

	result := readJSONResponse(t, resp)
	if result["error"] != "unauthorized" {
		t.Errorf("Expected error 'unauthorized', got %v", result["error"])
	}
	if result["message"] != "not authenticated" {
		t.Errorf("Expected message 'not authenticated', got %v", result["message"])
	}
}

func TestRespondNotFound(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)

	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.respondNotFound(c, "resource missing")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	result := readJSONResponse(t, resp)
	if result["error"] != "not_found" {
		t.Errorf("Expected error 'not_found', got %v", result["error"])
	}
	if result["message"] != "resource missing" {
		t.Errorf("Expected message 'resource missing', got %v", result["message"])
	}
}

func TestRespondInternalError(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)

	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.respondInternalError(c, "something broke")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	result := readJSONResponse(t, resp)
	if result["error"] != "internal_error" {
		t.Errorf("Expected error 'internal_error', got %v", result["error"])
	}
	if result["message"] != "something broke" {
		t.Errorf("Expected message 'something broke', got %v", result["message"])
	}
}

// --- getUserID Tests ---

func TestGetUserID_Success(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)
	expectedUID := uuid.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		uid, err := handler.getUserID(c)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"user_id": uid.String()})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Test-User-ID", expectedUID.String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	result := readJSONResponse(t, resp)
	if result["user_id"] != expectedUID.String() {
		t.Errorf("Expected user_id %s, got %v", expectedUID.String(), result["user_id"])
	}
}

func TestGetUserID_Missing(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)

	app.Get("/test", func(c *fiber.Ctx) error {
		_, err := handler.getUserID(c)
		if err == nil {
			t.Error("Expected error when userID is missing")
			return c.SendStatus(200)
		}
		if !strings.Contains(err.Error(), "user ID not found") {
			t.Errorf("Expected 'user ID not found' error, got: %v", err)
		}
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No X-Test-User-ID header
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestGetUserID_InvalidType(t *testing.T) {
	repo := newMockAIRepo()
	encryption := ai.NewNoOpEncryptionService()
	service := ai.NewAIService(repo, encryption)
	handler := NewAIHandler(service)

	app := fiber.New()
	t.Cleanup(func() { _ = app.Shutdown() })

	// Set Locals to a string instead of uuid.UUID
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("userID", "not-a-uuid-type")
		_, err := handler.getUserID(c)
		if err == nil {
			return c.JSON(fiber.Map{"error": "expected error"})
		}
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

// --- updateUserCredentials Tests ---

func TestUpdateUserCredentials_Success(t *testing.T) {
	repo := newMockAIRepo()
	providerID := addTestProvider(repo)
	app, handler := setupAITestApp(t, repo)
	userID := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		uid, _ := uuid.Parse(c.Get("X-Test-User-ID"))
		req := &SetUserCredentialsRequest{
			ProviderID: providerID.String(),
			APIKey:     "sk-test-key",
			Region:     "us-east-1",
		}
		if err := handler.updateUserCredentials(c, uid, req); err != nil {
			return err
		}
		return c.JSON(fiber.Map{"message": "ok"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		result := readJSONResponse(t, resp)
		t.Fatalf("Expected status 200, got %d: %v", resp.StatusCode, result)
	}

	// Verify credential was stored
	key := userID.String() + "-" + providerID.String()
	if _, ok := repo.credentials[key]; !ok {
		t.Error("Credential should have been stored in the repo")
	}
}

func TestUpdateUserCredentials_InvalidProviderID(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)
	userID := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		uid, _ := uuid.Parse(c.Get("X-Test-User-ID"))
		req := &SetUserCredentialsRequest{
			ProviderID: "not-a-valid-uuid",
			APIKey:     "sk-test-key",
		}
		return handler.updateUserCredentials(c, uid, req)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	result := readJSONResponse(t, resp)
	if result["message"] != "Invalid provider ID" {
		t.Errorf("Expected 'Invalid provider ID' message, got %v", result["message"])
	}
}

func TestUpdateUserCredentials_ProviderNotFound(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)
	userID := uuid.New()
	nonExistentProvider := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		uid, _ := uuid.Parse(c.Get("X-Test-User-ID"))
		req := &SetUserCredentialsRequest{
			ProviderID: nonExistentProvider.String(),
			APIKey:     "sk-test-key",
		}
		return handler.updateUserCredentials(c, uid, req)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status 500 (service returns provider not found as internal error), got %d", resp.StatusCode)
	}
}

// --- updateFeatureRouting Tests ---

func TestUpdateFeatureRouting_Success(t *testing.T) {
	repo := newMockAIRepo()
	providerID := addTestProvider(repo)
	app, handler := setupAITestApp(t, repo)
	userID := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		uid, _ := uuid.Parse(c.Get("X-Test-User-ID"))
		req := &SetModelRoutingRequest{
			Feature:    "chat",
			ProviderID: providerID.String(),
			ModelID:    "gpt-4",
			Priority:   1,
			IsEnabled:  true,
		}
		if err := handler.updateFeatureRouting(c, uid, req); err != nil {
			return err
		}
		return c.JSON(fiber.Map{"message": "ok"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		result := readJSONResponse(t, resp)
		t.Fatalf("Expected status 200, got %d: %v", resp.StatusCode, result)
	}

	// Verify routing was stored
	if len(repo.routings) != 1 {
		t.Errorf("Expected 1 routing, got %d", len(repo.routings))
	}
	for _, r := range repo.routings {
		if r.Feature != ai.FeatureChat {
			t.Errorf("Expected feature 'chat', got %s", r.Feature)
		}
		if r.ProviderID != providerID {
			t.Errorf("Expected provider ID %s, got %s", providerID, r.ProviderID)
		}
		if r.UserID == nil || *r.UserID != userID {
			t.Error("Routing should have the user ID set")
		}
		if r.ModelID != "gpt-4" {
			t.Errorf("Expected model 'gpt-4', got %s", r.ModelID)
		}
	}
}

func TestUpdateFeatureRouting_InvalidFeatureType(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)
	userID := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		uid, _ := uuid.Parse(c.Get("X-Test-User-ID"))
		req := &SetModelRoutingRequest{
			Feature:    "invalid_feature",
			ProviderID: uuid.New().String(),
			ModelID:    "gpt-4",
		}
		return handler.updateFeatureRouting(c, uid, req)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	result := readJSONResponse(t, resp)
	if result["message"] != "Invalid feature type" {
		t.Errorf("Expected 'Invalid feature type' message, got %v", result["message"])
	}
}

func TestUpdateFeatureRouting_InvalidProviderID(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)
	userID := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		uid, _ := uuid.Parse(c.Get("X-Test-User-ID"))
		req := &SetModelRoutingRequest{
			Feature:    "chat",
			ProviderID: "not-a-uuid",
			ModelID:    "gpt-4",
		}
		return handler.updateFeatureRouting(c, uid, req)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	result := readJSONResponse(t, resp)
	if result["message"] != "Invalid provider ID" {
		t.Errorf("Expected 'Invalid provider ID' message, got %v", result["message"])
	}
}

func TestUpdateFeatureRouting_ProviderNotFound(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)
	userID := uuid.New()
	nonExistentProvider := uuid.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		uid, _ := uuid.Parse(c.Get("X-Test-User-ID"))
		req := &SetModelRoutingRequest{
			Feature:    "chat",
			ProviderID: nonExistentProvider.String(),
			ModelID:    "gpt-4",
			Priority:   1,
			IsEnabled:  true,
		}
		return handler.updateFeatureRouting(c, uid, req)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Test-User-ID", userID.String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

// --- Error Response Helpers with Various Messages ---

func TestRespondHelpers_EmptyMessage(t *testing.T) {
	repo := newMockAIRepo()
	app, handler := setupAITestApp(t, repo)

	tests := []struct {
		name           string
		path           string
		respondFn      func(*AIHandler, *fiber.Ctx) error
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "bad request empty message",
			path:           "/bad",
			respondFn:      func(h *AIHandler, c *fiber.Ctx) error { return h.respondBadRequest(c, "") },
			expectedStatus: 400,
			expectedError:  "invalid_request",
		},
		{
			name:           "unauthorized empty message",
			path:           "/unauth",
			respondFn:      func(h *AIHandler, c *fiber.Ctx) error { return h.respondUnauthorized(c, "") },
			expectedStatus: 401,
			expectedError:  "unauthorized",
		},
		{
			name:           "not found empty message",
			path:           "/notfound",
			respondFn:      func(h *AIHandler, c *fiber.Ctx) error { return h.respondNotFound(c, "") },
			expectedStatus: 404,
			expectedError:  "not_found",
		},
		{
			name:           "internal error empty message",
			path:           "/internal",
			respondFn:      func(h *AIHandler, c *fiber.Ctx) error { return h.respondInternalError(c, "") },
			expectedStatus: 500,
			expectedError:  "internal_error",
		},
	}

	for _, tt := range tests {
		app.Get(tt.path, func(c *fiber.Ctx) error {
			// Capture the test case values
			for _, tc := range tests {
				if c.Path() == tc.path {
					return tc.respondFn(handler, c)
				}
			}
			return nil
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			result := readJSONResponse(t, resp)
			if result["error"] != tt.expectedError {
				t.Errorf("Expected error %q, got %v", tt.expectedError, result["error"])
			}
			if result["message"] != "" {
				t.Errorf("Expected empty message, got %v", result["message"])
			}
		})
	}
}

func TestUpdateFeatureRouting_AllValidFeatureTypes(t *testing.T) {
	validFeatures := []string{"summary", "search", "chat", "embed", "moderate", "translate"}

	for _, feature := range validFeatures {
		t.Run(fmt.Sprintf("feature_%s", feature), func(t *testing.T) {
			repo := newMockAIRepo()
			providerID := addTestProvider(repo)
			app, handler := setupAITestApp(t, repo)
			userID := uuid.New()

			app.Post("/test", func(c *fiber.Ctx) error {
				uid, _ := uuid.Parse(c.Get("X-Test-User-ID"))
				req := &SetModelRoutingRequest{
					Feature:    feature,
					ProviderID: providerID.String(),
					ModelID:    "test-model",
					Priority:   1,
					IsEnabled:  true,
				}
				if err := handler.updateFeatureRouting(c, uid, req); err != nil {
					return err
				}
				return c.JSON(fiber.Map{"message": "ok"})
			})

			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set("X-Test-User-ID", userID.String())
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp.StatusCode != fiber.StatusOK {
				t.Errorf("Expected status 200 for feature %s, got %d", feature, resp.StatusCode)
			}
		})
	}
}
