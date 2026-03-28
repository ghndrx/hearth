package services

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// AutoModRepository defines auto-mod data access operations
type AutoModRepository interface {
	CreateRule(ctx context.Context, rule *models.ModerationRule) error
	GetRuleByID(ctx context.Context, id uuid.UUID) (*models.ModerationRule, error)
	GetRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationRule, error)
	GetEnabledRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationRule, error)
	UpdateRule(ctx context.Context, rule *models.ModerationRule) error
	DeleteRule(ctx context.Context, id uuid.UUID) error
	CreateAlert(ctx context.Context, alert *models.AutoModAlert) error
	GetAlertByID(ctx context.Context, id uuid.UUID) (*models.AutoModAlert, error)
	GetAlertsByServerID(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.AutoModAlertSummary, error)
	GetAlertsByRuleID(ctx context.Context, ruleID uuid.UUID, limit, offset int) ([]*models.AutoModAlert, error)
	GetUnresolvedAlertsByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.AutoModAlert, error)
	ResolveAlert(ctx context.Context, alertID, resolvedBy uuid.UUID) error
	DeleteAlert(ctx context.Context, id uuid.UUID) error
	IncrementRuleTrigger(ctx context.Context, ruleID uuid.UUID) error
	GetRuleStats(ctx context.Context, ruleID uuid.UUID) (*models.AutoModRuleTriggerCount, error)
}

// AutoModMatchResult represents the result of matching content against rules
type AutoModMatchResult struct {
	Matched        bool
	Rule           *models.ModerationRule
	Action         models.AutoModAction
	MatchedKeyword *string
	MatchedPattern *string
	AlertID        *uuid.UUID
}

// AutoModService handles auto-moderation logic
type AutoModService struct {
	repo AutoModRepository
}

// NewAutoModService creates a new auto-mod service
func NewAutoModService(repo AutoModRepository) *AutoModService {
	return &AutoModService{repo: repo}
}

// CreateRule creates a new auto-mod rule
func (s *AutoModService) CreateRule(ctx context.Context, serverID, createdBy uuid.UUID, req *models.CreateAutoModRuleRequest) (*models.ModerationRule, error) {
	// Validate trigger type has appropriate trigger data
	if err := s.validateTrigger(req.TriggerType, &req.Trigger); err != nil {
		return nil, err
	}

	// Validate actions
	if len(req.Actions) == 0 {
		return nil, ErrInvalidInput
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := &models.ModerationRule{
		ServerID:    serverID,
		Name:        req.Name,
		EventType:   req.EventType,
		TriggerType: req.TriggerType,
		Trigger:     req.Trigger,
		Actions:     req.Actions,
		Enabled:     enabled,
		CreatedBy:   createdBy,
	}

	if err := s.repo.CreateRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// GetRule gets an auto-mod rule by ID
func (s *AutoModService) GetRule(ctx context.Context, id uuid.UUID) (*models.ModerationRule, error) {
	return s.repo.GetRuleByID(ctx, id)
}

// GetServerRules gets all auto-mod rules for a server
func (s *AutoModService) GetServerRules(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationRule, error) {
	return s.repo.GetRulesByServerID(ctx, serverID)
}

// UpdateRule updates an existing auto-mod rule
func (s *AutoModService) UpdateRule(ctx context.Context, id uuid.UUID, req *models.UpdateAutoModRuleRequest) (*models.ModerationRule, error) {
	rule, err := s.repo.GetRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrServerNotFound
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.EventType != nil {
		rule.EventType = *req.EventType
	}
	if req.TriggerType != nil {
		rule.TriggerType = *req.TriggerType
	}
	if req.Trigger != nil {
		if err := s.validateTrigger(rule.TriggerType, req.Trigger); err != nil {
			return nil, err
		}
		rule.Trigger = *req.Trigger
	}
	if req.Actions != nil {
		if len(*req.Actions) == 0 {
			return nil, ErrInvalidInput
		}
		rule.Actions = *req.Actions
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := s.repo.UpdateRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// DeleteRule deletes an auto-mod rule
func (s *AutoModService) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRule(ctx, id)
}

// GetServerAlerts gets alerts for a server with pagination
func (s *AutoModService) GetServerAlerts(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.AutoModAlertSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.GetAlertsByServerID(ctx, serverID, limit, offset)
}

// GetUnresolvedAlerts gets all unresolved alerts for a server
func (s *AutoModService) GetUnresolvedAlerts(ctx context.Context, serverID uuid.UUID) ([]*models.AutoModAlert, error) {
	return s.repo.GetUnresolvedAlertsByServerID(ctx, serverID)
}

// ResolveAlert marks an alert as resolved
func (s *AutoModService) ResolveAlert(ctx context.Context, alertID, resolvedBy uuid.UUID) error {
	return s.repo.ResolveAlert(ctx, alertID, resolvedBy)
}

// TestContent tests content against enabled rules and returns match info
func (s *AutoModService) TestContent(ctx context.Context, req *models.AutoModTestRequest) (*models.AutoModTestResult, error) {
	rules, err := s.repo.GetEnabledRulesByServerID(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if rule.EventType != models.EventTypeMessageSend {
			continue
		}

		result := s.matchRuleContent(rule, req.Content)
		if result.Matched {
			alertID := result.AlertID
			return &models.AutoModTestResult{
				Matched:  true,
				RuleID:   &rule.ID,
				RuleName: &rule.Name,
				Actions:  rule.Actions,
				Keyword:  result.MatchedKeyword,
				Pattern:  result.MatchedPattern,
				AlertID:  alertID,
			}, nil
		}
	}

	return &models.AutoModTestResult{Matched: false}, nil
}

// ScanMessage scans a message against enabled rules and returns match results
// Returns the match result and whether the message should be blocked
func (s *AutoModService) ScanMessage(ctx context.Context, serverID, memberID, channelID, messageID uuid.UUID, content string) (*AutoModMatchResult, error) {
	rules, err := s.repo.GetEnabledRulesByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if rule.EventType != models.EventTypeMessageSend {
			continue
		}

		result := s.matchRuleContent(rule, content)
		if result.Matched {
			// Record the alert
			alert := &models.AutoModAlert{
				RuleID:         rule.ID,
				ServerID:       serverID,
				MemberID:       memberID,
				ChannelID:      &channelID,
				MessageID:      &messageID,
				Content:        content,
				ActionTaken:    result.Action.Type,
				MatchedKeyword: result.MatchedKeyword,
			}

			if err := s.repo.CreateAlert(ctx, alert); err != nil {
				// Log but don't fail the scan
			} else {
				result.AlertID = &alert.ID
			}

			// Increment trigger count
			_ = s.repo.IncrementRuleTrigger(ctx, rule.ID)

			return result, nil
		}
	}

	return nil, nil
}

// ShouldBlockMessage determines if a message should be blocked based on automod rules
func (s *AutoModService) ShouldBlockMessage(ctx context.Context, serverID, memberID, channelID, messageID uuid.UUID, content string) (bool, *AutoModMatchResult, error) {
	result, err := s.ScanMessage(ctx, serverID, memberID, channelID, messageID, content)
	if err != nil {
		return false, nil, err
	}
	if result == nil {
		return false, nil, nil
	}

	shouldBlock := false
	for _, action := range result.Rule.Actions {
		if action.Type == models.ActionBlockMessage {
			shouldBlock = true
			break
		}
	}

	return shouldBlock, result, nil
}

func (s *AutoModService) matchRuleContent(rule *models.ModerationRule, content string) *AutoModMatchResult {
	switch rule.TriggerType {
	case models.TriggerKeyword:
		return s.matchKeywordTrigger(rule, content)
	case models.TriggerCustomRegex:
		return s.matchRegexTrigger(rule, content)
	case models.TriggerMentionSpam:
		return s.matchMentionSpamTrigger(rule, content)
	case models.TriggerSpam:
		return s.matchSpamTrigger(rule, content)
	default:
		return &AutoModMatchResult{Matched: false}
	}
}

func (s *AutoModService) matchKeywordTrigger(rule *models.ModerationRule, content string) *AutoModMatchResult {
	lowerContent := strings.ToLower(content)

	for _, keyword := range rule.Trigger.Keywords {
		lowerKeyword := strings.ToLower(keyword)

		// Check if keyword is in whitelist
		whitelisted := false
		for _, wl := range rule.Trigger.Whitelist {
			if strings.Contains(lowerContent, strings.ToLower(wl)) {
				whitelisted = true
				break
			}
		}
		if whitelisted {
			continue
		}

		// Check for keyword match
		if strings.Contains(lowerContent, lowerKeyword) {
			return &AutoModMatchResult{
				Matched:        true,
				Rule:           rule,
				Action:         rule.Actions[0],
				MatchedKeyword: &keyword,
			}
		}
	}

	return &AutoModMatchResult{Matched: false}
}

func (s *AutoModService) matchRegexTrigger(rule *models.ModerationRule, content string) *AutoModMatchResult {
	for _, pattern := range rule.Trigger.RegexPatterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			continue
		}

		if re.MatchString(content) {
			return &AutoModMatchResult{
				Matched:        true,
				Rule:           rule,
				Action:         rule.Actions[0],
				MatchedPattern: &pattern,
			}
		}
	}

	return &AutoModMatchResult{Matched: false}
}

func (s *AutoModService) matchMentionSpamTrigger(rule *models.ModerationRule, content string) *AutoModMatchResult {
	mentionCount := strings.Count(content, "@")

	if rule.Trigger.MentionLimit > 0 && mentionCount > rule.Trigger.MentionLimit {
		return &AutoModMatchResult{
			Matched: true,
			Rule:    rule,
			Action:  rule.Actions[0],
		}
	}

	return &AutoModMatchResult{Matched: false}
}

func (s *AutoModService) matchSpamTrigger(rule *models.ModerationRule, content string) *AutoModMatchResult {
	// Simple spam detection: very short messages with links
	// or repeated characters
	if utf8.RuneCountInString(content) < 10 && strings.Contains(content, "http") {
		return &AutoModMatchResult{
			Matched: true,
			Rule:    rule,
			Action:  rule.Actions[0],
		}
	}

	// Check for repeated characters (e.g., "aaaaaaaaa")
	repeated := regexp.MustCompile(`(.)\1{5,}`)
	if repeated.MatchString(content) {
		return &AutoModMatchResult{
			Matched: true,
			Rule:    rule,
			Action:  rule.Actions[0],
		}
	}

	return &AutoModMatchResult{Matched: false}
}

func (s *AutoModService) validateTrigger(triggerType models.AutoModTriggerType, trigger *models.AutoModTrigger) error {
	switch triggerType {
	case models.TriggerKeyword:
		if len(trigger.Keywords) == 0 {
			return ErrInvalidInput
		}
	case models.TriggerCustomRegex:
		if len(trigger.RegexPatterns) == 0 {
			return ErrInvalidInput
		}
		// Validate regex patterns
		for _, pattern := range trigger.RegexPatterns {
			if _, err := regexp.Compile(pattern); err != nil {
				return ErrInvalidInput
			}
		}
	case models.TriggerMentionSpam:
		if trigger.MentionLimit <= 0 {
			return ErrInvalidInput
		}
	}
	return nil
}

// GetRuleStats gets trigger statistics for a rule
func (s *AutoModService) GetRuleStats(ctx context.Context, ruleID uuid.UUID) (*models.AutoModRuleTriggerCount, error) {
	return s.repo.GetRuleStats(ctx, ruleID)
}
