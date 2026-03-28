// File: models/app_directory.go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// AppCategory represents the category of an app in the directory
type AppCategory int

const (
	AppCategoryModeration AppCategory = iota
	AppCategoryMusic
	AppCategoryGaming
	AppCategoryUtility
	AppCategoryFun
	AppCategoryEducation
	AppCategoryRoleplay
	AppCategoryEconomy
)

// String returns the string representation of the category
func (c AppCategory) String() string {
	switch c {
	case AppCategoryModeration:
		return "moderation"
	case AppCategoryMusic:
		return "music"
	case AppCategoryGaming:
		return "gaming"
	case AppCategoryUtility:
		return "utility"
	case AppCategoryFun:
		return "fun"
	case AppCategoryEducation:
		return "education"
	case AppCategoryRoleplay:
		return "roleplay"
	case AppCategoryEconomy:
		return "economy"
	default:
		return "unknown"
	}
}

// ParseAppCategory parses a string into an AppCategory
func ParseAppCategory(s string) (AppCategory, bool) {
	switch s {
	case "moderation":
		return AppCategoryModeration, true
	case "music":
		return AppCategoryMusic, true
	case "gaming":
		return AppCategoryGaming, true
	case "utility":
		return AppCategoryUtility, true
	case "fun":
		return AppCategoryFun, true
	case "education":
		return AppCategoryEducation, true
	case "roleplay":
		return AppCategoryRoleplay, true
	case "economy":
		return AppCategoryEconomy, true
	default:
		return 0, false
	}
}

// AppCategoryNames returns all valid category names
var AppCategoryNames = []string{
	"moderation",
	"music",
	"gaming",
	"utility",
	"fun",
	"education",
	"roleplay",
	"economy",
}

// AppStatus represents the approval status of an app
type AppStatus int

const (
	AppStatusPending AppStatus = iota
	AppStatusApproved
	AppStatusRejected
	AppStatusSuspended
)

// String returns the string representation of the status
func (s AppStatus) String() string {
	switch s {
	case AppStatusPending:
		return "pending"
	case AppStatusApproved:
		return "approved"
	case AppStatusRejected:
		return "rejected"
	case AppStatusSuspended:
		return "suspended"
	default:
		return "unknown"
	}
}

// App represents a bot/app in the App Directory
type App struct {
	ID                uuid.UUID      `json:"id" db:"id"`
	Name              string         `json:"name" db:"name"`
	Description       string         `json:"description" db:"description"`
	LongDescription   *string        `json:"long_description,omitempty" db:"long_description"`
	DeveloperID       uuid.UUID      `json:"developer_id" db:"developer_id"`
	OAuthAppID        *uuid.UUID     `json:"oauth_app_id,omitempty" db:"oauth_app_id"`
	Category          AppCategory    `json:"category" db:"category"`
	Tags              pq.StringArray `json:"tags" db:"tags"`
	IconURL           *string        `json:"icon_url,omitempty" db:"icon_url"`
	Screenshots       pq.StringArray `json:"screenshots" db:"screenshots"`
	InstallCount      int            `json:"install_count" db:"install_count"`
	Rating            float64        `json:"rating" db:"rating"`
	ReviewCount       int            `json:"review_count" db:"review_count"`
	Status            AppStatus      `json:"status" db:"status"`
	PrivacyPolicyURL  *string        `json:"privacy_policy_url,omitempty" db:"privacy_policy_url"`
	TermsOfServiceURL *string        `json:"terms_of_service_url,omitempty" db:"terms_of_service_url"`
	SupportServerID   *uuid.UUID     `json:"support_server_id,omitempty" db:"support_server_id"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
}

// AppInstallation represents a server's installation of an app
type AppInstallation struct {
	AppID       uuid.UUID `json:"app_id" db:"app_id"`
	ServerID    uuid.UUID `json:"server_id" db:"server_id"`
	InstallerID uuid.UUID `json:"installer_id" db:"installer_id"`
	InstalledAt time.Time `json:"installed_at" db:"installed_at"`

	// Populated fields
	App    *App    `json:"app,omitempty"`
	Server *Server `json:"server,omitempty"`
}

// AppReview represents a user review for an app
type AppReview struct {
	ID         uuid.UUID `json:"id" db:"id"`
	AppID      uuid.UUID `json:"app_id" db:"app_id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Rating     int       `json:"rating" db:"rating"`
	ReviewText *string   `json:"review_text,omitempty" db:"review_text"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`

	// Populated fields
	User *PublicUser `json:"user,omitempty"`
}

// AppDeveloperTeamMember represents a developer's role in an app
type AppDeveloperTeamMember struct {
	AppID  uuid.UUID `json:"app_id" db:"app_id"`
	UserID uuid.UUID `json:"user_id" db:"user_id"`
	Role   string    `json:"role" db:"role"`
}

// Developer roles for app teams
const (
	AppDeveloperRoleOwner  = "owner"
	AppDeveloperRoleAdmin  = "admin"
	AppDeveloperRoleMember = "member"
)

// --- Request/Response types ---

// CreateAppRequest is the input for creating a new app
type CreateAppRequest struct {
	Name              string   `json:"name" validate:"required,min=2,max=100"`
	Description       string   `json:"description" validate:"required,min=10,max=200"`
	LongDescription   *string  `json:"long_description,omitempty"`
	Category          string   `json:"category" validate:"required"`
	Tags              []string `json:"tags,omitempty"`
	IconURL           *string  `json:"icon_url,omitempty"`
	Screenshots       []string `json:"screenshots,omitempty"`
	PrivacyPolicyURL  *string  `json:"privacy_policy_url,omitempty"`
	TermsOfServiceURL *string  `json:"terms_of_service_url,omitempty"`
	SupportServerID   *string  `json:"support_server_id,omitempty"`
}

// UpdateAppRequest is the input for updating an app
type UpdateAppRequest struct {
	Name              *string  `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Description       *string  `json:"description,omitempty" validate:"omitempty,min=10,max=200"`
	LongDescription   *string  `json:"long_description,omitempty"`
	Category          *string  `json:"category,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	IconURL           *string  `json:"icon_url,omitempty"`
	Screenshots       []string `json:"screenshots,omitempty"`
	PrivacyPolicyURL  *string  `json:"privacy_policy_url,omitempty"`
	TermsOfServiceURL *string  `json:"terms_of_service_url,omitempty"`
	SupportServerID   *string  `json:"support_server_id,omitempty"`
}

// ListAppsRequest represents query parameters for listing apps
type ListAppsRequest struct {
	Category string `query:"category"`
	Query    string `query:"query"`
	Featured bool   `query:"featured"`
	Limit    int    `query:"limit"`
	Offset   int    `query:"offset"`
}

// CreateReviewRequest is the input for creating a review
type CreateReviewRequest struct {
	Rating     int    `json:"rating" validate:"required,min=1,max=5"`
	ReviewText string `json:"review_text,omitempty" validate:"max=2000"`
}

// UpdateReviewRequest is the input for updating a review
type UpdateReviewRequest struct {
	Rating     *int    `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	ReviewText *string `json:"review_text,omitempty" validate:"max=2000"`
}

// AppDeveloperAnalytics represents analytics data for a developer
type AppDeveloperAnalytics struct {
	TotalApps     int            `json:"total_apps"`
	TotalInstalls int            `json:"total_installs"`
	TotalReviews  int            `json:"total_reviews"`
	AverageRating float64        `json:"average_rating"`
	AppsByStatus  map[string]int `json:"apps_by_status"`
	InstallTrend  []int          `json:"install_trend"` // Last 7 days
	ReviewTrend   []int          `json:"review_trend"`  // Last 7 days
}
