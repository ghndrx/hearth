package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	
	"hearth/internal/models"
	"hearth/internal/services"
)

var ErrUserNotFound = services.ErrUserNotFound

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, discriminator, email, password_hash, avatar_url, banner_url, bio, status, mfa_enabled, verified, flags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Discriminator, user.Email, user.PasswordHash,
		user.AvatarURL, user.BannerURL, user.Bio, user.Status, user.MFAEnabled,
		user.Verified, user.Flags, user.CreatedAt, user.UpdatedAt,
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
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users SET
			username = $2, discriminator = $3, email = $4, password_hash = $5,
			avatar_url = $6, banner_url = $7, bio = $8, status = $9, 
			custom_status = $10, mfa_enabled = $11, verified = $12, flags = $13, updated_at = $14
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Discriminator, user.Email, user.PasswordHash,
		user.AvatarURL, user.BannerURL, user.Bio, user.Status, user.CustomStatus,
		user.MFAEnabled, user.Verified, user.Flags, user.UpdatedAt,
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

// GetRelationship gets the relationship between two users
func (r *UserRepository) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	var relType int
	query := `SELECT type FROM relationships WHERE user_id = $1 AND target_id = $2`
	err := r.db.GetContext(ctx, &relType, query, userID, targetID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return relType, err
}

// SendFriendRequest creates a pending friend request from sender to receiver
func (r *UserRepository) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	// Create outgoing request for sender
	_, err = tx.ExecContext(ctx, `
		INSERT INTO relationships (user_id, target_id, type, created_at)
		VALUES ($1, $2, 4, $3)
		ON CONFLICT (user_id, target_id) DO UPDATE SET type = 4, created_at = $3
	`, senderID, receiverID, now)
	if err != nil {
		return err
	}

	// Create incoming request for receiver
	_, err = tx.ExecContext(ctx, `
		INSERT INTO relationships (user_id, target_id, type, created_at)
		VALUES ($1, $2, 3, $3)
		ON CONFLICT (user_id, target_id) DO UPDATE SET type = 3, created_at = $3
	`, receiverID, senderID, now)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetIncomingFriendRequests gets all pending incoming friend requests for a user
func (r *UserRepository) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	query := `
		SELECT u.* FROM users u
		INNER JOIN relationships r ON r.user_id = $1 AND r.target_id = u.id AND r.type = 3
	`
	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userID)
	return users, err
}

// GetOutgoingFriendRequests gets all pending outgoing friend requests for a user
func (r *UserRepository) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	query := `
		SELECT u.* FROM users u
		INNER JOIN relationships r ON r.user_id = $1 AND r.target_id = u.id AND r.type = 4
	`
	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userID)
	return users, err
}

// AcceptFriendRequest accepts a pending friend request
func (r *UserRepository) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update both relationships to type 1 (friend)
	_, err = tx.ExecContext(ctx, `
		UPDATE relationships SET type = 1 WHERE user_id = $1 AND target_id = $2 AND type = 3
	`, receiverID, senderID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE relationships SET type = 1 WHERE user_id = $1 AND target_id = $2 AND type = 4
	`, senderID, receiverID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeclineFriendRequest declines/cancels a pending friend request
func (r *UserRepository) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove both pending relationships (works for both declining incoming and canceling outgoing)
	_, err = tx.ExecContext(ctx, `
		DELETE FROM relationships WHERE user_id = $1 AND target_id = $2 AND type IN (3, 4)
	`, userID, otherID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM relationships WHERE user_id = $1 AND target_id = $2 AND type IN (3, 4)
	`, otherID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
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
