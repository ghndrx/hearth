package models

import (
	"time"

	"github.com/google/uuid"
)

// ServerDiscoveryDailyStats represents daily aggregated discovery stats for a server
type ServerDiscoveryDailyStats struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ServerID     uuid.UUID `json:"server_id" db:"server_id"`
	StatDate     time.Time `json:"stat_date" db:"stat_date"`
	Views        int       `json:"views" db:"views"`
	Impressions  int       `json:"impressions" db:"impressions"`
	Joins        int       `json:"joins" db:"joins"`
	SearchClicks int       `json:"search_clicks" db:"search_clicks"`
}

// ServerRecommendation represents a recommended server for a user
type ServerRecommendation struct {
	DiscoverableServerSearchResult
	Reason            string   `json:"reason"`                   // Why this server is recommended
	MutualMemberCount int      `json:"mutual_member_count"`      // Number of mutual members
	MutualServers     []string `json:"mutual_servers,omitempty"` // Shared servers
}

// DiscoverySearchRequest represents advanced search parameters
type DiscoverySearchRequest struct {
	Query      string                    `json:"q,omitempty"`
	Category   ServerDiscoveryCategory   `json:"category,omitempty"`
	Categories []ServerDiscoveryCategory `json:"categories,omitempty"`
	Tags       []string                  `json:"tags,omitempty"`
	SortBy     string                    `json:"sort_by,omitempty"` // popular, new, active, recommended
	SortOrder  string                    `json:"sort_order,omitempty"`
	Page       int                       `json:"page,omitempty"`
	Limit      int                       `json:"limit,omitempty"`
}

// DiscoverySearchResponse represents paginated search results
type DiscoverySearchResponse struct {
	Servers    []*DiscoverableServerSearchResult `json:"servers"`
	Total      int                               `json:"total"`
	Page       int                               `json:"page"`
	Limit      int                               `json:"limit"`
	TotalPages int                               `json:"total_pages"`
}

// TrendingServerInfo represents trending server data
type TrendingServerInfo struct {
	Server             *DiscoverableServerSearchResult `json:"server"`
	TrendScore         float64                         `json:"trend_score"`
	GrowthRate         float64                         `json:"growth_rate"`
	ActiveMembersRatio float64                         `json:"active_members_ratio"`
	RankChange         int                             `json:"rank_change"` // Positive = moved up, negative = moved down
}

// CategoryWithStats represents a category with additional statistics
type CategoryWithStats struct {
	CategoryInfo
	TotalMembers   int     `json:"total_members"`
	AvgMemberCount float64 `json:"avg_member_count"`
	GrowthRate     float64 `json:"growth_rate"`
}

// DiscoveryHomePage represents the full discovery home page data
type DiscoveryHomePage struct {
	Featured    []*DiscoverableFeaturedServer `json:"featured"`
	Trending    []*TrendingServerInfo         `json:"trending"`
	Recommended []*ServerRecommendation       `json:"recommended"`
	Categories  []*CategoryWithStats          `json:"categories"`
	PopularTags []*DiscoveryTag               `json:"popular_tags"`
	Stats       *DiscoveryPageStats           `json:"stats"`
}

// DiscoveryPageStats represents statistics for the discovery page
type DiscoveryPageStats struct {
	TotalServers       int64 `json:"total_servers"`
	TotalMembers       int64 `json:"total_members"`
	TotalCategories    int   `json:"total_categories"`
	NewServersThisWeek int   `json:"new_servers_this_week"`
}

// DiscoverableServerWithStats is a discoverable server with additional statistics
type DiscoverableServerWithStats struct {
	DiscoverableServerSearchResult
	OnlineMembersRatio float64 `json:"online_members_ratio"`
	ActivityScore      float64 `json:"activity_score"`
	EngagementRate     float64 `json:"engagement_rate"`
}

// SearchSuggestion represents a search suggestion
type SearchSuggestion struct {
	Type  string `json:"type"` // category, tag, server
	Value string `json:"value"`
	Count int    `json:"count,omitempty"`
}
