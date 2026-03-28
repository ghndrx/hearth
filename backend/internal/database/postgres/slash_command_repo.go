package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"hearth/internal/models"
)

// SlashCommandRepository handles slash command database operations
type SlashCommandRepository struct {
	db *sqlx.DB
}

// NewSlashCommandRepository creates a new slash command repository
func NewSlashCommandRepository(db *sqlx.DB) *SlashCommandRepository {
	return &SlashCommandRepository{db: db}
}

// Create creates a new slash command
func (r *SlashCommandRepository) Create(ctx context.Context, cmd *models.SlashCommand) error {
	var optionsJSON []byte
	var err error
	if cmd.Options != nil {
		optionsJSON, err = json.Marshal(cmd.Options)
		if err != nil {
			return err
		}
	}
	var permissionsJSON []byte
	if cmd.Permissions != nil {
		permissionsJSON, err = json.Marshal(cmd.Permissions)
		if err != nil {
			return err
		}
	}
	query := `
		INSERT INTO slash_commands (id, type, app_id, server_id, name, description,
			options, permissions, version, creator_id, default_permission, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err = r.db.ExecContext(ctx, query,
		cmd.ID, cmd.Type, cmd.AppID, cmd.ServerID, cmd.Name, cmd.Description,
		optionsJSON, permissionsJSON, cmd.Version, cmd.CreatorID, cmd.Default, cmd.CreatedAt, cmd.UpdatedAt,
	)
	return err
}

// GetByID retrieves a slash command by ID
func (r *SlashCommandRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SlashCommand, error) {
	query := `
		SELECT id, type, app_id, server_id, name, description, options, permissions,
			   version, creator_id, default_permission, created_at, updated_at
		FROM slash_commands WHERE id = $1
	`
	return r.scanSlashCommand(r.db.QueryRowContext(ctx, query, id))
}

// GetByAppID retrieves all slash commands for an application
func (r *SlashCommandRepository) GetByAppID(ctx context.Context, appID uuid.UUID) ([]*models.SlashCommand, error) {
	query := `
		SELECT id, type, app_id, server_id, name, description, options, permissions,
			   version, creator_id, default_permission, created_at, updated_at
		FROM slash_commands WHERE app_id = $1 ORDER BY name
	`
	rows, err := r.db.QueryContext(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSlashCommands(rows)
}

// GetByServerID retrieves all slash commands available in a server (global + server-specific)
func (r *SlashCommandRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.SlashCommand, error) {
	query := `
		SELECT id, type, app_id, server_id, name, description, options, permissions,
			   version, creator_id, default_permission, created_at, updated_at
		FROM slash_commands
		WHERE server_id IS NULL OR server_id = $1
		ORDER BY name
	`
	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSlashCommands(rows)
}

// GetByName retrieves a command by app ID and name (optionally server-scoped)
func (r *SlashCommandRepository) GetByName(ctx context.Context, appID uuid.UUID, name string, serverID *uuid.UUID) (*models.SlashCommand, error) {
	var cmd *models.SlashCommand
	var err error
	if serverID != nil {
		query := `
			SELECT id, type, app_id, server_id, name, description, options, permissions,
				   version, creator_id, default_permission, created_at, updated_at
			FROM slash_commands WHERE app_id = $1 AND name = $2 AND server_id = $3
		`
		cmd, err = r.scanSlashCommand(r.db.QueryRowContext(ctx, query, appID, name, *serverID))
	} else {
		query := `
			SELECT id, type, app_id, server_id, name, description, options, permissions,
				   version, creator_id, default_permission, created_at, updated_at
			FROM slash_commands WHERE app_id = $1 AND name = $2 AND server_id IS NULL
		`
		cmd, err = r.scanSlashCommand(r.db.QueryRowContext(ctx, query, appID, name))
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return cmd, err
}

// Update updates an existing slash command
func (r *SlashCommandRepository) Update(ctx context.Context, cmd *models.SlashCommand) error {
	var optionsJSON []byte
	var err error
	if cmd.Options != nil {
		optionsJSON, err = json.Marshal(cmd.Options)
		if err != nil {
			return err
		}
	}
	var permissionsJSON []byte
	if cmd.Permissions != nil {
		permissionsJSON, err = json.Marshal(cmd.Permissions)
		if err != nil {
			return err
		}
	}
	query := `
		UPDATE slash_commands SET name = $2, description = $3, options = $4,
			permissions = $5, version = $6, default_permission = $7, updated_at = $8
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		cmd.ID, cmd.Name, cmd.Description, optionsJSON, permissionsJSON,
		cmd.Version, cmd.Default, cmd.UpdatedAt,
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

// Delete deletes a slash command
func (r *SlashCommandRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM slash_commands WHERE id = $1`
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

// CreateExecutionLog logs a command execution
func (r *SlashCommandRepository) CreateExecutionLog(ctx context.Context, log *models.CommandExecution) error {
	query := `
		INSERT INTO command_executions (id, command_id, app_id, server_id, channel_id,
			user_id, options, response_time_ms, status, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.CommandID, log.AppID, log.ServerID, log.ChannelID,
		log.UserID, log.Options, log.ResponseTime, log.Status, log.ErrorMsg, log.CreatedAt,
	)
	return err
}

func (r *SlashCommandRepository) scanSlashCommand(row *sql.Row) (*models.SlashCommand, error) {
	var cmd models.SlashCommand
	var optionsJSON, permissionsJSON []byte
	var serverID, creatorID sql.NullString
	err := row.Scan(
		&cmd.ID, &cmd.Type, &cmd.AppID, &serverID, &cmd.Name, &cmd.Description,
		&optionsJSON, &permissionsJSON, &cmd.Version, &creatorID,
		&cmd.Default, &cmd.CreatedAt, &cmd.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if serverID.Valid {
		id, _ := uuid.Parse(serverID.String)
		cmd.ServerID = &id
	}
	if creatorID.Valid {
		id, _ := uuid.Parse(creatorID.String)
		cmd.CreatorID = &id
	}
	if len(optionsJSON) > 0 {
		json.Unmarshal(optionsJSON, &cmd.Options)
	}
	if len(permissionsJSON) > 0 {
		json.Unmarshal(permissionsJSON, &cmd.Permissions)
	}
	return &cmd, nil
}

func (r *SlashCommandRepository) scanSlashCommands(rows *sql.Rows) ([]*models.SlashCommand, error) {
	var commands []*models.SlashCommand
	for rows.Next() {
		var cmd models.SlashCommand
		var optionsJSON, permissionsJSON []byte
		var serverID, creatorID sql.NullString
		err := rows.Scan(
			&cmd.ID, &cmd.Type, &cmd.AppID, &serverID, &cmd.Name, &cmd.Description,
			&optionsJSON, &permissionsJSON, &cmd.Version, &creatorID,
			&cmd.Default, &cmd.CreatedAt, &cmd.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if serverID.Valid {
			id, _ := uuid.Parse(serverID.String)
			cmd.ServerID = &id
		}
		if creatorID.Valid {
			id, _ := uuid.Parse(creatorID.String)
			cmd.CreatorID = &id
		}
		if len(optionsJSON) > 0 {
			json.Unmarshal(optionsJSON, &cmd.Options)
		}
		if len(permissionsJSON) > 0 {
			json.Unmarshal(permissionsJSON, &cmd.Permissions)
		}
		commands = append(commands, &cmd)
	}
	return commands, rows.Err()
}
