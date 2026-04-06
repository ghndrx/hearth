package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

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
