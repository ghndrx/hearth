package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
	"hearth/internal/services"
)

// SearchRepository implements advanced search queries
type SearchRepository struct {
	db *sqlx.DB
}

// NewSearchRepository creates a new search repository
func NewSearchRepository(db *sqlx.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

// SearchMessages performs advanced message search with filters
func (r *SearchRepository) SearchMessages(ctx context.Context, opts services.SearchMessageOptions) (*services.SearchResult, error) {
	// Build the query dynamically
	var conditions []string
	var args []interface{}
	argNum := 1

	// Base query
	query := `
		SELECT m.*, COUNT(*) OVER() as total_count 
		FROM messages m
		WHERE 1=1
	`

	// Text search
	if opts.Query != "" {
		// Use PostgreSQL full-text search for better performance
		// Fallback to ILIKE for simple searches
		conditions = append(conditions, fmt.Sprintf("(m.content ILIKE $%d OR to_tsvector('english', m.content) @@ plainto_tsquery('english', $%d))", argNum, argNum))
		args = append(args, "%"+opts.Query+"%")
		argNum++
	}

	// Channel filter
	if opts.ChannelID != nil {
		conditions = append(conditions, fmt.Sprintf("m.channel_id = $%d", argNum))
		args = append(args, *opts.ChannelID)
		argNum++
	}

	// Multiple channels filter
	if len(opts.ChannelIDs) > 0 {
		placeholders := make([]string, len(opts.ChannelIDs))
		for i := range opts.ChannelIDs {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, opts.ChannelIDs[i])
			argNum++
		}
		conditions = append(conditions, fmt.Sprintf("m.channel_id IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Server filter (via channel lookup)
	if opts.ServerID != nil && len(opts.ChannelIDs) == 0 && opts.ChannelID == nil {
		conditions = append(conditions, fmt.Sprintf(`m.channel_id IN (
			SELECT id FROM channels WHERE server_id = $%d
		)`, argNum))
		args = append(args, *opts.ServerID)
		argNum++
	}

	// Author filter
	if opts.AuthorID != nil {
		conditions = append(conditions, fmt.Sprintf("m.author_id = $%d", argNum))
		args = append(args, *opts.AuthorID)
		argNum++
	}

	// Time range filters
	if opts.Before != nil {
		conditions = append(conditions, fmt.Sprintf("m.created_at < $%d", argNum))
		args = append(args, *opts.Before)
		argNum++
	}

	if opts.After != nil {
		conditions = append(conditions, fmt.Sprintf("m.created_at > $%d", argNum))
		args = append(args, *opts.After)
		argNum++
	}

	// Content filters using subqueries
	if opts.HasAttachments != nil && *opts.HasAttachments {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM attachments a WHERE a.message_id = m.id
		)`)
	} else if opts.HasAttachments != nil && !*opts.HasAttachments {
		conditions = append(conditions, `NOT EXISTS (
			SELECT 1 FROM attachments a WHERE a.message_id = m.id
		)`)
	}

	if opts.HasReactions != nil && *opts.HasReactions {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM reactions r WHERE r.message_id = m.id
		)`)
	} else if opts.HasReactions != nil && !*opts.HasReactions {
		conditions = append(conditions, `NOT EXISTS (
			SELECT 1 FROM reactions r WHERE r.message_id = m.id
		)`)
	}

	// Pinned filter
	if opts.Pinned != nil {
		conditions = append(conditions, fmt.Sprintf("m.pinned = $%d", argNum))
		args = append(args, *opts.Pinned)
		argNum++
	}

	// Mentions filter
	if len(opts.Mentions) > 0 {
		placeholders := make([]string, len(opts.Mentions))
		for i := range opts.Mentions {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, opts.Mentions[i])
			argNum++
		}
		conditions = append(conditions, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM message_mentions mm 
			WHERE mm.message_id = m.id AND mm.user_id IN (%s)
		)`, strings.Join(placeholders, ", ")))
	}

	// Add conditions to query
	for _, condition := range conditions {
		query += " AND " + condition
	}

	// Order and limit
	query += fmt.Sprintf(" ORDER BY m.created_at DESC LIMIT $%d", argNum)
	args = append(args, opts.Limit+1) // Fetch one extra to check if there are more
	argNum++

	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, opts.Offset)
	}

	// Execute query
	var results []struct {
		models.Message
		TotalCount int `db:"total_count"`
	}

	err := r.db.SelectContext(ctx, &results, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search messages failed: %w", err)
	}

	// Process results
	total := 0
	hasMore := false
	if len(results) > 0 {
		total = results[0].TotalCount
		if len(results) > opts.Limit {
			hasMore = true
			results = results[:opts.Limit] // Remove the extra row
		}
	}

	messages := make([]*models.Message, len(results))
	for i, r := range results {
		msg := r.Message
		messages[i] = &msg
	}

	// Load attachments for found messages
	if len(messages) > 0 {
		r.loadAttachments(ctx, messages)
	}

	return &services.SearchResult{
		Messages: messages,
		Total:    total,
		HasMore:  hasMore,
	}, nil
}

// loadAttachments loads attachments for messages
func (r *SearchRepository) loadAttachments(ctx context.Context, messages []*models.Message) error {
	if len(messages) == 0 {
		return nil
	}

	messageIDs := make([]uuid.UUID, len(messages))
	for i, m := range messages {
		messageIDs[i] = m.ID
	}

	query, args, err := sqlx.In(`SELECT * FROM attachments WHERE message_id IN (?)`, messageIDs)
	if err != nil {
		return err
	}

	query = r.db.Rebind(query)
	var attachments []models.Attachment
	if err := r.db.SelectContext(ctx, &attachments, query, args...); err != nil {
		return err
	}

	// Map attachments to messages
	attMap := make(map[uuid.UUID][]models.Attachment)
	for _, att := range attachments {
		attMap[att.MessageID] = append(attMap[att.MessageID], att)
	}
	for _, m := range messages {
		m.Attachments = attMap[m.ID]
	}

	return nil
}

// SearchUsers searches for users by username or display name
func (r *SearchRepository) SearchUsers(ctx context.Context, query string, serverID *uuid.UUID, limit int) ([]*models.PublicUser, error) {
	var users []*models.User
	var args []interface{}

	sqlQuery := `
		SELECT * FROM users 
		WHERE (
			username ILIKE $1 
			OR display_name ILIKE $1
		)
	`
	args = append(args, "%"+query+"%")
	argNum := 2

	// Filter by server membership
	if serverID != nil {
		sqlQuery += fmt.Sprintf(` AND id IN (
			SELECT user_id FROM server_members WHERE server_id = $%d
		)`, argNum)
		args = append(args, *serverID)
		argNum++
	}

	sqlQuery += fmt.Sprintf(" LIMIT $%d", argNum)
	args = append(args, limit)

	err := r.db.SelectContext(ctx, &users, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search users failed: %w", err)
	}

	// Convert to PublicUser
	publicUsers := make([]*models.PublicUser, len(users))
	for i, u := range users {
		pu := u.ToPublic()
		publicUsers[i] = &pu
	}

	return publicUsers, nil
}

// SearchChannels searches for channels by name
func (r *SearchRepository) SearchChannels(ctx context.Context, query string, serverID *uuid.UUID, limit int) ([]*models.Channel, error) {
	var channels []*models.Channel
	var args []interface{}

	sqlQuery := `
		SELECT * FROM channels 
		WHERE name ILIKE $1
	`
	args = append(args, "%"+query+"%")
	argNum := 2

	if serverID != nil {
		sqlQuery += fmt.Sprintf(" AND server_id = $%d", argNum)
		args = append(args, *serverID)
		argNum++
	}

	sqlQuery += fmt.Sprintf(" LIMIT $%d", argNum)
	args = append(args, limit)

	err := r.db.SelectContext(ctx, &channels, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search channels failed: %w", err)
	}

	return channels, nil
}

// SearchMessagesSimple provides a simplified search for basic use cases
func (r *SearchRepository) SearchMessagesSimple(ctx context.Context, query string, channelID *uuid.UUID, limit int) ([]*models.Message, error) {
	opts := services.SearchMessageOptions{
		Query:     query,
		Limit:     limit,
		ChannelID: channelID,
	}

	result, err := r.SearchMessages(ctx, opts)
	if err != nil {
		return nil, err
	}

	return result.Messages, nil
}

// GetMessagesByDateRange retrieves messages within a date range
func (r *SearchRepository) GetMessagesByDateRange(ctx context.Context, channelID uuid.UUID, start, end time.Time, limit int) ([]*models.Message, error) {
	opts := services.SearchMessageOptions{
		ChannelID: &channelID,
		After:     &start,
		Before:    &end,
		Limit:     limit,
	}

	result, err := r.SearchMessages(ctx, opts)
	if err != nil {
		return nil, err
	}

	return result.Messages, nil
}

// GetMessagesWithAttachments retrieves messages that have attachments
func (r *SearchRepository) GetMessagesWithAttachments(ctx context.Context, channelID uuid.UUID, limit int) ([]*models.Message, error) {
	hasAttachments := true
	opts := services.SearchMessageOptions{
		ChannelID:      &channelID,
		HasAttachments: &hasAttachments,
		Limit:          limit,
	}

	result, err := r.SearchMessages(ctx, opts)
	if err != nil {
		return nil, err
	}

	return result.Messages, nil
}

// GetMessagesByAuthor retrieves all messages by a specific author in a channel/server
func (r *SearchRepository) GetMessagesByAuthor(ctx context.Context, authorID, channelID uuid.UUID, limit int) ([]*models.Message, error) {
	opts := services.SearchMessageOptions{
		AuthorID:  &authorID,
		ChannelID: &channelID,
		Limit:     limit,
	}

	result, err := r.SearchMessages(ctx, opts)
	if err != nil {
		return nil, err
	}

	return result.Messages, nil
}
