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
	anthropicDefaultBaseURL = "https://api.anthropic.com/v1"
	anthropicAPIVersion     = "2023-06-01"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude
type AnthropicProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	baseURL    string
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(config *ProviderConfig) (*AnthropicProvider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.APIKey == "" {
		return nil, ErrInvalidCredentials
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = anthropicDefaultBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second // Anthropic can be slower
	}

	return &AnthropicProvider{
		config:     config,
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
	}, nil
}

func (p *AnthropicProvider) Name() string { return "Anthropic" }
func (p *AnthropicProvider) Type() string { return "anthropic" }

// anthropicRequest is the Anthropic API request format
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
	TopP        *float64           `json:"top_p,omitempty"`
	StopSeqs    []string           `json:"stop_sequences,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type      string      `json:"type"`
	Text      string      `json:"text,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Input     interface{} `json:"input,omitempty"`
	Name      string      `json:"name,omitempty"`
	Content   string      `json:"content,omitempty"` // For tool_result
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// anthropicResponse is the Anthropic API response format
type anthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []anthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason,omitempty"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	anthropicReq := p.convertRequest(req)
	anthropicReq.Stream = false

	body, err := p.doRequest(ctx, "POST", "/messages", anthropicReq)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp anthropicResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertResponse(&resp), nil
}

func (p *AnthropicProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	anthropicReq := p.convertRequest(req)
	anthropicReq.Stream = true

	body, err := p.doRequest(ctx, "POST", "/messages", anthropicReq)
	if err != nil {
		return err
	}
	defer body.Close()

	reader := bufio.NewReader(body)
	var accumulated strings.Builder

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

		var event struct {
			Type  string `json:"type"`
			Index int    `json:"index,omitempty"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
			} `json:"delta,omitempty"`
			Message *anthropicResponse `json:"message,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta.Type == "text_delta" {
				accumulated.WriteString(event.Delta.Text)
				resp := &ChatResponse{
					Delta: &Message{
						Role:    RoleAssistant,
						Content: event.Delta.Text,
					},
				}
				if err := callback(resp); err != nil {
					return err
				}
			}
		case "message_stop":
			// Final message
			resp := &ChatResponse{
				FinishReason: "stop",
			}
			if err := callback(resp); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *AnthropicProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, ErrStreamingNotSupported // Anthropic doesn't have native embedding API
}

func (p *AnthropicProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Anthropic doesn't have a list models endpoint, return known models
	return []ModelInfo{
		{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", ContextWindow: 200000, Capabilities: []string{"chat", "vision", "function_calling"}},
		{ID: "claude-3-sonnet-20240229", Name: "Claude 3 Sonnet", ContextWindow: 200000, Capabilities: []string{"chat", "vision", "function_calling"}},
		{ID: "claude-3-haiku-20240307", Name: "Claude 3 Haiku", ContextWindow: 200000, Capabilities: []string{"chat", "vision", "function_calling"}},
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", ContextWindow: 200000, Capabilities: []string{"chat", "vision", "function_calling"}},
		{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", ContextWindow: 200000, Capabilities: []string{"chat", "vision", "function_calling"}},
	}, nil
}

func (p *AnthropicProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	status := &HealthStatus{CheckedAt: start}

	// Do a minimal request to check health
	testReq := &ChatRequest{
		Model:     "claude-3-haiku-20240307",
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

func (p *AnthropicProvider) SupportsStreaming() bool       { return true }
func (p *AnthropicProvider) SupportsEmbeddings() bool      { return false }
func (p *AnthropicProvider) SupportsFunctionCalling() bool { return true }
func (p *AnthropicProvider) Close() error                  { return nil }

func (p *AnthropicProvider) convertRequest(req *ChatRequest) *anthropicRequest {
	anthropicReq := &anthropicRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		StopSeqs:    req.Stop,
	}

	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}

	// Convert messages, extracting system prompt
	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			anthropicReq.System = msg.Content
			continue
		}

		anthropicMsg := anthropicMessage{
			Role: string(msg.Role),
		}

		if msg.Role == RoleTool {
			anthropicMsg.Role = "user"
			anthropicMsg.Content = []anthropicContent{{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   msg.Content,
			}}
		} else {
			anthropicMsg.Content = []anthropicContent{{
				Type: "text",
				Text: msg.Content,
			}}
		}

		anthropicReq.Messages = append(anthropicReq.Messages, anthropicMsg)
	}

	// Convert tools
	for _, tool := range req.Tools {
		anthropicReq.Tools = append(anthropicReq.Tools, anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}

	return anthropicReq
}

func (p *AnthropicProvider) convertResponse(resp *anthropicResponse) *ChatResponse {
	chatResp := &ChatResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Usage: &Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	// Convert content to message
	var content strings.Builder
	var toolCalls []ToolCall

	for _, c := range resp.Content {
		switch c.Type {
		case "text":
			content.WriteString(c.Text)
		case "tool_use":
			inputJSON, _ := json.Marshal(c.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:   c.ToolUseID,
				Type: "function",
				Function: FunctionCall{
					Name:      c.Name,
					Arguments: string(inputJSON),
				},
			})
		}
	}

	message := Message{
		Role:      RoleAssistant,
		Content:   content.String(),
		ToolCalls: toolCalls,
	}

	finishReason := "stop"
	if resp.StopReason == "tool_use" {
		finishReason = "tool_calls"
	} else if resp.StopReason == "max_tokens" {
		finishReason = "length"
	}

	chatResp.Choices = []Choice{{
		Index:        0,
		Message:      message,
		FinishReason: finishReason,
	}}
	chatResp.FinishReason = finishReason

	return chatResp
}

func (p *AnthropicProvider) doRequest(ctx context.Context, method, path string, body interface{}) (io.ReadCloser, error) {
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

	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
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

func (p *AnthropicProvider) parseError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
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
		case 503, 502, 504, 529:
			return ErrProviderUnavailable
		default:
			return fmt.Errorf("API error: %s", errResp.Error.Message)
		}
	}

	return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
}
