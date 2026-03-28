package ai

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"hearth/internal/ai/providers"
	"hearth/internal/models"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrTemplateNotFound     = errors.New("template not found")
	ErrNotAuthorized        = errors.New("not authorized to access this resource")
	ErrShareNotFound        = errors.New("share not found")
	ErrShareExpired         = errors.New("share has expired")
)

// ChatRepository defines the interface for AI chat data persistence
type ChatRepository interface {
	// Conversations
	CreateConversation(ctx context.Context, conv *models.AIConversation) error
	GetConversation(ctx context.Context, id uuid.UUID) (*models.AIConversation, error)
	GetConversationWithMessages(ctx context.Context, id uuid.UUID, limit int) (*models.AIConversationWithMessages, error)
	ListConversations(ctx context.Context, params *models.ConversationListParams) ([]*models.AIConversation, error)
	UpdateConversation(ctx context.Context, conv *models.AIConversation) error
	DeleteConversation(ctx context.Context, id uuid.UUID) error

	// Messages
	CreateMessage(ctx context.Context, msg *models.ConversationMessage) error
	GetMessage(ctx context.Context, id uuid.UUID) (*models.ConversationMessage, error)
	GetConversationMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*models.ConversationMessage, error)
	UpdateMessage(ctx context.Context, msg *models.ConversationMessage) error
	DeleteMessage(ctx context.Context, id uuid.UUID) error
	DeleteMessagesAfter(ctx context.Context, conversationID uuid.UUID, afterID uuid.UUID) error

	// Templates
	CreateTemplate(ctx context.Context, tmpl *models.AIChatTemplate) error
	GetTemplate(ctx context.Context, id uuid.UUID) (*models.AIChatTemplate, error)
	ListTemplates(ctx context.Context, userID *uuid.UUID, includePublic bool, category string) ([]*models.AIChatTemplate, error)
	UpdateTemplate(ctx context.Context, tmpl *models.AIChatTemplate) error
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	IncrementTemplateUsage(ctx context.Context, id uuid.UUID) error

	// Shares
	CreateShare(ctx context.Context, share *models.AIConversationShare) error
	GetShareByCode(ctx context.Context, code string) (*models.AIConversationShare, error)
	GetConversationShares(ctx context.Context, conversationID uuid.UUID) ([]*models.AIConversationShare, error)
	DeleteShare(ctx context.Context, id uuid.UUID) error
	IncrementShareViewCount(ctx context.Context, id uuid.UUID) error
}

// ChatService manages AI chat conversations
type ChatService struct {
	repo      ChatRepository
	aiService *AIService
}

// NewChatService creates a new ChatService
func NewChatService(repo ChatRepository, aiService *AIService) *ChatService {
	return &ChatService{
		repo:      repo,
		aiService: aiService,
	}
}

// --- Conversations ---

// CreateConversation creates a new AI conversation
func (s *ChatService) CreateConversation(ctx context.Context, userID uuid.UUID, req *models.CreateConversationRequest) (*models.AIConversation, error) {
	conv := &models.AIConversation{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "New Chat",
		Temperature: 0.7,
		MaxTokens:   2048,
		IsArchived:  false,
		IsPinned:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Apply request params
	if req.Title != "" {
		conv.Title = req.Title
	}
	if req.ModelID != nil {
		conv.ModelID = req.ModelID
	}
	if req.ProviderID != nil {
		providerID, err := uuid.Parse(*req.ProviderID)
		if err == nil {
			conv.ProviderID = &providerID
		}
	}
	if req.SystemPrompt != nil {
		conv.SystemPrompt = req.SystemPrompt
	}
	if req.Temperature > 0 {
		conv.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		conv.MaxTokens = req.MaxTokens
	}

	// Apply template if specified
	if req.TemplateID != nil {
		templateID, err := uuid.Parse(*req.TemplateID)
		if err == nil {
			template, err := s.repo.GetTemplate(ctx, templateID)
			if err == nil && template != nil {
				conv.SystemPrompt = &template.SystemPrompt
				if template.Name != "" && conv.Title == "New Chat" {
					conv.Title = template.Name
				}
				// Increment usage count
				s.repo.IncrementTemplateUsage(ctx, templateID)
			}
		}
	}

	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conv, nil
}

// GetConversation retrieves a conversation by ID
func (s *ChatService) GetConversation(ctx context.Context, userID, conversationID uuid.UUID) (*models.AIConversation, error) {
	conv, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, ErrConversationNotFound
	}
	if conv.UserID != userID {
		return nil, ErrNotAuthorized
	}
	return conv, nil
}

// GetConversationWithMessages retrieves a conversation with its messages
func (s *ChatService) GetConversationWithMessages(ctx context.Context, userID, conversationID uuid.UUID, limit int) (*models.AIConversationWithMessages, error) {
	conv, err := s.repo.GetConversationWithMessages(ctx, conversationID, limit)
	if err != nil {
		return nil, ErrConversationNotFound
	}
	if conv.UserID != userID {
		return nil, ErrNotAuthorized
	}
	return conv, nil
}

// ListConversations lists user's conversations
func (s *ChatService) ListConversations(ctx context.Context, params *models.ConversationListParams) ([]*models.AIConversation, error) {
	return s.repo.ListConversations(ctx, params)
}

// UpdateConversation updates a conversation
func (s *ChatService) UpdateConversation(ctx context.Context, userID, conversationID uuid.UUID, req *models.UpdateConversationRequest) (*models.AIConversation, error) {
	conv, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		conv.Title = *req.Title
	}
	if req.ModelID != nil {
		conv.ModelID = req.ModelID
	}
	if req.ProviderID != nil {
		providerID, err := uuid.Parse(*req.ProviderID)
		if err == nil {
			conv.ProviderID = &providerID
		}
	}
	if req.SystemPrompt != nil {
		conv.SystemPrompt = req.SystemPrompt
	}
	if req.Temperature != nil {
		conv.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		conv.MaxTokens = *req.MaxTokens
	}
	if req.IsArchived != nil {
		conv.IsArchived = *req.IsArchived
	}
	if req.IsPinned != nil {
		conv.IsPinned = *req.IsPinned
	}
	conv.UpdatedAt = time.Now()

	if err := s.repo.UpdateConversation(ctx, conv); err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	return conv, nil
}

// DeleteConversation deletes a conversation
func (s *ChatService) DeleteConversation(ctx context.Context, userID, conversationID uuid.UUID) error {
	conv, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return err
	}

	return s.repo.DeleteConversation(ctx, conv.ID)
}

// --- Messages ---

// SendMessage sends a message and gets AI response
func (s *ChatService) SendMessage(ctx context.Context, userID, conversationID uuid.UUID, req *models.SendChatMessageRequest) (*models.ConversationMessage, error) {
	// Get conversation
	conv, err := s.GetConversationWithMessages(ctx, userID, conversationID, 50)
	if err != nil {
		return nil, err
	}

	// Create user message
	userMsg := &models.ConversationMessage{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           models.ConvRoleUser,
		Content:        req.Content,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Build messages for AI request
	messages := s.buildMessagesForRequest(conv, req.Content)

	// Get model settings
	modelID := conv.ModelID
	if req.ModelID != "" {
		modelID = &req.ModelID
	}
	temperature := float64(conv.Temperature)
	if req.Temperature != nil {
		temperature = float64(*req.Temperature)
	}
	maxTokens := conv.MaxTokens
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	// Build chat request
	chatReq := &providers.ChatRequest{
		Messages:    messages,
		Temperature: &temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
	}
	if modelID != nil {
		chatReq.Model = *modelID
	}

	// Get AI response
	resp, err := s.aiService.Chat(ctx, FeatureChat, chatReq, nil, &userID)
	if err != nil {
		// Save error message
		errorMsg := err.Error()
		assistantMsg := &models.ConversationMessage{
			ID:             uuid.New(),
			ConversationID: conversationID,
			Role:           models.ConvRoleAssistant,
			Content:        "",
			ErrorMessage:   &errorMsg,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		s.repo.CreateMessage(ctx, assistantMsg)
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// Create assistant message
	assistantContent := ""
	finishReason := ""
	var tokensUsed *int
	var modelUsed *string

	if len(resp.Choices) > 0 {
		assistantContent = resp.Choices[0].Message.Content
		finishReason = resp.Choices[0].FinishReason
	}
	if resp.Usage != nil {
		total := resp.Usage.TotalTokens
		tokensUsed = &total
	}
	if resp.Model != "" {
		modelUsed = &resp.Model
	}

	assistantMsg := &models.ConversationMessage{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           models.ConvRoleAssistant,
		Content:        assistantContent,
		TokensUsed:     tokensUsed,
		ModelUsed:      modelUsed,
		FinishReason:   &finishReason,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	// Auto-generate title if this is the first real exchange
	if conv.MessageCount <= 1 && conv.Title == "New Chat" {
		// Use a separate context with timeout for background operation
		titleCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		go func() {
			defer cancel()
			s.autoGenerateTitle(titleCtx, userID, conversationID, req.Content, assistantContent)
		}()
	}

	return assistantMsg, nil
}

// SendMessageStream sends a message and streams the AI response
func (s *ChatService) SendMessageStream(ctx context.Context, userID, conversationID uuid.UUID, req *models.SendChatMessageRequest, callback func(*models.StreamChunk) error) (*models.ConversationMessage, error) {
	// Get conversation
	conv, err := s.GetConversationWithMessages(ctx, userID, conversationID, 50)
	if err != nil {
		return nil, err
	}

	// Create user message
	userMsg := &models.ConversationMessage{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           models.ConvRoleUser,
		Content:        req.Content,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Build messages for AI request
	messages := s.buildMessagesForRequest(conv, req.Content)

	// Get model settings
	modelID := conv.ModelID
	if req.ModelID != "" {
		modelID = &req.ModelID
	}
	temperature := float64(conv.Temperature)
	if req.Temperature != nil {
		temperature = float64(*req.Temperature)
	}
	maxTokens := conv.MaxTokens
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	// Build chat request
	chatReq := &providers.ChatRequest{
		Messages:    messages,
		Temperature: &temperature,
		MaxTokens:   maxTokens,
		Stream:      true,
	}
	if modelID != nil {
		chatReq.Model = *modelID
	}

	// Collect streamed content
	var contentBuilder strings.Builder
	var finishReason string
	var modelUsed string

	// Stream callback wrapper
	streamCallback := func(resp *providers.ChatResponse) error {
		if resp.Delta != nil && resp.Delta.Content != "" {
			contentBuilder.WriteString(resp.Delta.Content)
		}
		if resp.FinishReason != "" {
			finishReason = resp.FinishReason
		}
		if resp.Model != "" {
			modelUsed = resp.Model
		}

		// Convert to stream chunk
		chunk := &models.StreamChunk{
			ID:      resp.ID,
			Object:  "chat.completion.chunk",
			Created: resp.Created,
			Model:   resp.Model,
			Choices: []models.StreamChoice{
				{
					Index: 0,
					Delta: models.StreamDelta{
						Content: "",
					},
				},
			},
		}
		if resp.Delta != nil {
			chunk.Choices[0].Delta.Content = resp.Delta.Content
			chunk.Choices[0].Delta.Role = string(resp.Delta.Role)
		}
		if resp.FinishReason != "" {
			chunk.FinishReason = &resp.FinishReason
		}

		return callback(chunk)
	}

	// Stream the response
	err = s.aiService.ChatStream(ctx, FeatureChat, chatReq, streamCallback, nil, &userID)
	if err != nil {
		// Save error message
		errorMsg := err.Error()
		assistantMsg := &models.ConversationMessage{
			ID:             uuid.New(),
			ConversationID: conversationID,
			Role:           models.ConvRoleAssistant,
			Content:        contentBuilder.String(), // Save partial content
			ErrorMessage:   &errorMsg,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		s.repo.CreateMessage(ctx, assistantMsg)
		return nil, fmt.Errorf("AI stream failed: %w", err)
	}

	// Create assistant message with complete content
	assistantMsg := &models.ConversationMessage{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           models.ConvRoleAssistant,
		Content:        contentBuilder.String(),
		FinishReason:   &finishReason,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if modelUsed != "" {
		assistantMsg.ModelUsed = &modelUsed
	}

	if err := s.repo.CreateMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	// Auto-generate title if first exchange
	if conv.MessageCount <= 1 && conv.Title == "New Chat" {
		// Use a separate context with timeout for background operation
		titleCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		go func() {
			defer cancel()
			s.autoGenerateTitle(titleCtx, userID, conversationID, req.Content, assistantMsg.Content)
		}()
	}

	return assistantMsg, nil
}

// RegenerateMessage regenerates an assistant message
func (s *ChatService) RegenerateMessage(ctx context.Context, userID, conversationID, messageID uuid.UUID, stream bool, callback func(*models.StreamChunk) error) (*models.ConversationMessage, error) {
	// Get conversation
	conv, err := s.GetConversationWithMessages(ctx, userID, conversationID, 50)
	if err != nil {
		return nil, err
	}

	// Find the message to regenerate
	msg, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}
	if msg.ConversationID != conversationID {
		return nil, ErrNotAuthorized
	}
	if msg.Role != models.ConvRoleAssistant {
		return nil, errors.New("can only regenerate assistant messages")
	}

	// Delete the message and any messages after it
	if err := s.repo.DeleteMessagesAfter(ctx, conversationID, messageID); err != nil {
		return nil, fmt.Errorf("failed to delete messages: %w", err)
	}

	// Find the last user message
	var lastUserContent string
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		if conv.Messages[i].Role == models.ConvRoleUser && conv.Messages[i].ID != messageID {
			lastUserContent = conv.Messages[i].Content
			break
		}
	}

	if lastUserContent == "" {
		return nil, errors.New("no user message found to regenerate from")
	}

	// Send new message
	req := &models.SendChatMessageRequest{
		Content: lastUserContent,
		Stream:  stream,
	}

	if stream {
		return s.SendMessageStream(ctx, userID, conversationID, req, callback)
	}
	return s.SendMessage(ctx, userID, conversationID, req)
}

// GetConversationMessages gets messages for a conversation
func (s *ChatService) GetConversationMessages(ctx context.Context, userID, conversationID uuid.UUID, limit, offset int) ([]*models.ConversationMessage, error) {
	// Verify access
	_, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetConversationMessages(ctx, conversationID, limit, offset)
}

// DeleteMessage deletes a message
func (s *ChatService) DeleteMessage(ctx context.Context, userID, conversationID, messageID uuid.UUID) error {
	// Verify access
	_, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return err
	}

	msg, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return ErrMessageNotFound
	}
	if msg.ConversationID != conversationID {
		return ErrNotAuthorized
	}

	return s.repo.DeleteMessage(ctx, messageID)
}

// --- Templates ---

// GetTemplates lists available templates
func (s *ChatService) GetTemplates(ctx context.Context, userID *uuid.UUID, category string) ([]*models.AIChatTemplate, error) {
	return s.repo.ListTemplates(ctx, userID, true, category)
}

// GetTemplate gets a specific template
func (s *ChatService) GetTemplate(ctx context.Context, id uuid.UUID) (*models.AIChatTemplate, error) {
	return s.repo.GetTemplate(ctx, id)
}

// CreateTemplate creates a new template
func (s *ChatService) CreateTemplate(ctx context.Context, userID uuid.UUID, tmpl *models.AIChatTemplate) error {
	tmpl.ID = uuid.New()
	tmpl.UserID = &userID
	tmpl.CreatedAt = time.Now()
	tmpl.UpdatedAt = time.Now()
	return s.repo.CreateTemplate(ctx, tmpl)
}

// UpdateTemplate updates a template
func (s *ChatService) UpdateTemplate(ctx context.Context, userID uuid.UUID, tmpl *models.AIChatTemplate) error {
	existing, err := s.repo.GetTemplate(ctx, tmpl.ID)
	if err != nil {
		return ErrTemplateNotFound
	}
	if existing.UserID != nil && *existing.UserID != userID {
		return ErrNotAuthorized
	}
	tmpl.UpdatedAt = time.Now()
	return s.repo.UpdateTemplate(ctx, tmpl)
}

// DeleteTemplate deletes a template
func (s *ChatService) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	existing, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return ErrTemplateNotFound
	}
	if existing.UserID != nil && *existing.UserID != userID {
		return ErrNotAuthorized
	}
	return s.repo.DeleteTemplate(ctx, templateID)
}

// --- Shares ---

// ShareConversation creates a shareable link for a conversation
func (s *ChatService) ShareConversation(ctx context.Context, userID, conversationID uuid.UUID, isPublic, canContinue bool, expiresIn *time.Duration) (*models.AIConversationShare, error) {
	// Verify ownership
	_, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}

	// Generate share code
	codeBytes := make([]byte, 16)
	if _, err := rand.Read(codeBytes); err != nil {
		return nil, err
	}
	shareCode := hex.EncodeToString(codeBytes)

	share := &models.AIConversationShare{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SharedBy:       userID,
		ShareCode:      shareCode,
		IsPublic:       isPublic,
		CanContinue:    canContinue,
		CreatedAt:      time.Now(),
	}

	if expiresIn != nil {
		expires := time.Now().Add(*expiresIn)
		share.ExpiresAt = &expires
	}

	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// GetSharedConversation gets a conversation via share code
func (s *ChatService) GetSharedConversation(ctx context.Context, shareCode string) (*models.AIConversationWithMessages, error) {
	share, err := s.repo.GetShareByCode(ctx, shareCode)
	if err != nil {
		return nil, ErrShareNotFound
	}

	if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
		return nil, ErrShareExpired
	}

	// Increment view count (log error but don't fail)
	if err := s.repo.IncrementShareViewCount(ctx, share.ID); err != nil {
		log.Printf("Failed to increment share view count: %v", err)
	}

	conv, err := s.repo.GetConversationWithMessages(ctx, share.ConversationID, 100)
	if err != nil {
		return nil, ErrConversationNotFound
	}

	return conv, nil
}

// DeleteShare deletes a share
func (s *ChatService) DeleteShare(ctx context.Context, userID, shareID uuid.UUID) error {
	// We'd need to verify ownership through the share -> conversation -> user chain
	return s.repo.DeleteShare(ctx, shareID)
}

// --- Helpers ---

func (s *ChatService) buildMessagesForRequest(conv *models.AIConversationWithMessages, newContent string) []providers.Message {
	messages := make([]providers.Message, 0, len(conv.Messages)+2)

	// Add system prompt if present
	if conv.SystemPrompt != nil && *conv.SystemPrompt != "" {
		messages = append(messages, providers.Message{
			Role:    providers.RoleSystem,
			Content: *conv.SystemPrompt,
		})
	}

	// Add conversation history
	for _, msg := range conv.Messages {
		messages = append(messages, providers.Message{
			Role:    providers.Role(msg.Role),
			Content: msg.Content,
		})
	}

	// Add the new user message
	messages = append(messages, providers.Message{
		Role:    providers.RoleUser,
		Content: newContent,
	})

	return messages
}

func (s *ChatService) autoGenerateTitle(ctx context.Context, userID, conversationID uuid.UUID, userMsg, assistantMsg string) {
	// Use AI to generate a title
	prompt := fmt.Sprintf("Generate a very short title (max 6 words) for this conversation. Only respond with the title, no quotes or explanation.\n\nUser: %s\n\nAssistant: %s", userMsg, assistantMsg)

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{
				Role:    providers.RoleUser,
				Content: prompt,
			},
		},
		MaxTokens: 20,
	}

	resp, err := s.aiService.Chat(ctx, FeatureSummary, req, nil, &userID)
	if err != nil {
		return
	}

	if len(resp.Choices) > 0 {
		title := strings.TrimSpace(resp.Choices[0].Message.Content)
		title = strings.Trim(title, `"'`)
		if len(title) > 100 {
			title = title[:100]
		}
		if title != "" {
			conv, err := s.repo.GetConversation(ctx, conversationID)
			if err != nil {
				// Log but don't fail - this is background operation
				log.Printf("Failed to get conversation for title update: %v", err)
				return
			}
			if conv != nil {
				conv.Title = title
				conv.UpdatedAt = time.Now()
				if err := s.repo.UpdateConversation(ctx, conv); err != nil {
					// Log but don't fail - this is background operation
					log.Printf("Failed to update conversation title: %v", err)
				}
			}
		}
	}
}
