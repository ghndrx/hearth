package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

type TemplateRepository struct {
	db *sqlx.DB
}

func NewTemplateRepository(db *sqlx.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(ctx context.Context, tmpl *models.ServerTemplate) error {
	query := `
		INSERT INTO server_templates (id, code, name, description, source_server_id, creator_id, serialized_data, usage_count, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		tmpl.ID, tmpl.Code, tmpl.Name, tmpl.Description, tmpl.SourceServerID,
		tmpl.CreatorID, tmpl.SerializedData, tmpl.UsageCount, tmpl.IsPublic,
		tmpl.CreatedAt, tmpl.UpdatedAt,
	)
	return err
}

func (r *TemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ServerTemplate, error) {
	var tmpl models.ServerTemplate
	query := `SELECT * FROM server_templates WHERE id = $1`
	err := r.db.GetContext(ctx, &tmpl, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &tmpl, err
}

func (r *TemplateRepository) GetByCode(ctx context.Context, code string) (*models.ServerTemplate, error) {
	var tmpl models.ServerTemplate
	query := `SELECT * FROM server_templates WHERE code = $1`
	err := r.db.GetContext(ctx, &tmpl, query, code)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &tmpl, err
}

func (r *TemplateRepository) GetByCreator(ctx context.Context, creatorID uuid.UUID, limit int) ([]*models.ServerTemplate, error) {
	var templates []*models.ServerTemplate
	query := `SELECT * FROM server_templates WHERE creator_id = $1 ORDER BY created_at DESC LIMIT $2`
	err := r.db.SelectContext(ctx, &templates, query, creatorID, limit)
	return templates, err
}

func (r *TemplateRepository) ListPublic(ctx context.Context, cursor *uuid.UUID, limit int) ([]*models.ServerTemplate, *uuid.UUID, error) {
	var templates []*models.ServerTemplate
	var err error

	if cursor != nil {
		query := `SELECT * FROM server_templates WHERE is_public = TRUE AND id < $1 ORDER BY id DESC LIMIT $2`
		err = r.db.SelectContext(ctx, &templates, query, *cursor, limit)
	} else {
		query := `SELECT * FROM server_templates WHERE is_public = TRUE ORDER BY usage_count DESC, created_at DESC LIMIT $1`
		err = r.db.SelectContext(ctx, &templates, query, limit)
	}

	if err != nil {
		return nil, nil, err
	}

	var nextID *uuid.UUID
	if len(templates) == limit && templates[limit-1] != nil {
		nextID = &templates[limit-1].ID
	}

	return templates, nextID, nil
}

func (r *TemplateRepository) Update(ctx context.Context, tmpl *models.ServerTemplate) error {
	query := `
		UPDATE server_templates SET
			name = $2, description = $3, is_public = $4, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, tmpl.ID, tmpl.Name, tmpl.Description, tmpl.IsPublic)
	return err
}

func (r *TemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM server_templates WHERE id = $1`, id)
	return err
}

func (r *TemplateRepository) IncrementUsage(ctx context.Context, code string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE server_templates SET usage_count = usage_count + 1 WHERE code = $1`,
		code,
	)
	return err
}

func (r *TemplateRepository) GenerateUniqueCode(ctx context.Context) (string, error) {
	for attempt := 0; attempt < 10; attempt++ {
		var code string
		err := r.db.GetContext(ctx, &code,
			`SELECT generate_template_code()`)
		if err != nil {
			return "", fmt.Errorf("failed to generate code: %w", err)
		}

		// Check if code already exists
		var exists bool
		err = r.db.GetContext(ctx, &exists,
			`SELECT EXISTS(SELECT 1 FROM server_templates WHERE code = $1)`, code)
		if err != nil {
			return "", fmt.Errorf("failed to check code: %w", err)
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after 10 attempts")
}

// GetSerializedData parses and returns the SerializedData as a TemplateSerializedData struct
func (r *TemplateRepository) GetSerializedData(ctx context.Context, code string) (*models.TemplateSerializedData, error) {
	var data json.RawMessage
	query := `SELECT serialized_data FROM server_templates WHERE code = $1`
	err := r.db.GetContext(ctx, &data, query, code)
	if err != nil {
		return nil, err
	}

	var result models.TemplateSerializedData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse serialized data: %w", err)
	}

	return &result, nil
}
