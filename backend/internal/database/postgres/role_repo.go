package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

type RoleRepository struct {
	db *sqlx.DB
}

func NewRoleRepository(db *sqlx.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(ctx context.Context, role *models.Role) error {
	query := `
		INSERT INTO roles (id, server_id, name, color, hoist, position, permissions, mentionable, is_default, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		role.ID, role.ServerID, role.Name, role.Color, role.Hoist, role.Position,
		role.Permissions, role.Mentionable, role.IsDefault, role.CreatedAt,
	)
	return err
}

func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	var role models.Role
	query := `SELECT * FROM roles WHERE id = $1`
	err := r.db.GetContext(ctx, &role, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &role, err
}

func (r *RoleRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error) {
	var roles []*models.Role
	query := `SELECT * FROM roles WHERE server_id = $1 ORDER BY position DESC`
	err := r.db.SelectContext(ctx, &roles, query, serverID)
	return roles, err
}

func (r *RoleRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Role, error) {
	if len(ids) == 0 {
		return []*models.Role{}, nil
	}

	query, args, err := sqlx.In(`SELECT * FROM roles WHERE id IN (?) ORDER BY position DESC`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var roles []*models.Role
	err = r.db.SelectContext(ctx, &roles, query, args...)
	return roles, err
}

func (r *RoleRepository) Update(ctx context.Context, role *models.Role) error {
	query := `
		UPDATE roles SET
			name = $2, color = $3, hoist = $4, position = $5,
			permissions = $6, mentionable = $7
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		role.ID, role.Name, role.Color, role.Hoist, role.Position,
		role.Permissions, role.Mentionable,
	)
	return err
}

func (r *RoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM roles WHERE id = $1 AND is_default = false`, id)
	return err
}

func (r *RoleRepository) AddRoleToMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	query := `
		UPDATE members 
		SET roles = array_append(roles, $3)
		WHERE server_id = $1 AND user_id = $2 AND NOT ($3 = ANY(roles))
	`
	_, err := r.db.ExecContext(ctx, query, serverID, userID, roleID)
	return err
}

func (r *RoleRepository) RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error {
	query := `
		UPDATE members 
		SET roles = array_remove(roles, $3)
		WHERE server_id = $1 AND user_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, serverID, userID, roleID)
	return err
}

func (r *RoleRepository) UpdatePositions(ctx context.Context, serverID uuid.UUID, positions map[uuid.UUID]int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for roleID, position := range positions {
		_, err := tx.ExecContext(ctx,
			`UPDATE roles SET position = $1 WHERE id = $2 AND server_id = $3`,
			position, roleID, serverID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *RoleRepository) GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error) {
	query := `
		SELECT r.* FROM roles r
		INNER JOIN members m ON r.id = ANY(m.roles)
		WHERE m.server_id = $1 AND m.user_id = $2
		ORDER BY r.position DESC
	`
	var roles []*models.Role
	err := r.db.SelectContext(ctx, &roles, query, serverID, userID)
	return roles, err
}

// GetMemberPermissions calculates combined permissions for a member
func (r *RoleRepository) GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	query := `
		SELECT COALESCE(bit_or(r.permissions), 0) as permissions
		FROM members m
		INNER JOIN roles r ON r.id = ANY(m.roles)
		WHERE m.server_id = $1 AND m.user_id = $2
	`
	var permissions int64
	err := r.db.GetContext(ctx, &permissions, query, serverID, userID)
	return permissions, err
}

// GetDefaultRole returns the @everyone role for a server
func (r *RoleRepository) GetDefaultRole(ctx context.Context, serverID uuid.UUID) (*models.Role, error) {
	var role models.Role
	query := `SELECT * FROM roles WHERE server_id = $1 AND is_default = true`
	err := r.db.GetContext(ctx, &role, query, serverID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &role, err
}
