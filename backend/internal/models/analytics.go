package models

import (
	"time"

	"github.com/google/uuid"
)

// MemberGrowthPoint represents a single point in member growth history
type MemberGrowthPoint struct {
	Date   time.Time `json:"date" db:"date"`
	Count  int       `json:"count" db:"count"`
	Change int       `json:"change" db:"change"` // Change from previous day
}

// ActivityHourStat represents message activity for a specific hour of the week
type ActivityHourStat struct {
	DayOfWeek    int `json:"day_of_week" db:"day_of_week"` // 0=Sunday, 6=Saturday
	Hour         int `json:"hour" db:"hour"`               // 0-23
	MessageCount int `json:"message_count" db:"message_count"`
	UniqueUsers  int `json:"unique_users" db:"unique_users"`
}

// TopChannelStat represents channel ranking by activity
type TopChannelStat struct {
	ChannelID     uuid.UUID  `json:"channel_id" db:"channel_id"`
	ChannelName   string     `json:"channel_name" db:"channel_name"`
	ChannelType   string     `json:"channel_type" db:"channel_type"`
	MessageCount  int        `json:"message_count" db:"message_count"`
	UniqueAuthors int        `json:"unique_authors" db:"unique_authors"`
	LastActivity  *time.Time `json:"last_activity,omitempty" db:"last_activity"`
}

// DailyActiveUserPoint represents DAU for a single day
type DailyActiveUserPoint struct {
	Date  time.Time `json:"date" db:"date"`
	Count int       `json:"count" db:"count"`
}

// RetentionMetrics contains user retention and engagement data
type RetentionMetrics struct {
	DailyActiveUsers []*DailyActiveUserPoint `json:"daily_active_users"`
	MAU              int                     `json:"mau"`           // Monthly Active Users
	TotalMembers     int                     `json:"total_members"` // Current member count
	AverageDAU       float64                 `json:"average_dau"`   // Average DAU over period
	Stickiness       float64                 `json:"stickiness"`    // DAU/MAU ratio (0-1)
}

// AnalyticsSummary provides a quick overview of key metrics
type AnalyticsSummary struct {
	// Today's stats
	MessagesToday    int `json:"messages_today" db:"messages_today"`
	ActiveUsersToday int `json:"active_users_today" db:"active_users_today"`

	// Weekly stats
	MessagesWeek    int `json:"messages_week" db:"messages_week"`
	ActiveUsersWeek int `json:"active_users_week" db:"active_users_week"`

	// Member stats
	TotalMembers     int `json:"total_members" db:"total_members"`
	NewMembersWeek   int `json:"new_members_week" db:"new_members_week"`
	MemberChangeWeek int `json:"member_change_week" db:"member_change_week"`

	// Trends
	MessageChangePercent float64 `json:"message_change_percent"` // Week-over-week change
}

// PeakHour represents peak activity time
type PeakHour struct {
	Hour         int `json:"hour" db:"hour"`
	MessageCount int `json:"message_count" db:"message_count"`
}

// ActiveUserStat represents a user's activity statistics
type ActiveUserStat struct {
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	Username     string    `json:"username" db:"username"`
	DisplayName  *string   `json:"display_name,omitempty" db:"display_name"`
	AvatarURL    *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	MessageCount int       `json:"message_count" db:"message_count"`
	DaysActive   int       `json:"days_active" db:"days_active"`
}

// ServerInsightsResponse is the combined response for the insights dashboard
type ServerInsightsResponse struct {
	ServerID uuid.UUID         `json:"server_id"`
	Period   string            `json:"period"` // "7d", "30d", "90d"
	Summary  *AnalyticsSummary `json:"summary"`
}

// MemberGrowthResponse wraps member growth data
type MemberGrowthResponse struct {
	ServerID string               `json:"server_id"`
	Period   string               `json:"period"`
	Data     []*MemberGrowthPoint `json:"data"`
}

// ActivityHeatmapResponse wraps activity heatmap data
type ActivityHeatmapResponse struct {
	ServerID   string              `json:"server_id"`
	Period     string              `json:"period"`
	Data       []*ActivityHourStat `json:"data"`
	PeakHours  []*PeakHour         `json:"peak_hours,omitempty"`
	TotalStats struct {
		TotalMessages int     `json:"total_messages"`
		AvgPerHour    float64 `json:"avg_per_hour"`
	} `json:"total_stats"`
}

// TopChannelsResponse wraps top channels data
type TopChannelsResponse struct {
	ServerID string            `json:"server_id"`
	Period   string            `json:"period"`
	Data     []*TopChannelStat `json:"data"`
}

// RetentionResponse wraps retention metrics
type RetentionResponse struct {
	ServerID string            `json:"server_id"`
	Period   string            `json:"period"`
	Data     *RetentionMetrics `json:"data"`
}

// ServerMemberSnapshot represents a daily member count snapshot
type ServerMemberSnapshot struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ServerID     uuid.UUID `json:"server_id" db:"server_id"`
	SnapshotDate time.Time `json:"snapshot_date" db:"snapshot_date"`
	MemberCount  int       `json:"member_count" db:"member_count"`
	OnlineCount  int       `json:"online_count" db:"online_count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ServerActivityHourly represents hourly message activity
type ServerActivityHourly struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ServerID     uuid.UUID `json:"server_id" db:"server_id"`
	ActivityHour time.Time `json:"activity_hour" db:"activity_hour"`
	MessageCount int       `json:"message_count" db:"message_count"`
	UniqueUsers  int       `json:"unique_users" db:"unique_users"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// ServerDailyActiveUser tracks daily user activity
type ServerDailyActiveUser struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ServerID     uuid.UUID `json:"server_id" db:"server_id"`
	ActivityDate time.Time `json:"activity_date" db:"activity_date"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	MessageCount int       `json:"message_count" db:"message_count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// AnalyticsQueryParams holds common query parameters for analytics endpoints
type AnalyticsQueryParams struct {
	Days  int `query:"days"`  // Number of days to query (default 7, max 90)
	Limit int `query:"limit"` // Limit for paginated results
}

// Normalize applies defaults and limits to query params
func (p *AnalyticsQueryParams) Normalize() {
	if p.Days <= 0 {
		p.Days = 7
	}
	if p.Days > 90 {
		p.Days = 90
	}
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 50 {
		p.Limit = 50
	}
}
