package providers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// BedrockProvider implements the Provider interface for AWS Bedrock
type BedrockProvider struct {
	config       *ProviderConfig
	httpClient   *http.Client
	region       string
	accessKey    string
	secretKey    string
	sessionToken string
}

// NewBedrockProvider creates a new AWS Bedrock provider
func NewBedrockProvider(config *ProviderConfig) (*BedrockProvider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.APIKey == "" || config.SecretKey == "" {
		return nil, ErrInvalidCredentials
	}

	region := config.Region
	if region == "" {
		region = "us-east-1"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	var sessionToken string
	if config.Custom != nil {
		if token, ok := config.Custom["session_token"].(string); ok {
			sessionToken = token
		}
	}

	return &BedrockProvider{
		config:       config,
		httpClient:   &http.Client{Timeout: timeout},
		region:       region,
		accessKey:    config.APIKey,
		secretKey:    config.SecretKey,
		sessionToken: sessionToken,
	}, nil
}

func (p *BedrockProvider) Name() string { return "AWS Bedrock" }
func (p *BedrockProvider) Type() string { return "bedrock" }

// bedrockRequest is the Bedrock Converse API request format
type bedrockRequest struct {
	Messages        []bedrockMessage        `json:"messages"`
	System          []bedrockContent        `json:"system,omitempty"`
	InferenceConfig *bedrockInferenceConfig `json:"inferenceConfig,omitempty"`
}

type bedrockMessage struct {
	Role    string           `json:"role"`
	Content []bedrockContent `json:"content"`
}

type bedrockContent struct {
	Text string `json:"text,omitempty"`
}

type bedrockInferenceConfig struct {
	MaxTokens     int      `json:"maxTokens,omitempty"`
	Temperature   float64  `json:"temperature,omitempty"`
	TopP          float64  `json:"topP,omitempty"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

type bedrockResponse struct {
	Output struct {
		Message bedrockMessage `json:"message"`
	} `json:"output"`
	StopReason string `json:"stopReason"`
	Usage      struct {
		InputTokens  int `json:"inputTokens"`
		OutputTokens int `json:"outputTokens"`
	} `json:"usage"`
}

func (p *BedrockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	bedrockReq := p.convertRequest(req)

	body, err := p.doRequest(ctx, req.Model, bedrockReq)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var resp bedrockResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertResponse(req.Model, &resp), nil
}

func (p *BedrockProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	// Bedrock streaming requires different endpoint and handling
	// For simplicity, we do non-streaming here
	resp, err := p.Chat(ctx, req)
	if err != nil {
		return err
	}

	// Simulate streaming with single callback
	if len(resp.Choices) > 0 {
		streamResp := &ChatResponse{
			Delta: &Message{
				Role:    RoleAssistant,
				Content: resp.Choices[0].Message.Content,
			},
		}
		if err := callback(streamResp); err != nil {
			return err
		}
	}

	finalResp := &ChatResponse{
		FinishReason: resp.FinishReason,
		Usage:        resp.Usage,
	}
	return callback(finalResp)
}

func (p *BedrockProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	// Bedrock embedding models (Titan, Cohere)
	embedReq := struct {
		InputText string `json:"inputText"`
	}{
		InputText: strings.Join(req.Input, "\n"),
	}

	body, err := p.doRequest(ctx, req.Model, embedReq)
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

func (p *BedrockProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Bedrock doesn't have a simple list models API, return known models
	return []ModelInfo{
		{ID: "anthropic.claude-3-opus-20240229-v1:0", Name: "Claude 3 Opus", ContextWindow: 200000, Capabilities: []string{"chat", "vision"}},
		{ID: "anthropic.claude-3-sonnet-20240229-v1:0", Name: "Claude 3 Sonnet", ContextWindow: 200000, Capabilities: []string{"chat", "vision"}},
		{ID: "anthropic.claude-3-haiku-20240307-v1:0", Name: "Claude 3 Haiku", ContextWindow: 200000, Capabilities: []string{"chat", "vision"}},
		{ID: "anthropic.claude-3-5-sonnet-20241022-v2:0", Name: "Claude 3.5 Sonnet", ContextWindow: 200000, Capabilities: []string{"chat", "vision"}},
		{ID: "meta.llama3-1-405b-instruct-v1:0", Name: "Llama 3.1 405B", ContextWindow: 128000, Capabilities: []string{"chat"}},
		{ID: "meta.llama3-1-70b-instruct-v1:0", Name: "Llama 3.1 70B", ContextWindow: 128000, Capabilities: []string{"chat"}},
		{ID: "amazon.titan-embed-text-v2:0", Name: "Titan Embeddings V2", Capabilities: []string{"embed"}},
		{ID: "cohere.embed-english-v3", Name: "Cohere Embed English", Capabilities: []string{"embed"}},
	}, nil
}

func (p *BedrockProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	status := &HealthStatus{CheckedAt: start}

	// Try a minimal request
	testReq := &ChatRequest{
		Model:     "anthropic.claude-3-haiku-20240307-v1:0",
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

func (p *BedrockProvider) SupportsStreaming() bool       { return true }
func (p *BedrockProvider) SupportsEmbeddings() bool      { return true }
func (p *BedrockProvider) SupportsFunctionCalling() bool { return true }
func (p *BedrockProvider) Close() error                  { return nil }

func (p *BedrockProvider) convertRequest(req *ChatRequest) *bedrockRequest {
	bedrockReq := &bedrockRequest{
		InferenceConfig: &bedrockInferenceConfig{},
	}

	// Convert messages
	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			bedrockReq.System = append(bedrockReq.System, bedrockContent{Text: msg.Content})
			continue
		}

		role := string(msg.Role)
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}

		bedrockReq.Messages = append(bedrockReq.Messages, bedrockMessage{
			Role:    role,
			Content: []bedrockContent{{Text: msg.Content}},
		})
	}

	// Set inference config
	if req.MaxTokens > 0 {
		bedrockReq.InferenceConfig.MaxTokens = req.MaxTokens
	} else {
		bedrockReq.InferenceConfig.MaxTokens = 4096
	}
	if req.Temperature != nil {
		bedrockReq.InferenceConfig.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		bedrockReq.InferenceConfig.TopP = *req.TopP
	}
	if len(req.Stop) > 0 {
		bedrockReq.InferenceConfig.StopSequences = req.Stop
	}

	return bedrockReq
}

func (p *BedrockProvider) convertResponse(model string, resp *bedrockResponse) *ChatResponse {
	chatResp := &ChatResponse{
		Model:   model,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Usage: &Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	var content strings.Builder
	for _, c := range resp.Output.Message.Content {
		content.WriteString(c.Text)
	}

	finishReason := "stop"
	if resp.StopReason == "max_tokens" {
		finishReason = "length"
	}

	chatResp.Choices = []Choice{{
		Index: 0,
		Message: Message{
			Role:    RoleAssistant,
			Content: content.String(),
		},
		FinishReason: finishReason,
	}}
	chatResp.FinishReason = finishReason

	return chatResp
}

func (p *BedrockProvider) doRequest(ctx context.Context, modelID string, body interface{}) (io.ReadCloser, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Bedrock endpoint
	host := fmt.Sprintf("bedrock-runtime.%s.amazonaws.com", p.region)
	path := fmt.Sprintf("/model/%s/converse", modelID)
	url := fmt.Sprintf("https://%s%s", host, path)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Sign request with AWS Signature V4
	p.signRequest(req, jsonBody, host)

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

func (p *BedrockProvider) signRequest(req *http.Request, payload []byte, host string) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", amzDate)
	if p.sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", p.sessionToken)
	}

	// Create canonical request
	payloadHash := sha256Hash(payload)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	if p.sessionToken != "" {
		signedHeaders += ";x-amz-security-token"
	}

	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s",
		req.Header.Get("Content-Type"), host, payloadHash, amzDate)
	if p.sessionToken != "" {
		canonicalHeaders += fmt.Sprintf("\nx-amz-security-token:%s", p.sessionToken)
	}
	canonicalHeaders += "\n"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method, req.URL.Path, req.URL.RawQuery, canonicalHeaders, signedHeaders, payloadHash)

	// Create string to sign
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/bedrock/aws4_request", dateStamp, p.region)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm, amzDate, credentialScope, sha256Hash([]byte(canonicalRequest)))

	// Calculate signature
	signingKey := p.getSignatureKey(dateStamp, p.region, "bedrock")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Add Authorization header
	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, p.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func (p *BedrockProvider) getSignatureKey(dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+p.secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func sha256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func (p *BedrockProvider) parseError(statusCode int, body []byte) error {
	var errResp struct {
		Message string `json:"message"`
		Type    string `json:"__type"`
	}

	_ = json.Unmarshal(body, &errResp)
	errMsg := errResp.Message
	if errMsg == "" {
		errMsg = string(body)
	}

	switch statusCode {
	case 401, 403:
		return ErrInvalidCredentials
	case 429:
		return ErrRateLimitExceeded
	case 400:
		if strings.Contains(strings.ToLower(errMsg), "context") || strings.Contains(strings.ToLower(errMsg), "token") {
			return ErrContextLengthExceeded
		}
		return fmt.Errorf("%w: %s", ErrInvalidRequest, errMsg)
	case 404:
		return ErrModelNotFound
	case 503, 502, 504:
		return ErrProviderUnavailable
	default:
		return fmt.Errorf("bedrock error: %s", errMsg)
	}
}
