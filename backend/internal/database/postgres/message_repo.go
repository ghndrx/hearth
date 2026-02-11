package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	
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
		INSERT INTO messages (id, channel_id, author_id, content, encrypted, reply_to, pinned, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		message.ID, message.ChannelID, message.AuthorID, message.Content,
		message.Encrypted, message.ReplyTo, message.Pinned,
		message.CreatedAt, message.UpdatedAt,
	)
	if err != nil {
		return err
	}
	
	// Insert mentions
	if len(message.Mentions) > 0 {
		for _, userID := range message.Mentions {
			_, _ = r.db.ExecContext(ctx,
				`INSERT INTO message_mentions (message_id, user_id) VALUES ($1, $2)`,
				message.ID, userID,
			)
		}
	}
	
	// Insert attachments
	if len(message.Attachments) > 0 {
		for _, att := range message.Attachments {
			_, _ = r.db.ExecContext(ctx,
				`INSERT INTO attachments (id, message_id, filename, url, content_type, size) VALUES ($1, $2, $3, $4, $5, $6)`,
				att.ID, message.ID, att.Filename, att.URL, att.ContentType, att.Size,
			)
		}
	}
	
	return nil
}

func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	var message models.Message
	query := `SELECT * FROM messages WHERE id = $1`
	err := r.db.GetContext(ctx, &message, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	// Load attachments
	var attachments []*models.Attachment
	_ = r.db.SelectContext(ctx, &attachments, `SELECT * FROM attachments WHERE message_id = $1`, id)
	message.Attachments = attachments
	
	return &message, nil
}

func (r *MessageRepository) Update(ctx context.Context, message *models.Message) error {
	query := `
		UPDATE messages SET content = $2, pinned = $3, edited_at = $4, updated_at = $5
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		message.ID, message.Content, message.Pinned, message.EditedAt, message.UpdatedAt,
	)
	return err
}

func (r *MessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM messages WHERE id = $1`, id)
	return err
}

func (r *MessageRepository) GetChannelMessages(ctx context.Context, channelID uuid.UUID, before, after *uuid.UUID, limit int) ([]*models.Message, error) {
	var messages []*models.Message
	var query string
	var args []interface{}
	
	if before != nil {
		query = `
			SELECT * FROM messages 
			WHERE channel_id = $1 AND id < $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []interface{}{channelID, *before, limit}
	} else if after != nil {
		query = `
			SELECT * FROM messages 
			WHERE channel_id = $1 AND id > $2
			ORDER BY created_at ASC
			LIMIT $3
		`
		args = []interface{}{channelID, *after, limit}
	} else {
		query = `
			SELECT * FROM messages 
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
		
		var attachments []*models.Attachment
		query, args, _ := sqlx.In(`SELECT * FROM attachments WHERE message_id IN (?)`, messageIDs)
		query = r.db.Rebind(query)
		_ = r.db.SelectContext(ctx, &attachments, query, args...)
		
		// Map attachments to messages
		attMap := make(map[uuid.UUID][]*models.Attachment)
		for _, att := range attachments {
			attMap[att.MessageID] = append(attMap[att.MessageID], att)
		}
		for _, m := range messages {
			m.Attachments = attMap[m.ID]
		}
	}
	
	return messages, nil
}

func (r *MessageRepository) GetPinnedMessages(ctx context.Context, channelID uuid.UUID) ([]*models.Message, error) {
	var messages []*models.Message
	query := `SELECT * FROM messages WHERE channel_id = $1 AND pinned = true ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &messages, query, channelID)
	return messages, err
}

func (r *MessageRepository) SearchMessages(ctx context.Context, query string, channelID *uuid.UUID, authorID *uuid.UUID, limit int) ([]*models.Message, error) {
	var messages []*models.Message
	
	sqlQuery := `
		SELECT * FROM messages 
		WHERE content ILIKE $1
	`
	args := []interface{}{"%" + query + "%"}
	argNum := 2
	
	if channelID != nil {
		sqlQuery += ` AND channel_id = $` + string(rune('0'+argNum))
		args = append(args, *channelID)
		argNum++
	}
	
	if authorID != nil {
		sqlQuery += ` AND author_id = $` + string(rune('0'+argNum))
		args = append(args, *authorID)
		argNum++
	}
	
	sqlQuery += ` ORDER BY created_at DESC LIMIT $` + string(rune('0'+argNum))
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
	err := r.db.SelectContext(ctx, &reactions, `SELECT * FROM reactions WHERE message_id = $1`, messageID)
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
