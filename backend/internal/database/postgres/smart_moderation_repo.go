package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// SmartModerationRepository defines data access operations for smart moderation
type SmartModerationRepository interface {
	// ModerationSettings CRUD
	CreateModerationSettings(ctx context.Context, settings *models.ModerationSettings) error
	GetModerationSettings(ctx context.Context, serverID uuid.UUID) (*models.ModerationSettings, error)
	UpdateModerationSettings(ctx context.Context, settings *models.ModerationSettings) error
	DeleteModerationSettings(ctx context.Context, serverID uuid.UUID) error

	// Keyword Rules CRUD
	CreateKeywordRule(ctx context.Context, rule *models.KeywordRule) error
	GetKeywordRuleByID(ctx context.Context, id uuid.UUID) (*models.KeywordRule, error)
	GetKeywordRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.KeywordRule, error)
	GetEnabledKeywordRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.KeywordRule, error)
	GetKeywordRulesByCategory(ctx context.Context, serverID uuid.UUID, category models.ToxicityCategory) ([]*models.KeywordRule, error)
	UpdateKeywordRule(ctx context.Context, rule *models.KeywordRule) error
	DeleteKeywordRule(ctx context.Context, id uuid.UUID) error

	// Moderation Logs
	CreateModerationLog(ctx context.Context, log *models.ModerationLog) error
	GetModerationLogByID(ctx context.Context, id uuid.UUID) (*models.ModerationLog, error)
	GetModerationLogsByServerID(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.ModerationLogSummary, error)
	GetModerationLogsByMemberID(ctx context.Context, serverID, memberID uuid.UUID, limit, offset int) ([]*models.ModerationLog, error)
	GetUnresolvedModerationLogs(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationLog, error)
	ResolveModerationLog(ctx context.Context, logID, resolvedBy uuid.UUID) error
	UpdateModerationLog(ctx context.Context, log *models.ModerationLog) error
	GetModerationLogsByDateRange(ctx context.Context, serverID uuid.UUID, start, end time.Time) ([]*models.ModerationLog, error)

	// User Violation Summary
	GetUserViolationSummary(ctx context.Context, serverID, userID uuid.UUID) (*models.UserViolationSummary, error)
	UpdateUserViolationSummary(ctx context.Context, summary *models.UserViolationSummary) error
	IncrementViolation(ctx context.Context, serverID, userID uuid.UUID, score float64, actionType models.ModerationActionType) error
	ResetUserViolations(ctx context.Context, serverID, userID uuid.UUID) error
	GetTopOffenders(ctx context.Context, serverID uuid.UUID, limit int) ([]*models.UserViolationSummary, error)

	// Dashboard Stats
	GetModerationStats(ctx context.Context, serverID uuid.UUID, start, end time.Time) (*models.ModerationDashboardStats, error)
	GetDailyModerationCounts(ctx context.Context, serverID uuid.UUID, start, end time.Time) ([]*models.DailyModerationCount, error)

	// Rate Limiting
	GetRateLimitWindow(ctx context.Context, serverID, moderatorID uuid.UUID, actionType models.ModerationActionType) (int, time.Time, error)
	IncrementRateLimit(ctx context.Context, serverID, moderatorID uuid.UUID, actionType models.ModerationActionType) error
}

type smartModerationRepo struct {
	db *sql.DB
}

// NewSmartModerationRepository creates a new smart moderation repository
func NewSmartModerationRepository(db *sql.DB) SmartModerationRepository {
	return &smartModerationRepo{db: db}
}

// ModerationSettings methods

func (r *smartModerationRepo) CreateModerationSettings(ctx context.Context, settings *models.ModerationSettings) error {
	settings.ID = uuid.New()
	settings.CreatedAt = time.Now()
	settings.UpdatedAt = time.Now()

	exemptRolesJSON, _ := json.Marshal(settings.ExemptRoles)
	exemptChannelsJSON, _ := json.Marshal(settings.ExemptChannels)

	query := `
		INSERT INTO moderation_settings (id, server_id, enabled, sensitivity_level, ml_classification_enabled, 
			auto_moderation_enabled, violation_threshold, warning_threshold, mute_threshold, temp_ban_threshold,
			temp_ban_duration, mute_duration, log_channel_id, exempt_roles, exempt_channels, audit_retention_days,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err := r.db.ExecContext(ctx, query,
		settings.ID,
		settings.ServerID,
		settings.Enabled,
		settings.SensitivityLevel,
		settings.MLClassificationEnabled,
		settings.AutoModerationEnabled,
		settings.ViolationThreshold,
		settings.WarningThreshold,
		settings.MuteThreshold,
		settings.TempBanThreshold,
		int(settings.TempBanDuration.Seconds()),
		int(settings.MuteDuration.Seconds()),
		settings.LogChannelID,
		exemptRolesJSON,
		exemptChannelsJSON,
		settings.AuditRetentionDays,
		settings.CreatedAt,
		settings.UpdatedAt,
	)
	return err
}

func (r *smartModerationRepo) GetModerationSettings(ctx context.Context, serverID uuid.UUID) (*models.ModerationSettings, error) {
	query := `
		SELECT id, server_id, enabled, sensitivity_level, ml_classification_enabled, auto_moderation_enabled,
			violation_threshold, warning_threshold, mute_threshold, temp_ban_threshold, temp_ban_duration,
			mute_duration, log_channel_id, exempt_roles, exempt_channels, audit_retention_days, created_at, updated_at
		FROM moderation_settings
		WHERE server_id = $1
	`

	settings := &models.ModerationSettings{}
	var exemptRolesJSON, exemptChannelsJSON []byte
	var tempBanDuration, muteDuration int

	err := r.db.QueryRowContext(ctx, query, serverID).Scan(
		&settings.ID,
		&settings.ServerID,
		&settings.Enabled,
		&settings.SensitivityLevel,
		&settings.MLClassificationEnabled,
		&settings.AutoModerationEnabled,
		&settings.ViolationThreshold,
		&settings.WarningThreshold,
		&settings.MuteThreshold,
		&settings.TempBanThreshold,
		&tempBanDuration,
		&muteDuration,
		&settings.LogChannelID,
		&exemptRolesJSON,
		&exemptChannelsJSON,
		&settings.AuditRetentionDays,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	settings.TempBanDuration = time.Duration(tempBanDuration) * time.Second
	settings.MuteDuration = time.Duration(muteDuration) * time.Second
	json.Unmarshal(exemptRolesJSON, &settings.ExemptRoles)
	json.Unmarshal(exemptChannelsJSON, &settings.ExemptChannels)

	return settings, nil
}

func (r *smartModerationRepo) UpdateModerationSettings(ctx context.Context, settings *models.ModerationSettings) error {
	settings.UpdatedAt = time.Now()

	exemptRolesJSON, _ := json.Marshal(settings.ExemptRoles)
	exemptChannelsJSON, _ := json.Marshal(settings.ExemptChannels)

	query := `
		UPDATE moderation_settings
		SET enabled = $2, sensitivity_level = $3, ml_classification_enabled = $4, auto_moderation_enabled = $5,
			violation_threshold = $6, warning_threshold = $7, mute_threshold = $8, temp_ban_threshold = $9,
			temp_ban_duration = $10, mute_duration = $11, log_channel_id = $12, exempt_roles = $13,
			exempt_channels = $14, audit_retention_days = $15, updated_at = $16
		WHERE server_id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		settings.ServerID,
		settings.Enabled,
		settings.SensitivityLevel,
		settings.MLClassificationEnabled,
		settings.AutoModerationEnabled,
		settings.ViolationThreshold,
		settings.WarningThreshold,
		settings.MuteThreshold,
		settings.TempBanThreshold,
		int(settings.TempBanDuration.Seconds()),
		int(settings.MuteDuration.Seconds()),
		settings.LogChannelID,
		exemptRolesJSON,
		exemptChannelsJSON,
		settings.AuditRetentionDays,
		settings.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *smartModerationRepo) DeleteModerationSettings(ctx context.Context, serverID uuid.UUID) error {
	query := `DELETE FROM moderation_settings WHERE server_id = $1`
	result, err := r.db.ExecContext(ctx, query, serverID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// KeywordRules methods

func (r *smartModerationRepo) CreateKeywordRule(ctx context.Context, rule *models.KeywordRule) error {
	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	query := `
		INSERT INTO moderation_keyword_rules (id, server_id, name, pattern, is_regex, sensitivity, category, action, weight, enabled, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.ExecContext(ctx, query,
		rule.ID,
		rule.ServerID,
		rule.Name,
		rule.Pattern,
		rule.IsRegex,
		rule.Sensitivity,
		rule.Category,
		rule.Action,
		rule.Weight,
		rule.Enabled,
		rule.CreatedBy,
		rule.CreatedAt,
		rule.UpdatedAt,
	)
	return err
}

func (r *smartModerationRepo) GetKeywordRuleByID(ctx context.Context, id uuid.UUID) (*models.KeywordRule, error) {
	query := `
		SELECT id, server_id, name, pattern, is_regex, sensitivity, category, action, weight, enabled, created_by, created_at, updated_at
		FROM moderation_keyword_rules
		WHERE id = $1
	`

	rule := &models.KeywordRule{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rule.ID,
		&rule.ServerID,
		&rule.Name,
		&rule.Pattern,
		&rule.IsRegex,
		&rule.Sensitivity,
		&rule.Category,
		&rule.Action,
		&rule.Weight,
		&rule.Enabled,
		&rule.CreatedBy,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *smartModerationRepo) GetKeywordRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.KeywordRule, error) {
	query := `
		SELECT id, server_id, name, pattern, is_regex, sensitivity, category, action, weight, enabled, created_by, created_at, updated_at
		FROM moderation_keyword_rules
		WHERE server_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.KeywordRule
	for rows.Next() {
		rule := &models.KeywordRule{}
		err := rows.Scan(
			&rule.ID,
			&rule.ServerID,
			&rule.Name,
			&rule.Pattern,
			&rule.IsRegex,
			&rule.Sensitivity,
			&rule.Category,
			&rule.Action,
			&rule.Weight,
			&rule.Enabled,
			&rule.CreatedBy,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *smartModerationRepo) GetEnabledKeywordRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.KeywordRule, error) {
	query := `
		SELECT id, server_id, name, pattern, is_regex, sensitivity, category, action, weight, enabled, created_by, created_at, updated_at
		FROM moderation_keyword_rules
		WHERE server_id = $1 AND enabled = true
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.KeywordRule
	for rows.Next() {
		rule := &models.KeywordRule{}
		err := rows.Scan(
			&rule.ID,
			&rule.ServerID,
			&rule.Name,
			&rule.Pattern,
			&rule.IsRegex,
			&rule.Sensitivity,
			&rule.Category,
			&rule.Action,
			&rule.Weight,
			&rule.Enabled,
			&rule.CreatedBy,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *smartModerationRepo) GetKeywordRulesByCategory(ctx context.Context, serverID uuid.UUID, category models.ToxicityCategory) ([]*models.KeywordRule, error) {
	query := `
		SELECT id, server_id, name, pattern, is_regex, sensitivity, category, action, weight, enabled, created_by, created_at, updated_at
		FROM moderation_keyword_rules
		WHERE server_id = $1 AND category = $2 AND enabled = true
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.KeywordRule
	for rows.Next() {
		rule := &models.KeywordRule{}
		err := rows.Scan(
			&rule.ID,
			&rule.ServerID,
			&rule.Name,
			&rule.Pattern,
			&rule.IsRegex,
			&rule.Sensitivity,
			&rule.Category,
			&rule.Action,
			&rule.Weight,
			&rule.Enabled,
			&rule.CreatedBy,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *smartModerationRepo) UpdateKeywordRule(ctx context.Context, rule *models.KeywordRule) error {
	rule.UpdatedAt = time.Now()

	query := `
		UPDATE moderation_keyword_rules
		SET name = $2, pattern = $3, is_regex = $4, sensitivity = $5, category = $6, action = $7, weight = $8, enabled = $9, updated_at = $10
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		rule.ID,
		rule.Name,
		rule.Pattern,
		rule.IsRegex,
		rule.Sensitivity,
		rule.Category,
		rule.Action,
		rule.Weight,
		rule.Enabled,
		rule.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *smartModerationRepo) DeleteKeywordRule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM moderation_keyword_rules WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ModerationLogs methods

func (r *smartModerationRepo) CreateModerationLog(ctx context.Context, log *models.ModerationLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()

	violationScoreJSON, _ := json.Marshal(log.ViolationScore)

	query := `
		INSERT INTO moderation_logs (id, server_id, member_id, moderator_id, action_type, violation_score, reason, channel_id, message_id, rule_id, rule_name, duration, resolved, resolved_by, resolved_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	var durationSecs *int
	if log.Duration != nil {
		secs := int(log.Duration.Seconds())
		durationSecs = &secs
	}

	_, err := r.db.ExecContext(ctx, query,
		log.ID,
		log.ServerID,
		log.MemberID,
		log.ModeratorID,
		log.ActionType,
		violationScoreJSON,
		log.Reason,
		log.ChannelID,
		log.MessageID,
		log.RuleID,
		log.RuleName,
		durationSecs,
		log.Resolved,
		log.ResolvedBy,
		log.ResolvedAt,
		log.CreatedAt,
	)
	return err
}

func (r *smartModerationRepo) GetModerationLogByID(ctx context.Context, id uuid.UUID) (*models.ModerationLog, error) {
	query := `
		SELECT id, server_id, member_id, moderator_id, action_type, violation_score, reason, channel_id, message_id, rule_id, rule_name, duration, resolved, resolved_by, resolved_at, created_at
		FROM moderation_logs
		WHERE id = $1
	`

	log := &models.ModerationLog{}
	var violationScoreJSON []byte
	var durationSecs *int

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID,
		&log.ServerID,
		&log.MemberID,
		&log.ModeratorID,
		&log.ActionType,
		&violationScoreJSON,
		&log.Reason,
		&log.ChannelID,
		&log.MessageID,
		&log.RuleID,
		&log.RuleName,
		&durationSecs,
		&log.Resolved,
		&log.ResolvedBy,
		&log.ResolvedAt,
		&log.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if violationScoreJSON != nil {
		json.Unmarshal(violationScoreJSON, &log.ViolationScore)
	}
	if durationSecs != nil {
		dur := time.Duration(*durationSecs) * time.Second
		log.Duration = &dur
	}

	return log, nil
}

func (r *smartModerationRepo) GetModerationLogsByServerID(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.ModerationLogSummary, error) {
	query := `
		SELECT l.id, l.server_id, l.member_id, u.username, l.moderator_id, l.action_type, l.reason, l.channel_id, l.message_id, l.resolved, l.created_at
		FROM moderation_logs l
		JOIN users u ON l.member_id = u.id
		WHERE l.server_id = $1
		ORDER BY l.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.ModerationLogSummary
	for rows.Next() {
		log := &models.ModerationLogSummary{}
		err := rows.Scan(
			&log.ID,
			&log.ServerID,
			&log.MemberID,
			&log.MemberName,
			&log.ModeratorID,
			&log.ActionType,
			&log.Reason,
			&log.ChannelID,
			&log.MessageID,
			&log.Resolved,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func (r *smartModerationRepo) GetModerationLogsByMemberID(ctx context.Context, serverID, memberID uuid.UUID, limit, offset int) ([]*models.ModerationLog, error) {
	query := `
		SELECT id, server_id, member_id, moderator_id, action_type, violation_score, reason, channel_id, message_id, rule_id, rule_name, duration, resolved, resolved_by, resolved_at, created_at
		FROM moderation_logs
		WHERE server_id = $1 AND member_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, memberID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.ModerationLog
	for rows.Next() {
		log := &models.ModerationLog{}
		var violationScoreJSON []byte
		var durationSecs *int

		err := rows.Scan(
			&log.ID,
			&log.ServerID,
			&log.MemberID,
			&log.ModeratorID,
			&log.ActionType,
			&violationScoreJSON,
			&log.Reason,
			&log.ChannelID,
			&log.MessageID,
			&log.RuleID,
			&log.RuleName,
			&durationSecs,
			&log.Resolved,
			&log.ResolvedBy,
			&log.ResolvedAt,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if violationScoreJSON != nil {
			json.Unmarshal(violationScoreJSON, &log.ViolationScore)
		}
		if durationSecs != nil {
			dur := time.Duration(*durationSecs) * time.Second
			log.Duration = &dur
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func (r *smartModerationRepo) GetUnresolvedModerationLogs(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationLog, error) {
	query := `
		SELECT id, server_id, member_id, moderator_id, action_type, violation_score, reason, channel_id, message_id, rule_id, rule_name, duration, resolved, resolved_by, resolved_at, created_at
		FROM moderation_logs
		WHERE server_id = $1 AND resolved = false
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.ModerationLog
	for rows.Next() {
		log := &models.ModerationLog{}
		var violationScoreJSON []byte
		var durationSecs *int

		err := rows.Scan(
			&log.ID,
			&log.ServerID,
			&log.MemberID,
			&log.ModeratorID,
			&log.ActionType,
			&violationScoreJSON,
			&log.Reason,
			&log.ChannelID,
			&log.MessageID,
			&log.RuleID,
			&log.RuleName,
			&durationSecs,
			&log.Resolved,
			&log.ResolvedBy,
			&log.ResolvedAt,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if violationScoreJSON != nil {
			json.Unmarshal(violationScoreJSON, &log.ViolationScore)
		}
		if durationSecs != nil {
			dur := time.Duration(*durationSecs) * time.Second
			log.Duration = &dur
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func (r *smartModerationRepo) ResolveModerationLog(ctx context.Context, logID, resolvedBy uuid.UUID) error {
	query := `
		UPDATE moderation_logs
		SET resolved = true, resolved_by = $2, resolved_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, logID, resolvedBy, time.Now())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *smartModerationRepo) UpdateModerationLog(ctx context.Context, log *models.ModerationLog) error {
	violationScoreJSON, _ := json.Marshal(log.ViolationScore)

	var durationSecs *int
	if log.Duration != nil {
		secs := int(log.Duration.Seconds())
		durationSecs = &secs
	}

	query := `
		UPDATE moderation_logs
		SET action_type = $2, violation_score = $3, reason = $4, resolved = $5, resolved_by = $6, resolved_at = $7, duration = $8
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		log.ID,
		log.ActionType,
		violationScoreJSON,
		log.Reason,
		log.Resolved,
		log.ResolvedBy,
		log.ResolvedAt,
		durationSecs,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *smartModerationRepo) GetModerationLogsByDateRange(ctx context.Context, serverID uuid.UUID, start, end time.Time) ([]*models.ModerationLog, error) {
	query := `
		SELECT id, server_id, member_id, moderator_id, action_type, violation_score, reason, channel_id, message_id, rule_id, rule_name, duration, resolved, resolved_by, resolved_at, created_at
		FROM moderation_logs
		WHERE server_id = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.ModerationLog
	for rows.Next() {
		log := &models.ModerationLog{}
		var violationScoreJSON []byte
		var durationSecs *int

		err := rows.Scan(
			&log.ID,
			&log.ServerID,
			&log.MemberID,
			&log.ModeratorID,
			&log.ActionType,
			&violationScoreJSON,
			&log.Reason,
			&log.ChannelID,
			&log.MessageID,
			&log.RuleID,
			&log.RuleName,
			&durationSecs,
			&log.Resolved,
			&log.ResolvedBy,
			&log.ResolvedAt,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if violationScoreJSON != nil {
			json.Unmarshal(violationScoreJSON, &log.ViolationScore)
		}
		if durationSecs != nil {
			dur := time.Duration(*durationSecs) * time.Second
			log.Duration = &dur
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// UserViolationSummary methods

func (r *smartModerationRepo) GetUserViolationSummary(ctx context.Context, serverID, userID uuid.UUID) (*models.UserViolationSummary, error) {
	query := `
		SELECT user_id, server_id, violation_count, last_violation_at, warn_count, mute_count, temp_ban_count, total_score
		FROM user_violation_summary
		WHERE server_id = $1 AND user_id = $2
	`

	summary := &models.UserViolationSummary{}
	err := r.db.QueryRowContext(ctx, query, serverID, userID).Scan(
		&summary.UserID,
		&summary.ServerID,
		&summary.ViolationCount,
		&summary.LastViolationAt,
		&summary.WarnCount,
		&summary.MuteCount,
		&summary.TempBanCount,
		&summary.TotalScore,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func (r *smartModerationRepo) UpdateUserViolationSummary(ctx context.Context, summary *models.UserViolationSummary) error {
	query := `
		UPDATE user_violation_summary
		SET violation_count = $3, last_violation_at = $4, warn_count = $5, mute_count = $6, temp_ban_count = $7, total_score = $8
		WHERE server_id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query,
		summary.ServerID,
		summary.UserID,
		summary.ViolationCount,
		summary.LastViolationAt,
		summary.WarnCount,
		summary.MuteCount,
		summary.TempBanCount,
		summary.TotalScore,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *smartModerationRepo) IncrementViolation(ctx context.Context, serverID, userID uuid.UUID, score float64, actionType models.ModerationActionType) error {
	query := `
		INSERT INTO user_violation_summary (user_id, server_id, violation_count, last_violation_at, warn_count, mute_count, temp_ban_count, total_score)
		VALUES ($1, $2, 1, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, server_id) DO UPDATE SET
			violation_count = user_violation_summary.violation_count + 1,
			last_violation_at = $3,
			total_score = user_violation_summary.total_score + $7,
			warn_count = CASE WHEN $4 THEN user_violation_summary.warn_count + 1 ELSE user_violation_summary.warn_count END,
			mute_count = CASE WHEN $5 THEN user_violation_summary.mute_count + 1 ELSE user_violation_summary.mute_count END,
			temp_ban_count = CASE WHEN $6 THEN user_violation_summary.temp_ban_count + 1 ELSE user_violation_summary.temp_ban_count END
	`

	now := time.Now()
	isWarn := actionType == models.ModActionWarn
	isMute := actionType == models.ModActionMute
	isTempBan := actionType == models.ModActionTempBan

	_, err := r.db.ExecContext(ctx, query, userID, serverID, now, isWarn, isMute, isTempBan, score)
	return err
}

func (r *smartModerationRepo) ResetUserViolations(ctx context.Context, serverID, userID uuid.UUID) error {
	query := `
		UPDATE user_violation_summary
		SET violation_count = 0, last_violation_at = NULL, warn_count = 0, mute_count = 0, temp_ban_count = 0, total_score = 0
		WHERE server_id = $1 AND user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, serverID, userID)
	return err
}

func (r *smartModerationRepo) GetTopOffenders(ctx context.Context, serverID uuid.UUID, limit int) ([]*models.UserViolationSummary, error) {
	query := `
		SELECT user_id, server_id, violation_count, last_violation_at, warn_count, mute_count, temp_ban_count, total_score
		FROM user_violation_summary
		WHERE server_id = $1 AND violation_count > 0
		ORDER BY violation_count DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*models.UserViolationSummary
	for rows.Next() {
		summary := &models.UserViolationSummary{}
		err := rows.Scan(
			&summary.UserID,
			&summary.ServerID,
			&summary.ViolationCount,
			&summary.LastViolationAt,
			&summary.WarnCount,
			&summary.MuteCount,
			&summary.TempBanCount,
			&summary.TotalScore,
		)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// Dashboard Stats methods

func (r *smartModerationRepo) GetModerationStats(ctx context.Context, serverID uuid.UUID, start, end time.Time) (*models.ModerationDashboardStats, error) {
	stats := &models.ModerationDashboardStats{
		TopCategories: make(map[models.ToxicityCategory]int),
	}

	countQuery := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN action_type = 1 THEN 1 ELSE 0 END) as warnings,
			SUM(CASE WHEN action_type = 2 THEN 1 ELSE 0 END) as mutes,
			SUM(CASE WHEN action_type = 4 THEN 1 ELSE 0 END) as tempbans
		FROM moderation_logs
		WHERE server_id = $1 AND created_at >= $2 AND created_at <= $3
	`

	err := r.db.QueryRowContext(ctx, countQuery, serverID, start, end).Scan(
		&stats.TotalViolations,
		&stats.TotalWarnings,
		&stats.TotalMutes,
		&stats.TotalTempBans,
	)
	if err != nil {
		return nil, err
	}

	topOffenders, err := r.GetTopOffenders(ctx, serverID, 10)
	if err != nil {
		return nil, err
	}
	stats.TopOffenders = topOffenders

	recentLogs, err := r.GetModerationLogsByServerID(ctx, serverID, 10, 0)
	if err != nil {
		return nil, err
	}
	stats.RecentActions = recentLogs

	trendData, err := r.GetDailyModerationCounts(ctx, serverID, start, end)
	if err != nil {
		return nil, err
	}
	stats.TrendData = trendData

	return stats, nil
}

func (r *smartModerationRepo) GetDailyModerationCounts(ctx context.Context, serverID uuid.UUID, start, end time.Time) ([]*models.DailyModerationCount, error) {
	query := `
		SELECT 
			DATE(created_at) as date,
			SUM(CASE WHEN action_type = 1 THEN 1 ELSE 0 END) as warning_count,
			SUM(CASE WHEN action_type = 2 THEN 1 ELSE 0 END) as mute_count,
			SUM(CASE WHEN action_type = 4 THEN 1 ELSE 0 END) as tempban_count,
			SUM(CASE WHEN action_type = 3 THEN 1 ELSE 0 END) as delete_count,
			SUM(CASE WHEN action_type = 6 THEN 1 ELSE 0 END) as flag_count
		FROM moderation_logs
		WHERE server_id = $1 AND created_at >= $2 AND created_at <= $3
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []*models.DailyModerationCount
	for rows.Next() {
		count := &models.DailyModerationCount{}
		err := rows.Scan(
			&count.Date,
			&count.WarningCount,
			&count.MuteCount,
			&count.TempBanCount,
			&count.DeleteCount,
			&count.FlagCount,
		)
		if err != nil {
			return nil, err
		}
		counts = append(counts, count)
	}
	return counts, nil
}

// Rate Limiting methods

func (r *smartModerationRepo) GetRateLimitWindow(ctx context.Context, serverID, moderatorID uuid.UUID, actionType models.ModerationActionType) (int, time.Time, error) {
	query := `
		SELECT action_count, window_start
		FROM moderation_action_rate_limits
		WHERE server_id = $1 AND moderator_id = $2 AND action_type = $3
		  AND window_start > $4
		ORDER BY window_start DESC
		LIMIT 1
	`

	windowStart := time.Now().Add(-1 * time.Hour)
	var count int
	var start time.Time

	err := r.db.QueryRowContext(ctx, query, serverID, moderatorID, actionType, windowStart).Scan(&count, &start)
	if err == sql.ErrNoRows {
		return 0, time.Time{}, nil
	}
	if err != nil {
		return 0, time.Time{}, err
	}
	return count, start, nil
}

func (r *smartModerationRepo) IncrementRateLimit(ctx context.Context, serverID, moderatorID uuid.UUID, actionType models.ModerationActionType) error {
	query := `
		INSERT INTO moderation_action_rate_limits (id, server_id, moderator_id, action_type, action_count, window_start)
		VALUES ($1, $2, $3, $4, 1, $5)
		ON CONFLICT (server_id, moderator_id, action_type, window_start) DO UPDATE SET
			action_count = moderation_action_rate_limits.action_count + 1
	`

	_, err := r.db.ExecContext(ctx, query, uuid.New(), serverID, moderatorID, actionType, time.Now())
	return err
}
