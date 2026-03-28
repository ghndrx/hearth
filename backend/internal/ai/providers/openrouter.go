package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	openRouterDefaultBaseURL = "https://openrouter.ai/api/v1"
)

// OpenRouterProvider implements the Provider interface for OpenRouter
type OpenRouterProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	baseURL    string
	appName    string
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(config *ProviderConfig) (*OpenRouterProvider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.APIKey == "" {
		return nil, ErrInvalidCredentials
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = openRouterDefaultBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	appName := "Hearth"
	if config.Custom != nil {
		if name, ok := config.Custom["app_name"].(string); ok {
			appName = name
		}
	}

	return &OpenRouterProvider{
		config:     config,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		appName:    appName,
	}, nil
}

func (p *OpenRouterProvider) Name() string { return "OpenRouter" }
func (p *OpenRouterProvider) Type() string { return "openrouter" }

func (p *OpenRouterProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	body, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp ChatResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

func (p *OpenRouterProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	req.Stream = true
	body, err := p.doRequest(ctx, "POST", "/chat/completions", req)
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
		if line == "" || line == "data: [DONE]" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var resp ChatResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if err := callback(&resp); err != nil {
			return err
		}
	}

	return nil
}

func (p *OpenRouterProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	body, err := p.doRequest(ctx, "POST", "/embeddings", req)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp EmbeddingResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

func (p *OpenRouterProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	body, err := p.doRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			ContextLength int    `json:"context_length"`
			Pricing       struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
			TopProvider struct {
				MaxCompletionTokens int `json:"max_completion_tokens"`
			} `json:"top_provider"`
		} `json:"data"`
	}
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]ModelInfo, 0, len(resp.Data))
	for _, m := range resp.Data {
		info := ModelInfo{
			ID:            m.ID,
			Name:          m.Name,
			Description:   m.Description,
			ContextWindow: m.ContextLength,
			MaxTokens:     m.TopProvider.MaxCompletionTokens,
			Capabilities:  []string{"chat"},
		}

		// Infer capabilities from model ID
		id := strings.ToLower(m.ID)
		if strings.Contains(id, "vision") || strings.Contains(id, "gpt-4o") ||
			strings.Contains(id, "claude-3") || strings.Contains(id, "gemini") {
			info.Capabilities = append(info.Capabilities, "vision")
		}
		if strings.Contains(id, "gpt") || strings.Contains(id, "claude") ||
			strings.Contains(id, "gemini") || strings.Contains(id, "mistral") {
			info.Capabilities = append(info.Capabilities, "function_calling")
		}

		models = append(models, info)
	}

	return models, nil
}

func (p *OpenRouterProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	status := &HealthStatus{CheckedAt: start}

	_, err := p.ListModels(ctx)
	status.Latency = time.Since(start)

	if err != nil {
		status.Available = false
		status.Error = err.Error()
		return status, nil
	}

	status.Available = true
	return status, nil
}

func (p *OpenRouterProvider) SupportsStreaming() bool       { return true }
func (p *OpenRouterProvider) SupportsEmbeddings() bool      { return true }
func (p *OpenRouterProvider) SupportsFunctionCalling() bool { return true }
func (p *OpenRouterProvider) Close() error                  { return nil }

func (p *OpenRouterProvider) doRequest(ctx context.Context, method, path string, body interface{}) (io.ReadCloser, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://hearth.app")
	req.Header.Set("X-Title", p.appName)

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

func (p *OpenRouterProvider) parseError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		switch statusCode {
		case 401:
			return ErrInvalidCredentials
		case 429:
			return ErrRateLimitExceeded
		case 400:
			if strings.Contains(errResp.Error.Message, "context") {
				return ErrContextLengthExceeded
			}
			return fmt.Errorf("%w: %s", ErrInvalidRequest, errResp.Error.Message)
		case 503, 502, 504:
			return ErrProviderUnavailable
		default:
			return fmt.Errorf("API error: %s", errResp.Error.Message)
		}
	}

	return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
}
