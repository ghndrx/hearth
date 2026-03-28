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

// OAuthRepository handles OAuth provider database operations
type OAuthRepository struct {
	db *sqlx.DB
}

// NewOAuthRepository creates a new OAuth repository
func NewOAuthRepository(db *sqlx.DB) *OAuthRepository {
	return &OAuthRepository{db: db}
}

// Create creates a new OAuth provider link
func (r *OAuthRepository) Create(ctx context.Context, provider *models.OAuthProvider) error {
	query := `
		INSERT INTO oauth_providers (
			id, user_id, provider, provider_user_id, email, 
			username, display_name, avatar_url, access_token, refresh_token,
			token_expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	now := time.Now()
	if provider.ID == uuid.Nil {
		provider.ID = uuid.New()
	}
	if provider.CreatedAt.IsZero() {
		provider.CreatedAt = now
	}
	provider.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		provider.ID, provider.UserID, provider.Provider, provider.ProviderUserID, provider.Email,
		provider.Username, provider.DisplayName, provider.AvatarURL, provider.AccessToken, provider.RefreshToken,
		provider.TokenExpiresAt, provider.CreatedAt, provider.UpdatedAt,
	)
	return err
}

// GetByID retrieves an OAuth provider by ID
func (r *OAuthRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.OAuthProvider, error) {
	var provider models.OAuthProvider
	query := `SELECT * FROM oauth_providers WHERE id = $1`
	err := r.db.GetContext(ctx, &provider, query, id)
	if err == sql.ErrNoRows {
		return nil, services.ErrOAuthProviderNotFound
	}
	return &provider, err
}

// GetByUserID retrieves all OAuth providers for a user
func (r *OAuthRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OAuthProvider, error) {
	var providers []*models.OAuthProvider
	query := `SELECT * FROM oauth_providers WHERE user_id = $1 ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &providers, query, userID)
	return providers, err
}

// GetByUserAndProvider retrieves a specific OAuth provider for a user
func (r *OAuthRepository) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*models.OAuthProvider, error) {
	var oauthProvider models.OAuthProvider
	query := `SELECT * FROM oauth_providers WHERE user_id = $1 AND provider = $2`
	err := r.db.GetContext(ctx, &oauthProvider, query, userID, provider)
	if err == sql.ErrNoRows {
		return nil, services.ErrOAuthProviderNotFound
	}
	return &oauthProvider, err
}

// GetByProviderUserID retrieves an OAuth provider by provider and provider user ID
func (r *OAuthRepository) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*models.OAuthProvider, error) {
	var oauthProvider models.OAuthProvider
	query := `SELECT * FROM oauth_providers WHERE provider = $1 AND provider_user_id = $2`
	err := r.db.GetContext(ctx, &oauthProvider, query, provider, providerUserID)
	if err == sql.ErrNoRows {
		return nil, services.ErrOAuthProviderNotFound
	}
	return &oauthProvider, err
}

// Update updates an OAuth provider link
func (r *OAuthRepository) Update(ctx context.Context, provider *models.OAuthProvider) error {
	query := `
		UPDATE oauth_providers SET
			email = $2,
			username = $3,
			display_name = $4,
			avatar_url = $5,
			access_token = $6,
			refresh_token = $7,
			token_expires_at = $8,
			updated_at = $9
		WHERE id = $1
	`

	provider.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		provider.ID, provider.Email, provider.Username, provider.DisplayName,
		provider.AvatarURL, provider.AccessToken, provider.RefreshToken,
		provider.TokenExpiresAt, provider.UpdatedAt,
	)
	return err
}

// Delete removes an OAuth provider link
func (r *OAuthRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM oauth_providers WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return services.ErrOAuthProviderNotFound
	}
	return nil
}

// DeleteByUserAndProvider removes an OAuth provider link by user and provider
func (r *OAuthRepository) DeleteByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) error {
	query := `DELETE FROM oauth_providers WHERE user_id = $1 AND provider = $2`
	result, err := r.db.ExecContext(ctx, query, userID, provider)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return services.ErrOAuthProviderNotFound
	}
	return nil
}

// CountByUserID counts OAuth providers for a user
func (r *OAuthRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM oauth_providers WHERE user_id = $1`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

// ExistsByProviderUserID checks if an OAuth provider link exists
func (r *OAuthRepository) ExistsByProviderUserID(ctx context.Context, provider, providerUserID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM oauth_providers WHERE provider = $1 AND provider_user_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, provider, providerUserID)
	return exists, err
}

// GetUserIDByProviderUserID gets the user ID linked to an OAuth provider account
func (r *OAuthRepository) GetUserIDByProviderUserID(ctx context.Context, provider, providerUserID string) (uuid.UUID, error) {
	var userID uuid.UUID
	query := `SELECT user_id FROM oauth_providers WHERE provider = $1 AND provider_user_id = $2`
	err := r.db.GetContext(ctx, &userID, query, provider, providerUserID)
	if err == sql.ErrNoRows {
		return uuid.Nil, services.ErrOAuthProviderNotFound
	}
	return userID, err
}
