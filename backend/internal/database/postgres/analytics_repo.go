package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// AnalyticsRepository provides data access for server analytics
type AnalyticsRepository struct {
	db *sqlx.DB
}

// NewAnalyticsRepository creates a new analytics repository
func NewAnalyticsRepository(db *sqlx.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// GetMemberGrowthHistory returns daily member counts for the specified period
func (r *AnalyticsRepository) GetMemberGrowthHistory(ctx context.Context, serverID uuid.UUID, days int) ([]*models.MemberGrowthPoint, error) {
	query := `
		WITH date_series AS (
			SELECT generate_series(
				CURRENT_DATE - INTERVAL '1 day' * $2,
				CURRENT_DATE,
				INTERVAL '1 day'
			)::DATE AS snapshot_date
		),
		snapshots_with_fill AS (
			SELECT 
				ds.snapshot_date,
				COALESCE(
					sms.member_count,
					LAG(sms.member_count) OVER (ORDER BY ds.snapshot_date),
					(SELECT member_count FROM server_member_snapshots 
					 WHERE server_id = $1 AND snapshot_date <= ds.snapshot_date 
					 ORDER BY snapshot_date DESC LIMIT 1)
				) AS member_count
			FROM date_series ds
			LEFT JOIN server_member_snapshots sms 
				ON sms.server_id = $1 AND sms.snapshot_date = ds.snapshot_date
		)
		SELECT 
			snapshot_date AS date,
			COALESCE(member_count, 0) AS count,
			COALESCE(member_count, 0) - COALESCE(LAG(member_count) OVER (ORDER BY snapshot_date), member_count) AS change
		FROM snapshots_with_fill
		ORDER BY snapshot_date ASC
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.MemberGrowthPoint
	for rows.Next() {
		var point models.MemberGrowthPoint
		if err := rows.Scan(&point.Date, &point.Count, &point.Change); err != nil {
			return nil, err
		}
		result = append(result, &point)
	}

	return result, rows.Err()
}

// GetMessageActivityStats returns hourly message counts for activity heatmap
func (r *AnalyticsRepository) GetMessageActivityStats(ctx context.Context, serverID uuid.UUID, days int) ([]*models.ActivityHourStat, error) {
	query := `
		SELECT 
			EXTRACT(DOW FROM activity_hour)::INT AS day_of_week,
			EXTRACT(HOUR FROM activity_hour)::INT AS hour,
			SUM(message_count)::INT AS message_count,
			SUM(unique_users)::INT AS unique_users
		FROM server_activity_hourly
		WHERE server_id = $1 
			AND activity_hour >= NOW() - INTERVAL '1 day' * $2
		GROUP BY EXTRACT(DOW FROM activity_hour), EXTRACT(HOUR FROM activity_hour)
		ORDER BY day_of_week, hour
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.ActivityHourStat
	for rows.Next() {
		var stat models.ActivityHourStat
		if err := rows.Scan(&stat.DayOfWeek, &stat.Hour, &stat.MessageCount, &stat.UniqueUsers); err != nil {
			return nil, err
		}
		result = append(result, &stat)
	}

	return result, rows.Err()
}

// GetTopChannels returns channels ranked by message volume
func (r *AnalyticsRepository) GetTopChannels(ctx context.Context, serverID uuid.UUID, days int, limit int) ([]*models.TopChannelStat, error) {
	query := `
		WITH channel_stats AS (
			SELECT 
				c.id AS channel_id,
				c.name AS channel_name,
				c.type AS channel_type,
				COUNT(m.id)::INT AS message_count,
				COUNT(DISTINCT m.author_id)::INT AS unique_authors,
				MAX(m.created_at) AS last_activity
			FROM channels c
			LEFT JOIN messages m ON m.channel_id = c.id 
				AND m.created_at >= NOW() - INTERVAL '1 day' * $2
			WHERE c.server_id = $1 AND c.type IN ('text', 'announcement', 'forum')
			GROUP BY c.id, c.name, c.type
		)
		SELECT 
			channel_id,
			channel_name,
			channel_type,
			message_count,
			unique_authors,
			last_activity
		FROM channel_stats
		ORDER BY message_count DESC, channel_name ASC
		LIMIT $3
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.TopChannelStat
	for rows.Next() {
		var stat models.TopChannelStat
		if err := rows.Scan(
			&stat.ChannelID,
			&stat.ChannelName,
			&stat.ChannelType,
			&stat.MessageCount,
			&stat.UniqueAuthors,
			&stat.LastActivity,
		); err != nil {
			return nil, err
		}
		result = append(result, &stat)
	}

	return result, rows.Err()
}

// GetRetentionMetrics returns DAU/MAU and retention data
func (r *AnalyticsRepository) GetRetentionMetrics(ctx context.Context, serverID uuid.UUID, days int) (*models.RetentionMetrics, error) {
	// Get DAU for each day in the period
	dauQuery := `
		SELECT 
			activity_date AS date,
			COUNT(DISTINCT user_id)::INT AS active_users
		FROM server_daily_active_users
		WHERE server_id = $1 AND activity_date >= CURRENT_DATE - INTERVAL '1 day' * $2
		GROUP BY activity_date
		ORDER BY activity_date ASC
	`

	dauRows, err := r.db.QueryxContext(ctx, dauQuery, serverID, days)
	if err != nil {
		return nil, err
	}
	defer dauRows.Close()

	var dailyActiveUsers []*models.DailyActiveUserPoint
	for dauRows.Next() {
		var point models.DailyActiveUserPoint
		if err := dauRows.Scan(&point.Date, &point.Count); err != nil {
			return nil, err
		}
		dailyActiveUsers = append(dailyActiveUsers, &point)
	}
	if err := dauRows.Err(); err != nil {
		return nil, err
	}

	// Get MAU (monthly active users)
	mauQuery := `
		SELECT COUNT(DISTINCT user_id)::INT AS mau
		FROM server_daily_active_users
		WHERE server_id = $1 AND activity_date >= CURRENT_DATE - INTERVAL '30 days'
	`
	var mau int
	if err := r.db.GetContext(ctx, &mau, mauQuery, serverID); err != nil {
		return nil, err
	}

	// Get total member count
	memberCountQuery := `SELECT COUNT(*) FROM members WHERE server_id = $1`
	var totalMembers int
	if err := r.db.GetContext(ctx, &totalMembers, memberCountQuery, serverID); err != nil {
		return nil, err
	}

	// Calculate average DAU
	var avgDAU float64
	if len(dailyActiveUsers) > 0 {
		sum := 0
		for _, d := range dailyActiveUsers {
			sum += d.Count
		}
		avgDAU = float64(sum) / float64(len(dailyActiveUsers))
	}

	// Calculate DAU/MAU ratio (stickiness)
	var stickiness float64
	if mau > 0 {
		stickiness = avgDAU / float64(mau)
	}

	return &models.RetentionMetrics{
		DailyActiveUsers: dailyActiveUsers,
		MAU:              mau,
		TotalMembers:     totalMembers,
		AverageDAU:       avgDAU,
		Stickiness:       stickiness,
	}, nil
}

// GetServerAnalyticsSummary returns a quick summary of key metrics
func (r *AnalyticsRepository) GetServerAnalyticsSummary(ctx context.Context, serverID uuid.UUID) (*models.AnalyticsSummary, error) {
	query := `
		WITH recent_stats AS (
			SELECT 
				COUNT(DISTINCT m.id)::INT AS messages_today,
				COUNT(DISTINCT m.author_id)::INT AS active_users_today
			FROM messages m
			JOIN channels c ON c.id = m.channel_id
			WHERE c.server_id = $1 AND m.created_at >= CURRENT_DATE
		),
		weekly_stats AS (
			SELECT 
				COUNT(DISTINCT m.id)::INT AS messages_week,
				COUNT(DISTINCT m.author_id)::INT AS active_users_week
			FROM messages m
			JOIN channels c ON c.id = m.channel_id
			WHERE c.server_id = $1 AND m.created_at >= CURRENT_DATE - INTERVAL '7 days'
		),
		member_stats AS (
			SELECT 
				COUNT(*)::INT AS total_members,
				COUNT(*) FILTER (WHERE joined_at >= CURRENT_DATE - INTERVAL '7 days')::INT AS new_members_week
			FROM members
			WHERE server_id = $1
		),
		growth AS (
			SELECT 
				COALESCE(
					(SELECT member_count FROM server_member_snapshots 
					 WHERE server_id = $1 ORDER BY snapshot_date DESC LIMIT 1) -
					(SELECT member_count FROM server_member_snapshots 
					 WHERE server_id = $1 AND snapshot_date <= CURRENT_DATE - INTERVAL '7 days'
					 ORDER BY snapshot_date DESC LIMIT 1),
					0
				)::INT AS member_change_week
		)
		SELECT 
			r.messages_today,
			r.active_users_today,
			w.messages_week,
			w.active_users_week,
			m.total_members,
			m.new_members_week,
			g.member_change_week
		FROM recent_stats r, weekly_stats w, member_stats m, growth g
	`

	var summary models.AnalyticsSummary
	err := r.db.QueryRowxContext(ctx, query, serverID).Scan(
		&summary.MessagesToday,
		&summary.ActiveUsersToday,
		&summary.MessagesWeek,
		&summary.ActiveUsersWeek,
		&summary.TotalMembers,
		&summary.NewMembersWeek,
		&summary.MemberChangeWeek,
	)
	if err != nil {
		return nil, err
	}

	// Calculate week-over-week change percentage
	if summary.MessagesWeek > 0 {
		// Get previous week's messages for comparison
		prevWeekQuery := `
			SELECT COUNT(*)::INT
			FROM messages m
			JOIN channels c ON c.id = m.channel_id
			WHERE c.server_id = $1 
				AND m.created_at >= CURRENT_DATE - INTERVAL '14 days'
				AND m.created_at < CURRENT_DATE - INTERVAL '7 days'
		`
		var prevWeekMessages int
		if err := r.db.GetContext(ctx, &prevWeekMessages, prevWeekQuery, serverID); err == nil && prevWeekMessages > 0 {
			summary.MessageChangePercent = float64(summary.MessagesWeek-prevWeekMessages) / float64(prevWeekMessages) * 100
		}
	}

	return &summary, nil
}

// TakeMemberSnapshot creates a snapshot of current member counts for all servers
// Should be called daily via scheduler/cron
func (r *AnalyticsRepository) TakeMemberSnapshot(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "SELECT take_member_snapshots()")
	return err
}

// TakeMemberSnapshotForServer creates a snapshot for a specific server
func (r *AnalyticsRepository) TakeMemberSnapshotForServer(ctx context.Context, serverID uuid.UUID) error {
	query := `
		INSERT INTO server_member_snapshots (server_id, snapshot_date, member_count)
		SELECT $1, CURRENT_DATE, COUNT(*)
		FROM members
		WHERE server_id = $1
		ON CONFLICT (server_id, snapshot_date) DO UPDATE SET
			member_count = EXCLUDED.member_count,
			created_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, serverID)
	return err
}

// GetPeakActivityHours returns the most active hours for a server
func (r *AnalyticsRepository) GetPeakActivityHours(ctx context.Context, serverID uuid.UUID, days int) ([]*models.PeakHour, error) {
	query := `
		SELECT 
			EXTRACT(HOUR FROM activity_hour)::INT AS hour,
			SUM(message_count)::INT AS total_messages
		FROM server_activity_hourly
		WHERE server_id = $1 AND activity_hour >= NOW() - INTERVAL '1 day' * $2
		GROUP BY EXTRACT(HOUR FROM activity_hour)
		ORDER BY total_messages DESC
		LIMIT 5
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.PeakHour
	for rows.Next() {
		var peak models.PeakHour
		if err := rows.Scan(&peak.Hour, &peak.MessageCount); err != nil {
			return nil, err
		}
		result = append(result, &peak)
	}

	return result, rows.Err()
}

// GetMostActiveUsers returns the most active users in a server
func (r *AnalyticsRepository) GetMostActiveUsers(ctx context.Context, serverID uuid.UUID, days int, limit int) ([]*models.ActiveUserStat, error) {
	query := `
		SELECT 
			dau.user_id,
			u.username,
			u.display_name,
			u.avatar_url,
			SUM(dau.message_count)::INT AS message_count,
			COUNT(DISTINCT dau.activity_date)::INT AS days_active
		FROM server_daily_active_users dau
		JOIN users u ON u.id = dau.user_id
		WHERE dau.server_id = $1 AND dau.activity_date >= CURRENT_DATE - INTERVAL '1 day' * $2
		GROUP BY dau.user_id, u.username, u.display_name, u.avatar_url
		ORDER BY message_count DESC
		LIMIT $3
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.ActiveUserStat
	for rows.Next() {
		var stat models.ActiveUserStat
		if err := rows.Scan(
			&stat.UserID,
			&stat.Username,
			&stat.DisplayName,
			&stat.AvatarURL,
			&stat.MessageCount,
			&stat.DaysActive,
		); err != nil {
			return nil, err
		}
		result = append(result, &stat)
	}

	return result, rows.Err()
}

// CleanupOldAnalyticsData removes analytics data older than the retention period
func (r *AnalyticsRepository) CleanupOldAnalyticsData(ctx context.Context, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Clean up hourly activity data
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM server_activity_hourly WHERE activity_hour < $1`,
		cutoff,
	)
	if err != nil {
		return err
	}

	// Clean up daily active users data
	_, err = r.db.ExecContext(ctx,
		`DELETE FROM server_daily_active_users WHERE activity_date < $1`,
		cutoff.Format("2006-01-02"),
	)
	if err != nil {
		return err
	}

	// Keep member snapshots for longer (1 year)
	yearAgo := time.Now().AddDate(-1, 0, 0)
	_, err = r.db.ExecContext(ctx,
		`DELETE FROM server_member_snapshots WHERE snapshot_date < $1`,
		yearAgo.Format("2006-01-02"),
	)

	return err
}
