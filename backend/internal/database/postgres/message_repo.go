package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

type MessageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, message *models.Message) error {
	query := `
		INSERT INTO messages (id, channel_id, author_id, content, encrypted_content, type, reply_to_id, pinned, tts, flags, created_at, edited_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		message.ID, message.ChannelID, message.AuthorID, message.Content,
		message.EncryptedContent, message.Type, message.ReplyToID, message.Pinned,
		message.TTS, message.Flags, message.CreatedAt, message.EditedAt,
	)
	if err != nil {
		return err
	}

	// Insert mentions
	var errs []error
	if len(message.Mentions) > 0 {
		for _, userID := range message.Mentions {
			_, err = r.db.ExecContext(ctx,
				`INSERT INTO message_mentions (message_id, user_id) VALUES ($1, $2)`,
				message.ID, userID,
			)
			if err != nil {
				log.Printf("failed to insert mention for message %s, user %s: %v", message.ID, userID, err)
				errs = append(errs, err)
			}
		}
	}

	// Insert attachments
	if len(message.Attachments) > 0 {
		for _, att := range message.Attachments {
			_, err = r.db.ExecContext(ctx,
				`INSERT INTO attachments (id, message_id, filename, url, content_type, size, alt_text) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				att.ID, message.ID, att.Filename, att.URL, att.ContentType, att.Size, att.AltText,
			)
			if err != nil {
				log.Printf("failed to insert attachment %s for message %s: %v", att.ID, message.ID, err)
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("message created but %d related insert(s) failed: %w", len(errs), errs[0])
	}

	return nil
}

func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	var message models.Message
	query := `SELECT id, channel_id, author_id, content, encrypted_content, type, edited_at, pinned, pinned_at, tts, mentions_everyone, mention_everyone, reply_to_id, thread_id, flags, created_at FROM messages WHERE id = $1`
	err := r.db.GetContext(ctx, &message, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Load attachments
	var attachments []models.Attachment
	_ = r.db.SelectContext(ctx, &attachments, `SELECT * FROM attachments WHERE message_id = $1`, id)
	message.Attachments = attachments

	return &message, nil
}

func (r *MessageRepository) Update(ctx context.Context, message *models.Message) error {
	query := `
		UPDATE messages SET content = $2, pinned = $3, edited_at = $4, flags = $5
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		message.ID, message.Content, message.Pinned, message.EditedAt, message.Flags,
	)
	return err
}

func (r *MessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM messages WHERE id = $1`, id)
	return err
}

// CountRepliesTo counts how many messages reply to a given message ID
func (r *MessageRepository) CountRepliesTo(ctx context.Context, messageID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM messages WHERE reply_to_id = $1`, messageID)
	return count, err
}

func (r *MessageRepository) BulkDeleteMessages(ctx context.Context, messageIDs []uuid.UUID) error {
	if len(messageIDs) == 0 {
		return nil
	}
	query := `DELETE FROM messages WHERE id = ANY($1)`
	_, err := r.db.ExecContext(ctx, query, messageIDs)
	return err
}

func (r *MessageRepository) GetChannelMessages(ctx context.Context, channelID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
	var messages []*models.Message
	var query string
	var args []interface{}

	if before != nil {
		query = `
			SELECT id, channel_id, author_id, content, encrypted_content, type, edited_at, pinned, pinned_at, tts, mentions_everyone, mention_everyone, reply_to_id, thread_id, flags, created_at FROM messages 
			WHERE channel_id = $1 AND id < $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []interface{}{channelID, *before, limit}
	} else if after != nil {
		query = `
			SELECT id, channel_id, author_id, content, encrypted_content, type, edited_at, pinned, pinned_at, tts, mentions_everyone, mention_everyone, reply_to_id, thread_id, flags, created_at FROM messages 
			WHERE channel_id = $1 AND id > $2
			ORDER BY created_at ASC
			LIMIT $3
		`
		args = []interface{}{channelID, *after, limit}
	} else {
		query = `
			SELECT id, channel_id, author_id, content, encrypted_content, type, edited_at, pinned, pinned_at, tts, mentions_everyone, mention_everyone, reply_to_id, thread_id, flags, created_at FROM messages 
			WHERE channel_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []interface{}{channelID, limit}
	}

	err := r.db.SelectContext(ctx, &messages, query, args...)
	if err != nil {
		return nil, err
	}

	// Load attachments for all messages
	if len(messages) > 0 {
		messageIDs := make([]uuid.UUID, len(messages))
		for i, m := range messages {
			messageIDs[i] = m.ID
		}

		var attachments []models.Attachment
		query, args, err := sqlx.In(`SELECT * FROM attachments WHERE message_id IN (?)`, messageIDs)
		if err != nil {
			return messages, err
		}
		query = r.db.Rebind(query)
		err = r.db.SelectContext(ctx, &attachments, query, args...)
		if err != nil {
			return messages, err
		}

		// Map attachments to messages
		attMap := make(map[uuid.UUID][]models.Attachment)
		for _, att := range attachments {
			attMap[att.MessageID] = append(attMap[att.MessageID], att)
		}
		for _, m := range messages {
			m.Attachments = attMap[m.ID]
		}

		// Load authors for all messages
		authorIDs := make([]uuid.UUID, 0, len(messages))
		seen := make(map[uuid.UUID]bool)
		for _, m := range messages {
			if !seen[m.AuthorID] {
				authorIDs = append(authorIDs, m.AuthorID)
				seen[m.AuthorID] = true
			}
		}

		if len(authorIDs) > 0 {
			var authors []models.PublicUser
			authQuery, authArgs, err := sqlx.In(`SELECT id, username, discriminator, avatar_url, banner_url, bio, status, custom_status, flags FROM users WHERE id IN (?)`, authorIDs)
			if err != nil {
				return messages, err
			}
			authQuery = r.db.Rebind(authQuery)
			err = r.db.SelectContext(ctx, &authors, authQuery, authArgs...)
			if err != nil {
				return messages, err
			}

			// Map authors to messages
			authorMap := make(map[uuid.UUID]*models.PublicUser)
			for i := range authors {
				authorMap[authors[i].ID] = &authors[i]
			}
			for _, m := range messages {
				m.Author = authorMap[m.AuthorID]
			}
		}
	}

	return messages, nil
}

func (r *MessageRepository) GetPinnedMessages(ctx context.Context, channelID uuid.UUID) ([]*models.Message, error) {
	var messages []*models.Message
	query := `SELECT id, channel_id, author_id, content, encrypted_content, type, edited_at, pinned, pinned_at, tts, mentions_everyone, mention_everyone, reply_to_id, thread_id, flags, created_at FROM messages WHERE channel_id = $1 AND pinned = true ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &messages, query, channelID)
	return messages, err
}

func (r *MessageRepository) SearchMessages(ctx context.Context, query string, channelID *uuid.UUID, authorID *uuid.UUID, limit int) ([]*models.Message, error) {
	var messages []*models.Message

	sqlQuery := `
		SELECT id, channel_id, author_id, content, encrypted_content, type, edited_at, pinned, pinned_at, tts, mentions_everyone, mention_everyone, reply_to_id, thread_id, flags, created_at FROM messages 
		WHERE content ILIKE $1
	`
	args := []interface{}{"%" + query + "%"}
	argNum := 2

	if channelID != nil {
		sqlQuery += ` AND channel_id = $` + strconv.Itoa(argNum)
		args = append(args, *channelID)
		argNum++
	}

	if authorID != nil {
		sqlQuery += ` AND author_id = $` + strconv.Itoa(argNum)
		args = append(args, *authorID)
		argNum++
	}

	sqlQuery += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argNum)
	args = append(args, limit)

	err := r.db.SelectContext(ctx, &messages, sqlQuery, args...)
	return messages, err
}

// Reactions

func (r *MessageRepository) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	query := `
		INSERT INTO reactions (message_id, user_id, emoji, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id, user_id, emoji) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, messageID, userID, emoji, time.Now())
	return err
}

func (r *MessageRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	query := `DELETE FROM reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`
	_, err := r.db.ExecContext(ctx, query, messageID, userID, emoji)
	return err
}

func (r *MessageRepository) GetReactions(ctx context.Context, messageID uuid.UUID) ([]*models.Reaction, error) {
	var reactions []*models.Reaction
	query := `
		SELECT message_id, emoji, COUNT(*) as count
		FROM reactions
		WHERE message_id = $1
		GROUP BY message_id, emoji
		ORDER BY MIN(created_at)
	`
	err := r.db.SelectContext(ctx, &reactions, query, messageID)
	return reactions, err
}

func (r *MessageRepository) GetReactionUsers(ctx context.Context, messageID uuid.UUID, emoji string, limit int) ([]*models.ReactionUser, error) {
	var users []*models.ReactionUser
	query := `
		SELECT message_id, user_id, emoji, created_at
		FROM reactions
		WHERE message_id = $1 AND emoji = $2
		ORDER BY created_at ASC
		LIMIT $3
	`
	err := r.db.SelectContext(ctx, &users, query, messageID, emoji, limit)
	return users, err
}

func (r *MessageRepository) GetUserReactions(ctx context.Context, messageID, userID uuid.UUID) ([]string, error) {
	var emojis []string
	query := `SELECT emoji FROM reactions WHERE message_id = $1 AND user_id = $2`
	err := r.db.SelectContext(ctx, &emojis, query, messageID, userID)
	return emojis, err
}

func (r *MessageRepository) RemoveAllReactions(ctx context.Context, messageID uuid.UUID) error {
	query := `DELETE FROM reactions WHERE message_id = $1`
	_, err := r.db.ExecContext(ctx, query, messageID)
	return err
}

// GetTopReactions returns the most frequently used reactions across all messages
// limited to the specified count (default 10)
func (r *MessageRepository) GetTopReactions(ctx context.Context, limit int) ([]*models.Reaction, error) {
	if limit <= 0 {
		limit = 10
	}
	var reactions []*models.Reaction
	query := `
		SELECT emoji, COUNT(*) as count
		FROM reactions
		GROUP BY emoji
		ORDER BY count DESC, emoji ASC
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &reactions, query, limit)
	return reactions, err
}

// Bulk operations

func (r *MessageRepository) DeleteByChannel(ctx context.Context, channelID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM messages WHERE channel_id = $1`, channelID)
	return err
}

func (r *MessageRepository) DeleteByAuthor(ctx context.Context, channelID, authorID uuid.UUID, since time.Time) (int, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM messages WHERE channel_id = $1 AND author_id = $2 AND created_at >= $3`,
		channelID, authorID, since,
	)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

type ForwardedMessageRepository struct {
	db *sqlx.DB
}

func NewForwardedMessageRepository(db *sqlx.DB) *ForwardedMessageRepository {
	return &ForwardedMessageRepository{db: db}
}

// Create inserts a new forwarded message record
func (r *ForwardedMessageRepository) Create(ctx context.Context, fm *models.ForwardedMessage) error {
	query := `
		INSERT INTO forwarded_messages (id, original_message_id, forwarded_by_id, destination_channel_id, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		fm.ID, fm.OriginalMessageID, fm.ForwardedByID, fm.DestinationChannelID, fm.Comment, fm.CreatedAt,
	)
	return err
}

// GetByOriginalMessageID returns all forwarded message records for a given original message
func (r *ForwardedMessageRepository) GetByOriginalMessageID(ctx context.Context, originalMessageID uuid.UUID) ([]*models.ForwardedMessage, error) {
	var messages []*models.ForwardedMessage
	query := `
		SELECT id, original_message_id, forwarded_by_id, destination_channel_id, comment, created_at
		FROM forwarded_messages
		WHERE original_message_id = $1
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &messages, query, originalMessageID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return messages, err
}

// GetByDestinationChannelID returns all forwarded messages sent to a given channel
func (r *ForwardedMessageRepository) GetByDestinationChannelID(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.ForwardedMessage, int, error) {
	var count int
	countQuery := `SELECT COUNT(*) FROM forwarded_messages WHERE destination_channel_id = $1`
	if err := r.db.GetContext(ctx, &count, countQuery, channelID); err != nil {
		return nil, 0, err
	}

	var messages []*models.ForwardedMessage
	query := `
		SELECT id, original_message_id, forwarded_by_id, destination_channel_id, comment, created_at
		FROM forwarded_messages
		WHERE destination_channel_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &messages, query, channelID, limit, offset)
	if err == sql.ErrNoRows {
		return nil, count, nil
	}
	return messages, count, err
}

// GetByID returns a single forwarded message by ID
func (r *ForwardedMessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ForwardedMessage, error) {
	var message models.ForwardedMessage
	query := `
		SELECT id, original_message_id, forwarded_by_id, destination_channel_id, comment, created_at
		FROM forwarded_messages
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &message, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &message, err
}

// EmbedRepository handles embed template data access
type EmbedRepository struct {
	db *sqlx.DB
}

// NewEmbedRepository creates a new embed repository
func NewEmbedRepository(db *sqlx.DB) *EmbedRepository {
	return &EmbedRepository{db: db}
}

// CreateTemplate creates a new embed template
func (r *EmbedRepository) CreateTemplate(ctx context.Context, template *models.EmbedTemplate) error {
	query := `
		INSERT INTO embed_templates (
			id, user_id, name, title, description, url, color,
			author_name, author_url, author_icon,
			footer_text, footer_icon,
			image_url, thumbnail_url,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		template.ID, template.UserID, template.Name, template.Title, template.Description,
		template.URL, template.Color, template.AuthorName, template.AuthorURL, template.AuthorIcon,
		template.FooterText, template.FooterIcon, template.ImageURL, template.ThumbnailURL,
		template.CreatedAt, template.UpdatedAt,
	)
	return err
}

// GetTemplateByID retrieves an embed template by ID
func (r *EmbedRepository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*models.EmbedTemplate, error) {
	var template models.EmbedTemplate
	query := `
		SELECT 
			id, user_id, name, title, description, url, color,
			author_name, author_url, author_icon,
			footer_text, footer_icon,
			image_url, thumbnail_url,
			created_at, updated_at
		FROM embed_templates WHERE id = $1
	`
	err := r.db.GetContext(ctx, &template, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &template, err
}

// GetTemplatesByUserID retrieves all embed templates for a user
func (r *EmbedRepository) GetTemplatesByUserID(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error) {
	var templates []models.EmbedTemplate
	query := `
		SELECT 
			id, user_id, name, title, description, url, color,
			author_name, author_url, author_icon,
			footer_text, footer_icon,
			image_url, thumbnail_url,
			created_at, updated_at
		FROM embed_templates WHERE user_id = $1 ORDER BY name ASC
	`
	err := r.db.SelectContext(ctx, &templates, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return templates, err
}

// UpdateTemplate updates an embed template
func (r *EmbedRepository) UpdateTemplate(ctx context.Context, template *models.EmbedTemplate) error {
	query := `
		UPDATE embed_templates SET
			name = $2, title = $3, description = $4, url = $5, color = $6,
			author_name = $7, author_url = $8, author_icon = $9,
			footer_text = $10, footer_icon = $11,
			image_url = $12, thumbnail_url = $13,
			updated_at = $14
		WHERE id = $1 AND user_id = $15
	`
	result, err := r.db.ExecContext(ctx, query,
		template.ID, template.Name, template.Title, template.Description,
		template.URL, template.Color, template.AuthorName, template.AuthorURL, template.AuthorIcon,
		template.FooterText, template.FooterIcon, template.ImageURL, template.ThumbnailURL,
		time.Now(), template.UserID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteTemplate deletes an embed template
func (r *EmbedRepository) DeleteTemplate(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM embed_templates WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
