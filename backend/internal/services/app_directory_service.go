// File: services/app_directory_service.go
package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrAppNotFound        = errors.New("app not found")
	ErrAlreadyInstalled   = errors.New("app already installed")
	ErrNotInstalled       = errors.New("app not installed")
	ErrNotAppDeveloper    = errors.New("user is not a developer of this app")
	ErrNotAppOwner        = errors.New("user is not the owner of this app")
	ErrCannotDeleteApp    = errors.New("cannot delete app with installations")
	ErrAlreadyReviewed    = errors.New("user has already reviewed this app")
	ErrReviewNotFound     = errors.New("review not found")
	ErrInvalidAppCategory = errors.New("invalid app category")
	ErrInvalidStatus      = errors.New("invalid app status")
	ErrAppNotApproved     = errors.New("app is not approved")
)

// AppDirectoryRepository defines the interface for app directory data access
type AppDirectoryRepository interface {
	// App CRUD
	CreateApp(ctx context.Context, app *models.App) error
	GetAppByID(ctx context.Context, id uuid.UUID) (*models.App, error)
	GetAppByIDWithDeveloper(ctx context.Context, id uuid.UUID) (*models.App, error)
	UpdateApp(ctx context.Context, app *models.App) error
	UpdateAppStatus(ctx context.Context, appID uuid.UUID, status models.AppStatus) error
	DeleteApp(ctx context.Context, id uuid.UUID) error

	// Listing
	ListApps(ctx context.Context, params ListAppsParams) ([]*models.App, error)
	ListDeveloperApps(ctx context.Context, developerID uuid.UUID) ([]*models.App, error)

	// Installations
	InstallApp(ctx context.Context, appID, serverID, installerID uuid.UUID) error
	UninstallApp(ctx context.Context, appID, serverID uuid.UUID) error
	GetServerInstallations(ctx context.Context, serverID uuid.UUID) ([]*models.AppInstallation, error)
	IsAppInstalled(ctx context.Context, appID, serverID uuid.UUID) (bool, error)

	// Reviews
	CreateReview(ctx context.Context, review *models.AppReview) error
	UpdateReview(ctx context.Context, review *models.AppReview) error
	DeleteReview(ctx context.Context, reviewID, userID uuid.UUID) (uuid.UUID, error)
	GetReviewByID(ctx context.Context, id uuid.UUID) (*models.AppReview, error)
	ListAppReviews(ctx context.Context, appID uuid.UUID, limit, offset int) ([]*models.AppReview, error)
	GetUserReviewForApp(ctx context.Context, appID, userID uuid.UUID) (*models.AppReview, error)

	// Developer team
	AddDeveloperTeamMember(ctx context.Context, appID, userID uuid.UUID, role string) error
	RemoveDeveloperTeamMember(ctx context.Context, appID, userID uuid.UUID) error
	GetDeveloperRole(ctx context.Context, appID, userID uuid.UUID) (string, error)
	IsAppDeveloper(ctx context.Context, appID, userID uuid.UUID) (bool, error)
	ListDeveloperTeamMembers(ctx context.Context, appID uuid.UUID) ([]*models.AppDeveloperTeamMember, error)

	// Analytics
	GetDeveloperAnalytics(ctx context.Context, developerID uuid.UUID) (*models.AppDeveloperAnalytics, error)
}

// ListAppsParams holds parameters for listing apps
type ListAppsParams struct {
	Category     *models.AppCategory
	Query        string
	Featured     bool
	ApprovedOnly bool
	Limit        int
	Offset       int
}

// AppDirectoryService handles business logic for the App Directory
type AppDirectoryService struct {
	repo AppDirectoryRepository
}

// NewAppDirectoryService creates a new app directory service
func NewAppDirectoryService(repo AppDirectoryRepository) *AppDirectoryService {
	return &AppDirectoryService{repo: repo}
}

// --- App Management ---

// SubmitApp creates a new app submission (pending approval)
func (s *AppDirectoryService) SubmitApp(ctx context.Context, req *models.CreateAppRequest, developerID uuid.UUID) (*models.App, error) {
	category, ok := models.ParseAppCategory(req.Category)
	if !ok {
		return nil, ErrInvalidAppCategory
	}

	app := &models.App{
		ID:                uuid.New(),
		Name:              req.Name,
		Description:       req.Description,
		LongDescription:   req.LongDescription,
		DeveloperID:       developerID,
		Category:          category,
		Tags:              req.Tags,
		IconURL:           req.IconURL,
		Screenshots:       req.Screenshots,
		Status:            models.AppStatusPending,
		PrivacyPolicyURL:  req.PrivacyPolicyURL,
		TermsOfServiceURL: req.TermsOfServiceURL,
	}

	if req.SupportServerID != nil {
		supportID, err := uuid.Parse(*req.SupportServerID)
		if err == nil {
			app.SupportServerID = &supportID
		}
	}

	if err := s.repo.CreateApp(ctx, app); err != nil {
		return nil, err
	}

	// Add developer as owner in team
	if err := s.repo.AddDeveloperTeamMember(ctx, app.ID, developerID, models.AppDeveloperRoleOwner); err != nil {
		// Log but don't fail - app was created
	}

	return app, nil
}

// UpdateApp updates an existing app
func (s *AppDirectoryService) UpdateApp(ctx context.Context, appID uuid.UUID, req *models.UpdateAppRequest, userID uuid.UUID) (*models.App, error) {
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil || app == nil {
		return nil, ErrAppNotFound
	}

	// Check if user is a developer
	isDev, err := s.repo.IsAppDeveloper(ctx, appID, userID)
	if err != nil || !isDev {
		return nil, ErrNotAppDeveloper
	}

	if req.Name != nil {
		app.Name = *req.Name
	}
	if req.Description != nil {
		app.Description = *req.Description
	}
	if req.LongDescription != nil {
		app.LongDescription = req.LongDescription
	}
	if req.Category != nil {
		category, ok := models.ParseAppCategory(*req.Category)
		if !ok {
			return nil, ErrInvalidAppCategory
		}
		app.Category = category
	}
	if req.Tags != nil {
		app.Tags = req.Tags
	}
	if req.IconURL != nil {
		app.IconURL = req.IconURL
	}
	if req.Screenshots != nil {
		app.Screenshots = req.Screenshots
	}
	if req.PrivacyPolicyURL != nil {
		app.PrivacyPolicyURL = req.PrivacyPolicyURL
	}
	if req.TermsOfServiceURL != nil {
		app.TermsOfServiceURL = req.TermsOfServiceURL
	}
	if req.SupportServerID != nil {
		if *req.SupportServerID == "" {
			app.SupportServerID = nil
		} else {
			supportID, err := uuid.Parse(*req.SupportServerID)
			if err == nil {
				app.SupportServerID = &supportID
			}
		}
	}

	if err := s.repo.UpdateApp(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// DeleteApp deletes an app
func (s *AppDirectoryService) DeleteApp(ctx context.Context, appID uuid.UUID, userID uuid.UUID) error {
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil || app == nil {
		return ErrAppNotFound
	}

	// Check if user is the owner
	role, err := s.repo.GetDeveloperRole(ctx, appID, userID)
	if err != nil || role != models.AppDeveloperRoleOwner {
		return ErrNotAppOwner
	}

	// Check for installations
	if app.InstallCount > 0 {
		return ErrCannotDeleteApp
	}

	return s.repo.DeleteApp(ctx, appID)
}

// GetApp retrieves an app by ID
func (s *AppDirectoryService) GetApp(ctx context.Context, appID uuid.UUID) (*models.App, error) {
	app, err := s.repo.GetAppByIDWithDeveloper(ctx, appID)
	if err != nil || app == nil {
		return nil, ErrAppNotFound
	}
	return app, nil
}

// ListApps returns a list of apps with optional filtering
func (s *AppDirectoryService) ListApps(ctx context.Context, params *models.ListAppsRequest) ([]*models.App, error) {
	listParams := ListAppsParams{
		ApprovedOnly: true, // Only show approved apps to public
		Limit:        params.Limit,
		Offset:       params.Offset,
		Query:        params.Query,
		Featured:     params.Featured,
	}

	if params.Category != "" {
		cat, ok := models.ParseAppCategory(params.Category)
		if ok {
			listParams.Category = &cat
		}
	}

	return s.repo.ListApps(ctx, listParams)
}

// --- Approval Workflow ---

// ApproveApp approves a pending app
func (s *AppDirectoryService) ApproveApp(ctx context.Context, appID uuid.UUID) error {
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil || app == nil {
		return ErrAppNotFound
	}

	if app.Status != models.AppStatusPending {
		return ErrInvalidStatus
	}

	return s.repo.UpdateAppStatus(ctx, appID, models.AppStatusApproved)
}

// RejectApp rejects a pending app
func (s *AppDirectoryService) RejectApp(ctx context.Context, appID uuid.UUID, reason string) error {
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil || app == nil {
		return ErrAppNotFound
	}

	if app.Status != models.AppStatusPending {
		return ErrInvalidStatus
	}

	return s.repo.UpdateAppStatus(ctx, appID, models.AppStatusRejected)
}

// SuspendApp suspends an approved app
func (s *AppDirectoryService) SuspendApp(ctx context.Context, appID uuid.UUID) error {
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil || app == nil {
		return ErrAppNotFound
	}

	if app.Status != models.AppStatusApproved {
		return ErrInvalidStatus
	}

	return s.repo.UpdateAppStatus(ctx, appID, models.AppStatusSuspended)
}

// --- Installation ---

// InstallApp installs an app on a server
func (s *AppDirectoryService) InstallApp(ctx context.Context, appID, serverID, installerID uuid.UUID) error {
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil || app == nil {
		return ErrAppNotFound
	}

	if app.Status != models.AppStatusApproved {
		return ErrAppNotApproved
	}

	installed, err := s.repo.IsAppInstalled(ctx, appID, serverID)
	if err != nil {
		return err
	}
	if installed {
		return ErrAlreadyInstalled
	}

	return s.repo.InstallApp(ctx, appID, serverID, installerID)
}

// UninstallApp uninstalls an app from a server
func (s *AppDirectoryService) UninstallApp(ctx context.Context, appID, serverID uuid.UUID) error {
	installed, err := s.repo.IsAppInstalled(ctx, appID, serverID)
	if err != nil {
		return err
	}
	if !installed {
		return ErrNotInstalled
	}

	return s.repo.UninstallApp(ctx, appID, serverID)
}

// GetServerInstallations returns all apps installed on a server
func (s *AppDirectoryService) GetServerInstallations(ctx context.Context, serverID uuid.UUID) ([]*models.AppInstallation, error) {
	return s.repo.GetServerInstallations(ctx, serverID)
}

// --- Reviews ---

// CreateReview creates a review for an app
func (s *AppDirectoryService) CreateReview(ctx context.Context, appID, userID uuid.UUID, req *models.CreateReviewRequest) (*models.AppReview, error) {
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil || app == nil {
		return nil, ErrAppNotFound
	}

	if app.Status != models.AppStatusApproved {
		return nil, ErrAppNotApproved
	}

	// Check for existing review
	existing, err := s.repo.GetUserReviewForApp(ctx, appID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyReviewed
	}

	review := &models.AppReview{
		ID:         uuid.New(),
		AppID:      appID,
		UserID:     userID,
		Rating:     req.Rating,
		ReviewText: &req.ReviewText,
	}

	if review.ReviewText != nil && *review.ReviewText == "" {
		review.ReviewText = nil
	}

	if err := s.repo.CreateReview(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

// UpdateReview updates an existing review
func (s *AppDirectoryService) UpdateReview(ctx context.Context, appID, reviewID, userID uuid.UUID, req *models.UpdateReviewRequest) (*models.AppReview, error) {
	review, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil || review == nil {
		return nil, ErrReviewNotFound
	}

	if review.AppID != appID {
		return nil, ErrReviewNotFound
	}

	if review.UserID != userID {
		return nil, ErrNotAppDeveloper // Using this error as a proxy for "not the author"
	}

	if req.Rating != nil {
		review.Rating = *req.Rating
	}
	if req.ReviewText != nil {
		if *req.ReviewText == "" {
			review.ReviewText = nil
		} else {
			review.ReviewText = req.ReviewText
		}
	}

	if err := s.repo.UpdateReview(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

// DeleteReview deletes a review
func (s *AppDirectoryService) DeleteReview(ctx context.Context, appID, reviewID, userID uuid.UUID) error {
	review, err := s.repo.GetReviewByID(ctx, reviewID)
	if err != nil || review == nil {
		return ErrReviewNotFound
	}

	if review.AppID != appID {
		return ErrReviewNotFound
	}

	if review.UserID != userID {
		return ErrNotAppDeveloper
	}

	_, err = s.repo.DeleteReview(ctx, reviewID, userID)
	return err
}

// ListAppReviews returns all reviews for an app
func (s *AppDirectoryService) ListAppReviews(ctx context.Context, appID uuid.UUID, limit, offset int) ([]*models.AppReview, error) {
	return s.repo.ListAppReviews(ctx, appID, limit, offset)
}

// GetUserReviewForApp returns a user's review for a specific app
func (s *AppDirectoryService) GetUserReviewForApp(ctx context.Context, appID, userID uuid.UUID) (*models.AppReview, error) {
	return s.repo.GetUserReviewForApp(ctx, appID, userID)
}

// --- Developer Dashboard ---

// ListDeveloperApps returns all apps owned by a developer
func (s *AppDirectoryService) ListDeveloperApps(ctx context.Context, developerID uuid.UUID) ([]*models.App, error) {
	return s.repo.ListDeveloperApps(ctx, developerID)
}

// GetDeveloperAnalytics returns analytics for a developer's apps
func (s *AppDirectoryService) GetDeveloperAnalytics(ctx context.Context, developerID uuid.UUID) (*models.AppDeveloperAnalytics, error) {
	return s.repo.GetDeveloperAnalytics(ctx, developerID)
}

// --- Developer Team Management ---

// AddDeveloper adds a user to the app's developer team
func (s *AppDirectoryService) AddDeveloper(ctx context.Context, appID, targetUserID, requestingUserID uuid.UUID, role string) error {
	// Check if requester is owner or admin
	requestingRole, err := s.repo.GetDeveloperRole(ctx, appID, requestingUserID)
	if err != nil {
		return err
	}
	if requestingRole != models.AppDeveloperRoleOwner && requestingRole != models.AppDeveloperRoleAdmin {
		return ErrNotAppDeveloper
	}

	// Only owners can add other owners or admins
	if role == models.AppDeveloperRoleOwner || role == models.AppDeveloperRoleAdmin {
		if requestingRole != models.AppDeveloperRoleOwner {
			return ErrNotAppOwner
		}
	}

	return s.repo.AddDeveloperTeamMember(ctx, appID, targetUserID, role)
}

// RemoveDeveloper removes a user from the app's developer team
func (s *AppDirectoryService) RemoveDeveloper(ctx context.Context, appID, targetUserID, requestingUserID uuid.UUID) error {
	// Only owners can remove developers
	requestingRole, err := s.repo.GetDeveloperRole(ctx, appID, requestingUserID)
	if err != nil {
		return err
	}
	if requestingRole != models.AppDeveloperRoleOwner {
		return ErrNotAppOwner
	}

	// Can't remove yourself if you're the only owner
	if targetUserID == requestingUserID {
		allMembers, err := s.repo.ListDeveloperTeamMembers(ctx, appID)
		if err != nil {
			return err
		}
		ownerCount := 0
		for _, m := range allMembers {
			if m.Role == models.AppDeveloperRoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return ErrCannotDeleteApp // Reusing error - can't remove only owner
		}
	}

	return s.repo.RemoveDeveloperTeamMember(ctx, appID, targetUserID)
}

// ListDeveloperTeamMembers returns all members of an app's developer team
func (s *AppDirectoryService) ListDeveloperTeamMembers(ctx context.Context, appID uuid.UUID) ([]*models.AppDeveloperTeamMember, error) {
	return s.repo.ListDeveloperTeamMembers(ctx, appID)
}
