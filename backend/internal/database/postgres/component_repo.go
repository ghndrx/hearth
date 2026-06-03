package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

type ComponentRepository struct {
	db *sqlx.DB
}

func NewComponentRepository(db *sqlx.DB) *ComponentRepository {
	return &ComponentRepository{db: db}
}

// CreateComponent creates a new message component
func (r *ComponentRepository) CreateComponent(ctx context.Context, c *models.MessageComponent) error {
	optionsJSON, err := json.Marshal(c.Options)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO message_components (id, message_id, type, style, label, custom_id, url, disabled, emoji_id, emoji_name, options, min_values, max_values, placeholder, required, value, min_length, max_length, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	_, err = r.db.ExecContext(ctx, query,
		c.ID, c.MessageID, c.Type, c.Style, c.Label, c.CustomID, c.URL, c.Disabled,
		c.EmojiID, c.EmojiName, optionsJSON, c.MinValues, c.MaxValues, c.Placeholder,
		c.Required, c.Value, c.MinLength, c.MaxLength, c.CreatedAt,
	)
	return err
}

// GetComponentByID retrieves a component by ID
func (r *ComponentRepository) GetComponentByID(ctx context.Context, id uuid.UUID) (*models.MessageComponent, error) {
	var c models.MessageComponent
	var optionsJSON []byte

	query := `
		SELECT id, message_id, type, style, label, custom_id, url, disabled, emoji_id, emoji_name, options, min_values, max_values, placeholder, required, value, min_length, max_length, created_at
		FROM message_components WHERE id = $1
	`
	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&c.ID, &c.MessageID, &c.Type, &c.Style, &c.Label, &c.CustomID, &c.URL, &c.Disabled,
		&c.EmojiID, &c.EmojiName, &optionsJSON, &c.MinValues, &c.MaxValues, &c.Placeholder,
		&c.Required, &c.Value, &c.MinLength, &c.MaxLength, &c.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(optionsJSON) > 0 {
		if err := json.Unmarshal(optionsJSON, &c.Options); err != nil {
			return nil, err
		}
	}

	return &c, nil
}

// GetComponentsByMessageID retrieves all components for a message
func (r *ComponentRepository) GetComponentsByMessageID(ctx context.Context, messageID uuid.UUID) ([]*models.MessageComponent, error) {
	var components []*models.MessageComponent

	query := `
		SELECT id, message_id, type, style, label, custom_id, url, disabled, emoji_id, emoji_name, options, min_values, max_values, placeholder, required, value, min_length, max_length, created_at
		FROM message_components WHERE message_id = $1 ORDER BY created_at ASC
	`
	rows, err := r.db.QueryxContext(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c models.MessageComponent
		var optionsJSON []byte

		err := rows.Scan(
			&c.ID, &c.MessageID, &c.Type, &c.Style, &c.Label, &c.CustomID, &c.URL, &c.Disabled,
			&c.EmojiID, &c.EmojiName, &optionsJSON, &c.MinValues, &c.MaxValues, &c.Placeholder,
			&c.Required, &c.Value, &c.MinLength, &c.MaxLength, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(optionsJSON) > 0 {
			if err := json.Unmarshal(optionsJSON, &c.Options); err != nil {
				return nil, err
			}
		}

		components = append(components, &c)
	}

	return components, rows.Err()
}

// UpdateComponent updates a component
func (r *ComponentRepository) UpdateComponent(ctx context.Context, c *models.MessageComponent) error {
	optionsJSON, err := json.Marshal(c.Options)
	if err != nil {
		return err
	}

	query := `
		UPDATE message_components SET 
			style = $2, label = $3, custom_id = $4, url = $5, disabled = $6,
			emoji_id = $7, emoji_name = $8, options = $9, min_values = $10, max_values = $11,
			placeholder = $12, required = $13, value = $14, min_length = $15, max_length = $16
		WHERE id = $1
	`
	_, err = r.db.ExecContext(ctx, query,
		c.ID, c.Style, c.Label, c.CustomID, c.URL, c.Disabled,
		c.EmojiID, c.EmojiName, optionsJSON, c.MinValues, c.MaxValues,
		c.Placeholder, c.Required, c.Value, c.MinLength, c.MaxLength,
	)
	return err
}

// DeleteComponent deletes a component by ID
func (r *ComponentRepository) DeleteComponent(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM message_components WHERE id = $1`, id)
	return err
}

// DeleteComponentsByMessageID deletes all components for a message
func (r *ComponentRepository) DeleteComponentsByMessageID(ctx context.Context, messageID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM message_components WHERE message_id = $1`, messageID)
	return err
}

// CreateInteraction creates a new component interaction
func (r *ComponentRepository) CreateInteraction(ctx context.Context, i *models.ComponentInteraction) error {
	query := `
		INSERT INTO component_interactions (id, type, user_id, channel_id, message_id, component_id, custom_id, values, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		i.ID, i.Type, i.UserID, i.ChannelID, i.MessageID, i.ComponentID, i.CustomID, i.Values, i.CreatedAt,
	)
	return err
}

// GetInteractionsByComponentID retrieves all interactions for a component
func (r *ComponentRepository) GetInteractionsByComponentID(ctx context.Context, componentID uuid.UUID) ([]*models.ComponentInteraction, error) {
	var interactions []*models.ComponentInteraction

	query := `
		SELECT id, type, user_id, channel_id, message_id, component_id, custom_id, values, created_at
		FROM component_interactions WHERE component_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.QueryxContext(ctx, query, componentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var i models.ComponentInteraction
		err := rows.Scan(
			&i.ID, &i.Type, &i.UserID, &i.ChannelID, &i.MessageID, &i.ComponentID, &i.CustomID, &i.Values, &i.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		interactions = append(interactions, &i)
	}

	return interactions, rows.Err()
}

// GetComponentsByCustomID retrieves components by custom_id
func (r *ComponentRepository) GetComponentsByCustomID(ctx context.Context, customID string) ([]*models.MessageComponent, error) {
	var components []*models.MessageComponent

	query := `
		SELECT id, message_id, type, style, label, custom_id, url, disabled, emoji_id, emoji_name, options, min_values, max_values, placeholder, required, value, min_length, max_length, created_at
		FROM message_components WHERE custom_id = $1 ORDER BY created_at ASC
	`
	rows, err := r.db.QueryxContext(ctx, query, customID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c models.MessageComponent
		var optionsJSON []byte

		err := rows.Scan(
			&c.ID, &c.MessageID, &c.Type, &c.Style, &c.Label, &c.CustomID, &c.URL, &c.Disabled,
			&c.EmojiID, &c.EmojiName, &optionsJSON, &c.MinValues, &c.MaxValues, &c.Placeholder,
			&c.Required, &c.Value, &c.MinLength, &c.MaxLength, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(optionsJSON) > 0 {
			if err := json.Unmarshal(optionsJSON, &c.Options); err != nil {
				return nil, err
			}
		}

		components = append(components, &c)
	}

	return components, rows.Err()
}

// GetInteractionByID retrieves a component interaction by ID
func (r *ComponentRepository) GetInteractionByID(ctx context.Context, id uuid.UUID) (*models.ComponentInteraction, error) {
	var i models.ComponentInteraction

	query := `
		SELECT id, type, user_id, channel_id, message_id, component_id, custom_id, values, created_at
		FROM component_interactions WHERE id = $1
	`
	err := r.db.GetContext(ctx, &i, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &i, nil
}

