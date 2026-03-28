package ai

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

// Helper to create a mock DB
func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db := sqlx.NewDb(mockDB, "postgres")
	return db, mock
}

// --- Conversation Repository Tests ---

func TestPostgresChatRepository_CreateConversation(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	conv := &models.AIConversation{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Title:       "Test Conversation",
		Temperature: 0.7,
		MaxTokens:   2048,
		IsArchived:  false,
		IsPinned:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO ai_conversations").
		WithArgs(
			conv.ID, conv.UserID, conv.Title, conv.ModelID, conv.ProviderID,
			conv.SystemPrompt, conv.Temperature, conv.MaxTokens, conv.IsArchived,
			conv.IsPinned, conv.MessageCount, conv.LastMessageAt, conv.CreatedAt, conv.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateConversation(context.Background(), conv)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_GetConversation(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	convID := uuid.New()
	userID := uuid.New()

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "title", "model_id", "provider_id", "system_prompt",
			"temperature", "max_tokens", "is_archived", "is_pinned", "message_count",
			"last_message_at", "created_at", "updated_at",
		}).AddRow(
			convID, userID, "Test Conv", nil, nil, nil,
			0.7, 2048, false, false, 0,
			nil, time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT \\* FROM ai_conversations WHERE id = \\$1").
			WithArgs(convID).
			WillReturnRows(rows)

		conv, err := repo.GetConversation(context.Background(), convID)
		require.NoError(t, err)
		assert.Equal(t, convID, conv.ID)
		assert.Equal(t, "Test Conv", conv.Title)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM ai_conversations WHERE id = \\$1").
			WithArgs(convID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetConversation(context.Background(), convID)
		assert.Equal(t, ErrConversationNotFound, err)
	})
}

func TestPostgresChatRepository_UpdateConversation(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	conv := &models.AIConversation{
		ID:          uuid.New(),
		Title:       "Updated Title",
		Temperature: 0.8,
		MaxTokens:   4096,
		IsArchived:  true,
		IsPinned:    true,
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec("UPDATE ai_conversations SET").
		WithArgs(
			conv.ID, conv.Title, conv.ModelID, conv.ProviderID, conv.SystemPrompt,
			conv.Temperature, conv.MaxTokens, conv.IsArchived, conv.IsPinned, conv.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateConversation(context.Background(), conv)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_DeleteConversation(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	convID := uuid.New()

	mock.ExpectExec("DELETE FROM ai_conversations WHERE id = \\$1").
		WithArgs(convID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteConversation(context.Background(), convID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_ListConversations(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	userID := uuid.New()

	t.Run("basic list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "title", "model_id", "provider_id", "system_prompt",
			"temperature", "max_tokens", "is_archived", "is_pinned", "message_count",
			"last_message_at", "created_at", "updated_at",
		}).
			AddRow(uuid.New(), userID, "Conv 1", nil, nil, nil, 0.7, 2048, false, false, 5, time.Now(), time.Now(), time.Now()).
			AddRow(uuid.New(), userID, "Conv 2", nil, nil, nil, 0.7, 2048, false, true, 10, time.Now(), time.Now(), time.Now())

		mock.ExpectQuery("SELECT \\* FROM ai_conversations").
			WillReturnRows(rows)

		params := &models.ConversationListParams{
			UserID: userID,
			Limit:  50,
		}

		convs, err := repo.ListConversations(context.Background(), params)
		require.NoError(t, err)
		assert.Len(t, convs, 2)
	})

	t.Run("with search filter", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "title", "model_id", "provider_id", "system_prompt",
			"temperature", "max_tokens", "is_archived", "is_pinned", "message_count",
			"last_message_at", "created_at", "updated_at",
		}).AddRow(uuid.New(), userID, "Search Result", nil, nil, nil, 0.7, 2048, false, false, 1, time.Now(), time.Now(), time.Now())

		mock.ExpectQuery("SELECT \\* FROM ai_conversations").
			WillReturnRows(rows)

		params := &models.ConversationListParams{
			UserID: userID,
			Search: "Search",
			Limit:  50,
		}

		convs, err := repo.ListConversations(context.Background(), params)
		require.NoError(t, err)
		assert.Len(t, convs, 1)
	})
}

// --- Message Repository Tests ---

func TestPostgresChatRepository_CreateMessage(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	msg := &models.ConversationMessage{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		Role:           models.ConvRoleUser,
		Content:        "Hello!",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	mock.ExpectExec("INSERT INTO ai_chat_messages").
		WithArgs(
			msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.ToolCalls,
			msg.ToolCallID, msg.Name, msg.TokensUsed, msg.ModelUsed, msg.ProviderUsed,
			msg.FinishReason, msg.IsEdited, msg.ParentMessageID, msg.ErrorMessage,
			msg.CreatedAt, msg.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_GetMessage(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	msgID := uuid.New()
	convID := uuid.New()

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "conversation_id", "role", "content", "tool_calls", "tool_call_id",
			"name", "tokens_used", "model_used", "provider_used", "finish_reason",
			"is_edited", "parent_message_id", "error_message", "created_at", "updated_at",
		}).AddRow(
			msgID, convID, "user", "Test message", nil, nil,
			nil, nil, nil, nil, nil,
			false, nil, nil, time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT \\* FROM ai_chat_messages WHERE id = \\$1").
			WithArgs(msgID).
			WillReturnRows(rows)

		msg, err := repo.GetMessage(context.Background(), msgID)
		require.NoError(t, err)
		assert.Equal(t, msgID, msg.ID)
		assert.Equal(t, "Test message", msg.Content)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM ai_chat_messages WHERE id = \\$1").
			WithArgs(msgID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetMessage(context.Background(), msgID)
		assert.Equal(t, ErrMessageNotFound, err)
	})
}

func TestPostgresChatRepository_GetConversationMessages(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	convID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "conversation_id", "role", "content", "tool_calls", "tool_call_id",
		"name", "tokens_used", "model_used", "provider_used", "finish_reason",
		"is_edited", "parent_message_id", "error_message", "created_at", "updated_at",
	}).
		AddRow(uuid.New(), convID, "user", "Hello", nil, nil, nil, nil, nil, nil, nil, false, nil, nil, time.Now(), time.Now()).
		AddRow(uuid.New(), convID, "assistant", "Hi there!", nil, nil, nil, 50, "gpt-4", "openai", "stop", false, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery("SELECT \\* FROM ai_chat_messages").
		WithArgs(convID, 50, 0).
		WillReturnRows(rows)

	messages, err := repo.GetConversationMessages(context.Background(), convID, 50, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, models.ConvRoleUser, messages[0].Role)
	assert.Equal(t, models.ConvRoleAssistant, messages[1].Role)
}

func TestPostgresChatRepository_DeleteMessage(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	msgID := uuid.New()

	mock.ExpectExec("DELETE FROM ai_chat_messages WHERE id = \\$1").
		WithArgs(msgID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteMessage(context.Background(), msgID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- Template Repository Tests ---

func TestPostgresChatRepository_CreateTemplate(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	userID := uuid.New()
	tmpl := &models.AIChatTemplate{
		ID:           uuid.New(),
		UserID:       &userID,
		Name:         "Test Template",
		SystemPrompt: "You are a test assistant",
		IsPublic:     false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mock.ExpectExec("INSERT INTO ai_chat_templates").
		WithArgs(
			tmpl.ID, tmpl.UserID, tmpl.Name, tmpl.Description, tmpl.SystemPrompt,
			tmpl.InitialMessages, tmpl.SuggestedPrompts, tmpl.Icon, tmpl.Category,
			tmpl.IsPublic, tmpl.UsageCount, tmpl.CreatedAt, tmpl.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateTemplate(context.Background(), tmpl)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_GetTemplate(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	tmplID := uuid.New()

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "system_prompt",
			"initial_messages", "suggested_prompts", "icon", "category",
			"is_public", "usage_count", "created_at", "updated_at",
		}).AddRow(
			tmplID, nil, "Public Template", "Description", "System prompt",
			nil, nil, "💬", "general",
			true, 100, time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT \\* FROM ai_chat_templates WHERE id = \\$1").
			WithArgs(tmplID).
			WillReturnRows(rows)

		tmpl, err := repo.GetTemplate(context.Background(), tmplID)
		require.NoError(t, err)
		assert.Equal(t, "Public Template", tmpl.Name)
		assert.True(t, tmpl.IsPublic)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM ai_chat_templates WHERE id = \\$1").
			WithArgs(tmplID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetTemplate(context.Background(), tmplID)
		assert.Equal(t, ErrTemplateNotFound, err)
	})
}

func TestPostgresChatRepository_ListTemplates(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	userID := uuid.New()

	t.Run("user templates with public", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "system_prompt",
			"initial_messages", "suggested_prompts", "icon", "category",
			"is_public", "usage_count", "created_at", "updated_at",
		}).
			AddRow(uuid.New(), nil, "Public", nil, "prompt", nil, nil, nil, "general", true, 50, time.Now(), time.Now()).
			AddRow(uuid.New(), userID, "Private", nil, "prompt", nil, nil, nil, "custom", false, 10, time.Now(), time.Now())

		mock.ExpectQuery("SELECT \\* FROM ai_chat_templates").
			WillReturnRows(rows)

		templates, err := repo.ListTemplates(context.Background(), &userID, true, "")
		require.NoError(t, err)
		assert.Len(t, templates, 2)
	})

	t.Run("with category filter", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_id", "name", "description", "system_prompt",
			"initial_messages", "suggested_prompts", "icon", "category",
			"is_public", "usage_count", "created_at", "updated_at",
		}).AddRow(uuid.New(), nil, "Coding Template", nil, "prompt", nil, nil, "💻", "coding", true, 100, time.Now(), time.Now())

		mock.ExpectQuery("SELECT \\* FROM ai_chat_templates").
			WillReturnRows(rows)

		templates, err := repo.ListTemplates(context.Background(), nil, true, "coding")
		require.NoError(t, err)
		assert.Len(t, templates, 1)
		assert.Equal(t, "Coding Template", templates[0].Name)
	})
}

func TestPostgresChatRepository_IncrementTemplateUsage(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	tmplID := uuid.New()

	mock.ExpectExec("UPDATE ai_chat_templates SET usage_count = usage_count \\+ 1").
		WithArgs(tmplID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.IncrementTemplateUsage(context.Background(), tmplID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- Share Repository Tests ---

func TestPostgresChatRepository_CreateShare(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	share := &models.AIConversationShare{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		SharedBy:       uuid.New(),
		ShareCode:      "abc123xyz",
		IsPublic:       true,
		CanContinue:    false,
		ViewCount:      0,
		CreatedAt:      time.Now(),
	}

	mock.ExpectExec("INSERT INTO ai_conversation_shares").
		WithArgs(
			share.ID, share.ConversationID, share.SharedBy, share.ShareCode,
			share.IsPublic, share.CanContinue, share.ExpiresAt, share.ViewCount,
			share.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateShare(context.Background(), share)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_GetShareByCode(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	shareCode := "testcode123"

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "conversation_id", "shared_by", "share_code",
			"is_public", "can_continue", "expires_at", "view_count", "created_at",
		}).AddRow(
			uuid.New(), uuid.New(), uuid.New(), shareCode,
			true, false, nil, 5, time.Now(),
		)

		mock.ExpectQuery("SELECT \\* FROM ai_conversation_shares WHERE share_code = \\$1").
			WithArgs(shareCode).
			WillReturnRows(rows)

		share, err := repo.GetShareByCode(context.Background(), shareCode)
		require.NoError(t, err)
		assert.Equal(t, shareCode, share.ShareCode)
		assert.True(t, share.IsPublic)
		assert.Equal(t, 5, share.ViewCount)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM ai_conversation_shares WHERE share_code = \\$1").
			WithArgs(shareCode).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetShareByCode(context.Background(), shareCode)
		assert.Equal(t, ErrShareNotFound, err)
	})
}

func TestPostgresChatRepository_IncrementShareViewCount(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	shareID := uuid.New()

	mock.ExpectExec("UPDATE ai_conversation_shares SET view_count = view_count \\+ 1").
		WithArgs(shareID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.IncrementShareViewCount(context.Background(), shareID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_DeleteShare(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	shareID := uuid.New()

	mock.ExpectExec("DELETE FROM ai_conversation_shares WHERE id = \\$1").
		WithArgs(shareID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteShare(context.Background(), shareID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Edge Cases

func TestPostgresChatRepository_DeleteMessagesAfter(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	convID := uuid.New()
	afterID := uuid.New()
	createdAt := time.Now()

	// First query gets the created_at of the target message
	mock.ExpectQuery("SELECT created_at FROM ai_chat_messages WHERE id = \\$1").
		WithArgs(afterID).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	// Second query deletes messages after that time
	mock.ExpectExec("DELETE FROM ai_chat_messages WHERE conversation_id = \\$1 AND created_at >= \\$2").
		WithArgs(convID, createdAt).
		WillReturnResult(sqlmock.NewResult(1, 3))

	err := repo.DeleteMessagesAfter(context.Background(), convID, afterID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_GetConversationShares(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	convID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "conversation_id", "shared_by", "share_code",
		"is_public", "can_continue", "expires_at", "view_count", "created_at",
	}).
		AddRow(uuid.New(), convID, uuid.New(), "code1", true, false, nil, 10, time.Now()).
		AddRow(uuid.New(), convID, uuid.New(), "code2", false, true, nil, 5, time.Now())

	mock.ExpectQuery("SELECT \\* FROM ai_conversation_shares WHERE conversation_id = \\$1").
		WithArgs(convID).
		WillReturnRows(rows)

	shares, err := repo.GetConversationShares(context.Background(), convID)
	require.NoError(t, err)
	assert.Len(t, shares, 2)
}

func TestPostgresChatRepository_UpdateMessage(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	msg := &models.ConversationMessage{
		ID:        uuid.New(),
		Content:   "Edited content",
		IsEdited:  true,
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec("UPDATE ai_chat_messages SET").
		WithArgs(msg.ID, msg.Content, msg.IsEdited, msg.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_UpdateTemplate(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	tmpl := &models.AIChatTemplate{
		ID:           uuid.New(),
		Name:         "Updated Template",
		SystemPrompt: "New prompt",
		IsPublic:     true,
		UpdatedAt:    time.Now(),
	}

	mock.ExpectExec("UPDATE ai_chat_templates SET").
		WithArgs(
			tmpl.ID, tmpl.Name, tmpl.Description, tmpl.SystemPrompt,
			tmpl.InitialMessages, tmpl.SuggestedPrompts, tmpl.Icon,
			tmpl.Category, tmpl.IsPublic, tmpl.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateTemplate(context.Background(), tmpl)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresChatRepository_DeleteTemplate(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewPostgresChatRepository(db)
	tmplID := uuid.New()

	mock.ExpectExec("DELETE FROM ai_chat_templates WHERE id = \\$1").
		WithArgs(tmplID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteTemplate(context.Background(), tmplID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
