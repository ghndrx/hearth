package models

import (
	"time"

	"github.com/google/uuid"
)

// ServerCategory represents a discovery category
type ServerCategory string

const (
	CategoryGaming        ServerCategory = "gaming"
	CategoryMusic         ServerCategory = "music"
	CategoryTechnology    ServerCategory = "technology"
	CategoryArt           ServerCategory = "art"
	CategoryEducation     ServerCategory = "education"
	CategoryScience       ServerCategory = "science"
	CategoryEntertainment ServerCategory = "entertainment"
	CategorySocial        ServerCategory = "social"
	CategorySports        ServerCategory = "sports"
	CategoryAnime         ServerCategory = "anime"
	CategoryFashion       ServerCategory = "fashion"
	CategoryFood          ServerCategory = "food"
	CategoryBusiness      ServerCategory = "business"
	CategoryLanguage      ServerCategory = "language"
)

// ApprovalStatus for server discovery listings
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

// DiscoveryListing represents a server's discovery listing
type DiscoveryListing struct {
	ID                  uuid.UUID      `json:"id" db:"id"`
	ServerID            uuid.UUID      `json:"server_id" db:"server_id"`
	ShortDescription    string         `json:"short_description" db:"short_description"`
	IsListed            bool           `json:"is_listed" db:"is_listed"`
	IsFeatured          bool           `json:"is_featured" db:"is_featured"`
	FeaturedAt          *time.Time     `json:"featured_at,omitempty" db:"featured_at"`
	ApprovalStatus      ApprovalStatus `json:"approval_status" db:"approval_status"`
	ApprovedAt          *time.Time     `json:"approved_at,omitempty" db:"approved_at"`
	ApprovedBy          *uuid.UUID     `json:"approved_by,omitempty" db:"approved_by"`
	RejectionReason     *string        `json:"rejection_reason,omitempty" db:"rejection_reason"`
	MemberCountSnapshot int            `json:"member_count_snapshot" db:"member_count_snapshot"`
	OnlineCountSnapshot int            `json:"online_count_snapshot" db:"online_count_snapshot"`
	WeeklyGrowthRate    float64        `json:"weekly_growth_rate" db:"weekly_growth_rate"`
	EngagementScore     float64        `json:"engagement_score" db:"engagement_score"`
	Region              *string        `json:"region,omitempty" db:"region"`
	Language            string         `json:"language" db:"language"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`

	// Populated via joins
	Server     *Server              `json:"server,omitempty" db:"-"`
	Categories []*DiscoveryCategory `json:"categories,omitempty" db:"-"`
	Tags       []string             `json:"tags,omitempty" db:"-"`
}

// ServerListingResult is returned by discovery search/listings
type ServerListingResult struct {
	ID                  uuid.UUID        `json:"id"`
	ServerID            uuid.UUID        `json:"server_id"`
	Name                string           `json:"name"`
	IconURL             *string          `json:"icon_url,omitempty"`
	BannerURL           *string          `json:"banner_url,omitempty"`
	Description         *string          `json:"description,omitempty"`
	ShortDescription    string           `json:"short_description"`
	Category            ServerCategory   `json:"primary_category"`
	Categories          []ServerCategory `json:"categories"`
	Tags                []string         `json:"tags"`
	MemberCount         int              `json:"member_count"`
	MemberCountSnapshot int              `json:"member_count_snapshot"`
	OnlineCount         int              `json:"online_count"`
	OnlineCountSnapshot int              `json:"online_count_snapshot"`
	IsFeatured          bool             `json:"is_featured"`
	IsVerified          bool             `json:"is_verified"`
	InviteCode          string           `json:"invite_code,omitempty"`
	Region              *string          `json:"region,omitempty"`
	Language            string           `json:"language"`
	WeeklyGrowthRate    float64          `json:"weekly_growth_rate"`
	EngagementScore     float64          `json:"engagement_score"`
	CreatedAt           time.Time        `json:"created_at"`
}

// FeaturedServer represents a featured server for the discovery page
type FeaturedServer struct {
	ServerListingResult
	BannerURL           *string   `json:"banner_url,omitempty"`
	FeaturedAt          time.Time `json:"featured_at"`
	MemberCountSnapshot int       `json:"member_count_snapshot" db:"member_count_snapshot"`
	OnlineCountSnapshot int       `json:"online_count_snapshot" db:"online_count_snapshot"`
}

// DiscoveryCategory represents a server discovery category
type DiscoveryCategory struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Icon        string    `json:"icon" db:"icon"`
	Description *string   `json:"description,omitempty" db:"description"`
	SortOrder   int       `json:"sort_order" db:"sort_order"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	ServerCount int       `json:"server_count,omitempty" db:"server_count"`
}

// DiscoveryTag represents a tag for servers
type DiscoveryTag struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Slug       string    `json:"slug" db:"slug"`
	UsageCount int       `json:"usage_count" db:"usage_count"`
}

// DiscoveryReport represents a user report for a server
type DiscoveryReport struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	ListingID       uuid.UUID  `json:"listing_id" db:"listing_id"`
	ReporterID      uuid.UUID  `json:"reporter_id" db:"reporter_id"`
	Reason          string     `json:"reason" db:"reason"`
	Details         *string    `json:"details,omitempty" db:"details"`
	Status          string     `json:"status" db:"status"`
	ReviewedBy      *uuid.UUID `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ResolutionNotes *string    `json:"resolution_notes,omitempty" db:"resolution_notes"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// DiscoveryFilters represents search/filter options
type DiscoveryFilters struct {
	Query      string           `json:"q,omitempty"`
	Category   ServerCategory   `json:"category,omitempty"`
	Categories []ServerCategory `json:"categories,omitempty"`
	Tags       []string         `json:"tags,omitempty"`
	Region     string           `json:"region,omitempty"`
	Language   string           `json:"language,omitempty"`
	MinMembers int              `json:"min_members,omitempty"`
	MaxMembers int              `json:"max_members,omitempty"`
	OnlineOnly bool             `json:"online_only,omitempty"`
	SortBy     string           `json:"sort_by,omitempty"`    // members, growth, engagement, newest
	SortOrder  string           `json:"sort_order,omitempty"` // asc, desc
	Featured   *bool            `json:"featured,omitempty"`
	Limit      int              `json:"limit,omitempty"`
	Offset     int              `json:"offset,omitempty"`
}

// SubmitDiscoveryRequest is the request to submit a server for discovery
type SubmitDiscoveryRequest struct {
	ShortDescription string           `json:"short_description" validate:"required,max=160"`
	Categories       []ServerCategory `json:"categories" validate:"required,min=1,max=3"`
	Tags             []string         `json:"tags,omitempty" validate:"max=5"`
	Region           *string          `json:"region,omitempty"`
	Language         string           `json:"language,omitempty"`
}

// UpdateDiscoveryRequest is the request to update a discovery listing
type UpdateDiscoveryRequest struct {
	ShortDescription *string          `json:"short_description,omitempty" validate:"omitempty,max=160"`
	Categories       []ServerCategory `json:"categories,omitempty" validate:"omitempty,min=1,max=3"`
	Tags             []string         `json:"tags,omitempty" validate:"omitempty,max=5"`
	Region           *string          `json:"region,omitempty"`
	Language         *string          `json:"language,omitempty"`
	IsListed         *bool            `json:"is_listed,omitempty"`
}

// ReportServerRequest is the request to report a server
type ReportServerRequest struct {
	Reason  string `json:"reason" validate:"required"`
	Details string `json:"details,omitempty"`
}

// DiscoveryStats represents discovery page statistics
type DiscoveryStats struct {
	TotalServers    int64 `json:"total_servers"`
	TotalCategories int   `json:"total_categories"`
	TotalMembers    int64 `json:"total_members"`
}

// TrendingServer represents a trending server in discovery
type TrendingServer struct {
	ServerListingResult
	BannerURL        *string   `json:"banner_url,omitempty"`
	TrendScore       float64   `json:"trend_score"`
	GrowthPercentage float64   `json:"growth_percentage"`
	LastTrendAt      time.Time `json:"last_trend_at"`
}

// TrendingCategory represents trending data for a category
type TrendingCategory struct {
	Category    *DiscoveryCategory `json:"category"`
	ServerCount int                `json:"server_count"`
	GrowthRate  float64            `json:"growth_rate"`
}

// DiscoveryPage represents the main discovery page data
type DiscoveryPage struct {
	Featured   []*FeaturedServer    `json:"featured"`
	Trending   []*TrendingServer    `json:"trending"`
	Categories []*DiscoveryCategory `json:"categories"`
	Stats      *DiscoveryStats      `json:"stats"`
}
