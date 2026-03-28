package ai

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// PostgresChatRepository implements ChatRepository using PostgreSQL
type PostgresChatRepository struct {
	db *sqlx.DB
}

// NewPostgresChatRepository creates a new PostgresChatRepository
func NewPostgresChatRepository(db *sqlx.DB) *PostgresChatRepository {
	return &PostgresChatRepository{db: db}
}

// --- Conversations ---

// CreateConversation creates a new conversation
func (r *PostgresChatRepository) CreateConversation(ctx context.Context, conv *models.AIConversation) error {
	query := `
		INSERT INTO ai_conversations (
			id, user_id, title, model_id, provider_id, system_prompt,
			temperature, max_tokens, is_archived, is_pinned, message_count,
			last_message_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	_, err := r.db.ExecContext(ctx, query,
		conv.ID, conv.UserID, conv.Title, conv.ModelID, conv.ProviderID,
		conv.SystemPrompt, conv.Temperature, conv.MaxTokens, conv.IsArchived,
		conv.IsPinned, conv.MessageCount, conv.LastMessageAt, conv.CreatedAt, conv.UpdatedAt,
	)
	return err
}

// GetConversation retrieves a conversation by ID
func (r *PostgresChatRepository) GetConversation(ctx context.Context, id uuid.UUID) (*models.AIConversation, error) {
	query := `SELECT * FROM ai_conversations WHERE id = $1`
	var conv models.AIConversation
	if err := r.db.GetContext(ctx, &conv, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	return &conv, nil
}

// GetConversationWithMessages retrieves a conversation with its messages
func (r *PostgresChatRepository) GetConversationWithMessages(ctx context.Context, id uuid.UUID, limit int) (*models.AIConversationWithMessages, error) {
	conv, err := r.GetConversation(ctx, id)
	if err != nil {
		return nil, err
	}

	messages, err := r.GetConversationMessages(ctx, id, limit, 0)
	if err != nil {
		return nil, err
	}

	// Convert pointers to values
	messageValues := make([]models.ConversationMessage, len(messages))
	for i, msg := range messages {
		messageValues[i] = *msg
	}

	return &models.AIConversationWithMessages{
		AIConversation: *conv,
		Messages:       messageValues,
	}, nil
}

// ListConversations lists conversations with filters
func (r *PostgresChatRepository) ListConversations(ctx context.Context, params *models.ConversationListParams) ([]*models.AIConversation, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
	args = append(args, params.UserID)
	argIdx++

	if !params.IncludeArchived {
		conditions = append(conditions, fmt.Sprintf("is_archived = $%d", argIdx))
		args = append(args, false)
		argIdx++
	}

	if params.OnlyPinned {
		conditions = append(conditions, fmt.Sprintf("is_pinned = $%d", argIdx))
		args = append(args, true)
		argIdx++
	}

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("title ILIKE $%d", argIdx))
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT * FROM ai_conversations 
		WHERE %s 
		ORDER BY is_pinned DESC, COALESCE(last_message_at, created_at) DESC
		LIMIT $%d OFFSET $%d`,
		strings.Join(conditions, " AND "), argIdx, argIdx+1)

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	args = append(args, limit, params.Offset)

	var conversations []*models.AIConversation
	if err := r.db.SelectContext(ctx, &conversations, query, args...); err != nil {
		return nil, err
	}
	return conversations, nil
}

// UpdateConversation updates a conversation
func (r *PostgresChatRepository) UpdateConversation(ctx context.Context, conv *models.AIConversation) error {
	query := `
		UPDATE ai_conversations SET
			title = $2,
			model_id = $3,
			provider_id = $4,
			system_prompt = $5,
			temperature = $6,
			max_tokens = $7,
			is_archived = $8,
			is_pinned = $9,
			updated_at = $10
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		conv.ID, conv.Title, conv.ModelID, conv.ProviderID, conv.SystemPrompt,
		conv.Temperature, conv.MaxTokens, conv.IsArchived, conv.IsPinned, conv.UpdatedAt,
	)
	return err
}

// DeleteConversation deletes a conversation
func (r *PostgresChatRepository) DeleteConversation(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_conversations WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// --- Messages ---

// CreateMessage creates a new message
func (r *PostgresChatRepository) CreateMessage(ctx context.Context, msg *models.ConversationMessage) error {
	query := `
		INSERT INTO ai_chat_messages (
			id, conversation_id, role, content, tool_calls, tool_call_id, name,
			tokens_used, model_used, provider_used, finish_reason, is_edited,
			parent_message_id, error_message, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`

	_, err := r.db.ExecContext(ctx, query,
		msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.ToolCalls,
		msg.ToolCallID, msg.Name, msg.TokensUsed, msg.ModelUsed, msg.ProviderUsed,
		msg.FinishReason, msg.IsEdited, msg.ParentMessageID, msg.ErrorMessage,
		msg.CreatedAt, msg.UpdatedAt,
	)
	return err
}

// GetMessage retrieves a message by ID
func (r *PostgresChatRepository) GetMessage(ctx context.Context, id uuid.UUID) (*models.ConversationMessage, error) {
	query := `SELECT * FROM ai_chat_messages WHERE id = $1`
	var msg models.ConversationMessage
	if err := r.db.GetContext(ctx, &msg, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	return &msg, nil
}

// GetConversationMessages retrieves messages for a conversation
func (r *PostgresChatRepository) GetConversationMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*models.ConversationMessage, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT * FROM ai_chat_messages 
		WHERE conversation_id = $1 
		ORDER BY created_at ASC 
		LIMIT $2 OFFSET $3`

	var messages []*models.ConversationMessage
	if err := r.db.SelectContext(ctx, &messages, query, conversationID, limit, offset); err != nil {
		return nil, err
	}
	return messages, nil
}

// UpdateMessage updates a message
func (r *PostgresChatRepository) UpdateMessage(ctx context.Context, msg *models.ConversationMessage) error {
	query := `
		UPDATE ai_chat_messages SET
			content = $2,
			is_edited = $3,
			updated_at = $4
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.Content, msg.IsEdited, msg.UpdatedAt)
	return err
}

// DeleteMessage deletes a message
func (r *PostgresChatRepository) DeleteMessage(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_chat_messages WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteMessagesAfter deletes a message and all messages after it
func (r *PostgresChatRepository) DeleteMessagesAfter(ctx context.Context, conversationID uuid.UUID, afterID uuid.UUID) error {
	// Get the created_at of the target message
	var createdAt sql.NullTime
	if err := r.db.GetContext(ctx, &createdAt,
		`SELECT created_at FROM ai_chat_messages WHERE id = $1`, afterID); err != nil {
		return err
	}

	query := `DELETE FROM ai_chat_messages WHERE conversation_id = $1 AND created_at >= $2`
	_, err := r.db.ExecContext(ctx, query, conversationID, createdAt.Time)
	return err
}

// --- Templates ---

// CreateTemplate creates a new template
func (r *PostgresChatRepository) CreateTemplate(ctx context.Context, tmpl *models.AIChatTemplate) error {
	query := `
		INSERT INTO ai_chat_templates (
			id, user_id, name, description, system_prompt, initial_messages,
			suggested_prompts, icon, category, is_public, usage_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	_, err := r.db.ExecContext(ctx, query,
		tmpl.ID, tmpl.UserID, tmpl.Name, tmpl.Description, tmpl.SystemPrompt,
		tmpl.InitialMessages, tmpl.SuggestedPrompts, tmpl.Icon, tmpl.Category,
		tmpl.IsPublic, tmpl.UsageCount, tmpl.CreatedAt, tmpl.UpdatedAt,
	)
	return err
}

// GetTemplate retrieves a template by ID
func (r *PostgresChatRepository) GetTemplate(ctx context.Context, id uuid.UUID) (*models.AIChatTemplate, error) {
	query := `SELECT * FROM ai_chat_templates WHERE id = $1`
	var tmpl models.AIChatTemplate
	if err := r.db.GetContext(ctx, &tmpl, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	return &tmpl, nil
}

// ListTemplates lists templates
func (r *PostgresChatRepository) ListTemplates(ctx context.Context, userID *uuid.UUID, includePublic bool, category string) ([]*models.AIChatTemplate, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if userID != nil {
		if includePublic {
			conditions = append(conditions, fmt.Sprintf("(user_id = $%d OR is_public = true)", argIdx))
		} else {
			conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		}
		args = append(args, *userID)
		argIdx++
	} else if includePublic {
		conditions = append(conditions, "is_public = true")
	}

	if category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, category)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT * FROM ai_chat_templates 
		%s 
		ORDER BY usage_count DESC, created_at DESC`, where)

	var templates []*models.AIChatTemplate
	if err := r.db.SelectContext(ctx, &templates, query, args...); err != nil {
		return nil, err
	}
	return templates, nil
}

// UpdateTemplate updates a template
func (r *PostgresChatRepository) UpdateTemplate(ctx context.Context, tmpl *models.AIChatTemplate) error {
	query := `
		UPDATE ai_chat_templates SET
			name = $2,
			description = $3,
			system_prompt = $4,
			initial_messages = $5,
			suggested_prompts = $6,
			icon = $7,
			category = $8,
			is_public = $9,
			updated_at = $10
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		tmpl.ID, tmpl.Name, tmpl.Description, tmpl.SystemPrompt,
		tmpl.InitialMessages, tmpl.SuggestedPrompts, tmpl.Icon,
		tmpl.Category, tmpl.IsPublic, tmpl.UpdatedAt,
	)
	return err
}

// DeleteTemplate deletes a template
func (r *PostgresChatRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_chat_templates WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// IncrementTemplateUsage increments the usage count
func (r *PostgresChatRepository) IncrementTemplateUsage(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE ai_chat_templates SET usage_count = usage_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// --- Shares ---

// CreateShare creates a new share
func (r *PostgresChatRepository) CreateShare(ctx context.Context, share *models.AIConversationShare) error {
	query := `
		INSERT INTO ai_conversation_shares (
			id, conversation_id, shared_by, share_code, is_public,
			can_continue, expires_at, view_count, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err := r.db.ExecContext(ctx, query,
		share.ID, share.ConversationID, share.SharedBy, share.ShareCode,
		share.IsPublic, share.CanContinue, share.ExpiresAt, share.ViewCount,
		share.CreatedAt,
	)
	return err
}

// GetShareByCode retrieves a share by its code
func (r *PostgresChatRepository) GetShareByCode(ctx context.Context, code string) (*models.AIConversationShare, error) {
	query := `SELECT * FROM ai_conversation_shares WHERE share_code = $1`
	var share models.AIConversationShare
	if err := r.db.GetContext(ctx, &share, query, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrShareNotFound
		}
		return nil, err
	}
	return &share, nil
}

// GetConversationShares retrieves all shares for a conversation
func (r *PostgresChatRepository) GetConversationShares(ctx context.Context, conversationID uuid.UUID) ([]*models.AIConversationShare, error) {
	query := `SELECT * FROM ai_conversation_shares WHERE conversation_id = $1 ORDER BY created_at DESC`
	var shares []*models.AIConversationShare
	if err := r.db.SelectContext(ctx, &shares, query, conversationID); err != nil {
		return nil, err
	}
	return shares, nil
}

// DeleteShare deletes a share
func (r *PostgresChatRepository) DeleteShare(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_conversation_shares WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// IncrementShareViewCount increments the view count
func (r *PostgresChatRepository) IncrementShareViewCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE ai_conversation_shares SET view_count = view_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
