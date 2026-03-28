package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

type EventRepository struct {
	db *sqlx.DB
}

func NewEventRepository(db *sqlx.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) CreateEvent(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO events (id, server_id, channel_id, creator_id, name, description, image_url,
			scheduled_start, scheduled_end, entity_type, location, status, user_count, recurrence_rule, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.ServerID, event.ChannelID, event.CreatorID, event.Name, event.Description,
		event.ImageURL, event.ScheduledStart, event.ScheduledEnd, event.EntityType, event.Location,
		event.Status, event.UserCount, event.RecurrenceRule, event.CreatedAt,
	)
	return err
}

func (r *EventRepository) GetEventByID(ctx context.Context, id uuid.UUID) (*models.Event, error) {
	var event models.Event
	query := `SELECT * FROM events WHERE id = $1`
	err := r.db.GetContext(ctx, &event, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *EventRepository) UpdateEvent(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*models.Event, error) {
	// Build dynamic update query
	query := "UPDATE events SET "
	args := []interface{}{}
	argIndex := 1
	var setClauses []string

	for key, value := range updates {
		setClauses = append(setClauses, key+" = $"+string(rune('0'+argIndex)))
		args = append(args, value)
		argIndex++
	}

	if len(setClauses) == 0 {
		return r.GetEventByID(ctx, id)
	}

	for i, clause := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}
	query += " WHERE id = $" + string(rune('0'+argIndex))
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return r.GetEventByID(ctx, id)
}

func (r *EventRepository) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM events WHERE id = $1`, id)
	return err
}

func (r *EventRepository) ListServerEvents(ctx context.Context, serverID uuid.UUID, statusFilter *int) ([]*models.Event, error) {
	var events []*models.Event
	var query string
	var args []interface{}

	if statusFilter != nil {
		query = `SELECT * FROM events WHERE server_id = $1 AND status = $2 ORDER BY scheduled_start ASC`
		args = []interface{}{serverID, *statusFilter}
	} else {
		query = `SELECT * FROM events WHERE server_id = $1 ORDER BY scheduled_start ASC`
		args = []interface{}{serverID}
	}

	err := r.db.SelectContext(ctx, &events, query, args...)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *EventRepository) ListUserRSVPEvents(ctx context.Context, userID uuid.UUID) ([]*models.Event, error) {
	var events []*models.Event
	query := `
		SELECT e.* FROM events e
		INNER JOIN event_rsvps r ON r.event_id = e.id
		WHERE r.user_id = $1
		ORDER BY e.scheduled_start ASC
	`
	err := r.db.SelectContext(ctx, &events, query, userID)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *EventRepository) RSVP(ctx context.Context, eventID, userID uuid.UUID, status models.RSVPStatus) error {
	query := `
		INSERT INTO event_rsvps (event_id, user_id, status, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id, user_id) DO UPDATE SET status = $3
	`
	_, err := r.db.ExecContext(ctx, query, eventID, userID, status, time.Now())
	return err
}

func (r *EventRepository) RemoveRSVP(ctx context.Context, eventID, userID uuid.UUID) error {
	query := `DELETE FROM event_rsvps WHERE event_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, eventID, userID)
	return err
}

func (r *EventRepository) GetEventUsers(ctx context.Context, eventID uuid.UUID) ([]*models.EventRSVP, error) {
	var rsvps []*models.EventRSVP
	query := `SELECT * FROM event_rsvps WHERE event_id = $1 ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &rsvps, query, eventID)
	if err != nil {
		return nil, err
	}
	return rsvps, nil
}

func (r *EventRepository) GetUserRSVP(ctx context.Context, eventID, userID uuid.UUID) (*models.EventRSVP, error) {
	var rsvp models.EventRSVP
	query := `SELECT * FROM event_rsvps WHERE event_id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &rsvp, query, eventID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rsvp, nil
}

func (r *EventRepository) IncrementUserCount(ctx context.Context, eventID uuid.UUID, delta int) error {
	query := `UPDATE events SET user_count = user_count + $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, delta, eventID)
	return err
}

// GetEventByIDWithDetails returns an event with additional details
func (r *EventRepository) GetEventByIDWithDetails(ctx context.Context, id uuid.UUID) (*models.Event, error) {
	return r.GetEventByID(ctx, id)
}

// MarshalRecurrenceRule converts a recurrence rule map to JSON
func MarshalRecurrenceRule(rule map[string]interface{}) (json.RawMessage, error) {
	if rule == nil {
		return nil, nil
	}
	return json.Marshal(rule)
}
