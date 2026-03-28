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
	ollamaDefaultBaseURL = "http://localhost:11434"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	baseURL    string
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(config *ProviderConfig) (*OllamaProvider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = ollamaDefaultBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second // Local models can be slow
	}

	return &OllamaProvider{
		config:     config,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
	}, nil
}

func (p *OllamaProvider) Name() string { return "Ollama" }
func (p *OllamaProvider) Type() string { return "ollama" }

// ollamaRequest is the Ollama API request format
type ollamaRequest struct {
	Model    string                 `json:"model"`
	Messages []ollamaMessage        `json:"messages,omitempty"`
	Prompt   string                 `json:"prompt,omitempty"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Model              string         `json:"model"`
	CreatedAt          string         `json:"created_at"`
	Message            *ollamaMessage `json:"message,omitempty"`
	Response           string         `json:"response,omitempty"`
	Done               bool           `json:"done"`
	TotalDuration      int64          `json:"total_duration,omitempty"`
	LoadDuration       int64          `json:"load_duration,omitempty"`
	PromptEvalCount    int            `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64          `json:"prompt_eval_duration,omitempty"`
	EvalCount          int            `json:"eval_count,omitempty"`
	EvalDuration       int64          `json:"eval_duration,omitempty"`
}

func (p *OllamaProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	ollamaReq := p.convertRequest(req)
	ollamaReq.Stream = false

	body, err := p.doRequest(ctx, "POST", "/api/chat", ollamaReq)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp ollamaResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertResponse(&resp), nil
}

func (p *OllamaProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	ollamaReq := p.convertRequest(req)
	ollamaReq.Stream = true

	body, err := p.doRequest(ctx, "POST", "/api/chat", ollamaReq)
	if err != nil {
		return err
	}
	defer body.Close()

	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}

		var resp ollamaResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		chatResp := &ChatResponse{
			Model: resp.Model,
		}

		if resp.Message != nil {
			chatResp.Delta = &Message{
				Role:    Role(resp.Message.Role),
				Content: resp.Message.Content,
			}
		}

		if resp.Done {
			chatResp.FinishReason = "stop"
			chatResp.Usage = &Usage{
				PromptTokens:     resp.PromptEvalCount,
				CompletionTokens: resp.EvalCount,
				TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
			}
		}

		if err := callback(chatResp); err != nil {
			return err
		}
	}

	return nil
}

func (p *OllamaProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	// Ollama uses different embedding endpoint
	embedReq := struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}{
		Model:  req.Model,
		Prompt: strings.Join(req.Input, "\n"),
	}

	body, err := p.doRequest(ctx, "POST", "/api/embeddings", embedReq)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(resp.Embedding))
	for i, v := range resp.Embedding {
		embedding[i] = float32(v)
	}

	return &EmbeddingResponse{
		Object: "list",
		Model:  req.Model,
		Data: []Embedding{{
			Object:    "embedding",
			Index:     0,
			Embedding: embedding,
		}},
	}, nil
}

func (p *OllamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	body, err := p.doRequest(ctx, "GET", "/api/tags", nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp struct {
		Models []struct {
			Name       string `json:"name"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
			Details    struct {
				Format            string   `json:"format"`
				Family            string   `json:"family"`
				Families          []string `json:"families"`
				ParameterSize     string   `json:"parameter_size"`
				QuantizationLevel string   `json:"quantization_level"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]ModelInfo, 0, len(resp.Models))
	for _, m := range resp.Models {
		info := ModelInfo{
			ID:           m.Name,
			Name:         m.Name,
			Description:  fmt.Sprintf("%s (%s)", m.Details.Family, m.Details.ParameterSize),
			Capabilities: []string{"chat"},
		}

		// Check for embedding models
		name := strings.ToLower(m.Name)
		if strings.Contains(name, "embed") || strings.Contains(name, "nomic") {
			info.Capabilities = []string{"embed"}
		}
		// Check for vision models
		if strings.Contains(name, "llava") || strings.Contains(name, "vision") {
			info.Capabilities = append(info.Capabilities, "vision")
		}

		models = append(models, info)
	}

	return models, nil
}

func (p *OllamaProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	status := &HealthStatus{CheckedAt: start}

	body, err := p.doRequest(ctx, "GET", "/api/tags", nil)
	status.Latency = time.Since(start)

	if err != nil {
		status.Available = false
		status.Error = err.Error()
		return status, nil
	}
	defer body.Close()

	status.Available = true
	return status, nil
}

func (p *OllamaProvider) SupportsStreaming() bool       { return true }
func (p *OllamaProvider) SupportsEmbeddings() bool      { return true }
func (p *OllamaProvider) SupportsFunctionCalling() bool { return false } // Ollama has limited tool support
func (p *OllamaProvider) Close() error                  { return nil }

func (p *OllamaProvider) convertRequest(req *ChatRequest) *ollamaRequest {
	ollamaReq := &ollamaRequest{
		Model:   req.Model,
		Options: make(map[string]interface{}),
	}

	// Convert messages
	for _, msg := range req.Messages {
		ollamaReq.Messages = append(ollamaReq.Messages, ollamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	// Convert options
	if req.Temperature != nil {
		ollamaReq.Options["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		ollamaReq.Options["top_p"] = *req.TopP
	}
	if req.MaxTokens > 0 {
		ollamaReq.Options["num_predict"] = req.MaxTokens
	}
	if len(req.Stop) > 0 {
		ollamaReq.Options["stop"] = req.Stop
	}

	return ollamaReq
}

func (p *OllamaProvider) convertResponse(resp *ollamaResponse) *ChatResponse {
	chatResp := &ChatResponse{
		Model:   resp.Model,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Usage: &Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
		FinishReason: "stop",
	}

	content := ""
	if resp.Message != nil {
		content = resp.Message.Content
	} else {
		content = resp.Response
	}

	chatResp.Choices = []Choice{{
		Index: 0,
		Message: Message{
			Role:    RoleAssistant,
			Content: content,
		},
		FinishReason: "stop",
	}}

	return chatResp
}

func (p *OllamaProvider) doRequest(ctx context.Context, method, path string, body interface{}) (io.ReadCloser, error) {
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

func (p *OllamaProvider) parseError(statusCode int, body []byte) error {
	var errResp struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		switch statusCode {
		case 404:
			return ErrModelNotFound
		case 400:
			return fmt.Errorf("%w: %s", ErrInvalidRequest, errResp.Error)
		default:
			return fmt.Errorf("ollama error: %s", errResp.Error)
		}
	}

	return fmt.Errorf("ollama error (status %d): %s", statusCode, string(body))
}
