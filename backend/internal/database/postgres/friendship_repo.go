package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// FriendshipRepository handles friendship-related database operations
type FriendshipRepository struct {
	db *sqlx.DB
}

// NewFriendshipRepository creates a new friendship repository
func NewFriendshipRepository(db *sqlx.DB) *FriendshipRepository {
	return &FriendshipRepository{db: db}
}

// Create creates a new friendship record
func (r *FriendshipRepository) Create(ctx context.Context, friendship *models.Friendship) error {
	query := `
		INSERT INTO friendships (id, user_id_1, user_id_2, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		friendship.ID, friendship.UserID1, friendship.UserID2, friendship.Status, friendship.CreatedAt, friendship.UpdatedAt)
	return err
}

// FetchByMembers retrieves a friendship by the two member IDs
func (r *FriendshipRepository) FetchByMembers(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*models.Friendship, error) {
	query := `
		SELECT id, user_id_1, user_id_2, status, created_at, updated_at
		FROM friendships
		WHERE (user_id_1 = $1 AND user_id_2 = $2) OR (user_id_1 = $2 AND user_id_2 = $1)
	`
	var friendship models.Friendship
	err := r.db.GetContext(ctx, &friendship, query, user1ID, user2ID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" || err.Error() == "sql: no rows in result set" {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}
	return &friendship, nil
}

// ListFriends retrieves all friends for a user
func (r *FriendshipRepository) ListFriends(ctx context.Context, userID uuid.UUID) ([]models.User, error) {
	query := `
		SELECT u.id, u.username, u.discriminator, u.email, u.password_hash, u.avatar_url, u.banner_url, u.bio, u.status, u.custom_status, u.mfa_enabled, u.verified, u.flags, u.created_at, u.updated_at
		FROM users u
		JOIN friendships f ON (u.id = f.user_id_1 OR u.id = f.user_id_2)
		WHERE (f.user_id_1 = $1 OR f.user_id_2 = $1)
		AND u.id != $1
		AND f.status = 'accepted'
	`
	var users []models.User
	err := r.db.SelectContext(ctx, &users, query, userID)
	return users, err
}

// Remove deletes a friendship by ID
func (r *FriendshipRepository) Remove(ctx context.Context, friendshipID uuid.UUID) error {
	query := `DELETE FROM friendships WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, friendshipID)
	return err
}

// PendingRequests retrieves all pending friend requests for a user
func (r *FriendshipRepository) PendingRequests(ctx context.Context, userID uuid.UUID) ([]models.Friendship, error) {
	query := `
		SELECT id, user_id_1, user_id_2, status, created_at, updated_at
		FROM friendships
		WHERE (user_id_1 = $1 OR user_id_2 = $1)
		AND status = 'pending'
	`
	var friendships []models.Friendship
	err := r.db.SelectContext(ctx, &friendships, query, userID)
	return friendships, err
}
