package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ServerDiscoveryCategory represents categories for the public server directory
type ServerDiscoveryCategory string

const (
	DiscoveryCategoryGaming        ServerDiscoveryCategory = "gaming"
	DiscoveryCategoryTechnology    ServerDiscoveryCategory = "technology"
	DiscoveryCategoryArt           ServerDiscoveryCategory = "art"
	DiscoveryCategoryMusic         ServerDiscoveryCategory = "music"
	DiscoveryCategorySports        ServerDiscoveryCategory = "sports"
	DiscoveryCategoryEducation     ServerDiscoveryCategory = "education"
	DiscoveryCategoryEntertainment ServerDiscoveryCategory = "entertainment"
	DiscoveryCategoryCommunity     ServerDiscoveryCategory = "community"
	DiscoveryCategoryOther         ServerDiscoveryCategory = "other"
)

// AllDiscoveryCategories returns all available discovery categories
func AllDiscoveryCategories() []ServerDiscoveryCategory {
	return []ServerDiscoveryCategory{
		DiscoveryCategoryGaming,
		DiscoveryCategoryTechnology,
		DiscoveryCategoryArt,
		DiscoveryCategoryMusic,
		DiscoveryCategorySports,
		DiscoveryCategoryEducation,
		DiscoveryCategoryEntertainment,
		DiscoveryCategoryCommunity,
		DiscoveryCategoryOther,
	}
}

// IsValidCategory checks if a category is valid
func IsValidCategory(c string) bool {
	for _, cat := range AllDiscoveryCategories() {
		if string(cat) == c {
			return true
		}
	}
	return false
}

// DiscoverableServer represents a server in the public directory
type DiscoverableServer struct {
	ID          uuid.UUID               `json:"id" db:"id"`
	ServerID    uuid.UUID               `json:"server_id" db:"server_id"`
	Name        string                  `json:"name" db:"name"`
	Description *string                 `json:"description,omitempty" db:"description"`
	Category    ServerDiscoveryCategory `json:"category" db:"category"`
	IconURL     *string                 `json:"icon_url,omitempty" db:"icon_url"`
	BannerURL   *string                 `json:"banner_url,omitempty" db:"banner_url"`
	Tags        pq.StringArray          `json:"tags" db:"tags"`
	MemberCount int                     `json:"member_count" db:"member_count"`
	IsVerified  bool                    `json:"is_verified" db:"is_verified"`
	IsPublic    bool                    `json:"is_public" db:"is_public"`
	IsFeatured  bool                    `json:"is_featured" db:"is_featured"`
	FeaturedAt  *time.Time              `json:"featured_at,omitempty" db:"featured_at"`
	CreatedAt   time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at" db:"updated_at"`
}

// DiscoverableServerDetail is the detailed view of a discoverable server
type DiscoverableServerDetail struct {
	DiscoverableServer
	InviteCode *string `json:"invite_code,omitempty"`
}

// DiscoverableServerSearchResult is returned when searching/discovering servers
type DiscoverableServerSearchResult struct {
	ID          uuid.UUID               `json:"id" db:"id"`
	ServerID    uuid.UUID               `json:"server_id" db:"server_id"`
	Name        string                  `json:"name" db:"name"`
	Description *string                 `json:"description,omitempty" db:"description"`
	Category    ServerDiscoveryCategory `json:"category" db:"category"`
	IconURL     *string                 `json:"icon_url,omitempty" db:"icon_url"`
	BannerURL   *string                 `json:"banner_url,omitempty" db:"banner_url"`
	Tags        pq.StringArray          `json:"tags" db:"tags"`
	MemberCount int                     `json:"member_count" db:"member_count"`
	IsVerified  bool                    `json:"is_verified" db:"is_verified"`
	IsFeatured  bool                    `json:"is_featured" db:"is_featured"`
	CreatedAt   time.Time               `json:"created_at" db:"created_at"`
}

// RegisterServerRequest is the request to register a server for discovery
type RegisterServerRequest struct {
	Name        string                  `json:"name" validate:"required,min=2,max=100"`
	Description string                  `json:"description" validate:"max=1000"`
	Category    ServerDiscoveryCategory `json:"category" validate:"required"`
	Tags        []string                `json:"tags,omitempty" validate:"max=10"`
}

// UpdateDiscoverableServerRequest is the request to update a discoverable server listing
type UpdateDiscoverableServerRequest struct {
	Name        *string                  `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Description *string                  `json:"description,omitempty" validate:"omitempty,max=1000"`
	Category    *ServerDiscoveryCategory `json:"category,omitempty"`
	Tags        []string                 `json:"tags,omitempty" validate:"omitempty,max=10"`
}

// RegisterServerRequest is the request to register a server for discovery
type RegisterServerRequest struct {
	Name        string                 `json:"name" validate:"required,min=2,max=100"`
	Description string                 `json:"description" validate:"max=1000"`
	Category    ServerDiscoveryCategory `json:"category" validate:"required"`
	Tags        []string               `json:"tags,omitempty" validate:"max=10"`
}

// UpdateDiscoverableServerRequest is the request to update a discoverable server listing
type UpdateDiscoverableServerRequest struct {
	Name        *string                 `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Description *string                 `json:"description,omitempty" validate:"omitempty,max=1000"`
	Category    *ServerDiscoveryCategory `json:"category,omitempty"`
	Tags        []string                `json:"tags,omitempty" validate:"omitempty,max=10"`
}

// PaginatedDiscoverableServers is the response for paginated server listings
type PaginatedDiscoverableServers struct {
	Servers    []*DiscoverableServerSearchResult `json:"servers"`
	Total      int                               `json:"total"`
	Page       int                               `json:"page"`
	Limit      int                               `json:"limit"`
	TotalPages int                               `json:"total_pages"`
}

// DiscoverableFeaturedServer represents a featured server in the discovery
type DiscoverableFeaturedServer struct {
	ID          uuid.UUID               `json:"id" db:"id"`
	ServerID    uuid.UUID               `json:"server_id" db:"server_id"`
	Name        string                  `json:"name" db:"name"`
	Description *string                 `json:"description,omitempty" db:"description"`
	Category    ServerDiscoveryCategory `json:"category" db:"category"`
	IconURL     *string                 `json:"icon_url,omitempty" db:"icon_url"`
	BannerURL   *string                 `json:"banner_url,omitempty" db:"banner_url"`
	MemberCount int                     `json:"member_count" db:"member_count"`
	IsVerified  bool                    `json:"is_verified" db:"is_verified"`
	FeaturedAt  time.Time               `json:"featured_at" db:"featured_at"`
	CreatedAt   time.Time               `json:"created_at" db:"created_at"`
}

// CategoryInfo represents a discovery category with server count
type CategoryInfo struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	ServerCount int    `json:"server_count"`
}

// DiscoverFilters represents search/filter options for server discovery
type DiscoverFilters struct {
	Query    string                  `json:"q,omitempty"`
	Category ServerDiscoveryCategory `json:"category,omitempty"`
	Page     int                     `json:"page,omitempty"`
	Limit    int                     `json:"limit,omitempty"`
}

// NormalizeDiscoverFilters sets default values and caps limits
func NormalizeDiscoverFilters(f *DiscoverFilters) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}
