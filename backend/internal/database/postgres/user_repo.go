package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	
	"hearth/internal/models"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, display_name, email, password_hash, avatar, banner, bio, pronouns, bot, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Username, user.DisplayName, user.Email, user.PasswordHash,
		user.Avatar, user.Banner, user.Bio, user.Pronouns, user.Bot,
		user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE username = $1`
	err := r.db.GetContext(ctx, &user, query, username)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users SET
			username = $2, display_name = $3, email = $4, password_hash = $5,
			avatar = $6, banner = $7, bio = $8, pronouns = $9, updated_at = $10
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Username, user.DisplayName, user.Email, user.PasswordHash,
		user.Avatar, user.Banner, user.Bio, user.Pronouns, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// Friends

func (r *UserRepository) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	query := `
		SELECT u.* FROM users u
		INNER JOIN relationships r ON (r.user_id = $1 AND r.target_id = u.id AND r.type = 1)
		OR (r.target_id = $1 AND r.user_id = u.id AND r.type = 1)
	`
	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userID)
	return users, err
}

func (r *UserRepository) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	query := `
		INSERT INTO relationships (user_id, target_id, type, created_at)
		VALUES ($1, $2, 1, $3)
		ON CONFLICT (user_id, target_id) DO UPDATE SET type = 1
	`
	_, err := r.db.ExecContext(ctx, query, userID, friendID, time.Now())
	return err
}

func (r *UserRepository) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	query := `DELETE FROM relationships WHERE (user_id = $1 AND target_id = $2) OR (user_id = $2 AND target_id = $1)`
	_, err := r.db.ExecContext(ctx, query, userID, friendID)
	return err
}

func (r *UserRepository) GetBlockedUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	query := `
		SELECT u.* FROM users u
		INNER JOIN relationships r ON r.user_id = $1 AND r.target_id = u.id AND r.type = 2
	`
	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userID)
	return users, err
}

func (r *UserRepository) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	query := `
		INSERT INTO relationships (user_id, target_id, type, created_at)
		VALUES ($1, $2, 2, $3)
		ON CONFLICT (user_id, target_id) DO UPDATE SET type = 2
	`
	_, err := r.db.ExecContext(ctx, query, userID, blockedID, time.Now())
	return err
}

func (r *UserRepository) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	query := `DELETE FROM relationships WHERE user_id = $1 AND target_id = $2 AND type = 2`
	_, err := r.db.ExecContext(ctx, query, userID, blockedID)
	return err
}

// Presence

func (r *UserRepository) UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus) error {
	query := `
		INSERT INTO presence (user_id, status, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET status = $2, updated_at = $3
	`
	_, err := r.db.ExecContext(ctx, query, userID, status, time.Now())
	return err
}

func (r *UserRepository) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	var presence models.Presence
	query := `SELECT * FROM presence WHERE user_id = $1`
	err := r.db.GetContext(ctx, &presence, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &presence, err
}

func (r *UserRepository) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]*models.Presence), nil
	}
	
	query, args, err := sqlx.In(`SELECT * FROM presence WHERE user_id IN (?)`, userIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	
	var presences []*models.Presence
	if err := r.db.SelectContext(ctx, &presences, query, args...); err != nil {
		return nil, err
	}
	
	result := make(map[uuid.UUID]*models.Presence)
	for _, p := range presences {
		result[p.UserID] = p
	}
	return result, nil
}
