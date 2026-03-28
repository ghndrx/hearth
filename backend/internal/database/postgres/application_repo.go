package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"hearth/internal/models"
)

// ApplicationRepository handles application data storage
type ApplicationRepository struct {
	db *sqlx.DB
}

// NewApplicationRepository creates a new application repository
func NewApplicationRepository(db *sqlx.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

// Create creates a new application
func (r *ApplicationRepository) Create(ctx context.Context, app *models.Application) error {
	query := `
		INSERT INTO applications (id, name, description, icon, owner_id, verified, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		app.ID, app.Name, app.Description, app.Icon, app.OwnerID, app.Verified, app.CreatedAt,
	)
	return err
}

// GetByID retrieves an application by ID
func (r *ApplicationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Application, error) {
	query := `
		SELECT id, name, description, icon, owner_id, verified, created_at
		FROM applications WHERE id = $1
	`
	var app models.Application
	var icon sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&app.ID, &app.Name, &app.Description, &icon, &app.OwnerID, &app.Verified, &app.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if icon.Valid {
		app.Icon = icon.String
	}
	return &app, nil
}

// GetByOwnerID retrieves all applications for an owner
func (r *ApplicationRepository) GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*models.Application, error) {
	query := `
		SELECT id, name, description, icon, owner_id, verified, created_at
		FROM applications WHERE owner_id = $1 ORDER BY name
	`
	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*models.Application
	for rows.Next() {
		var app models.Application
		var icon sql.NullString
		err := rows.Scan(&app.ID, &app.Name, &app.Description, &icon, &app.OwnerID, &app.Verified, &app.CreatedAt)
		if err != nil {
			return nil, err
		}
		if icon.Valid {
			app.Icon = icon.String
		}
		apps = append(apps, &app)
	}
	return apps, rows.Err()
}

// Update updates an existing application
func (r *ApplicationRepository) Update(ctx context.Context, app *models.Application) error {
	query := `
		UPDATE applications SET name = $2, description = $3, icon = $4, verified = $5
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		app.ID, app.Name, app.Description, app.Icon, app.Verified,
	)
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

// Delete deletes an application
func (r *ApplicationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM applications WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
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

// SetVerified sets the verified status of an application
func (r *ApplicationRepository) SetVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	query := `UPDATE applications SET verified = $2 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, verified)
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

// CommandPermissionsRepository handles command permission data storage
type CommandPermissionsRepository struct {
	db *sqlx.DB
}

// NewCommandPermissionsRepository creates a new command permissions repository
func NewCommandPermissionsRepository(db *sqlx.DB) *CommandPermissionsRepository {
	return &CommandPermissionsRepository{db: db}
}

// CommandPermission represents a command permission entry
type CommandPermission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CommandID   uuid.UUID `json:"command_id" db:"command_id"`
	GuildID     uuid.UUID `json:"guild_id" db:"guild_id"`
	Permissions string    `json:"permissions" db:"permissions"` // JSONB stored as string
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// SetPermissions sets or updates permissions for a command in a guild
func (r *CommandPermissionsRepository) SetPermissions(ctx context.Context, commandID, guildID uuid.UUID, permissionsJSON []byte) error {
	query := `
		INSERT INTO command_permissions (id, command_id, guild_id, permissions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (command_id, guild_id)
		DO UPDATE SET permissions = $4, updated_at = $6
	`
	id := uuid.New()
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query, id, commandID, guildID, permissionsJSON, now, now)
	return err
}

// GetPermissions retrieves permissions for a command in a guild
func (r *CommandPermissionsRepository) GetPermissions(ctx context.Context, commandID, guildID uuid.UUID) ([]byte, error) {
	query := `SELECT permissions FROM command_permissions WHERE command_id = $1 AND guild_id = $2`
	var perms []byte
	err := r.db.QueryRowContext(ctx, query, commandID, guildID).Scan(&perms)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return perms, err
}

// GetByCommandID retrieves all permission entries for a command as JSON
func (r *CommandPermissionsRepository) GetByCommandID(ctx context.Context, commandID uuid.UUID) ([]byte, error) {
	query := `SELECT permissions FROM command_permissions WHERE command_id = $1 LIMIT 1`
	var perms []byte
	err := r.db.QueryRowContext(ctx, query, commandID).Scan(&perms)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return perms, err
}

// DeletePermissions deletes permissions for a command in a guild
func (r *CommandPermissionsRepository) DeletePermissions(ctx context.Context, commandID, guildID uuid.UUID) error {
	query := `DELETE FROM command_permissions WHERE command_id = $1 AND guild_id = $2`
	_, err := r.db.ExecContext(ctx, query, commandID, guildID)
	return err
}
