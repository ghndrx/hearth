package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/ai"
)

var (
	ErrAIProviderNotFound   = errors.New("AI provider not found")
	ErrAICredentialNotFound = errors.New("AI credential not found")
	ErrModelRoutingNotFound = errors.New("model routing not found")
)

// AIRepository handles AI provider and credential persistence
type AIRepository struct {
	db *sqlx.DB
}

// NewAIRepository creates a new AIRepository
func NewAIRepository(db *sqlx.DB) *AIRepository {
	return &AIRepository{db: db}
}

// --- Provider Configurations ---

// CreateProviderConfig creates a new provider configuration
func (r *AIRepository) CreateProviderConfig(ctx context.Context, config *ai.AIProviderConfig) error {
	query := `
		INSERT INTO ai_providers (
			id, provider_type, name, display_name, base_url,
			is_enabled, is_default, priority, config, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.ProviderType, config.Name, config.DisplayName,
		config.BaseURL, config.IsEnabled, config.IsDefault, config.Priority,
		config.Config, config.CreatedAt, config.UpdatedAt,
	)
	return err
}

// GetProviderConfig retrieves a provider configuration by ID
func (r *AIRepository) GetProviderConfig(ctx context.Context, id uuid.UUID) (*ai.AIProviderConfig, error) {
	var config ai.AIProviderConfig
	query := `
		SELECT id, provider_type, name, display_name, base_url,
			is_enabled, is_default, priority, config, created_at, updated_at
		FROM ai_providers
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &config, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAIProviderNotFound
	}
	return &config, err
}

// GetProviderConfigByName retrieves a provider configuration by name
func (r *AIRepository) GetProviderConfigByName(ctx context.Context, name string) (*ai.AIProviderConfig, error) {
	var config ai.AIProviderConfig
	query := `
		SELECT id, provider_type, name, display_name, base_url,
			is_enabled, is_default, priority, config, created_at, updated_at
		FROM ai_providers
		WHERE name = $1
	`
	err := r.db.GetContext(ctx, &config, query, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAIProviderNotFound
	}
	return &config, err
}

// GetAllProviderConfigs retrieves all provider configurations
func (r *AIRepository) GetAllProviderConfigs(ctx context.Context) ([]*ai.AIProviderConfig, error) {
	var configs []*ai.AIProviderConfig
	query := `
		SELECT id, provider_type, name, display_name, base_url,
			is_enabled, is_default, priority, config, created_at, updated_at
		FROM ai_providers
		ORDER BY priority ASC, name ASC
	`
	err := r.db.SelectContext(ctx, &configs, query)
	return configs, err
}

// GetEnabledProviderConfigs retrieves all enabled provider configurations
func (r *AIRepository) GetEnabledProviderConfigs(ctx context.Context) ([]*ai.AIProviderConfig, error) {
	var configs []*ai.AIProviderConfig
	query := `
		SELECT id, provider_type, name, display_name, base_url,
			is_enabled, is_default, priority, config, created_at, updated_at
		FROM ai_providers
		WHERE is_enabled = TRUE
		ORDER BY priority ASC, name ASC
	`
	err := r.db.SelectContext(ctx, &configs, query)
	return configs, err
}

// GetDefaultProvider retrieves the default provider configuration
func (r *AIRepository) GetDefaultProvider(ctx context.Context) (*ai.AIProviderConfig, error) {
	var config ai.AIProviderConfig
	query := `
		SELECT id, provider_type, name, display_name, base_url,
			is_enabled, is_default, priority, config, created_at, updated_at
		FROM ai_providers
		WHERE is_default = TRUE AND is_enabled = TRUE
		ORDER BY priority ASC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &config, query)
	if errors.Is(err, sql.ErrNoRows) {
		// Fall back to first enabled provider
		query = `
			SELECT id, provider_type, name, display_name, base_url,
				is_enabled, is_default, priority, config, created_at, updated_at
			FROM ai_providers
			WHERE is_enabled = TRUE
			ORDER BY priority ASC
			LIMIT 1
		`
		err = r.db.GetContext(ctx, &config, query)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAIProviderNotFound
	}
	return &config, err
}

// UpdateProviderConfig updates a provider configuration
func (r *AIRepository) UpdateProviderConfig(ctx context.Context, config *ai.AIProviderConfig) error {
	query := `
		UPDATE ai_providers SET
			provider_type = $2,
			name = $3,
			display_name = $4,
			base_url = $5,
			is_enabled = $6,
			is_default = $7,
			priority = $8,
			config = $9,
			updated_at = $10
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		config.ID, config.ProviderType, config.Name, config.DisplayName,
		config.BaseURL, config.IsEnabled, config.IsDefault, config.Priority,
		config.Config, config.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAIProviderNotFound
	}
	return nil
}

// DeleteProviderConfig deletes a provider configuration
func (r *AIRepository) DeleteProviderConfig(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_providers WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAIProviderNotFound
	}
	return nil
}

// --- User Credentials ---

// CreateUserCredential creates a new user credential
func (r *AIRepository) CreateUserCredential(ctx context.Context, cred *ai.UserAICredential) error {
	query := `
		INSERT INTO user_ai_credentials (
			id, user_id, provider_id, provider_type, credentials,
			is_enabled, last_used_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		cred.ID, cred.UserID, cred.ProviderID, cred.ProviderType,
		cred.Credentials, cred.IsEnabled, cred.LastUsedAt,
		cred.CreatedAt, cred.UpdatedAt,
	)
	return err
}

// GetUserCredential retrieves a user credential by user and provider ID
func (r *AIRepository) GetUserCredential(ctx context.Context, userID, providerID uuid.UUID) (*ai.UserAICredential, error) {
	var cred ai.UserAICredential
	query := `
		SELECT id, user_id, provider_id, provider_type, credentials,
			is_enabled, last_used_at, created_at, updated_at
		FROM user_ai_credentials
		WHERE user_id = $1 AND provider_id = $2
	`
	err := r.db.GetContext(ctx, &cred, query, userID, providerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAICredentialNotFound
	}
	return &cred, err
}

// GetUserCredentials retrieves all credentials for a user
func (r *AIRepository) GetUserCredentials(ctx context.Context, userID uuid.UUID) ([]*ai.UserAICredential, error) {
	var creds []*ai.UserAICredential
	query := `
		SELECT id, user_id, provider_id, provider_type, credentials,
			is_enabled, last_used_at, created_at, updated_at
		FROM user_ai_credentials
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &creds, query, userID)
	return creds, err
}

// UpdateUserCredential updates a user credential
func (r *AIRepository) UpdateUserCredential(ctx context.Context, cred *ai.UserAICredential) error {
	query := `
		UPDATE user_ai_credentials SET
			credentials = $2,
			is_enabled = $3,
			updated_at = $4
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		cred.ID, cred.Credentials, cred.IsEnabled, cred.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAICredentialNotFound
	}
	return nil
}

// DeleteUserCredential deletes a user credential
func (r *AIRepository) DeleteUserCredential(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_ai_credentials WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAICredentialNotFound
	}
	return nil
}

// UpdateLastUsed updates the last used timestamp for a credential
func (r *AIRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE user_ai_credentials SET last_used_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, time.Now())
	return err
}

// --- Model Routing ---

// CreateModelRouting creates a new model routing
func (r *AIRepository) CreateModelRouting(ctx context.Context, routing *ai.ModelRouting) error {
	query := `
		INSERT INTO ai_model_routing (
			id, server_id, user_id, feature, provider_id, model_id,
			priority, is_enabled, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		routing.ID, routing.ServerID, routing.UserID, routing.Feature,
		routing.ProviderID, routing.ModelID, routing.Priority,
		routing.IsEnabled, routing.CreatedAt, routing.UpdatedAt,
	)
	return err
}

// GetModelRouting retrieves model routing for a feature
// Priority: user > server > global
func (r *AIRepository) GetModelRouting(ctx context.Context, feature ai.FeatureType, serverID, userID *uuid.UUID) (*ai.ModelRouting, error) {
	var routing ai.ModelRouting

	// Try user-specific routing first
	if userID != nil {
		query := `
			SELECT id, server_id, user_id, feature, provider_id, model_id,
				priority, is_enabled, created_at, updated_at
			FROM ai_model_routing
			WHERE feature = $1 AND user_id = $2 AND is_enabled = TRUE
			ORDER BY priority ASC
			LIMIT 1
		`
		err := r.db.GetContext(ctx, &routing, query, feature, userID)
		if err == nil {
			return &routing, nil
		}
	}

	// Try server-specific routing
	if serverID != nil {
		query := `
			SELECT id, server_id, user_id, feature, provider_id, model_id,
				priority, is_enabled, created_at, updated_at
			FROM ai_model_routing
			WHERE feature = $1 AND server_id = $2 AND user_id IS NULL AND is_enabled = TRUE
			ORDER BY priority ASC
			LIMIT 1
		`
		err := r.db.GetContext(ctx, &routing, query, feature, serverID)
		if err == nil {
			return &routing, nil
		}
	}

	// Try global routing
	query := `
		SELECT id, server_id, user_id, feature, provider_id, model_id,
			priority, is_enabled, created_at, updated_at
		FROM ai_model_routing
		WHERE feature = $1 AND server_id IS NULL AND user_id IS NULL AND is_enabled = TRUE
		ORDER BY priority ASC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &routing, query, feature)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrModelRoutingNotFound
	}
	return &routing, err
}

// GetAllModelRoutings retrieves all model routings
func (r *AIRepository) GetAllModelRoutings(ctx context.Context) ([]*ai.ModelRouting, error) {
	var routings []*ai.ModelRouting
	query := `
		SELECT id, server_id, user_id, feature, provider_id, model_id,
			priority, is_enabled, created_at, updated_at
		FROM ai_model_routing
		ORDER BY feature, priority ASC
	`
	err := r.db.SelectContext(ctx, &routings, query)
	return routings, err
}

// UpdateModelRouting updates a model routing
func (r *AIRepository) UpdateModelRouting(ctx context.Context, routing *ai.ModelRouting) error {
	query := `
		UPDATE ai_model_routing SET
			provider_id = $2,
			model_id = $3,
			priority = $4,
			is_enabled = $5,
			updated_at = $6
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		routing.ID, routing.ProviderID, routing.ModelID,
		routing.Priority, routing.IsEnabled, routing.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrModelRoutingNotFound
	}
	return nil
}

// DeleteModelRouting deletes a model routing
func (r *AIRepository) DeleteModelRouting(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_model_routing WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrModelRoutingNotFound
	}
	return nil
}
