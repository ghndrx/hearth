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

// OpenAICompatibleProvider implements the Provider interface for OpenAI-compatible APIs
// This works with LM Studio, vLLM, LocalAI, llama.cpp server, and other compatible servers
type OpenAICompatibleProvider struct {
	config       *ProviderConfig
	httpClient   *http.Client
	baseURL      string
	providerName string
	providerType string
	hasAuth      bool
}

// OpenAICompatibleConfig contains configuration options for OpenAI-compatible providers
type OpenAICompatibleConfig struct {
	ProviderName string // Display name (e.g., "LM Studio", "vLLM")
	ProviderType string // Type identifier (e.g., "lm_studio", "vllm")
	BaseURL      string // API base URL
	APIKey       string // Optional API key
	Timeout      time.Duration
	ExtraHeaders map[string]string
}

// NewOpenAICompatibleProvider creates a new OpenAI-compatible provider
func NewOpenAICompatibleProvider(cfg *OpenAICompatibleConfig) (*OpenAICompatibleProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second // Local models can be slow
	}

	return &OpenAICompatibleProvider{
		config: &ProviderConfig{
			BaseURL:      cfg.BaseURL,
			APIKey:       cfg.APIKey,
			Timeout:      timeout,
			ExtraHeaders: cfg.ExtraHeaders,
		},
		httpClient:   &http.Client{Timeout: timeout},
		baseURL:      strings.TrimSuffix(cfg.BaseURL, "/"),
		providerName: cfg.ProviderName,
		providerType: cfg.ProviderType,
		hasAuth:      cfg.APIKey != "",
	}, nil
}

// NewLMStudioProvider creates a new LM Studio provider
func NewLMStudioProvider(config *ProviderConfig) (*OpenAICompatibleProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:1234/v1"
	}

	return NewOpenAICompatibleProvider(&OpenAICompatibleConfig{
		ProviderName: "LM Studio",
		ProviderType: "lm_studio",
		BaseURL:      baseURL,
		APIKey:       config.APIKey,
		Timeout:      config.Timeout,
		ExtraHeaders: config.ExtraHeaders,
	})
}

// NewVLLMProvider creates a new vLLM provider
func NewVLLMProvider(config *ProviderConfig) (*OpenAICompatibleProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8000/v1"
	}

	return NewOpenAICompatibleProvider(&OpenAICompatibleConfig{
		ProviderName: "vLLM",
		ProviderType: "vllm",
		BaseURL:      baseURL,
		APIKey:       config.APIKey,
		Timeout:      config.Timeout,
		ExtraHeaders: config.ExtraHeaders,
	})
}

// NewLocalAIProvider creates a new LocalAI provider
func NewLocalAIProvider(config *ProviderConfig) (*OpenAICompatibleProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080/v1"
	}

	return NewOpenAICompatibleProvider(&OpenAICompatibleConfig{
		ProviderName: "LocalAI",
		ProviderType: "local_ai",
		BaseURL:      baseURL,
		APIKey:       config.APIKey,
		Timeout:      config.Timeout,
		ExtraHeaders: config.ExtraHeaders,
	})
}

// NewLlamaCppProvider creates a new llama.cpp server provider
func NewLlamaCppProvider(config *ProviderConfig) (*OpenAICompatibleProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080/v1"
	}

	return NewOpenAICompatibleProvider(&OpenAICompatibleConfig{
		ProviderName: "llama.cpp",
		ProviderType: "llama_cpp",
		BaseURL:      baseURL,
		APIKey:       config.APIKey,
		Timeout:      config.Timeout,
		ExtraHeaders: config.ExtraHeaders,
	})
}

func (p *OpenAICompatibleProvider) Name() string { return p.providerName }
func (p *OpenAICompatibleProvider) Type() string { return p.providerType }

func (p *OpenAICompatibleProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
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

func (p *OpenAICompatibleProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
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

func (p *OpenAICompatibleProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
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

func (p *OpenAICompatibleProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	body, err := p.doRequest(ctx, "GET", "/models", nil)
	if err != nil {
		// Some servers don't support this endpoint
		return []ModelInfo{}, nil
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
		return []ModelInfo{}, nil
	}

	models := make([]ModelInfo, 0, len(resp.Data))
	for _, m := range resp.Data {
		models = append(models, ModelInfo{
			ID:           m.ID,
			Name:         m.ID,
			Capabilities: []string{"chat"},
		})
	}

	return models, nil
}

func (p *OpenAICompatibleProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	status := &HealthStatus{CheckedAt: start}

	// Try to list models
	_, err := p.ListModels(ctx)
	status.Latency = time.Since(start)

	if err != nil {
		// Try a simple health endpoint
		body, err2 := p.doRequest(ctx, "GET", "/health", nil)
		if err2 == nil {
			body.Close()
			status.Available = true
			return status, nil
		}

		status.Available = false
		status.Error = err.Error()
		return status, nil
	}

	status.Available = true
	return status, nil
}

func (p *OpenAICompatibleProvider) SupportsStreaming() bool       { return true }
func (p *OpenAICompatibleProvider) SupportsEmbeddings() bool      { return true }
func (p *OpenAICompatibleProvider) SupportsFunctionCalling() bool { return false } // Varies by backend
func (p *OpenAICompatibleProvider) Close() error                  { return nil }

func (p *OpenAICompatibleProvider) doRequest(ctx context.Context, method, path string, body interface{}) (io.ReadCloser, error) {
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

	req.Header.Set("Content-Type", "application/json")
	if p.hasAuth {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

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

func (p *OpenAICompatibleProvider) parseError(statusCode int, body []byte) error {
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
			if strings.Contains(errResp.Error.Message, "context") {
				return ErrContextLengthExceeded
			}
			return fmt.Errorf("%w: %s", ErrInvalidRequest, errResp.Error.Message)
		case 404:
			return ErrModelNotFound
		case 503, 502, 504:
			return ErrProviderUnavailable
		default:
			return fmt.Errorf("API error: %s", errResp.Error.Message)
		}
	}

	return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
}
