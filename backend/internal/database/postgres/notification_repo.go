package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// NotificationRepository handles notification data access
type NotificationRepository struct {
	db *sqlx.DB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	notification.ID = uuid.New()
	notification.CreatedAt = time.Now()
	notification.Read = false

	query := `
		INSERT INTO notifications (
			id, user_id, type, title, body, read, data,
			actor_id, server_id, channel_id, message_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		notification.ID, notification.UserID, notification.Type, notification.Title,
		notification.Body, notification.Read, notification.Data,
		notification.ActorID, notification.ServerID, notification.ChannelID,
		notification.MessageID, notification.CreatedAt,
	)
	return err
}

// GetByID retrieves a notification by ID
func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	var notification models.Notification
	query := `
		SELECT id, user_id, type, title, body, read, data,
		       actor_id, server_id, channel_id, message_id, created_at
		FROM notifications
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &notification, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &notification, err
}

// GetByIDWithActor retrieves a notification with actor info
func (r *NotificationRepository) GetByIDWithActor(ctx context.Context, id uuid.UUID) (*models.NotificationWithActor, error) {
	var notification models.NotificationWithActor
	query := `
		SELECT n.id, n.user_id, n.type, n.title, n.body, n.read, n.data,
		       n.actor_id, n.server_id, n.channel_id, n.message_id, n.created_at,
		       u.username as actor_username, u.avatar as actor_avatar,
		       s.name as server_name, c.name as channel_name
		FROM notifications n
		LEFT JOIN users u ON n.actor_id = u.id
		LEFT JOIN servers s ON n.server_id = s.id
		LEFT JOIN channels c ON n.channel_id = c.id
		WHERE n.id = $1
	`
	err := r.db.GetContext(ctx, &notification, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &notification, err
}

// List retrieves notifications for a user with options
func (r *NotificationRepository) List(ctx context.Context, userID uuid.UUID, opts models.NotificationListOptions) ([]models.NotificationWithActor, error) {
	// Set defaults
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}

	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("n.user_id = $%d", argNum))
	args = append(args, userID)
	argNum++

	if opts.Unread != nil {
		conditions = append(conditions, fmt.Sprintf("n.read = $%d", argNum))
		args = append(args, !*opts.Unread)
		argNum++
	}

	if len(opts.Types) > 0 {
		placeholders := make([]string, len(opts.Types))
		for i, t := range opts.Types {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, t)
			argNum++
		}
		conditions = append(conditions, fmt.Sprintf("n.type IN (%s)", strings.Join(placeholders, ",")))
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT n.id, n.user_id, n.type, n.title, n.body, n.read, n.data,
		       n.actor_id, n.server_id, n.channel_id, n.message_id, n.created_at,
		       u.username as actor_username, u.avatar as actor_avatar,
		       s.name as server_name, c.name as channel_name
		FROM notifications n
		LEFT JOIN users u ON n.actor_id = u.id
		LEFT JOIN servers s ON n.server_id = s.id
		LEFT JOIN channels c ON n.channel_id = c.id
		WHERE %s
		ORDER BY n.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, opts.Limit, opts.Offset)

	var notifications []models.NotificationWithActor
	err := r.db.SelectContext(ctx, &notifications, query, args...)
	if err != nil {
		return nil, err
	}

	if notifications == nil {
		notifications = []models.NotificationWithActor{}
	}
	return notifications, nil
}

// GetStats retrieves notification statistics for a user
func (r *NotificationRepository) GetStats(ctx context.Context, userID uuid.UUID) (*models.NotificationStats, error) {
	var stats models.NotificationStats
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE read = false) as unread
		FROM notifications
		WHERE user_id = $1
	`
	err := r.db.GetContext(ctx, &stats, query, userID)
	return &stats, err
}

// MarkAsRead marks a single notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Delete deletes a notification
func (r *NotificationRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteAllRead deletes all read notifications for a user
func (r *NotificationRepository) DeleteAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `DELETE FROM notifications WHERE user_id = $1 AND read = true`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteOlderThan deletes notifications older than the given duration
func (r *NotificationRepository) DeleteOlderThan(ctx context.Context, userID uuid.UUID, before time.Time) (int64, error) {
	query := `DELETE FROM notifications WHERE user_id = $1 AND created_at < $2`
	result, err := r.db.ExecContext(ctx, query, userID, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
