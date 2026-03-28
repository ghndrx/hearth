// File: postgres/app_directory_repo.go
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"hearth/internal/models"
	"hearth/internal/services"
)

// AppDirectoryRepository handles app directory data operations
type AppDirectoryRepository struct {
	db *sqlx.DB
}

// NewAppDirectoryRepository creates a new app directory repository
func NewAppDirectoryRepository(db *sqlx.DB) *AppDirectoryRepository {
	return &AppDirectoryRepository{db: db}
}

// --- App CRUD ---

// CreateApp creates a new app in the directory
func (r *AppDirectoryRepository) CreateApp(ctx context.Context, app *models.App) error {
	query := `
		INSERT INTO apps (id, name, description, long_description, developer_id, oauth_app_id, 
			category, tags, icon_url, screenshots, status, privacy_policy_url, terms_of_service_url, support_server_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowxContext(ctx, query,
		app.ID, app.Name, app.Description, app.LongDescription, app.DeveloperID, app.OAuthAppID,
		app.Category, app.Tags, app.IconURL, app.Screenshots, app.Status,
		app.PrivacyPolicyURL, app.TermsOfServiceURL, app.SupportServerID,
	).Scan(&app.CreatedAt, &app.UpdatedAt)
}

// GetAppByID retrieves an app by ID
func (r *AppDirectoryRepository) GetAppByID(ctx context.Context, id uuid.UUID) (*models.App, error) {
	var app models.App
	query := `
		SELECT id, name, description, long_description, developer_id, oauth_app_id, 
			category, tags, icon_url, screenshots, install_count, rating, review_count, 
			status, privacy_policy_url, terms_of_service_url, support_server_id, created_at, updated_at
		FROM apps WHERE id = $1
	`
	err := r.db.GetContext(ctx, &app, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &app, err
}

// GetAppByIDWithDeveloper retrieves an app by ID with developer info populated
func (r *AppDirectoryRepository) GetAppByIDWithDeveloper(ctx context.Context, id uuid.UUID) (*models.App, error) {
	var app models.App
	query := `
		SELECT a.id, a.name, a.description, a.long_description, a.developer_id, a.oauth_app_id, 
			a.category, a.tags, a.icon_url, a.screenshots, a.install_count, a.rating, a.review_count, 
			a.status, a.privacy_policy_url, a.terms_of_service_url, a.support_server_id, a.created_at, a.updated_at,
			u.username as developer_username, u.discriminator as developer_discriminator, u.avatar_url as developer_avatar
		FROM apps a
		JOIN users u ON u.id = a.developer_id
		WHERE a.id = $1
	`
	var devUsername, devDiscriminator, devAvatar sql.NullString
	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&app.ID, &app.Name, &app.Description, &app.LongDescription, &app.DeveloperID, &app.OAuthAppID,
		&app.Category, &app.Tags, &app.IconURL, &app.Screenshots, &app.InstallCount, &app.Rating, &app.ReviewCount,
		&app.Status, &app.PrivacyPolicyURL, &app.TermsOfServiceURL, &app.SupportServerID, &app.CreatedAt, &app.UpdatedAt,
		&devUsername, &devDiscriminator, &devAvatar,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Developer info would need to be populated via a join - for now return the app
	_ = devUsername
	_ = devDiscriminator
	_ = devAvatar
	return &app, nil
}

// UpdateApp updates an existing app
func (r *AppDirectoryRepository) UpdateApp(ctx context.Context, app *models.App) error {
	query := `
		UPDATE apps SET 
			name = $2, description = $3, long_description = $4, category = $5, tags = $6,
			icon_url = $7, screenshots = $8, status = $9, privacy_policy_url = $10, 
			terms_of_service_url = $11, support_server_id = $12, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRowxContext(ctx, query,
		app.ID, app.Name, app.Description, app.LongDescription, app.Category, app.Tags,
		app.IconURL, app.Screenshots, app.Status, app.PrivacyPolicyURL,
		app.TermsOfServiceURL, app.SupportServerID,
	).Scan(&app.UpdatedAt)
}

// UpdateAppStatus updates the status of an app (for approval/rejection)
func (r *AppDirectoryRepository) UpdateAppStatus(ctx context.Context, appID uuid.UUID, status models.AppStatus) error {
	query := `UPDATE apps SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, appID, status)
	return err
}

// DeleteApp deletes an app
func (r *AppDirectoryRepository) DeleteApp(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM apps WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListApps returns a list of apps with optional filtering
func (r *AppDirectoryRepository) ListApps(ctx context.Context, params services.ListAppsParams) ([]*models.App, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	// Only show approved apps in public listings
	if params.ApprovedOnly {
		conditions = append(conditions, "status = $1")
		args = append(args, models.AppStatusApproved)
		argIdx++
	}

	if params.Category != nil {
		conditions = append(conditions, "category = $"+itoa(argIdx))
		args = append(args, *params.Category)
		argIdx++
	}

	if params.Query != "" {
		conditions = append(conditions, "(name ILIKE $"+itoa(argIdx)+" OR description ILIKE $"+itoa(argIdx)+")")
		args = append(args, "%"+params.Query+"%")
		argIdx++
	}

	if params.Featured {
		conditions = append(conditions, "install_count > 100")
	}

	// Override to only approved for featured
	if params.Featured && params.ApprovedOnly {
		conditions = []string{"status = 1", "install_count > 100"}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := `
		SELECT id, name, description, long_description, developer_id, oauth_app_id, 
			category, tags, icon_url, screenshots, install_count, rating, review_count, 
			status, privacy_policy_url, terms_of_service_url, support_server_id, created_at, updated_at
		FROM apps
		` + whereClause + `
		ORDER BY rating DESC, install_count DESC
		LIMIT $` + itoa(argIdx) + ` OFFSET $` + itoa(argIdx+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*models.App
	for rows.Next() {
		var app models.App
		err := rows.Scan(
			&app.ID, &app.Name, &app.Description, &app.LongDescription, &app.DeveloperID, &app.OAuthAppID,
			&app.Category, &app.Tags, &app.IconURL, &app.Screenshots, &app.InstallCount, &app.Rating, &app.ReviewCount,
			&app.Status, &app.PrivacyPolicyURL, &app.TermsOfServiceURL, &app.SupportServerID, &app.CreatedAt, &app.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		apps = append(apps, &app)
	}
	return apps, nil
}

// itoa converts an integer to a string for use in query parameter placeholders
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

// ListDeveloperApps returns all apps owned by a developer
func (r *AppDirectoryRepository) ListDeveloperApps(ctx context.Context, developerID uuid.UUID) ([]*models.App, error) {
	query := `
		SELECT id, name, description, long_description, developer_id, oauth_app_id, 
			category, tags, icon_url, screenshots, install_count, rating, review_count, 
			status, privacy_policy_url, terms_of_service_url, support_server_id, created_at, updated_at
		FROM apps
		WHERE developer_id = $1
		ORDER BY created_at DESC
	`
	var apps []*models.App
	err := r.db.SelectContext(ctx, &apps, query, developerID)
	return apps, err
}

// --- App Installations ---

// InstallApp creates an app installation for a server
func (r *AppDirectoryRepository) InstallApp(ctx context.Context, appID, serverID, installerID uuid.UUID) error {
	query := `
		INSERT INTO app_installations (app_id, server_id, installer_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (app_id, server_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, appID, serverID, installerID)
	if err != nil {
		return err
	}

	// Increment install count
	incQuery := `UPDATE apps SET install_count = install_count + 1 WHERE id = $1`
	_, err = r.db.ExecContext(ctx, incQuery, appID)
	return err
}

// UninstallApp removes an app installation from a server
func (r *AppDirectoryRepository) UninstallApp(ctx context.Context, appID, serverID uuid.UUID) error {
	query := `DELETE FROM app_installations WHERE app_id = $1 AND server_id = $2`
	result, err := r.db.ExecContext(ctx, query, appID, serverID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		// Decrement install count
		decQuery := `UPDATE apps SET install_count = GREATEST(0, install_count - 1) WHERE id = $1`
		_, err = r.db.ExecContext(ctx, decQuery, appID)
	}
	return err
}

// GetServerInstallations returns all apps installed on a server
func (r *AppDirectoryRepository) GetServerInstallations(ctx context.Context, serverID uuid.UUID) ([]*models.AppInstallation, error) {
	query := `
		SELECT ai.app_id, ai.server_id, ai.installer_id, ai.installed_at,
			a.id, a.name, a.description, a.category, a.icon_url, a.install_count, a.rating, a.status
		FROM app_installations ai
		JOIN apps a ON a.id = ai.app_id
		WHERE ai.server_id = $1
		ORDER BY ai.installed_at DESC
	`
	rows, err := r.db.QueryxContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []*models.AppInstallation
	for rows.Next() {
		var inst models.AppInstallation
		var app models.App
		err := rows.Scan(
			&inst.AppID, &inst.ServerID, &inst.InstallerID, &inst.InstalledAt,
			&app.ID, &app.Name, &app.Description, &app.Category, &app.IconURL, &app.InstallCount, &app.Rating, &app.Status,
		)
		if err != nil {
			return nil, err
		}
		inst.App = &app
		installations = append(installations, &inst)
	}
	return installations, nil
}

// IsAppInstalled checks if an app is installed on a server
func (r *AppDirectoryRepository) IsAppInstalled(ctx context.Context, appID, serverID uuid.UUID) (bool, error) {
	query := `SELECT 1 FROM app_installations WHERE app_id = $1 AND server_id = $2`
	var exists int
	err := r.db.GetContext(ctx, &exists, query, appID, serverID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// --- App Reviews ---

// CreateReview creates a new review for an app
func (r *AppDirectoryRepository) CreateReview(ctx context.Context, review *models.AppReview) error {
	query := `
		INSERT INTO app_reviews (id, app_id, user_id, rating, review_text)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRowxContext(ctx, query,
		review.ID, review.AppID, review.UserID, review.Rating, review.ReviewText,
	).Scan(&review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		return err
	}

	// Update app rating
	return r.updateAppRating(ctx, review.AppID)
}

// UpdateReview updates an existing review
func (r *AppDirectoryRepository) UpdateReview(ctx context.Context, review *models.AppReview) error {
	query := `
		UPDATE app_reviews SET rating = $3, review_text = $4, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_at
	`
	err := r.db.QueryRowxContext(ctx, query,
		review.ID, review.UserID, review.Rating, review.ReviewText,
	).Scan(&review.UpdatedAt)
	if err != nil {
		return err
	}
	return r.updateAppRating(ctx, review.AppID)
}

// DeleteReview deletes a review
func (r *AppDirectoryRepository) DeleteReview(ctx context.Context, reviewID, userID uuid.UUID) (uuid.UUID, error) {
	// Get app ID first
	var appID uuid.UUID
	getQuery := `SELECT app_id FROM app_reviews WHERE id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &appID, getQuery, reviewID, userID)
	if err != nil {
		return uuid.Nil, err
	}

	query := `DELETE FROM app_reviews WHERE id = $1 AND user_id = $2`
	_, err = r.db.ExecContext(ctx, query, reviewID, userID)
	if err != nil {
		return uuid.Nil, err
	}

	// Update app rating
	_ = r.updateAppRating(ctx, appID)
	return appID, nil
}

// GetReviewByID retrieves a review by ID
func (r *AppDirectoryRepository) GetReviewByID(ctx context.Context, id uuid.UUID) (*models.AppReview, error) {
	var review models.AppReview
	query := `
		SELECT id, app_id, user_id, rating, review_text, created_at, updated_at
		FROM app_reviews WHERE id = $1
	`
	err := r.db.GetContext(ctx, &review, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &review, err
}

// ListAppReviews returns all reviews for an app
func (r *AppDirectoryRepository) ListAppReviews(ctx context.Context, appID uuid.UUID, limit, offset int) ([]*models.AppReview, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT r.id, r.app_id, r.user_id, r.rating, r.review_text, r.created_at, r.updated_at,
			u.username, u.discriminator, u.avatar_url
		FROM app_reviews r
		JOIN users u ON u.id = r.user_id
		WHERE r.app_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryxContext(ctx, query, appID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*models.AppReview
	for rows.Next() {
		var review models.AppReview
		var username, discriminator, avatarURL sql.NullString
		err := rows.Scan(
			&review.ID, &review.AppID, &review.UserID, &review.Rating, &review.ReviewText, &review.CreatedAt, &review.UpdatedAt,
			&username, &discriminator, &avatarURL,
		)
		if err != nil {
			return nil, err
		}
		if username.Valid {
			avatar := avatarURL.String
			review.User = &models.PublicUser{
				ID:            review.UserID,
				Username:      username.String,
				Discriminator: discriminator.String,
				AvatarURL:     &avatar,
			}
		}
		reviews = append(reviews, &review)
	}
	return reviews, nil
}

// GetUserReviewForApp returns a user's review for a specific app
func (r *AppDirectoryRepository) GetUserReviewForApp(ctx context.Context, appID, userID uuid.UUID) (*models.AppReview, error) {
	var review models.AppReview
	query := `SELECT id, app_id, user_id, rating, review_text, created_at, updated_at FROM app_reviews WHERE app_id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &review, query, appID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &review, err
}

// updateAppRating recalculates and updates the average rating for an app
func (r *AppDirectoryRepository) updateAppRating(ctx context.Context, appID uuid.UUID) error {
	query := `
		UPDATE apps SET 
			rating = COALESCE((SELECT AVG(rating) FROM app_reviews WHERE app_id = $1), 0),
			review_count = (SELECT COUNT(*) FROM app_reviews WHERE app_id = $1)
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, appID)
	return err
}

// --- Developer Team ---

// AddDeveloperTeamMember adds a user to the app's developer team
func (r *AppDirectoryRepository) AddDeveloperTeamMember(ctx context.Context, appID, userID uuid.UUID, role string) error {
	query := `
		INSERT INTO app_developer_teams (app_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (app_id, user_id) DO UPDATE SET role = $3
	`
	_, err := r.db.ExecContext(ctx, query, appID, userID, role)
	return err
}

// RemoveDeveloperTeamMember removes a user from the app's developer team
func (r *AppDirectoryRepository) RemoveDeveloperTeamMember(ctx context.Context, appID, userID uuid.UUID) error {
	query := `DELETE FROM app_developer_teams WHERE app_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, appID, userID)
	return err
}

// GetDeveloperRole returns the role of a user in an app's developer team
func (r *AppDirectoryRepository) GetDeveloperRole(ctx context.Context, appID, userID uuid.UUID) (string, error) {
	var role string
	query := `SELECT role FROM app_developer_teams WHERE app_id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &role, query, appID, userID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return role, err
}

// IsAppDeveloper checks if a user is a developer of an app
func (r *AppDirectoryRepository) IsAppDeveloper(ctx context.Context, appID, userID uuid.UUID) (bool, error) {
	var exists int
	query := `SELECT 1 FROM app_developer_teams WHERE app_id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &exists, query, appID, userID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// ListDeveloperTeamMembers returns all members of an app's developer team
func (r *AppDirectoryRepository) ListDeveloperTeamMembers(ctx context.Context, appID uuid.UUID) ([]*models.AppDeveloperTeamMember, error) {
	query := `
		SELECT t.app_id, t.user_id, t.role, u.username, u.discriminator, u.avatar_url
		FROM app_developer_teams t
		JOIN users u ON u.id = t.user_id
		WHERE t.app_id = $1
		ORDER BY t.role, t.user_id
	`
	rows, err := r.db.QueryxContext(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.AppDeveloperTeamMember
	for rows.Next() {
		var m models.AppDeveloperTeamMember
		var username, discriminator, avatarURL sql.NullString
		err := rows.Scan(&m.AppID, &m.UserID, &m.Role, &username, &discriminator, &avatarURL)
		if err != nil {
			return nil, err
		}
		_ = username
		_ = discriminator
		_ = avatarURL
		members = append(members, &m)
	}
	return members, nil
}

// --- Analytics ---

// GetDeveloperAnalytics returns analytics for a developer's apps
func (r *AppDirectoryRepository) GetDeveloperAnalytics(ctx context.Context, developerID uuid.UUID) (*models.AppDeveloperAnalytics, error) {
	analytics := &models.AppDeveloperAnalytics{
		AppsByStatus: make(map[string]int),
	}

	// Get total apps
	countQuery := `SELECT COUNT(*) FROM apps WHERE developer_id = $1`
	err := r.db.GetContext(ctx, &analytics.TotalApps, countQuery, developerID)
	if err != nil {
		return nil, err
	}

	// Get apps by status
	statusQuery := `SELECT status, COUNT(*) FROM apps WHERE developer_id = $1 GROUP BY status`
	var statusRows []struct {
		Status int `db:"status"`
		Count  int `db:"count"`
	}
	err = r.db.SelectContext(ctx, &statusRows, statusQuery, developerID)
	if err != nil {
		return nil, err
	}
	for _, row := range statusRows {
		statusName := models.AppStatus(row.Status).String()
		analytics.AppsByStatus[statusName] = row.Count
	}

	// Get total installs
	installQuery := `
		SELECT COALESCE(SUM(a.install_count), 0) 
		FROM apps a 
		WHERE a.developer_id = $1 AND a.status = $2
	`
	err = r.db.GetContext(ctx, &analytics.TotalInstalls, installQuery, developerID, models.AppStatusApproved)
	if err != nil {
		return nil, err
	}

	// Get total reviews and average rating
	reviewQuery := `
		SELECT COUNT(*), COALESCE(AVG(r.rating), 0)
		FROM app_reviews r
		JOIN apps a ON a.id = r.app_id
		WHERE a.developer_id = $1 AND a.status = $2
	`
	var avgRating sql.NullFloat64
	err = r.db.GetContext(ctx, &analytics.TotalReviews, reviewQuery, developerID, models.AppStatusApproved)
	if err != nil {
		return nil, err
	}
	if avgRating.Valid {
		analytics.AverageRating = avgRating.Float64
	}

	// Get install trend (last 7 days)
	analytics.InstallTrend = make([]int, 7)
	trendQuery := `
		SELECT DATE(i.installed_at) as date, COUNT(*)
		FROM app_installations i
		JOIN apps a ON a.id = i.app_id
		WHERE a.developer_id = $1 AND i.installed_at > NOW() - INTERVAL '7 days'
		GROUP BY DATE(i.installed_at)
		ORDER BY date
	`
	var trendRows []struct {
		Date  time.Time `db:"date"`
		Count int       `db:"count"`
	}
	err = r.db.SelectContext(ctx, &trendRows, trendQuery, developerID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	for _, row := range trendRows {
		dayIdx := int(time.Since(row.Date).Hours() / 24)
		if dayIdx >= 0 && dayIdx < 7 {
			analytics.InstallTrend[6-dayIdx] = row.Count
		}
	}

	// Get review trend (last 7 days)
	analytics.ReviewTrend = make([]int, 7)
	reviewTrendQuery := `
		SELECT DATE(r.created_at) as date, COUNT(*)
		FROM app_reviews r
		JOIN apps a ON a.id = r.app_id
		WHERE a.developer_id = $1 AND r.created_at > NOW() - INTERVAL '7 days'
		GROUP BY DATE(r.created_at)
		ORDER BY date
	`
	var reviewTrendRows []struct {
		Date  time.Time `db:"date"`
		Count int       `db:"count"`
	}
	err = r.db.SelectContext(ctx, &reviewTrendRows, reviewTrendQuery, developerID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	for _, row := range reviewTrendRows {
		dayIdx := int(time.Since(row.Date).Hours() / 24)
		if dayIdx >= 0 && dayIdx < 7 {
			analytics.ReviewTrend[6-dayIdx] = row.Count
		}
	}

	return analytics, nil
}

// Ensure pq types are used to avoid import issues
var _ = pq.StringArray{}
