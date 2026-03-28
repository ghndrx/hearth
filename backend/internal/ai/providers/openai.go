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
	openAIDefaultBaseURL = "https://api.openai.com/v1"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	baseURL    string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config *ProviderConfig) (*OpenAIProvider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.APIKey == "" {
		return nil, ErrInvalidCredentials
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = openAIDefaultBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &OpenAIProvider{
		config:     config,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
	}, nil
}

func (p *OpenAIProvider) Name() string { return "OpenAI" }
func (p *OpenAIProvider) Type() string { return "openai" }

func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
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

func (p *OpenAIProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
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
			continue // Skip malformed chunks
		}

		if err := callback(&resp); err != nil {
			return err
		}
	}

	return nil
}

func (p *OpenAIProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
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

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	body, err := p.doRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]ModelInfo, 0, len(resp.Data))
	for _, m := range resp.Data {
		info := ModelInfo{
			ID:   m.ID,
			Name: m.ID,
		}
		// Infer capabilities from model name
		if strings.Contains(m.ID, "gpt-4") || strings.Contains(m.ID, "gpt-3.5") {
			info.Capabilities = []string{"chat", "function_calling"}
			if strings.Contains(m.ID, "vision") {
				info.Capabilities = append(info.Capabilities, "vision")
			}
		} else if strings.Contains(m.ID, "embedding") {
			info.Capabilities = []string{"embed"}
		} else if strings.Contains(m.ID, "text-moderation") {
			info.Capabilities = []string{"moderate"}
		}
		models = append(models, info)
	}

	return models, nil
}

func (p *OpenAIProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	status := &HealthStatus{CheckedAt: start}

	// Try to list models as a health check
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

func (p *OpenAIProvider) SupportsStreaming() bool       { return true }
func (p *OpenAIProvider) SupportsEmbeddings() bool      { return true }
func (p *OpenAIProvider) SupportsFunctionCalling() bool { return true }
func (p *OpenAIProvider) Close() error                  { return nil }

func (p *OpenAIProvider) doRequest(ctx context.Context, method, path string, body interface{}) (io.ReadCloser, error) {
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

	// Add extra headers
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

func (p *OpenAIProvider) parseError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		switch statusCode {
		case 401:
			return ErrInvalidCredentials
		case 429:
			return ErrRateLimitExceeded
		case 400:
			if strings.Contains(errResp.Error.Message, "context_length") {
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
