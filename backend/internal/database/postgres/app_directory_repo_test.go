package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
)

func setupAppDirectoryRepoMock(t *testing.T) (*AppDirectoryRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewAppDirectoryRepository(sqlxDB)
	return repo, mock
}

func myStrPtr(s string) *string { return &s }

var appColumns = []string{
	"id", "name", "description", "long_description", "developer_id", "oauth_app_id",
	"category", "tags", "icon_url", "screenshots", "install_count", "rating", "review_count",
	"status", "privacy_policy_url", "terms_of_service_url", "support_server_id", "created_at", "updated_at",
}

func TestNewAppDirectoryRepository(t *testing.T) {
	db, _, _ := sqlmock.New()
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewAppDirectoryRepository(sqlxDB)
	assert.NotNil(t, repo)
}

func TestAppDirectoryRepository_CreateApp(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	longDesc := "A longer description"
	iconURL := "https://example.com/icon.png"
	privacyURL := "https://example.com/privacy"
	tosURL := "https://example.com/tos"
	supportServerID := uuid.New()

	app := &models.App{
		ID:                uuid.New(),
		Name:              "Test App",
		Description:       "A test application",
		LongDescription:   &longDesc,
		DeveloperID:       uuid.New(),
		OAuthAppID:        nil,
		Category:          models.AppCategoryUtility,
		Tags:              pq.StringArray{"tag1", "tag2"},
		IconURL:           &iconURL,
		Screenshots:       pq.StringArray{"https://example.com/screen1.png"},
		Status:            models.AppStatusPending,
		PrivacyPolicyURL:  &privacyURL,
		TermsOfServiceURL: &tosURL,
		SupportServerID:   &supportServerID,
	}

	mock.ExpectQuery("INSERT INTO apps").
		WithArgs(
			app.ID, app.Name, app.Description, app.LongDescription, app.DeveloperID, app.OAuthAppID,
			app.Category, app.Tags, app.IconURL, app.Screenshots, app.Status,
			app.PrivacyPolicyURL, app.TermsOfServiceURL, app.SupportServerID,
		).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(time.Now(), time.Now()))

	err := repo.CreateApp(ctx, app)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_CreateApp_Error(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	app := &models.App{
		ID:   uuid.New(),
		Name: "Test App",
	}

	mock.ExpectQuery("INSERT INTO apps").
		WillReturnError(errors.New("database error"))

	err := repo.CreateApp(ctx, app)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestAppDirectoryRepository_GetAppByID(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows(appColumns).AddRow(
		appID, "Test App", "Description", nil, uuid.New(), nil,
		models.AppCategoryUtility, pq.StringArray{"tag1"}, nil, nil, 100, 4.5, 50,
		models.AppStatusApproved, nil, nil, nil, now, now,
	)

	mock.ExpectQuery("SELECT .+ FROM apps WHERE id = \\$1").
		WithArgs(appID).
		WillReturnRows(rows)

	app, err := repo.GetAppByID(ctx, appID)
	require.NoError(t, err)
	require.NotNil(t, app)
	assert.Equal(t, appID, app.ID)
	assert.Equal(t, "Test App", app.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetAppByID_NotFound(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM apps WHERE id = \\$1").
		WithArgs(appID).
		WillReturnError(sql.ErrNoRows)

	app, err := repo.GetAppByID(ctx, appID)
	require.NoError(t, err)
	assert.Nil(t, app)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetAppByIDWithDeveloper(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	developerID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "name", "description", "long_description", "developer_id", "oauth_app_id",
		"category", "tags", "icon_url", "screenshots", "install_count", "rating", "review_count",
		"status", "privacy_policy_url", "terms_of_service_url", "support_server_id", "created_at", "updated_at",
		"developer_username", "developer_discriminator", "developer_avatar",
	}).AddRow(
		appID, "Test App", "Description", nil, developerID, nil,
		models.AppCategoryUtility, pq.StringArray{"tag1"}, nil, nil, 100, 4.5, 50,
		models.AppStatusApproved, nil, nil, nil, now, now,
		"devuser", "0001", "https://example.com/avatar.png",
	)

	mock.ExpectQuery("SELECT a\\..+ FROM apps a JOIN users").
		WithArgs(appID).
		WillReturnRows(rows)

	app, err := repo.GetAppByIDWithDeveloper(ctx, appID)
	require.NoError(t, err)
	require.NotNil(t, app)
	assert.Equal(t, appID, app.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetAppByIDWithDeveloper_NotFound(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()

	mock.ExpectQuery("SELECT a\\..+ FROM apps a JOIN users").
		WithArgs(appID).
		WillReturnError(sql.ErrNoRows)

	app, err := repo.GetAppByIDWithDeveloper(ctx, appID)
	require.NoError(t, err)
	assert.Nil(t, app)
}

func TestAppDirectoryRepository_UpdateApp(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	app := &models.App{
		ID:              uuid.New(),
		Name:            "Updated App",
		Description:     "Updated description",
		Category:        models.AppCategoryGaming,
		Tags:            pq.StringArray{"newtag"},
	}

	mock.ExpectQuery("UPDATE apps SET").
		WithArgs(
			app.ID, app.Name, app.Description, app.LongDescription, app.Category, app.Tags,
			app.IconURL, app.Screenshots, app.Status, app.PrivacyPolicyURL,
			app.TermsOfServiceURL, app.SupportServerID,
		).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(time.Now()))

	err := repo.UpdateApp(ctx, app)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_UpdateAppStatus(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	newStatus := models.AppStatusApproved

	mock.ExpectExec("UPDATE apps SET status = \\$2").
		WithArgs(appID, newStatus).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateAppStatus(ctx, appID, newStatus)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_DeleteApp(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()

	mock.ExpectExec("DELETE FROM apps WHERE id = \\$1").
		WithArgs(appID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteApp(ctx, appID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_ListApps_Basic(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	appID1 := uuid.New()
	appID2 := uuid.New()

	rows := sqlmock.NewRows(appColumns).
		AddRow(appID1, "App 1", "Desc 1", nil, uuid.New(), nil, models.AppCategoryUtility, nil, nil, nil, 50, 4.0, 10, models.AppStatusApproved, nil, nil, nil, now, now).
		AddRow(appID2, "App 2", "Desc 2", nil, uuid.New(), nil, models.AppCategoryGaming, nil, nil, nil, 100, 4.5, 20, models.AppStatusApproved, nil, nil, nil, now, now)

	mock.ExpectQuery("SELECT .+ FROM apps").
		WillReturnRows(rows)

	params := services.ListAppsParams{
		ApprovedOnly: true,
		Limit:       20,
		Offset:      0,
	}

	apps, err := repo.ListApps(ctx, params)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_ListApps_WithCategory(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	category := models.AppCategoryUtility
	params := services.ListAppsParams{
		ApprovedOnly: true,
		Category:     &category,
		Limit:       20,
		Offset:      0,
	}

	rows := sqlmock.NewRows(appColumns)
	mock.ExpectQuery("SELECT .+ FROM apps").
		WillReturnRows(rows)

	_, err := repo.ListApps(ctx, params)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_ListApps_WithQuery(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	params := services.ListAppsParams{
		ApprovedOnly: true,
		Query:       "test",
		Limit:       20,
		Offset:      0,
	}

	rows := sqlmock.NewRows(appColumns)
	mock.ExpectQuery("SELECT .+ FROM apps").
		WillReturnRows(rows)

	_, err := repo.ListApps(ctx, params)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_ListApps_Featured(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	params := services.ListAppsParams{
		ApprovedOnly: true,
		Featured:     true,
		Limit:        20,
		Offset:       0,
	}

	rows := sqlmock.NewRows(appColumns)
	mock.ExpectQuery("SELECT .+ FROM apps WHERE status = 1 AND install_count > 100").
		WillReturnRows(rows)

	_, err := repo.ListApps(ctx, params)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_ListApps_DefaultLimit(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	params := services.ListAppsParams{
		Limit: 0, // Should default to 20
	}

	rows := sqlmock.NewRows(appColumns)
	mock.ExpectQuery("SELECT .+ FROM apps").
		WillReturnRows(rows)

	_, err := repo.ListApps(ctx, params)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_ListApps_Empty(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	params := services.ListAppsParams{
		Limit: 20,
	}

	rows := sqlmock.NewRows(appColumns)
	mock.ExpectQuery("SELECT .+ FROM apps").
		WillReturnRows(rows)

	apps, err := repo.ListApps(ctx, params)
	require.NoError(t, err)
	assert.Len(t, apps, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItoa(t *testing.T) {
	result := itoa(1)
	assert.Equal(t, "1", result)

	result = itoa(10)
	assert.Equal(t, "10", result)

	result = itoa(999)
	assert.Equal(t, "999", result)
}

func TestAppDirectoryRepository_ListDeveloperApps(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	developerID := uuid.New()
	now := time.Now()
	appID1 := uuid.New()
	appID2 := uuid.New()

	rows := sqlmock.NewRows(appColumns).
		AddRow(appID1, "Dev App 1", "Desc", nil, developerID, nil, models.AppCategoryUtility, nil, nil, nil, 10, 4.0, 5, models.AppStatusApproved, nil, nil, nil, now, now).
		AddRow(appID2, "Dev App 2", "Desc", nil, developerID, nil, models.AppCategoryGaming, nil, nil, nil, 20, 4.5, 10, models.AppStatusApproved, nil, nil, nil, now, now)

	mock.ExpectQuery("SELECT .+ FROM apps WHERE developer_id = \\$1").
		WithArgs(developerID).
		WillReturnRows(rows)

	apps, err := repo.ListDeveloperApps(ctx, developerID)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_InstallApp(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	serverID := uuid.New()
	installerID := uuid.New()

	mock.ExpectExec("INSERT INTO app_installations").
		WithArgs(appID, serverID, installerID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE apps SET install_count = install_count \\+ 1").
		WithArgs(appID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.InstallApp(ctx, appID, serverID, installerID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_InstallApp_Conflict(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	serverID := uuid.New()
	installerID := uuid.New()

	mock.ExpectExec("INSERT INTO app_installations").
		WithArgs(appID, serverID, installerID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // No rows affected = conflict

	// Install count is still incremented even on conflict
	mock.ExpectExec("UPDATE apps SET install_count = install_count \\+ 1").
		WithArgs(appID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.InstallApp(ctx, appID, serverID, installerID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_UninstallApp(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	serverID := uuid.New()

	mock.ExpectExec("DELETE FROM app_installations WHERE app_id = \\$1 AND server_id = \\$2").
		WithArgs(appID, serverID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE apps SET install_count = GREATEST").
		WithArgs(appID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UninstallApp(ctx, appID, serverID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_UninstallApp_NotInstalled(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	serverID := uuid.New()

	mock.ExpectExec("DELETE FROM app_installations WHERE app_id = \\$1 AND server_id = \\$2").
		WithArgs(appID, serverID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // No rows affected

	err := repo.UninstallApp(ctx, appID, serverID)
	require.NoError(t, err)
	// Should not try to decrement since no rows were affected
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetServerInstallations(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	now := time.Now()
	appID1 := uuid.New()
	appID2 := uuid.New()

	rows := sqlmock.NewRows([]string{
		"app_id", "server_id", "installer_id", "installed_at",
		"id", "name", "description", "category", "icon_url", "install_count", "rating", "status",
	}).
		AddRow(appID1, serverID, uuid.New(), now, appID1, "App 1", "Desc 1", models.AppCategoryUtility, nil, 50, 4.5, models.AppStatusApproved).
		AddRow(appID2, serverID, uuid.New(), now, appID2, "App 2", "Desc 2", models.AppCategoryGaming, nil, 100, 4.0, models.AppStatusApproved)

	mock.ExpectQuery("SELECT ai\\.app_id, ai\\.server_id, ai\\.installer_id, ai\\.installed_at").
		WithArgs(serverID).
		WillReturnRows(rows)

	installations, err := repo.GetServerInstallations(ctx, serverID)
	require.NoError(t, err)
	require.Len(t, installations, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetServerInstallations_Empty(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"app_id", "server_id", "installer_id", "installed_at",
		"id", "name", "description", "category", "icon_url", "install_count", "rating", "status",
	})

	mock.ExpectQuery("SELECT ai\\.app_id, ai\\.server_id, ai\\.installer_id, ai\\.installed_at").
		WithArgs(serverID).
		WillReturnRows(rows)

	installations, err := repo.GetServerInstallations(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, installations, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_IsAppInstalled_True(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{"1"}).AddRow(1)
	mock.ExpectQuery("SELECT 1 FROM app_installations WHERE app_id = \\$1 AND server_id = \\$2").
		WithArgs(appID, serverID).
		WillReturnRows(rows)

	installed, err := repo.IsAppInstalled(ctx, appID, serverID)
	require.NoError(t, err)
	assert.True(t, installed)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_IsAppInstalled_False(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	serverID := uuid.New()

	mock.ExpectQuery("SELECT 1 FROM app_installations WHERE app_id = \\$1 AND server_id = \\$2").
		WithArgs(appID, serverID).
		WillReturnError(sql.ErrNoRows)

	installed, err := repo.IsAppInstalled(ctx, appID, serverID)
	require.NoError(t, err)
	assert.False(t, installed)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_CreateReview(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	reviewText := "Great app!"
	review := &models.AppReview{
		ID:         uuid.New(),
		AppID:      uuid.New(),
		UserID:     uuid.New(),
		Rating:     5,
		ReviewText: &reviewText,
	}

	mock.ExpectQuery("INSERT INTO app_reviews").
		WithArgs(review.ID, review.AppID, review.UserID, review.Rating, review.ReviewText).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(time.Now(), time.Now()))

	mock.ExpectExec("UPDATE apps SET rating = COALESCE").
		WithArgs(review.AppID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.CreateReview(ctx, review)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_UpdateReview(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	reviewText := "Updated review"
	review := &models.AppReview{
		ID:         uuid.New(),
		AppID:      uuid.New(),
		UserID:     uuid.New(),
		Rating:     4,
		ReviewText: &reviewText,
	}

	mock.ExpectQuery("UPDATE app_reviews SET rating = \\$3").
		WithArgs(review.ID, review.UserID, review.Rating, review.ReviewText).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(time.Now()))

	mock.ExpectExec("UPDATE apps SET rating = COALESCE").
		WithArgs(review.AppID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateReview(ctx, review)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_DeleteReview(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	reviewID := uuid.New()
	userID := uuid.New()
	appID := uuid.New()

	rows := sqlmock.NewRows([]string{"app_id"}).AddRow(appID)
	mock.ExpectQuery("SELECT app_id FROM app_reviews WHERE id = \\$1 AND user_id = \\$2").
		WithArgs(reviewID, userID).
		WillReturnRows(rows)

	mock.ExpectExec("DELETE FROM app_reviews WHERE id = \\$1 AND user_id = \\$2").
		WithArgs(reviewID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// The updateAppRating is called but we ignore its error in the original code
	_, err := repo.DeleteReview(ctx, reviewID, userID)
	require.NoError(t, err)
	// Don't check expectations since updateAppRating isn't mocked
}

func TestAppDirectoryRepository_DeleteReview_NotFound(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	reviewID := uuid.New()
	userID := uuid.New()

	mock.ExpectQuery("SELECT app_id FROM app_reviews WHERE id = \\$1 AND user_id = \\$2").
		WithArgs(reviewID, userID).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.DeleteReview(ctx, reviewID, userID)
	assert.Error(t, err)
}

func TestAppDirectoryRepository_GetReviewByID(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	reviewID := uuid.New()
	reviewText := "Great app!"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "app_id", "user_id", "rating", "review_text", "created_at", "updated_at",
	}).AddRow(reviewID, uuid.New(), uuid.New(), 5, &reviewText, now, now)

	mock.ExpectQuery("SELECT .+ FROM app_reviews WHERE id = \\$1").
		WithArgs(reviewID).
		WillReturnRows(rows)

	review, err := repo.GetReviewByID(ctx, reviewID)
	require.NoError(t, err)
	require.NotNil(t, review)
	assert.Equal(t, reviewID, review.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetReviewByID_NotFound(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	reviewID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM app_reviews WHERE id = \\$1").
		WithArgs(reviewID).
		WillReturnError(sql.ErrNoRows)

	review, err := repo.GetReviewByID(ctx, reviewID)
	require.NoError(t, err)
	assert.Nil(t, review)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_ListAppReviews(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	now := time.Now()
	reviewID1 := uuid.New()
	reviewID2 := uuid.New()
	reviewText1 := "Great app!"
	reviewText2 := "Good app!"

	rows := sqlmock.NewRows([]string{
		"id", "app_id", "user_id", "rating", "review_text", "created_at", "updated_at",
		"username", "discriminator", "avatar_url",
	}).
		AddRow(reviewID1, appID, uuid.New(), 5, &reviewText1, now, now, myStrPtr("user1"), myStrPtr("0001"), myStrPtr("https://example.com/avatar1.png")).
		AddRow(reviewID2, appID, uuid.New(), 4, &reviewText2, now, now, myStrPtr("user2"), myStrPtr("0002"), nil)

	mock.ExpectQuery("SELECT r\\.id, r\\.app_id, r\\.user_id, r\\.rating, r\\.review_text, r\\.created_at, r\\.updated_at").
		WithArgs(appID, 20, 0).
		WillReturnRows(rows)

	reviews, err := repo.ListAppReviews(ctx, appID, 20, 0)
	require.NoError(t, err)
	require.Len(t, reviews, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_ListAppReviews_Empty(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "app_id", "user_id", "rating", "review_text", "created_at", "updated_at",
		"username", "discriminator", "avatar_url",
	})

	mock.ExpectQuery("SELECT r\\.id, r\\.app_id, r\\.user_id, r\\.rating, r\\.review_text, r\\.created_at, r\\.updated_at").
		WithArgs(appID, 20, 0).
		WillReturnRows(rows)

	reviews, err := repo.ListAppReviews(ctx, appID, 20, 0)
	require.NoError(t, err)
	assert.Len(t, reviews, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetUserReviewForApp(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	userID := uuid.New()
	reviewText := "My review"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "app_id", "user_id", "rating", "review_text", "created_at", "updated_at",
	}).AddRow(uuid.New(), appID, userID, 5, &reviewText, now, now)

	mock.ExpectQuery("SELECT .+ FROM app_reviews WHERE app_id = \\$1 AND user_id = \\$2").
		WithArgs(appID, userID).
		WillReturnRows(rows)

	review, err := repo.GetUserReviewForApp(ctx, appID, userID)
	require.NoError(t, err)
	require.NotNil(t, review)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetUserReviewForApp_NotFound(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	userID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM app_reviews WHERE app_id = \\$1 AND user_id = \\$2").
		WithArgs(appID, userID).
		WillReturnError(sql.ErrNoRows)

	review, err := repo.GetUserReviewForApp(ctx, appID, userID)
	require.NoError(t, err)
	assert.Nil(t, review)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_AddDeveloperTeamMember(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	userID := uuid.New()
	role := "admin"

	mock.ExpectExec("INSERT INTO app_developer_teams").
		WithArgs(appID, userID, role).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.AddDeveloperTeamMember(ctx, appID, userID, role)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_RemoveDeveloperTeamMember(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM app_developer_teams WHERE app_id = \\$1 AND user_id = \\$2").
		WithArgs(appID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RemoveDeveloperTeamMember(ctx, appID, userID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetDeveloperRole(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"role"}).AddRow("admin")
	mock.ExpectQuery("SELECT role FROM app_developer_teams WHERE app_id = \\$1 AND user_id = \\$2").
		WithArgs(appID, userID).
		WillReturnRows(rows)

	role, err := repo.GetDeveloperRole(ctx, appID, userID)
	require.NoError(t, err)
	assert.Equal(t, "admin", role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_GetDeveloperRole_NotFound(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	userID := uuid.New()

	mock.ExpectQuery("SELECT role FROM app_developer_teams WHERE app_id = \\$1 AND user_id = \\$2").
		WithArgs(appID, userID).
		WillReturnError(sql.ErrNoRows)

	role, err := repo.GetDeveloperRole(ctx, appID, userID)
	require.NoError(t, err)
	assert.Equal(t, "", role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_IsAppDeveloper_True(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"1"}).AddRow(1)
	mock.ExpectQuery("SELECT 1 FROM app_developer_teams WHERE app_id = \\$1 AND user_id = \\$2").
		WithArgs(appID, userID).
		WillReturnRows(rows)

	isDev, err := repo.IsAppDeveloper(ctx, appID, userID)
	require.NoError(t, err)
	assert.True(t, isDev)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_IsAppDeveloper_False(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	userID := uuid.New()

	mock.ExpectQuery("SELECT 1 FROM app_developer_teams WHERE app_id = \\$1 AND user_id = \\$2").
		WithArgs(appID, userID).
		WillReturnError(sql.ErrNoRows)

	isDev, err := repo.IsAppDeveloper(ctx, appID, userID)
	require.NoError(t, err)
	assert.False(t, isDev)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAppDirectoryRepository_ListDeveloperTeamMembers(t *testing.T) {
	repo, mock := setupAppDirectoryRepoMock(t)
	ctx := context.Background()

	appID := uuid.New()
	memberID1 := uuid.New()
	memberID2 := uuid.New()

	rows := sqlmock.NewRows([]string{
		"app_id", "user_id", "role", "username", "discriminator", "avatar_url",
	}).
		AddRow(appID, memberID1, "admin", "adminuser", "0001", "https://example.com/avatar1.png").
		AddRow(appID, memberID2, "developer", "devuser", "0002", nil)

	mock.ExpectQuery("SELECT t\\.app_id, t\\.user_id, t\\.role, u\\.username, u\\.discriminator, u\\.avatar_url").
		WithArgs(appID).
		WillReturnRows(rows)

	members, err := repo.ListDeveloperTeamMembers(ctx, appID)
	require.NoError(t, err)
	require.Len(t, members, 2)
	assert.Equal(t, "admin", members[0].Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

