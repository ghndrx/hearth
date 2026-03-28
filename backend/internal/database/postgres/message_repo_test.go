package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func setupMessageRepoMock(t *testing.T) (*MessageRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewMessageRepository(sqlxDB)
	return repo, mock
}

var messageColumns = []string{
	"id", "channel_id", "author_id", "content", "encrypted_content",
	"type", "edited_at", "pinned", "pinned_at", "tts",
	"mentions_everyone", "mention_everyone", "reply_to_id", "thread_id", "flags", "created_at",
}

func TestMessageRepository_Create(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msg := &models.Message{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		AuthorID:  uuid.New(),
		Content:   "Hello, world!",
		Type:      models.MessageTypeDefault,
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO messages").
		WithArgs(
			msg.ID, msg.ChannelID, msg.AuthorID, msg.Content,
			msg.EncryptedContent, msg.Type, msg.ReplyToID, msg.Pinned,
			msg.TTS, msg.Flags, msg.CreatedAt, msg.EditedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, msg)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_Create_WithMentions(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()
	msg := &models.Message{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		AuthorID:  uuid.New(),
		Content:   "Hey @user1 @user2",
		Type:      models.MessageTypeDefault,
		Mentions:  []uuid.UUID{user1, user2},
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO messages").
		WithArgs(
			msg.ID, msg.ChannelID, msg.AuthorID, msg.Content,
			msg.EncryptedContent, msg.Type, msg.ReplyToID, msg.Pinned,
			msg.TTS, msg.Flags, msg.CreatedAt, msg.EditedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO message_mentions").
		WithArgs(msg.ID, user1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO message_mentions").
		WithArgs(msg.ID, user2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, msg)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_Create_MentionInsertFails(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()
	msg := &models.Message{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		AuthorID:  uuid.New(),
		Content:   "Hey @user1 @user2",
		Type:      models.MessageTypeDefault,
		Mentions:  []uuid.UUID{user1, user2},
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO messages").
		WithArgs(
			msg.ID, msg.ChannelID, msg.AuthorID, msg.Content,
			msg.EncryptedContent, msg.Type, msg.ReplyToID, msg.Pinned,
			msg.TTS, msg.Flags, msg.CreatedAt, msg.EditedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// First mention succeeds
	mock.ExpectExec("INSERT INTO message_mentions").
		WithArgs(msg.ID, user1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Second mention fails
	mock.ExpectExec("INSERT INTO message_mentions").
		WithArgs(msg.ID, user2).
		WillReturnError(fmt.Errorf("foreign key violation"))

	err := repo.Create(ctx, msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 related insert(s) failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_Create_WithAttachments(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	att1 := models.Attachment{
		ID:       uuid.New(),
		Filename: "image.png",
		URL:      "https://cdn.example.com/image.png",
		Size:     1024,
	}
	msg := &models.Message{
		ID:          uuid.New(),
		ChannelID:   uuid.New(),
		AuthorID:    uuid.New(),
		Content:     "Check this out",
		Type:        models.MessageTypeDefault,
		Attachments: []models.Attachment{att1},
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO messages").
		WithArgs(
			msg.ID, msg.ChannelID, msg.AuthorID, msg.Content,
			msg.EncryptedContent, msg.Type, msg.ReplyToID, msg.Pinned,
			msg.TTS, msg.Flags, msg.CreatedAt, msg.EditedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO attachments").
		WithArgs(att1.ID, msg.ID, att1.Filename, att1.URL, att1.ContentType, att1.Size, att1.AltText).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, msg)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_Create_AttachmentInsertFails(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	att1 := models.Attachment{
		ID:       uuid.New(),
		Filename: "image.png",
		URL:      "https://cdn.example.com/image.png",
		Size:     1024,
	}
	msg := &models.Message{
		ID:          uuid.New(),
		ChannelID:   uuid.New(),
		AuthorID:    uuid.New(),
		Content:     "Check this out",
		Type:        models.MessageTypeDefault,
		Attachments: []models.Attachment{att1},
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO messages").
		WithArgs(
			msg.ID, msg.ChannelID, msg.AuthorID, msg.Content,
			msg.EncryptedContent, msg.Type, msg.ReplyToID, msg.Pinned,
			msg.TTS, msg.Flags, msg.CreatedAt, msg.EditedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO attachments").
		WithArgs(att1.ID, msg.ID, att1.Filename, att1.URL, att1.ContentType, att1.Size, att1.AltText).
		WillReturnError(fmt.Errorf("storage error"))

	err := repo.Create(ctx, msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 related insert(s) failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_Create_MentionAndAttachmentBothFail(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	user1 := uuid.New()
	att1 := models.Attachment{
		ID:       uuid.New(),
		Filename: "file.txt",
		URL:      "https://cdn.example.com/file.txt",
		Size:     512,
	}
	msg := &models.Message{
		ID:          uuid.New(),
		ChannelID:   uuid.New(),
		AuthorID:    uuid.New(),
		Content:     "Hey @user",
		Type:        models.MessageTypeDefault,
		Mentions:    []uuid.UUID{user1},
		Attachments: []models.Attachment{att1},
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO messages").
		WithArgs(
			msg.ID, msg.ChannelID, msg.AuthorID, msg.Content,
			msg.EncryptedContent, msg.Type, msg.ReplyToID, msg.Pinned,
			msg.TTS, msg.Flags, msg.CreatedAt, msg.EditedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO message_mentions").
		WithArgs(msg.ID, user1).
		WillReturnError(fmt.Errorf("mention error"))

	mock.ExpectExec("INSERT INTO attachments").
		WithArgs(att1.ID, msg.ID, att1.Filename, att1.URL, att1.ContentType, att1.Size, att1.AltText).
		WillReturnError(fmt.Errorf("attachment error"))

	err := repo.Create(ctx, msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2 related insert(s) failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_Create_MessageInsertFails(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msg := &models.Message{
		ID:        uuid.New(),
		ChannelID: uuid.New(),
		AuthorID:  uuid.New(),
		Content:   "Hello",
		Type:      models.MessageTypeDefault,
		Mentions:  []uuid.UUID{uuid.New()},
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO messages").
		WithArgs(
			msg.ID, msg.ChannelID, msg.AuthorID, msg.Content,
			msg.EncryptedContent, msg.Type, msg.ReplyToID, msg.Pinned,
			msg.TTS, msg.Flags, msg.CreatedAt, msg.EditedAt,
		).
		WillReturnError(fmt.Errorf("insert failed"))

	err := repo.Create(ctx, msg)
	require.Error(t, err)
	assert.Equal(t, "insert failed", err.Error())
	// Mentions should not be attempted since the message insert failed
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_GetByID(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()
	channelID := uuid.New()
	authorID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows(messageColumns).AddRow(
		msgID, channelID, authorID, "Hello!", "",
		models.MessageTypeDefault, nil, false, nil, false,
		false, false, nil, nil, 0, now,
	)

	mock.ExpectQuery("SELECT .+ FROM messages WHERE id = \\$1").
		WithArgs(msgID).
		WillReturnRows(rows)

	// Attachment query
	attRows := sqlmock.NewRows([]string{
		"id", "message_id", "filename", "url", "proxy_url", "size",
		"content_type", "width", "height", "alt_text", "ephemeral",
		"encrypted", "encrypted_key", "iv", "created_at",
	})
	mock.ExpectQuery("SELECT \\* FROM attachments WHERE message_id = \\$1").
		WithArgs(msgID).
		WillReturnRows(attRows)

	msg, err := repo.GetByID(ctx, msgID)
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, msgID, msg.ID)
	assert.Equal(t, channelID, msg.ChannelID)
	assert.Equal(t, "Hello!", msg.Content)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM messages WHERE id = \\$1").
		WithArgs(msgID).
		WillReturnError(sql.ErrNoRows)

	msg, err := repo.GetByID(ctx, msgID)
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_GetByID_DBError(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM messages WHERE id = \\$1").
		WithArgs(msgID).
		WillReturnError(fmt.Errorf("connection refused"))

	msg, err := repo.GetByID(ctx, msgID)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_GetByID_WithAttachments(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()
	channelID := uuid.New()
	authorID := uuid.New()
	attID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows(messageColumns).AddRow(
		msgID, channelID, authorID, "Check this", "",
		models.MessageTypeDefault, nil, false, nil, false,
		false, false, nil, nil, 0, now,
	)

	mock.ExpectQuery("SELECT .+ FROM messages WHERE id = \\$1").
		WithArgs(msgID).
		WillReturnRows(rows)

	attRows := sqlmock.NewRows([]string{
		"id", "message_id", "filename", "url", "proxy_url", "size",
		"content_type", "width", "height", "alt_text", "ephemeral",
		"encrypted", "encrypted_key", "iv", "created_at",
	}).AddRow(
		attID, msgID, "photo.jpg", "https://cdn.example.com/photo.jpg", nil, int64(2048),
		nil, nil, nil, nil, false,
		false, "", "", now,
	)

	mock.ExpectQuery("SELECT \\* FROM attachments WHERE message_id = \\$1").
		WithArgs(msgID).
		WillReturnRows(attRows)

	msg, err := repo.GetByID(ctx, msgID)
	require.NoError(t, err)
	require.NotNil(t, msg)
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "photo.jpg", msg.Attachments[0].Filename)
	assert.Equal(t, int64(2048), msg.Attachments[0].Size)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_Update(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	msg := &models.Message{
		ID:       uuid.New(),
		Content:  "Updated content",
		Pinned:   true,
		EditedAt: &now,
		Flags:    0,
	}

	mock.ExpectExec("UPDATE messages SET").
		WithArgs(msg.ID, msg.Content, msg.Pinned, msg.EditedAt, msg.Flags).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(ctx, msg)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_Delete(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()

	mock.ExpectExec("DELETE FROM messages WHERE id = \\$1").
		WithArgs(msgID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, msgID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_BulkDeleteMessages_Empty(t *testing.T) {
	repo, _ := setupMessageRepoMock(t)
	ctx := context.Background()

	err := repo.BulkDeleteMessages(ctx, []uuid.UUID{})
	require.NoError(t, err)
}

func TestMessageRepository_BulkDeleteMessages_NilSlice(t *testing.T) {
	repo, _ := setupMessageRepoMock(t)
	ctx := context.Background()

	err := repo.BulkDeleteMessages(ctx, nil)
	require.NoError(t, err)
}

func TestMessageRepository_AddReaction(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("INSERT INTO reactions").
		WithArgs(msgID, userID, "👍", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.AddReaction(ctx, msgID, userID, "👍")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_RemoveReaction(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM reactions WHERE message_id = \\$1 AND user_id = \\$2 AND emoji = \\$3").
		WithArgs(msgID, userID, "👍").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RemoveReaction(ctx, msgID, userID, "👍")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_GetReactions(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()

	rows := sqlmock.NewRows([]string{"message_id", "emoji", "count"}).
		AddRow(msgID, "👍", 3).
		AddRow(msgID, "❤️", 1)

	mock.ExpectQuery("SELECT message_id, emoji, COUNT").
		WithArgs(msgID).
		WillReturnRows(rows)

	reactions, err := repo.GetReactions(ctx, msgID)
	require.NoError(t, err)
	require.Len(t, reactions, 2)
	assert.Equal(t, "👍", reactions[0].Emoji)
	assert.Equal(t, 3, reactions[0].Count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_DeleteByChannel(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()

	mock.ExpectExec("DELETE FROM messages WHERE channel_id = \\$1").
		WithArgs(channelID).
		WillReturnResult(sqlmock.NewResult(0, 5))

	err := repo.DeleteByChannel(ctx, channelID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_DeleteByAuthor(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	authorID := uuid.New()
	since := time.Now().Add(-24 * time.Hour)

	mock.ExpectExec("DELETE FROM messages WHERE channel_id = \\$1 AND author_id = \\$2 AND created_at >= \\$3").
		WithArgs(channelID, authorID, since).
		WillReturnResult(sqlmock.NewResult(0, 7))

	count, err := repo.DeleteByAuthor(ctx, channelID, authorID, since)
	require.NoError(t, err)
	assert.Equal(t, 7, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_GetPinnedMessages(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	msgID := uuid.New()
	authorID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows(messageColumns).AddRow(
		msgID, channelID, authorID, "Pinned message", "",
		models.MessageTypeDefault, nil, true, &now, false,
		false, false, nil, nil, 0, now,
	)

	mock.ExpectQuery("SELECT .+ FROM messages WHERE channel_id = \\$1 AND pinned = true").
		WithArgs(channelID).
		WillReturnRows(rows)

	messages, err := repo.GetPinnedMessages(ctx, channelID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.True(t, messages[0].Pinned)
	assert.Equal(t, "Pinned message", messages[0].Content)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_GetReactionUsers(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"message_id", "user_id", "emoji", "created_at"}).
		AddRow(msgID, userID, "👍", now)

	mock.ExpectQuery("SELECT message_id, user_id, emoji, created_at FROM reactions").
		WithArgs(msgID, "👍", 10).
		WillReturnRows(rows)

	users, err := repo.GetReactionUsers(ctx, msgID, "👍", 10)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, userID, users[0].UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_GetUserReactions(t *testing.T) {
	repo, mock := setupMessageRepoMock(t)
	ctx := context.Background()

	msgID := uuid.New()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"emoji"}).
		AddRow("👍").
		AddRow("❤️")

	mock.ExpectQuery("SELECT emoji FROM reactions WHERE message_id = \\$1 AND user_id = \\$2").
		WithArgs(msgID, userID).
		WillReturnRows(rows)

	emojis, err := repo.GetUserReactions(ctx, msgID, userID)
	require.NoError(t, err)
	assert.Len(t, emojis, 2)
	assert.Contains(t, emojis, "👍")
	assert.Contains(t, emojis, "❤️")
	assert.NoError(t, mock.ExpectationsWereMet())
}
