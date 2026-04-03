package postgres

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AuditLog represents an audit log entry stored in PostgreSQL
type AuditLog struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	ServerID       uuid.UUID       `db:"server_id" json:"server_id"`
	ActorID        uuid.UUID       `db:"actor_id" json:"actor_id"`
	ActionType     string          `db:"action_type" json:"action_type"`
	ActionCategory int             `db:"action_category" json:"action_category"`
	TargetID       *uuid.UUID      `db:"target_id" json:"target_id,omitempty"`
	TargetType     *string         `db:"target_type" json:"target_type,omitempty"`
	Reason         *string         `db:"reason" json:"reason,omitempty"`
	Changes        json.RawMessage `db:"changes" json:"changes,omitempty"`
	Metadata       json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	IPAddress      *string         `db:"ip_address" json:"ip_address,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

// AuditLogRepository provides PostgreSQL persistence for audit logs
type AuditLogRepository struct {
	db *sqlx.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// AuditLogFilter contains filtering options for audit log queries
type AuditLogFilter struct {
	ActionType    string     // Filter by specific action type
	ActionCategory int       // Filter by action category (10-89)
	ActorID       *uuid.UUID // Filter by the user who performed the action
	TargetID      *uuid.UUID // Filter by the target of the action
	TargetType    string     // Filter by target type (member, message, channel, role)
	Before        *time.Time // Filter entries before this time
	After         *time.Time // Filter entries after this time
	ReasonKeyword string     // Filter by keyword in reason
	Limit         int        // Maximum number of entries (default 50, max 100)
	Offset        int        // Offset for pagination
}

// Create inserts a new audit log entry
func (r *AuditLogRepository) Create(ctx context.Context, log *AuditLog) error {
	query := `
		INSERT INTO audit_logs (server_id, actor_id, action_type, action_category, target_id, target_type, reason, changes, metadata, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	row := r.db.QueryRowxContext(ctx, query,
		log.ServerID,
		log.ActorID,
		log.ActionType,
		log.ActionCategory,
		log.TargetID,
		log.TargetType,
		log.Reason,
		log.Changes,
		log.Metadata,
		log.IPAddress,
		log.CreatedAt,
	)

	err := row.Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// CreateBatch inserts multiple audit log entries in a single transaction
func (r *AuditLogRepository) CreateBatch(ctx context.Context, logs []AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PreparexContext(ctx, `
		INSERT INTO audit_logs (id, server_id, actor_id, action_type, action_category, target_id, target_type, reason, changes, metadata, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for i := range logs {
		if logs[i].ID == uuid.Nil {
			logs[i].ID = uuid.New()
		}
		if logs[i].CreatedAt.IsZero() {
			logs[i].CreatedAt = time.Now()
		}

		_, err := stmt.ExecContext(ctx,
			logs[i].ID,
			logs[i].ServerID,
			logs[i].ActorID,
			logs[i].ActionType,
			logs[i].ActionCategory,
			logs[i].TargetID,
			logs[i].TargetType,
			logs[i].Reason,
			logs[i].Changes,
			logs[i].Metadata,
			logs[i].IPAddress,
			logs[i].CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert audit log: %w", err)
		}
	}

	return tx.Commit()
}

// GetByID retrieves a specific audit log entry
func (r *AuditLogRepository) GetByID(ctx context.Context, serverID, logID uuid.UUID) (*AuditLog, error) {
	query := `
		SELECT id, server_id, actor_id, action_type, action_category, target_id, target_type, reason, changes, metadata, ip_address, created_at
		FROM audit_logs
		WHERE server_id = $1 AND id = $2
	`

	var log AuditLog
	err := r.db.GetContext(ctx, &log, query, serverID, logID)
	if err != nil {
		return nil, err
	}

	return &log, nil
}

// GetByServer retrieves audit logs for a server with filtering and pagination
func (r *AuditLogRepository) GetByServer(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) ([]AuditLog, int, error) {
	// Set defaults
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	// Build query dynamically
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("server_id = $%d", argNum))
	args = append(args, serverID)
	argNum++

	if filter.ActionType != "" {
		conditions = append(conditions, fmt.Sprintf("action_type = $%d", argNum))
		args = append(args, filter.ActionType)
		argNum++
	}

	if filter.ActionCategory > 0 {
		conditions = append(conditions, fmt.Sprintf("action_category = $%d", argNum))
		args = append(args, filter.ActionCategory)
		argNum++
	}

	if filter.ActorID != nil {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argNum))
		args = append(args, *filter.ActorID)
		argNum++
	}

	if filter.TargetID != nil {
		conditions = append(conditions, fmt.Sprintf("target_id = $%d", argNum))
		args = append(args, *filter.TargetID)
		argNum++
	}

	if filter.TargetType != "" {
		conditions = append(conditions, fmt.Sprintf("target_type = $%d", argNum))
		args = append(args, filter.TargetType)
		argNum++
	}

	if filter.Before != nil {
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", argNum))
		args = append(args, *filter.Before)
		argNum++
	}

	if filter.After != nil {
		conditions = append(conditions, fmt.Sprintf("created_at > $%d", argNum))
		args = append(args, *filter.After)
		argNum++
	}

	if filter.ReasonKeyword != "" {
		conditions = append(conditions, fmt.Sprintf("reason ILIKE $%d", argNum))
		args = append(args, "%"+filter.ReasonKeyword+"%")
		argNum++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs WHERE %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, server_id, actor_id, action_type, action_category, target_id, target_type, reason, changes, metadata, ip_address, created_at
		FROM audit_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, filter.Limit, filter.Offset)

	var logs []AuditLog
	err = r.db.SelectContext(ctx, &logs, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %w", err)
	}

	return logs, total, nil
}

// DeleteOlderThan deletes audit logs older than the specified time for a server
func (r *AuditLogRepository) DeleteOlderThan(ctx context.Context, serverID uuid.UUID, before time.Time) (int64, error) {
	query := `DELETE FROM audit_logs WHERE server_id = $1 AND created_at < $2`
	result, err := r.db.ExecContext(ctx, query, serverID, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	return result.RowsAffected()
}

// DeleteAllOlderThan deletes all audit logs older than the specified time (across all servers)
func (r *AuditLogRepository) DeleteAllOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < $1`
	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	return result.RowsAffected()
}

// GetByActor retrieves audit logs for a specific actor within a server
func (r *AuditLogRepository) GetByActor(ctx context.Context, serverID, actorID uuid.UUID, limit, offset int) ([]AuditLog, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE server_id = $1 AND actor_id = $2`
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, serverID, actorID)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, server_id, actor_id, action_type, action_category, target_id, target_type, reason, changes, metadata, ip_address, created_at
		FROM audit_logs
		WHERE server_id = $1 AND actor_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	var logs []AuditLog
	err = r.db.SelectContext(ctx, &logs, query, serverID, actorID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByTarget retrieves audit logs for a specific target within a server
func (r *AuditLogRepository) GetByTarget(ctx context.Context, serverID, targetID uuid.UUID, limit, offset int) ([]AuditLog, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE server_id = $1 AND target_id = $2`
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, serverID, targetID)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, server_id, actor_id, action_type, action_category, target_id, target_type, reason, changes, metadata, ip_address, created_at
		FROM audit_logs
		WHERE server_id = $1 AND target_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	var logs []AuditLog
	err = r.db.SelectContext(ctx, &logs, query, serverID, targetID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetActionCounts returns counts of each action type for a server in a time range
func (r *AuditLogRepository) GetActionCounts(ctx context.Context, serverID uuid.UUID, since time.Time) (map[string]int, error) {
	query := `
		SELECT action_type, COUNT(*) as count
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= $2
		GROUP BY action_type
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var actionType string
		var count int
		if err := rows.Scan(&actionType, &count); err != nil {
			return nil, err
		}
		result[actionType] = count
	}

	return result, rows.Err()
}

// GetActionCategoryCounts returns counts of each action category for a server in a time range
func (r *AuditLogRepository) GetActionCategoryCounts(ctx context.Context, serverID uuid.UUID, since time.Time) (map[int]int, error) {
	query := `
		SELECT action_category, COUNT(*) as count
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= $2 AND action_category > 0
		GROUP BY action_category
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]int)
	for rows.Next() {
		var category int
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, err
		}
		result[category] = count
	}

	return result, rows.Err()
}

// GetModeratorActivity returns moderation activity metrics per moderator
func (r *AuditLogRepository) GetModeratorActivity(ctx context.Context, serverID uuid.UUID, since time.Time) ([]ModeratorActivity, error) {
	query := `
		SELECT 
			actor_id,
			COUNT(*) as total_actions,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_BAN') as bans,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_UNBAN') as unbans,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_KICK') as kicks,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_TIMEOUT') as mutes,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_TIMEOUT_REMOVE') as unmutes,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_UPDATE') as warns,
			COUNT(*) FILTER (WHERE action_type = 'MESSAGE_DELETE') as message_deletes
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= $2
		GROUP BY actor_id
		ORDER BY total_actions DESC
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ModeratorActivity
	for rows.Next() {
		var ma ModeratorActivity
		if err := rows.Scan(
			&ma.ActorID,
			&ma.TotalActions,
			&ma.Bans,
			&ma.Unbans,
			&ma.Kicks,
			&ma.Mutes,
			&ma.Unmutes,
			&ma.Warns,
			&ma.MessageDeletes,
		); err != nil {
			return nil, err
		}
		results = append(results, ma)
	}

	return results, rows.Err()
}

// ModeratorActivity represents moderation activity metrics for a single moderator
type ModeratorActivity struct {
	ActorID        uuid.UUID `db:"actor_id" json:"actor_id"`
	TotalActions   int       `db:"total_actions" json:"total_actions"`
	Bans           int       `db:"bans" json:"bans"`
	Unbans         int       `db:"unbans" json:"unbans"`
	Kicks          int       `db:"kicks" json:"kicks"`
	Mutes          int       `db:"mutes" json:"mutes"`
	Unmutes        int       `db:"unmutes" json:"unmutes"`
	Warns          int       `db:"warns" json:"warns"`
	MessageDeletes int       `db:"message_deletes" json:"message_deletes"`
}

// GetRepeatOffenders returns users who have been moderated multiple times
func (r *AuditLogRepository) GetRepeatOffenders(ctx context.Context, serverID uuid.UUID, since time.Time, minCount int) ([]RepeatOffender, error) {
	query := `
		SELECT 
			target_id,
			COUNT(*) as moderation_count,
			COUNT(DISTINCT actor_id) as different_moderators,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_BAN') as bans,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_UPDATE') as warns,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_TIMEOUT') as mutes
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= $2 AND target_id IS NOT NULL
		GROUP BY target_id
		HAVING COUNT(*) >= $3
		ORDER BY moderation_count DESC
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, since, minCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RepeatOffender
	for rows.Next() {
		var ro RepeatOffender
		if err := rows.Scan(
			&ro.TargetID,
			&ro.ModerationCount,
			&ro.DifferentModerators,
			&ro.Bans,
			&ro.Warns,
			&ro.Mutes,
		); err != nil {
			return nil, err
		}
		results = append(results, ro)
	}

	return results, rows.Err()
}

// RepeatOffender represents a user who has been moderated multiple times
type RepeatOffender struct {
	TargetID            uuid.UUID `db:"target_id" json:"target_id"`
	ModerationCount     int       `db:"moderation_count" json:"moderation_count"`
	DifferentModerators int       `db:"different_moderators" json:"different_moderators"`
	Bans                int       `db:"bans" json:"bans"`
	Warns               int       `db:"warns" json:"warns"`
	Mutes               int       `db:"mutes" json:"mutes"`
}

// GetTrendData returns daily counts of moderation actions for trend analysis
func (r *AuditLogRepository) GetTrendData(ctx context.Context, serverID uuid.UUID, days int) ([]DailyTrendPoint, error) {
	query := `
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as total_actions,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_BAN') as bans,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_KICK') as kicks,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_TIMEOUT') as mutes,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_UPDATE') as warns,
			COUNT(*) FILTER (WHERE action_type = 'MESSAGE_DELETE') as message_deletes
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= NOW() - INTERVAL '1 day' * $2
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DailyTrendPoint
	for rows.Next() {
		var dt DailyTrendPoint
		if err := rows.Scan(
			&dt.Date,
			&dt.TotalActions,
			&dt.Bans,
			&dt.Kicks,
			&dt.Mutes,
			&dt.Warns,
			&dt.MessageDeletes,
		); err != nil {
			return nil, err
		}
		results = append(results, dt)
	}

	return results, rows.Err()
}

// DailyTrendPoint represents daily moderation action counts
type DailyTrendPoint struct {
	Date           time.Time `db:"date" json:"date"`
	TotalActions  int       `db:"total_actions" json:"total_actions"`
	Bans          int       `db:"bans" json:"bans"`
	Kicks         int       `db:"kicks" json:"kicks"`
	Mutes         int       `db:"mutes" json:"mutes"`
	Warns         int       `db:"warns" json:"warns"`
	MessageDeletes int       `db:"message_deletes" json:"message_deletes"`
}

// GetHourlyTrendData returns hourly counts for detailed trend analysis
func (r *AuditLogRepository) GetHourlyTrendData(ctx context.Context, serverID uuid.UUID, days int) ([]HourlyTrendPoint, error) {
	query := `
		SELECT 
			DATE_TRUNC('hour', created_at) as hour,
			COUNT(*) as total_actions
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= NOW() - INTERVAL '1 day' * $2
		GROUP BY DATE_TRUNC('hour', created_at)
		ORDER BY hour ASC
	`

	rows, err := r.db.QueryxContext(ctx, query, serverID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []HourlyTrendPoint
	for rows.Next() {
		var ht HourlyTrendPoint
		if err := rows.Scan(&ht.Hour, &ht.TotalActions); err != nil {
			return nil, err
		}
		results = append(results, ht)
	}

	return results, rows.Err()
}

// HourlyTrendPoint represents hourly moderation action counts
type HourlyTrendPoint struct {
	Hour         time.Time `db:"hour" json:"hour"`
	TotalActions int      `db:"total_actions" json:"total_actions"`
}

// GetModerationRatios returns the ratios of different moderation actions
func (r *AuditLogRepository) GetModerationRatios(ctx context.Context, serverID uuid.UUID, since time.Time) (*ModerationRatios, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_BAN') as bans,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_UNBAN') as unbans,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_TIMEOUT') as mutes,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_TIMEOUT_REMOVE') as unmutes,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_UPDATE') as warns,
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_KICK') as kicks,
			COUNT(*) FILTER (WHERE action_type = 'MESSAGE_DELETE') as message_deletes,
			COUNT(*) FILTER (WHERE action_type = 'MESSAGE_BULK_DELETE') as bulk_deletes
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= $2
	`

	var ratios ModerationRatios
	err := r.db.GetContext(ctx, &ratios, query, serverID, since)
	if err != nil {
		return nil, err
	}

	// Calculate percentages
	if ratios.Total > 0 {
		ratios.BanRatio = float64(ratios.Bans) / float64(ratios.Total) * 100
		ratios.MuteRatio = float64(ratios.Mutes) / float64(ratios.Total) * 100
		ratios.WarnRatio = float64(ratios.Warns) / float64(ratios.Total) * 100
		ratios.KickRatio = float64(ratios.Kicks) / float64(ratios.Total) * 100
	}

	return &ratios, nil
}

// ModerationRatios contains moderation action ratios
type ModerationRatios struct {
	Total          int     `db:"total" json:"total"`
	Bans           int     `db:"bans" json:"bans"`
	Unbans         int     `db:"unbans" json:"unbans"`
	Mutes          int     `db:"mutes" json:"mutes"`
	Unmutes        int     `db:"unmutes" json:"unmutes"`
	Warns          int     `db:"warns" json:"warns"`
	Kicks          int     `db:"kicks" json:"kicks"`
	MessageDeletes int     `db:"message_deletes" json:"message_deletes"`
	BulkDeletes    int     `db:"bulk_deletes" json:"bulk_deletes"`
	BanRatio       float64 `json:"ban_ratio"`
	MuteRatio      float64 `json:"mute_ratio"`
	WarnRatio      float64 `json:"warn_ratio"`
	KickRatio      float64 `json:"kick_ratio"`
}

// ExportForGDPR exports all audit logs for a user across all servers they own/admin
func (r *AuditLogRepository) ExportForGDPR(ctx context.Context, userID uuid.UUID) ([]AuditLog, error) {
	query := `
		SELECT DISTINCT al.id, al.server_id, al.actor_id, al.action_type, al.action_category, al.target_id, al.target_type, al.reason, al.changes, al.metadata, al.ip_address, al.created_at
		FROM audit_logs al
		JOIN servers s ON s.id = al.server_id
		WHERE al.actor_id = $1 OR al.target_id = $1 OR s.owner_id = $1
		ORDER BY al.created_at DESC
		LIMIT 10000
	`

	var logs []AuditLog
	err := r.db.SelectContext(ctx, &logs, query, userID)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

// ExportToCSV exports audit logs to CSV format
func (r *AuditLogRepository) ExportToCSV(ctx context.Context, serverID uuid.UUID, filter AuditLogFilter) (*strings.Builder, error) {
	logs, _, err := r.GetByServer(ctx, serverID, filter)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Write header
	header := []string{"ID", "ServerID", "ActorID", "ActionType", "ActionCategory", "TargetID", "TargetType", "Reason", "CreatedAt"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write data
	for _, log := range logs {
		targetIDStr := ""
		if log.TargetID != nil {
			targetIDStr = log.TargetID.String()
		}
		reason := ""
		if log.Reason != nil {
			reason = *log.Reason
		}
		targetType := ""
		if log.TargetType != nil {
			targetType = *log.TargetType
		}

		row := []string{
			log.ID.String(),
			log.ServerID.String(),
			log.ActorID.String(),
			log.ActionType,
			fmt.Sprintf("%d", log.ActionCategory),
			targetIDStr,
			targetType,
			reason,
			log.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return &sb, nil
}

// GetDashboardSummary returns a quick summary for the moderation dashboard
func (r *AuditLogRepository) GetDashboardSummary(ctx context.Context, serverID uuid.UUID, since time.Time) (*DashboardSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_actions,
			COUNT(DISTINCT actor_id) as unique_moderators,
			COUNT(DISTINCT target_id) FILTER (WHERE target_id IS NOT NULL) as unique_targets
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= $2
	`

	var summary DashboardSummary
	summary.ServerID = serverID
	err := r.db.GetContext(ctx, &summary, query, serverID, since)
	if err != nil {
		return nil, err
	}

	// Get top action type
	topQuery := `
		SELECT action_type, COUNT(*) as count
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= $2
		GROUP BY action_type
		ORDER BY count DESC
		LIMIT 1
	`

	var topAction struct {
		ActionType string `db:"action_type"`
		Count      int    `db:"count"`
	}
	err = r.db.GetContext(ctx, &topAction, topQuery, serverID, since)
	if err == nil {
		summary.TopAction = topAction.ActionType
		summary.TopActionCount = topAction.Count
	}

	// Get action breakdown
	breakdown, err := r.GetActionCounts(ctx, serverID, since)
	if err == nil {
		summary.ActionBreakdown = breakdown
	}

	return &summary, nil
}

// DashboardSummary contains summary statistics for the moderation dashboard
type DashboardSummary struct {
	ServerID         uuid.UUID    `db:"server_id" json:"server_id"`
	TotalActions     int          `db:"total_actions" json:"total_actions"`
	UniqueModerators int          `db:"unique_moderators" json:"unique_moderators"`
	UniqueTargets    int          `db:"unique_targets" json:"unique_targets"`
	TopAction        string       `json:"top_action"`
	TopActionCount   int          `json:"top_action_count"`
	ActionBreakdown  map[string]int `json:"action_breakdown"`
}

// GetAutoModStats returns auto-moderation statistics
func (r *AuditLogRepository) GetAutoModStats(ctx context.Context, serverID uuid.UUID, since time.Time) (*AutoModStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_triggers,
			COUNT(*) FILTER (WHERE action_type = 'AUTOMOD_BLOCK') as blocks,
			COUNT(*) FILTER (WHERE action_type = 'AUTOMOD_WARN') as warns,
			COUNT(*) FILTER (WHERE action_type = 'AUTOMOD_TIMEOUT') as timeouts,
			COUNT(*) FILTER (WHERE action_type = 'AUTOMOD_KICK') as kicks,
			COUNT(*) FILTER (WHERE action_type = 'AUTOMOD_BAN') as bans,
			COUNT(*) FILTER (WHERE action_type = 'AUTOMOD_MESSAGE_DELETE') as message_deletes,
			COUNT(*) FILTER (WHERE action_type = 'AUTOMOD_MENTION_SPAM') as mention_spam,
			COUNT(*) FILTER (WHERE action_type = 'AUTOMOD_WORD_FILTER') as word_filter,
			COUNT(*) FILTER (WHERE action_type = 'AUTOMOD_LINK_FILTER') as link_filter
		FROM audit_logs
		WHERE server_id = $1 AND created_at >= $2 AND action_category = 80
	`

	var stats AutoModStats
	err := r.db.GetContext(ctx, &stats, query, serverID, since)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// AutoModStats contains auto-moderation statistics
type AutoModStats struct {
	TotalTriggers   int `json:"total_triggers"`
	Blocks          int `json:"blocks"`
	Warns           int `json:"warns"`
	Timeouts        int `json:"timeouts"`
	Kicks           int `json:"kicks"`
	Bans            int `json:"bans"`
	MessageDeletes  int `json:"message_deletes"`
	MentionSpam     int `json:"mention_spam"`
	WordFilter      int `json:"word_filter"`
	LinkFilter      int `json:"link_filter"`
}

// UpdateDailyAnalytics updates the daily analytics summary table
func (r *AuditLogRepository) UpdateDailyAnalytics(ctx context.Context, serverID uuid.UUID, date time.Time) error {
	query := `
		INSERT INTO moderation_analytics_daily (
			server_id, date, total_actions,
			member_bans, member_unbans, member_kicks, member_mutes, member_unmutes, member_warns,
			message_deletes, message_bulk_deletes,
			channel_creates, channel_updates, channel_deletes,
			role_creates, role_updates, role_deletes,
			webhook_creates, webhook_updates, webhook_deletes,
			emoji_creates, emoji_updates, emoji_deletes,
			invite_creates, invite_deletes,
			automod_triggers, automod_actions
		)
		SELECT 
			$1, $2,
			COUNT(*),
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_BAN'),
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_UNBAN'),
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_KICK'),
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_TIMEOUT'),
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_TIMEOUT_REMOVE'),
			COUNT(*) FILTER (WHERE action_type = 'MEMBER_UPDATE'),
			COUNT(*) FILTER (WHERE action_type = 'MESSAGE_DELETE'),
			COUNT(*) FILTER (WHERE action_type = 'MESSAGE_BULK_DELETE'),
			COUNT(*) FILTER (WHERE action_type = 'CHANNEL_CREATE'),
			COUNT(*) FILTER (WHERE action_type = 'CHANNEL_UPDATE'),
			COUNT(*) FILTER (WHERE action_type = 'CHANNEL_DELETE'),
			COUNT(*) FILTER (WHERE action_type = 'ROLE_CREATE'),
			COUNT(*) FILTER (WHERE action_type = 'ROLE_UPDATE'),
			COUNT(*) FILTER (WHERE action_type = 'ROLE_DELETE'),
			COUNT(*) FILTER (WHERE action_type = 'WEBHOOK_CREATE'),
			COUNT(*) FILTER (WHERE action_type = 'WEBHOOK_UPDATE'),
			COUNT(*) FILTER (WHERE action_type = 'WEBHOOK_DELETE'),
			COUNT(*) FILTER (WHERE action_type = 'EMOJI_CREATE'),
			COUNT(*) FILTER (WHERE action_type = 'EMOJI_UPDATE'),
			COUNT(*) FILTER (WHERE action_type = 'EMOJI_DELETE'),
			COUNT(*) FILTER (WHERE action_type = 'INVITE_CREATE'),
			COUNT(*) FILTER (WHERE action_type = 'INVITE_DELETE'),
			COUNT(*) FILTER (WHERE action_category = 80),
			COUNT(*) FILTER (WHERE action_type LIKE 'AUTOMOD_%' AND action_type != 'AUTOMOD_FLAG')
		FROM audit_logs
		WHERE server_id = $1 AND DATE(created_at) = $2
		ON CONFLICT (server_id, date) DO UPDATE SET
			total_actions = EXCLUDED.total_actions,
			member_bans = EXCLUDED.member_bans,
			member_unbans = EXCLUDED.member_unbans,
			member_kicks = EXCLUDED.member_kicks,
			member_mutes = EXCLUDED.member_mutes,
			member_unmutes = EXCLUDED.member_unmutes,
			member_warns = EXCLUDED.member_warns,
			message_deletes = EXCLUDED.message_deletes,
			message_bulk_deletes = EXCLUDED.message_bulk_deletes,
			channel_creates = EXCLUDED.channel_creates,
			channel_updates = EXCLUDED.channel_updates,
			channel_deletes = EXCLUDED.channel_deletes,
			role_creates = EXCLUDED.role_creates,
			role_updates = EXCLUDED.role_updates,
			role_deletes = EXCLUDED.role_deletes,
			webhook_creates = EXCLUDED.webhook_creates,
			webhook_updates = EXCLUDED.webhook_updates,
			webhook_deletes = EXCLUDED.webhook_deletes,
			emoji_creates = EXCLUDED.emoji_creates,
			emoji_updates = EXCLUDED.emoji_updates,
			emoji_deletes = EXCLUDED.emoji_deletes,
			invite_creates = EXCLUDED.invite_creates,
						invite_deletes = EXCLUDED.invite_deletes,
			automod_triggers = EXCLUDED.automod_triggers,
			automod_actions = EXCLUDED.automod_actions
	`

	_, err := r.db.ExecContext(ctx, query, serverID, date)
	return err
}
