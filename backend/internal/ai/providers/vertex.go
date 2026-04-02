package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// VertexAIProvider implements the Provider interface for Google Vertex AI
type VertexAIProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	projectID  string
	location   string
}

// NewVertexAIProvider creates a new Vertex AI provider
func NewVertexAIProvider(config *ProviderConfig) (*VertexAIProvider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	projectID := config.ProjectID
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required for Vertex AI")
	}

	location := config.Region
	if location == "" {
		location = "us-central1"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	provider := &VertexAIProvider{
		config:     config,
		httpClient: &http.Client{Timeout: timeout},
		projectID:  projectID,
		location:   location,
	}

	return provider, nil
}

// tokenCache caches the access token and expiry
type tokenCache struct {
	mu     sync.RWMutex
	token  string
	expiry time.Time
}

var vertexTokenCache = &tokenCache{}

// getAccessToken returns a valid access token, refreshing if necessary
func (p *VertexAIProvider) getAccessToken() (string, error) {
	// Check for explicit access token in config
	if p.config.APIKey != "" {
		return p.config.APIKey, nil
	}

	// Check cache
	vertexTokenCache.mu.RLock()
	if vertexTokenCache.token != "" && time.Now().Before(vertexTokenCache.expiry) {
		token := vertexTokenCache.token
		vertexTokenCache.mu.RUnlock()
		return token, nil
	}
	vertexTokenCache.mu.RUnlock()

	// Try to get token from gcloud CLI (works with ADC)
	vertexTokenCache.mu.Lock()
	defer vertexTokenCache.mu.Unlock()

	// Double-check after acquiring write lock
	if vertexTokenCache.token != "" && time.Now().Before(vertexTokenCache.expiry) {
		return vertexTokenCache.token, nil
	}

	cmd := exec.Command("gcloud", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w (ensure gcloud is installed and authenticated)", err)
	}

	token := strings.TrimSpace(string(output))
	vertexTokenCache.token = token
	vertexTokenCache.expiry = time.Now().Add(50 * time.Minute) // Tokens typically last 1 hour

	return token, nil
}

func (p *VertexAIProvider) Name() string { return "Google Vertex AI" }
func (p *VertexAIProvider) Type() string { return "vertex_ai" }

// vertexRequest is the Vertex AI Gemini request format
type vertexRequest struct {
	Contents          []vertexContent         `json:"contents"`
	SystemInstruction *vertexContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *vertexGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings    []vertexSafetySetting   `json:"safetySettings,omitempty"`
}

type vertexContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []vertexPart `json:"parts"`
}

type vertexPart struct {
	Text string `json:"text,omitempty"`
}

type vertexGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type vertexSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type vertexResponse struct {
	Candidates []struct {
		Content struct {
			Role  string       `json:"role"`
			Parts []vertexPart `json:"parts"`
		} `json:"content"`
		FinishReason  string `json:"finishReason"`
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func (p *VertexAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	vertexReq := p.convertRequest(req)

	body, err := p.doRequest(ctx, req.Model, "generateContent", vertexReq)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp vertexResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertResponse(req.Model, &resp), nil
}

func (p *VertexAIProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	vertexReq := p.convertRequest(req)

	body, err := p.doRequest(ctx, req.Model, "streamGenerateContent?alt=sse", vertexReq)
	if err != nil {
		return err
	}
	defer body.Close()

	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var resp vertexResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if len(resp.Candidates) > 0 {
			var text strings.Builder
			for _, part := range resp.Candidates[0].Content.Parts {
				text.WriteString(part.Text)
			}

			chatResp := &ChatResponse{
				Delta: &Message{
					Role:    RoleAssistant,
					Content: text.String(),
				},
			}

			if resp.Candidates[0].FinishReason != "" {
				chatResp.FinishReason = p.mapFinishReason(resp.Candidates[0].FinishReason)
			}

			if err := callback(chatResp); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *VertexAIProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	// Vertex AI embedding request format
	embedReq := struct {
		Instances []struct {
			Content string `json:"content"`
		} `json:"instances"`
	}{}

	for _, text := range req.Input {
		embedReq.Instances = append(embedReq.Instances, struct {
			Content string `json:"content"`
		}{Content: text})
	}

	body, err := p.doRequest(ctx, req.Model, "predict", embedReq)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp struct {
		Predictions []struct {
			Embeddings struct {
				Values []float64 `json:"values"`
			} `json:"embeddings"`
		} `json:"predictions"`
	}
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embedResp := &EmbeddingResponse{
		Object: "list",
		Model:  req.Model,
	}

	for i, pred := range resp.Predictions {
		embedding := make([]float32, len(pred.Embeddings.Values))
		for j, v := range pred.Embeddings.Values {
			embedding[j] = float32(v)
		}
		embedResp.Data = append(embedResp.Data, Embedding{
			Object:    "embedding",
			Index:     i,
			Embedding: embedding,
		})
	}

	return embedResp, nil
}

func (p *VertexAIProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Return known Vertex AI models
	return []ModelInfo{
		{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", ContextWindow: 2097152, Capabilities: []string{"chat", "vision", "function_calling"}},
		{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", ContextWindow: 1048576, Capabilities: []string{"chat", "vision", "function_calling"}},
		{ID: "gemini-1.0-pro", Name: "Gemini 1.0 Pro", ContextWindow: 32768, Capabilities: []string{"chat", "function_calling"}},
		{ID: "gemini-1.0-pro-vision", Name: "Gemini 1.0 Pro Vision", ContextWindow: 16384, Capabilities: []string{"chat", "vision"}},
		{ID: "text-embedding-004", Name: "Text Embedding", Capabilities: []string{"embed"}},
		{ID: "textembedding-gecko@003", Name: "Gecko Embedding", Capabilities: []string{"embed"}},
	}, nil
}

func (p *VertexAIProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	status := &HealthStatus{CheckedAt: start}

	// Try a minimal request
	testReq := &ChatRequest{
		Model:     "gemini-1.5-flash",
		Messages:  []Message{{Role: RoleUser, Content: "Hi"}},
		MaxTokens: 1,
	}

	_, err := p.Chat(ctx, testReq)
	status.Latency = time.Since(start)

	if err != nil {
		status.Available = false
		status.Error = err.Error()
		return status, nil
	}

	status.Available = true
	return status, nil
}

func (p *VertexAIProvider) SupportsStreaming() bool       { return true }
func (p *VertexAIProvider) SupportsEmbeddings() bool      { return true }
func (p *VertexAIProvider) SupportsFunctionCalling() bool { return true }
func (p *VertexAIProvider) Close() error                  { return nil }

func (p *VertexAIProvider) convertRequest(req *ChatRequest) *vertexRequest {
	vertexReq := &vertexRequest{
		GenerationConfig: &vertexGenerationConfig{},
	}

	// Extract system instruction
	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			vertexReq.SystemInstruction = &vertexContent{
				Parts: []vertexPart{{Text: msg.Content}},
			}
			break
		}
	}

	// Convert messages
	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			continue
		}

		role := "user"
		if msg.Role == RoleAssistant {
			role = "model"
		}

		vertexReq.Contents = append(vertexReq.Contents, vertexContent{
			Role:  role,
			Parts: []vertexPart{{Text: msg.Content}},
		})
	}

	// Set generation config
	if req.MaxTokens > 0 {
		vertexReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
	}
	vertexReq.GenerationConfig.Temperature = req.Temperature
	vertexReq.GenerationConfig.TopP = req.TopP
	if len(req.Stop) > 0 {
		vertexReq.GenerationConfig.StopSequences = req.Stop
	}

	return vertexReq
}

func (p *VertexAIProvider) convertResponse(model string, resp *vertexResponse) *ChatResponse {
	chatResp := &ChatResponse{
		Model:   model,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Usage: &Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}

	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		var content strings.Builder
		for _, part := range candidate.Content.Parts {
			content.WriteString(part.Text)
		}

		finishReason := p.mapFinishReason(candidate.FinishReason)

		chatResp.Choices = []Choice{{
			Index: 0,
			Message: Message{
				Role:    RoleAssistant,
				Content: content.String(),
			},
			FinishReason: finishReason,
		}}
		chatResp.FinishReason = finishReason
	}

	return chatResp
}

func (p *VertexAIProvider) mapFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	default:
		return reason
	}
}

func (p *VertexAIProvider) doRequest(ctx context.Context, modelID, action string, body interface{}) (io.ReadCloser, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Vertex AI endpoint
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:%s",
		p.location, p.projectID, p.location, modelID, action)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Get access token for authentication
	token, err := p.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	for k, v := range p.config.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, p.parseError(resp.StatusCode, body)
	}

	return resp.Body, nil
}

func (p *VertexAIProvider) parseError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		switch statusCode {
		case 401, 403:
			return ErrInvalidCredentials
		case 429:
			return ErrRateLimitExceeded
		case 400:
			if strings.Contains(strings.ToLower(errResp.Error.Message), "context") || strings.Contains(strings.ToLower(errResp.Error.Message), "token") {
				return ErrContextLengthExceeded
			}
			return fmt.Errorf("%w: %s", ErrInvalidRequest, errResp.Error.Message)
		case 404:
			return ErrModelNotFound
		case 503, 502, 504:
			return ErrProviderUnavailable
		default:
			return fmt.Errorf("vertex AI error: %s", errResp.Error.Message)
		}
	}

	return fmt.Errorf("vertex AI error (status %d): %s", statusCode, string(body))
}
