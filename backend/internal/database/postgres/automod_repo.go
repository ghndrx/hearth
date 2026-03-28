package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// AutoModRepository defines auto-mod data access operations
type AutoModRepository interface {
	// Rule CRUD
	CreateRule(ctx context.Context, rule *models.ModerationRule) error
	GetRuleByID(ctx context.Context, id uuid.UUID) (*models.ModerationRule, error)
	GetRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationRule, error)
	GetEnabledRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationRule, error)
	UpdateRule(ctx context.Context, rule *models.ModerationRule) error
	DeleteRule(ctx context.Context, id uuid.UUID) error

	// Alert CRUD
	CreateAlert(ctx context.Context, alert *models.AutoModAlert) error
	GetAlertByID(ctx context.Context, id uuid.UUID) (*models.AutoModAlert, error)
	GetAlertsByServerID(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.AutoModAlertSummary, error)
	GetAlertsByRuleID(ctx context.Context, ruleID uuid.UUID, limit, offset int) ([]*models.AutoModAlert, error)
	GetUnresolvedAlertsByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.AutoModAlert, error)
	ResolveAlert(ctx context.Context, alertID, resolvedBy uuid.UUID) error
	DeleteAlert(ctx context.Context, id uuid.UUID) error

	// Analytics
	IncrementRuleTrigger(ctx context.Context, ruleID uuid.UUID) error
	GetRuleStats(ctx context.Context, ruleID uuid.UUID) (*models.AutoModRuleTriggerCount, error)
}

type autoModRepo struct {
	db *sql.DB
}

// NewAutoModRepository creates a new auto-mod repository
func NewAutoModRepository(db *sql.DB) AutoModRepository {
	return &autoModRepo{db: db}
}

func (r *autoModRepo) CreateRule(ctx context.Context, rule *models.ModerationRule) error {
	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	query := `
		INSERT INTO automod_rules (id, server_id, name, event_type, trigger_type, trigger, actions, enabled, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		rule.ID,
		rule.ServerID,
		rule.Name,
		rule.EventType,
		rule.TriggerType,
		rule.Trigger,
		rule.Actions,
		rule.Enabled,
		rule.CreatedBy,
		rule.CreatedAt,
		rule.UpdatedAt,
	)
	return err
}

func (r *autoModRepo) GetRuleByID(ctx context.Context, id uuid.UUID) (*models.ModerationRule, error) {
	query := `
		SELECT id, server_id, name, event_type, trigger_type, trigger, actions, enabled, created_by, created_at, updated_at
		FROM automod_rules
		WHERE id = $1
	`

	rule := &models.ModerationRule{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rule.ID,
		&rule.ServerID,
		&rule.Name,
		&rule.EventType,
		&rule.TriggerType,
		&rule.Trigger,
		&rule.Actions,
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

func (r *autoModRepo) GetRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationRule, error) {
	query := `
		SELECT id, server_id, name, event_type, trigger_type, trigger, actions, enabled, created_by, created_at, updated_at
		FROM automod_rules
		WHERE server_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.ModerationRule
	for rows.Next() {
		rule := &models.ModerationRule{}
		err := rows.Scan(
			&rule.ID,
			&rule.ServerID,
			&rule.Name,
			&rule.EventType,
			&rule.TriggerType,
			&rule.Trigger,
			&rule.Actions,
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

func (r *autoModRepo) GetEnabledRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationRule, error) {
	query := `
		SELECT id, server_id, name, event_type, trigger_type, trigger, actions, enabled, created_by, created_at, updated_at
		FROM automod_rules
		WHERE server_id = $1 AND enabled = true
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.ModerationRule
	for rows.Next() {
		rule := &models.ModerationRule{}
		err := rows.Scan(
			&rule.ID,
			&rule.ServerID,
			&rule.Name,
			&rule.EventType,
			&rule.TriggerType,
			&rule.Trigger,
			&rule.Actions,
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

func (r *autoModRepo) UpdateRule(ctx context.Context, rule *models.ModerationRule) error {
	rule.UpdatedAt = time.Now()

	query := `
		UPDATE automod_rules
		SET name = $2, event_type = $3, trigger_type = $4, trigger = $5, actions = $6, enabled = $7, updated_at = $8
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		rule.ID,
		rule.Name,
		rule.EventType,
		rule.TriggerType,
		rule.Trigger,
		rule.Actions,
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

func (r *autoModRepo) DeleteRule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM automod_rules WHERE id = $1`
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

// Alert methods

func (r *autoModRepo) CreateAlert(ctx context.Context, alert *models.AutoModAlert) error {
	alert.ID = uuid.New()
	alert.CreatedAt = time.Now()
	alert.Resolved = false

	query := `
		INSERT INTO automod_alerts (id, rule_id, server_id, member_id, channel_id, message_id, content, action_taken, resolved, matched_keyword, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		alert.ID,
		alert.RuleID,
		alert.ServerID,
		alert.MemberID,
		alert.ChannelID,
		alert.MessageID,
		alert.Content,
		alert.ActionTaken,
		alert.Resolved,
		alert.MatchedKeyword,
		alert.CreatedAt,
	)
	return err
}

func (r *autoModRepo) GetAlertByID(ctx context.Context, id uuid.UUID) (*models.AutoModAlert, error) {
	query := `
		SELECT a.id, a.rule_id, a.server_id, a.member_id, a.channel_id, a.message_id, a.content, a.action_taken, a.resolved, a.resolved_by, a.resolved_at, a.matched_keyword, a.created_at,
			   r.id, r.server_id, r.name, r.event_type, r.trigger_type, r.trigger, r.actions, r.enabled, r.created_by, r.created_at, r.updated_at
		FROM automod_alerts a
		JOIN automod_rules r ON a.rule_id = r.id
		WHERE a.id = $1
	`

	alert := &models.AutoModAlert{}
	rule := &models.ModerationRule{}

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&alert.ID,
		&alert.RuleID,
		&alert.ServerID,
		&alert.MemberID,
		&alert.ChannelID,
		&alert.MessageID,
		&alert.Content,
		&alert.ActionTaken,
		&alert.Resolved,
		&alert.ResolvedBy,
		&alert.ResolvedAt,
		&alert.MatchedKeyword,
		&alert.CreatedAt,
		&rule.ID,
		&rule.ServerID,
		&rule.Name,
		&rule.EventType,
		&rule.TriggerType,
		&rule.Trigger,
		&rule.Actions,
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
	alert.Rule = rule
	return alert, nil
}

func (r *autoModRepo) GetAlertsByServerID(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.AutoModAlertSummary, error) {
	query := `
		SELECT a.id, a.rule_id, r.name, a.server_id, a.member_id, u.username, a.channel_id, a.message_id, a.content, a.action_taken, a.resolved, a.created_at
		FROM automod_alerts a
		JOIN automod_rules r ON a.rule_id = r.id
		JOIN users u ON a.member_id = u.id
		WHERE a.server_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.AutoModAlertSummary
	for rows.Next() {
		alert := &models.AutoModAlertSummary{}
		err := rows.Scan(
			&alert.ID,
			&alert.RuleID,
			&alert.RuleName,
			&alert.ServerID,
			&alert.MemberID,
			&alert.MemberName,
			&alert.ChannelID,
			&alert.MessageID,
			&alert.Content,
			&alert.ActionTaken,
			&alert.Resolved,
			&alert.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}

func (r *autoModRepo) GetAlertsByRuleID(ctx context.Context, ruleID uuid.UUID, limit, offset int) ([]*models.AutoModAlert, error) {
	query := `
		SELECT id, rule_id, server_id, member_id, channel_id, message_id, content, action_taken, resolved, resolved_by, resolved_at, matched_keyword, created_at
		FROM automod_alerts
		WHERE rule_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, ruleID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.AutoModAlert
	for rows.Next() {
		alert := &models.AutoModAlert{}
		err := rows.Scan(
			&alert.ID,
			&alert.RuleID,
			&alert.ServerID,
			&alert.MemberID,
			&alert.ChannelID,
			&alert.MessageID,
			&alert.Content,
			&alert.ActionTaken,
			&alert.Resolved,
			&alert.ResolvedBy,
			&alert.ResolvedAt,
			&alert.MatchedKeyword,
			&alert.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}

func (r *autoModRepo) GetUnresolvedAlertsByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.AutoModAlert, error) {
	query := `
		SELECT id, rule_id, server_id, member_id, channel_id, message_id, content, action_taken, resolved, resolved_by, resolved_at, matched_keyword, created_at
		FROM automod_alerts
		WHERE server_id = $1 AND resolved = false
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.AutoModAlert
	for rows.Next() {
		alert := &models.AutoModAlert{}
		err := rows.Scan(
			&alert.ID,
			&alert.RuleID,
			&alert.ServerID,
			&alert.MemberID,
			&alert.ChannelID,
			&alert.MessageID,
			&alert.Content,
			&alert.ActionTaken,
			&alert.Resolved,
			&alert.ResolvedBy,
			&alert.ResolvedAt,
			&alert.MatchedKeyword,
			&alert.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}

func (r *autoModRepo) ResolveAlert(ctx context.Context, alertID, resolvedBy uuid.UUID) error {
	query := `
		UPDATE automod_alerts
		SET resolved = true, resolved_by = $2, resolved_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, alertID, resolvedBy, time.Now())
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

func (r *autoModRepo) DeleteAlert(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM automod_alerts WHERE id = $1`
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

func (r *autoModRepo) IncrementRuleTrigger(ctx context.Context, ruleID uuid.UUID) error {
	query := `
		INSERT INTO automod_rule_stats (rule_id, trigger_count, block_count, flag_count)
		VALUES ($1, 1, 0, 0)
		ON CONFLICT (rule_id) DO UPDATE SET trigger_count = automod_rule_stats.trigger_count + 1
	`
	_, err := r.db.ExecContext(ctx, query, ruleID)
	return err
}

func (r *autoModRepo) GetRuleStats(ctx context.Context, ruleID uuid.UUID) (*models.AutoModRuleTriggerCount, error) {
	query := `
		SELECT rule_id, trigger_count, block_count, flag_count
		FROM automod_rule_stats
		WHERE rule_id = $1
	`

	stats := &models.AutoModRuleTriggerCount{}
	err := r.db.QueryRowContext(ctx, query, ruleID).Scan(
		&stats.RuleID,
		&stats.TriggerCount,
		&stats.BlockCount,
		&stats.FlagCount,
	)
	if err == sql.ErrNoRows {
		return &models.AutoModRuleTriggerCount{RuleID: ruleID}, nil
	}
	if err != nil {
		return nil, err
	}
	return stats, nil
}
