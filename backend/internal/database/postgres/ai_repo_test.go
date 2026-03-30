package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/ai"
)

func setupAIRepoMock(t *testing.T) (*AIRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewAIRepository(sqlxDB)
	return repo, mock
}

var providerConfigColumns = []string{
	"id", "provider_type", "name", "display_name", "base_url",
	"is_enabled", "is_default", "priority", "config", "created_at", "updated_at",
}

func TestAIRepository_CreateProviderConfig(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	config := &ai.AIProviderConfig{
		ID:           uuid.New(),
		ProviderType: ai.ProviderOpenAI,
		Name:         "openai-main",
		DisplayName:  "OpenAI Main",
		BaseURL:      stringPtr("https://api.openai.com"),
		IsEnabled:    true,
		IsDefault:   true,
		Priority:    1,
		Config:      stringPtr(`{"api_key":"secret"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO ai_providers").
		WithArgs(
			config.ID, config.ProviderType, config.Name, config.DisplayName,
			config.BaseURL, config.IsEnabled, config.IsDefault, config.Priority,
			config.Config, config.CreatedAt, config.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateProviderConfig(ctx, config)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_CreateProviderConfig_Error(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	config := &ai.AIProviderConfig{
		ID:           uuid.New(),
		ProviderType: ai.ProviderOpenAI,
		Name:         "openai-main",
		DisplayName:  "OpenAI Main",
		IsEnabled:    true,
		IsDefault:   true,
		Priority:    1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO ai_providers").
		WillReturnError(fmt.Errorf("database error"))

	err := repo.CreateProviderConfig(ctx, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestAIRepository_GetProviderConfig(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()
	now := time.Now()
	baseURL := "https://api.openai.com"
	configJSON := `{"api_key":"secret"}`

	rows := sqlmock.NewRows(providerConfigColumns).AddRow(
		providerID, ai.ProviderOpenAI, "openai-main", "OpenAI Main",
		&baseURL, true, true, 1, &configJSON, now, now,
	)

	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE id = \\$1").
		WithArgs(providerID).
		WillReturnRows(rows)

	config, err := repo.GetProviderConfig(ctx, providerID)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, providerID, config.ID)
	assert.Equal(t, ai.ProviderOpenAI, config.ProviderType)
	assert.Equal(t, "openai-main", config.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetProviderConfig_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE id = \\$1").
		WithArgs(providerID).
		WillReturnError(sql.ErrNoRows)

	config, err := repo.GetProviderConfig(ctx, providerID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAIProviderNotFound))
	assert.Nil(t, config)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetProviderConfigByName(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows(providerConfigColumns).AddRow(
		providerID, ai.ProviderOpenAI, "openai-main", "OpenAI Main",
		nil, true, true, 1, nil, now, now,
	)

	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE name = \\$1").
		WithArgs("openai-main").
		WillReturnRows(rows)

	config, err := repo.GetProviderConfigByName(ctx, "openai-main")
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "openai-main", config.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetProviderConfigByName_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE name = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	config, err := repo.GetProviderConfigByName(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAIProviderNotFound))
	assert.Nil(t, config)
}

func TestAIRepository_GetAllProviderConfigs(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	id1 := uuid.New()
	id2 := uuid.New()

	rows := sqlmock.NewRows(providerConfigColumns).
		AddRow(id1, ai.ProviderOpenAI, "openai-main", "OpenAI Main", nil, true, true, 1, nil, now, now).
		AddRow(id2, ai.ProviderAnthropic, "anthropic-main", "Anthropic Main", nil, true, false, 2, nil, now, now)

	mock.ExpectQuery("SELECT .+ FROM ai_providers").
		WillReturnRows(rows)

	configs, err := repo.GetAllProviderConfigs(ctx)
	require.NoError(t, err)
	require.Len(t, configs, 2)
	assert.Equal(t, "openai-main", configs[0].Name)
	assert.Equal(t, "anthropic-main", configs[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetEnabledProviderConfigs(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	id1 := uuid.New()

	rows := sqlmock.NewRows(providerConfigColumns).
		AddRow(id1, ai.ProviderOpenAI, "openai-main", "OpenAI Main", nil, true, true, 1, nil, now, now)

	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE is_enabled = TRUE").
		WillReturnRows(rows)

	configs, err := repo.GetEnabledProviderConfigs(ctx)
	require.NoError(t, err)
	require.Len(t, configs, 1)
	assert.True(t, configs[0].IsEnabled)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetDefaultProvider(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	id1 := uuid.New()

	rows := sqlmock.NewRows(providerConfigColumns).AddRow(
		id1, ai.ProviderOpenAI, "openai-main", "OpenAI Main", nil, true, true, 1, nil, now, now,
	)

	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE is_default = TRUE AND is_enabled = TRUE").
		WillReturnRows(rows)

	config, err := repo.GetDefaultProvider(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.True(t, config.IsDefault)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetDefaultProvider_Fallback(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	id1 := uuid.New()

	// First query returns no rows
	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE is_default = TRUE AND is_enabled = TRUE").
		WillReturnError(sql.ErrNoRows)

	// Fallback query returns a provider
	rows2 := sqlmock.NewRows(providerConfigColumns).AddRow(
		id1, ai.ProviderOpenAI, "openai-main", "OpenAI Main", nil, true, false, 1, nil, now, now,
	)
	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE is_enabled = TRUE").
		WillReturnRows(rows2)

	config, err := repo.GetDefaultProvider(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetDefaultProvider_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	// First query returns no rows
	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE is_default = TRUE AND is_enabled = TRUE").
		WillReturnError(sql.ErrNoRows)

	// Fallback also returns no rows
	mock.ExpectQuery("SELECT .+ FROM ai_providers WHERE is_enabled = TRUE").
		WillReturnError(sql.ErrNoRows)

	config, err := repo.GetDefaultProvider(ctx)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAIProviderNotFound))
	assert.Nil(t, config)
}

func TestAIRepository_UpdateProviderConfig(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	config := &ai.AIProviderConfig{
		ID:           uuid.New(),
		ProviderType: ai.ProviderOpenAI,
		Name:         "openai-updated",
		DisplayName:  "OpenAI Updated",
		BaseURL:      stringPtr("https://api.openai.com/v1"),
		IsEnabled:    true,
		IsDefault:   false,
		Priority:    2,
		Config:      stringPtr(`{"api_key":"newkey"}`),
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec("UPDATE ai_providers SET").
		WithArgs(
			config.ID, config.ProviderType, config.Name, config.DisplayName,
			config.BaseURL, config.IsEnabled, config.IsDefault, config.Priority,
			config.Config, config.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateProviderConfig(ctx, config)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_UpdateProviderConfig_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	config := &ai.AIProviderConfig{
		ID:        uuid.New(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec("UPDATE ai_providers SET").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateProviderConfig(ctx, config)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAIProviderNotFound))
}

func TestAIRepository_DeleteProviderConfig(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()

	mock.ExpectExec("DELETE FROM ai_providers WHERE id = \\$1").
		WithArgs(providerID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteProviderConfig(ctx, providerID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_DeleteProviderConfig_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()

	mock.ExpectExec("DELETE FROM ai_providers WHERE id = \\$1").
		WithArgs(providerID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeleteProviderConfig(ctx, providerID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAIProviderNotFound))
}

func TestAIRepository_CreateUserCredential(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	cred := &ai.UserAICredential{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		ProviderID:   uuid.New(),
		ProviderType: ai.ProviderOpenAI,
		Credentials:  `{"api_key":"secret"}`,
		IsEnabled:    true,
		LastUsedAt:   nil,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mock.ExpectExec("INSERT INTO user_ai_credentials").
		WithArgs(
			cred.ID, cred.UserID, cred.ProviderID, cred.ProviderType,
			cred.Credentials, cred.IsEnabled, cred.LastUsedAt,
			cred.CreatedAt, cred.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateUserCredential(ctx, cred)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetUserCredential(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	providerID := uuid.New()
	credID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "provider_id", "provider_type", "credentials",
		"is_enabled", "last_used_at", "created_at", "updated_at",
	}).AddRow(credID, userID, providerID, ai.ProviderOpenAI, `{"api_key":"secret"}`, true, nil, now, now)

	mock.ExpectQuery("SELECT .+ FROM user_ai_credentials WHERE user_id = \\$1 AND provider_id = \\$2").
		WithArgs(userID, providerID).
		WillReturnRows(rows)

	cred, err := repo.GetUserCredential(ctx, userID, providerID)
	require.NoError(t, err)
	require.NotNil(t, cred)
	assert.Equal(t, userID, cred.UserID)
	assert.Equal(t, providerID, cred.ProviderID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetUserCredential_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	providerID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM user_ai_credentials WHERE user_id = \\$1 AND provider_id = \\$2").
		WithArgs(userID, providerID).
		WillReturnError(sql.ErrNoRows)

	cred, err := repo.GetUserCredential(ctx, userID, providerID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAICredentialNotFound))
	assert.Nil(t, cred)
}

func TestAIRepository_GetUserCredentials(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()
	credID1 := uuid.New()
	credID2 := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "provider_id", "provider_type", "credentials",
		"is_enabled", "last_used_at", "created_at", "updated_at",
	}).
		AddRow(credID1, userID, uuid.New(), ai.ProviderOpenAI, `{}`, true, nil, now, now).
		AddRow(credID2, userID, uuid.New(), ai.ProviderAnthropic, `{}`, true, nil, now, now)

	mock.ExpectQuery("SELECT .+ FROM user_ai_credentials WHERE user_id = \\$1").
		WithArgs(userID).
		WillReturnRows(rows)

	creds, err := repo.GetUserCredentials(ctx, userID)
	require.NoError(t, err)
	require.Len(t, creds, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_UpdateUserCredential(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	cred := &ai.UserAICredential{
		ID:          uuid.New(),
		Credentials: `{"api_key":"newkey"}`,
		IsEnabled:   false,
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec("UPDATE user_ai_credentials SET").
		WithArgs(cred.ID, cred.Credentials, cred.IsEnabled, cred.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateUserCredential(ctx, cred)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_UpdateUserCredential_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	cred := &ai.UserAICredential{
		ID:        uuid.New(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec("UPDATE user_ai_credentials SET").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateUserCredential(ctx, cred)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAICredentialNotFound))
}

func TestAIRepository_DeleteUserCredential(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	credID := uuid.New()

	mock.ExpectExec("DELETE FROM user_ai_credentials WHERE id = \\$1").
		WithArgs(credID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteUserCredential(ctx, credID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_DeleteUserCredential_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	credID := uuid.New()

	mock.ExpectExec("DELETE FROM user_ai_credentials WHERE id = \\$1").
		WithArgs(credID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeleteUserCredential(ctx, credID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAICredentialNotFound))
}

func TestAIRepository_UpdateLastUsed(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	credID := uuid.New()

	mock.ExpectExec("UPDATE user_ai_credentials SET last_used_at = \\$2 WHERE id = \\$1").
		WithArgs(credID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateLastUsed(ctx, credID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_CreateModelRouting(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	routing := &ai.ModelRouting{
		ID:         uuid.New(),
		ServerID:   nil,
		UserID:     nil,
		Feature:    ai.FeatureChat,
		ProviderID: uuid.New(),
		ModelID:    "gpt-4",
		Priority:   1,
		IsEnabled:  true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mock.ExpectExec("INSERT INTO ai_model_routing").
		WithArgs(
			routing.ID, routing.ServerID, routing.UserID, routing.Feature,
			routing.ProviderID, routing.ModelID, routing.Priority,
			routing.IsEnabled, routing.CreatedAt, routing.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateModelRouting(ctx, routing)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetModelRouting_UserLevel(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "user_id", "feature", "provider_id", "model_id",
		"priority", "is_enabled", "created_at", "updated_at",
	}).AddRow(uuid.New(), nil, userID, ai.FeatureChat, providerID, "gpt-4", 1, true, now, now)

	mock.ExpectQuery("SELECT .+ FROM ai_model_routing WHERE feature = \\$1 AND user_id = \\$2 AND is_enabled = TRUE").
		WithArgs(ai.FeatureChat, userID).
		WillReturnRows(rows)

	routing, err := repo.GetModelRouting(ctx, ai.FeatureChat, nil, &userID)
	require.NoError(t, err)
	require.NotNil(t, routing)
	assert.Equal(t, "gpt-4", routing.ModelID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetModelRouting_ServerLevel(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	// User-level query is skipped since userID is nil
	// But we need to expect the server-level query since that's what gets called
	rows := sqlmock.NewRows([]string{
		"id", "server_id", "user_id", "feature", "provider_id", "model_id",
		"priority", "is_enabled", "created_at", "updated_at",
	}).AddRow(uuid.New(), serverID, nil, ai.FeatureChat, providerID, "gpt-4-turbo", 1, true, now, now)

	mock.ExpectQuery("SELECT .+ FROM ai_model_routing WHERE feature = \\$1 AND server_id = \\$2 AND user_id IS NULL AND is_enabled = TRUE").
		WithArgs(ai.FeatureChat, serverID).
		WillReturnRows(rows)

	routing, err := repo.GetModelRouting(ctx, ai.FeatureChat, &serverID, nil)
	require.NoError(t, err)
	require.NotNil(t, routing)
	assert.Equal(t, "gpt-4-turbo", routing.ModelID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetModelRouting_GlobalLevel(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	// Server-level returns no rows (user-level is skipped since userID is nil)
	mock.ExpectQuery("SELECT .+ FROM ai_model_routing WHERE feature = \\$1 AND server_id = \\$2 AND user_id IS NULL AND is_enabled = TRUE").
		WithArgs(ai.FeatureChat, serverID).
		WillReturnError(sql.ErrNoRows)

	// Global-level returns a routing
	rows := sqlmock.NewRows([]string{
		"id", "server_id", "user_id", "feature", "provider_id", "model_id",
		"priority", "is_enabled", "created_at", "updated_at",
	}).AddRow(uuid.New(), nil, nil, ai.FeatureChat, providerID, "gpt-3.5-turbo", 1, true, now, now)

	mock.ExpectQuery("SELECT .+ FROM ai_model_routing WHERE feature = \\$1 AND server_id IS NULL AND user_id IS NULL AND is_enabled = TRUE").
		WithArgs(ai.FeatureChat).
		WillReturnRows(rows)

	routing, err := repo.GetModelRouting(ctx, ai.FeatureChat, &serverID, nil)
	require.NoError(t, err)
	require.NotNil(t, routing)
	assert.Equal(t, "gpt-3.5-turbo", routing.ModelID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_GetModelRouting_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	// With both nil, only global query should be called
	mock.ExpectQuery("SELECT .+ FROM ai_model_routing WHERE feature = \\$1 AND server_id IS NULL AND user_id IS NULL AND is_enabled = TRUE").
		WillReturnError(sql.ErrNoRows)

	routing, err := repo.GetModelRouting(ctx, ai.FeatureChat, nil, nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelRoutingNotFound))
	assert.Nil(t, routing)
}

func TestAIRepository_GetAllModelRoutings(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	id1 := uuid.New()
	id2 := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "user_id", "feature", "provider_id", "model_id",
		"priority", "is_enabled", "created_at", "updated_at",
	}).
		AddRow(id1, nil, nil, ai.FeatureChat, uuid.New(), "gpt-4", 1, true, now, now).
		AddRow(id2, nil, nil, ai.FeatureSearch, uuid.New(), "gpt-4-turbo", 1, true, now, now)

	mock.ExpectQuery("SELECT .+ FROM ai_model_routing").
		WillReturnRows(rows)

	routings, err := repo.GetAllModelRoutings(ctx)
	require.NoError(t, err)
	require.Len(t, routings, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_UpdateModelRouting(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	routing := &ai.ModelRouting{
		ID:         uuid.New(),
		ProviderID: uuid.New(),
		ModelID:    "gpt-4-turbo",
		Priority:   2,
		IsEnabled:  true,
		UpdatedAt:  time.Now(),
	}

	mock.ExpectExec("UPDATE ai_model_routing SET").
		WithArgs(routing.ID, routing.ProviderID, routing.ModelID, routing.Priority, routing.IsEnabled, routing.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateModelRouting(ctx, routing)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_UpdateModelRouting_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	routing := &ai.ModelRouting{
		ID:        uuid.New(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec("UPDATE ai_model_routing SET").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateModelRouting(ctx, routing)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelRoutingNotFound))
}

func TestAIRepository_DeleteModelRouting(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	routingID := uuid.New()

	mock.ExpectExec("DELETE FROM ai_model_routing WHERE id = \\$1").
		WithArgs(routingID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteModelRouting(ctx, routingID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAIRepository_DeleteModelRouting_NotFound(t *testing.T) {
	repo, mock := setupAIRepoMock(t)
	ctx := context.Background()

	routingID := uuid.New()

	mock.ExpectExec("DELETE FROM ai_model_routing WHERE id = \\$1").
		WithArgs(routingID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeleteModelRouting(ctx, routingID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelRoutingNotFound))
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
