package services

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// SmartModerationRepository defines the repository interface for smart moderation
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

// ModerationRateLimitConfig holds rate limit configuration for moderation actions
type ModerationRateLimitConfig struct {
	WarnLimit   int // Max warns per window
	MuteLimit   int // Max mutes per window
	BanLimit    int // Max bans per window
	GeneralLimit int // Max general moderation actions per window
	Window      time.Duration
}

// DefaultModerationRateLimitConfig provides sensible defaults
var DefaultModerationRateLimitConfig = ModerationRateLimitConfig{
	WarnLimit:    50,
	MuteLimit:    20,
	BanLimit:    10,
	GeneralLimit: 100,
	Window:       1 * time.Hour,
}

// SmartModerationService handles AI-powered smart moderation
type SmartModerationService struct {
	repo   SmartModerationRepository
	config ModerationRateLimitConfig
}

// NewSmartModerationService creates a new smart moderation service
func NewSmartModerationService(repo SmartModerationRepository) *SmartModerationService {
	return &SmartModerationService{
		repo:   repo,
		config: DefaultModerationRateLimitConfig,
	}
}

// NewSmartModerationServiceWithConfig creates a new smart moderation service with custom config
func NewSmartModerationServiceWithConfig(repo SmartModerationRepository, config ModerationRateLimitConfig) *SmartModerationService {
	return &SmartModerationService{
		repo:   repo,
		config: config,
	}
}

// GetOrCreateSettings gets or creates moderation settings for a server
func (s *SmartModerationService) GetOrCreateSettings(ctx context.Context, serverID uuid.UUID) (*models.ModerationSettings, error) {
	settings, err := s.repo.GetModerationSettings(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if settings != nil {
		return settings, nil
	}

	// Create default settings
	defaultSettings := &models.ModerationSettings{
		ServerID:                 serverID,
		Enabled:                  true,
		SensitivityLevel:        models.SensitivityMedium,
		MLClassificationEnabled: false, // Placeholder for ML
		AutoModerationEnabled:   true,
		ViolationThreshold:      50,
		WarningThreshold:        3,
		MuteThreshold:           5,
		TempBanThreshold:        10,
		TempBanDuration:         24 * time.Hour,
		MuteDuration:            1 * time.Hour,
		AuditRetentionDays:     90,
		ExemptRoles:             []uuid.UUID{},
		ExemptChannels:          []uuid.UUID{},
	}

	if err := s.repo.CreateModerationSettings(ctx, defaultSettings); err != nil {
		return nil, err
	}

	return defaultSettings, nil
}

// UpdateSettings updates moderation settings for a server
func (s *SmartModerationService) UpdateSettings(ctx context.Context, serverID uuid.UUID, req *models.UpdateModerationSettingsRequest) (*models.ModerationSettings, error) {
	settings, err := s.GetOrCreateSettings(ctx, serverID)
	if err != nil {
		return nil, err
	}

	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.SensitivityLevel != nil {
		settings.SensitivityLevel = *req.SensitivityLevel
	}
	if req.MLClassificationEnabled != nil {
		settings.MLClassificationEnabled = *req.MLClassificationEnabled
	}
	if req.AutoModerationEnabled != nil {
		settings.AutoModerationEnabled = *req.AutoModerationEnabled
	}
	if req.ViolationThreshold != nil {
		settings.ViolationThreshold = *req.ViolationThreshold
	}
	if req.WarningThreshold != nil {
		settings.WarningThreshold = *req.WarningThreshold
	}
	if req.MuteThreshold != nil {
		settings.MuteThreshold = *req.MuteThreshold
	}
	if req.TempBanThreshold != nil {
		settings.TempBanThreshold = *req.TempBanThreshold
	}
	if req.TempBanDuration != nil {
		settings.TempBanDuration = *req.TempBanDuration
	}
	if req.MuteDuration != nil {
		settings.MuteDuration = *req.MuteDuration
	}
	if req.LogChannelID != nil {
		settings.LogChannelID = req.LogChannelID
	}
	if req.ExemptRoles != nil {
		settings.ExemptRoles = *req.ExemptRoles
	}
	if req.ExemptChannels != nil {
		settings.ExemptChannels = *req.ExemptChannels
	}
	if req.AuditRetentionDays != nil {
		settings.AuditRetentionDays = *req.AuditRetentionDays
	}

	if err := s.repo.UpdateModerationSettings(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// CreateKeywordRule creates a new keyword/regex rule
func (s *SmartModerationService) CreateKeywordRule(ctx context.Context, serverID, createdBy uuid.UUID, req *models.CreateKeywordRuleRequest) (*models.KeywordRule, error) {
	weight := 0.5
	if req.Weight != nil {
		weight = *req.Weight
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := &models.KeywordRule{
		ServerID:    serverID,
		Name:        req.Name,
		Pattern:     req.Pattern,
		IsRegex:     req.IsRegex,
		Sensitivity: req.Sensitivity,
		Category:    req.Category,
		Action:      req.Action,
		Weight:      weight,
		Enabled:     enabled,
		CreatedBy:   createdBy,
	}

	if err := s.repo.CreateKeywordRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// UpdateKeywordRule updates an existing keyword rule
func (s *SmartModerationService) UpdateKeywordRule(ctx context.Context, ruleID uuid.UUID, req *models.UpdateKeywordRuleRequest) (*models.KeywordRule, error) {
	rule, err := s.repo.GetKeywordRuleByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrModerationRuleNotFound
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Pattern != nil {
		rule.Pattern = *req.Pattern
	}
	if req.IsRegex != nil {
		rule.IsRegex = *req.IsRegex
	}
	if req.Sensitivity != nil {
		rule.Sensitivity = *req.Sensitivity
	}
	if req.Category != nil {
		rule.Category = *req.Category
	}
	if req.Action != nil {
		rule.Action = *req.Action
	}
	if req.Weight != nil {
		rule.Weight = *req.Weight
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := s.repo.UpdateKeywordRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// DeleteKeywordRule deletes a keyword rule
func (s *SmartModerationService) DeleteKeywordRule(ctx context.Context, ruleID uuid.UUID) error {
	err := s.repo.DeleteKeywordRule(ctx, ruleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrModerationRuleNotFound
		}
		return err
	}
	return nil
}

// GetKeywordRules gets all keyword rules for a server
func (s *SmartModerationService) GetKeywordRules(ctx context.Context, serverID uuid.UUID) ([]*models.KeywordRule, error) {
	return s.repo.GetKeywordRulesByServerID(ctx, serverID)
}

// AnalyzeContent analyzes content for violations and returns detailed results
func (s *SmartModerationService) AnalyzeContent(ctx context.Context, req *models.AnalyzeContentRequest) (*models.AnalyzeContentResult, error) {
	settings, err := s.GetOrCreateSettings(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}

	if !settings.Enabled || !settings.AutoModerationEnabled {
		return &models.AnalyzeContentResult{
			Violations:  []models.ViolationDetail{},
			TotalScore:  0,
			ShouldBlock: false,
			Actions:     []models.ModerationActionType{},
		}, nil
	}

	// Check if channel is exempt
	if req.ChannelID != nil {
		for _, exemptChannel := range settings.ExemptChannels {
			if exemptChannel == *req.ChannelID {
				return &models.AnalyzeContentResult{
					Violations:  []models.ViolationDetail{},
					TotalScore:  0,
					ShouldBlock: false,
					Actions:     []models.ModerationActionType{},
				}, nil
			}
		}
	}

	rules, err := s.repo.GetEnabledKeywordRulesByServerID(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}

	result := &models.AnalyzeContentResult{
		Violations: []models.ViolationDetail{},
		Actions:    []models.ModerationActionType{},
	}

	for _, rule := range rules {
		// Check sensitivity match
		if rule.Sensitivity > int(settings.SensitivityLevel) {
			continue
		}

		violation := s.matchRule(rule, req.Content)
		if violation != nil {
			violation.RuleID = rule.ID
			violation.RuleName = rule.Name
			violation.Category = rule.Category
			violation.Score = rule.Weight * 100 * float64(rule.Sensitivity)
			violation.Action = rule.Action
			result.Violations = append(result.Violations, *violation)
			result.TotalScore += violation.Score
		}
	}

	// Apply ML classification placeholder (always returns 0 for now)
	if settings.MLClassificationEnabled {
		mlScore := s.calculateMLScorePlaceholder(req.Content)
		result.TotalScore += mlScore
	}

	result.ShouldBlock = result.TotalScore >= float64(settings.ViolationThreshold)

	// Determine actions based on violations
	actionSet := make(map[models.ModerationActionType]bool)
	for _, v := range result.Violations {
		actionSet[v.Action] = true
	}
	for action := range actionSet {
		result.Actions = append(result.Actions, action)
	}

	return result, nil
}

// ScanMessage scans a message and returns whether it should be blocked and the analysis result
func (s *SmartModerationService) ScanMessage(ctx context.Context, serverID, memberID, channelID, messageID uuid.UUID, content string) (*models.AnalyzeContentResult, error) {
	req := &models.AnalyzeContentRequest{
		ServerID:  serverID,
		Content:   content,
		ChannelID: &channelID,
		MemberID:  &memberID,
	}
	return s.AnalyzeContent(ctx, req)
}

// RecordViolation records a moderation action and updates user violation summary
func (s *SmartModerationService) RecordViolation(ctx context.Context, serverID, memberID, moderatorID uuid.UUID, actionType models.ModerationActionType, reason string, score float64, channelID, messageID, ruleID *uuid.UUID, ruleName *string, duration *time.Duration) (*models.ModerationLog, error) {
	log := &models.ModerationLog{
		ServerID:     serverID,
		MemberID:     memberID,
		ModeratorID: &moderatorID,
		ActionType:  actionType,
		Reason:      reason,
		ChannelID:   channelID,
		MessageID:   messageID,
		RuleID:      ruleID,
		RuleName:    ruleName,
		Duration:    duration,
		Resolved:    false,
	}

	if score > 0 {
		log.ViolationScore = &models.ToxicityScore{
			Overall: score,
			Categories: make(map[models.ToxicityCategory]float64),
		}
	}

	if err := s.repo.CreateModerationLog(ctx, log); err != nil {
		return nil, err
	}

	// Update user violation summary
	if err := s.repo.IncrementViolation(ctx, serverID, memberID, score, actionType); err != nil {
		// Log but don't fail
	}

	return log, nil
}

// GetAutoModActions determines automatic actions based on violation count
func (s *SmartModerationService) GetAutoModActions(ctx context.Context, serverID, memberID uuid.UUID) ([]models.ModerationActionType, error) {
	settings, err := s.GetOrCreateSettings(ctx, serverID)
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.GetUserViolationSummary(ctx, serverID, memberID)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return []models.ModerationActionType{}, nil
	}

	var actions []models.ModerationActionType

	// Check thresholds
	if settings.WarningThreshold > 0 && summary.ViolationCount >= settings.WarningThreshold && summary.ViolationCount%settings.WarningThreshold == 0 {
		actions = append(actions, models.ModActionWarn)
	}

	if settings.MuteThreshold > 0 && summary.ViolationCount >= settings.MuteThreshold && summary.ViolationCount%settings.MuteThreshold == 0 {
		actions = append(actions, models.ModActionMute)
	}

	if settings.TempBanThreshold > 0 && summary.ViolationCount >= settings.TempBanThreshold && summary.ViolationCount%settings.TempBanThreshold == 0 {
		actions = append(actions, models.ModActionTempBan)
	}

	return actions, nil
}

// TakeModerationAction applies a moderation action with rate limiting
func (s *SmartModerationService) TakeModerationAction(ctx context.Context, serverID, moderatorID uuid.UUID, req *models.ModerationActionRequest) (*models.ModerationLog, error) {
	// Check rate limit
	allowed, err := s.CheckRateLimit(ctx, serverID, moderatorID, req.Action)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrModerationRateLimited
	}

	// Increment rate limit counter
	if err := s.repo.IncrementRateLimit(ctx, serverID, moderatorID, req.Action); err != nil {
		// Log but don't fail
	}

	var duration *time.Duration
	if req.Duration != nil {
		duration = req.Duration
	}

	// Record the action
	log, err := s.RecordViolation(ctx, serverID, req.MemberID, moderatorID, req.Action, req.Reason, 0, req.ChannelID, req.MessageID, nil, nil, duration)
	if err != nil {
		return nil, err
	}

	return log, nil
}

// CheckRateLimit checks if a moderator can perform an action
func (s *SmartModerationService) CheckRateLimit(ctx context.Context, serverID, moderatorID uuid.UUID, actionType models.ModerationActionType) (bool, error) {
	count, windowStart, err := s.repo.GetRateLimitWindow(ctx, serverID, moderatorID, actionType)
	if err != nil {
		return false, err
	}

	// If window has expired, reset
	if time.Since(windowStart) > s.config.Window {
		return true, nil
	}

	limit := s.getLimitForAction(actionType)
	return count < limit, nil
}

func (s *SmartModerationService) getLimitForAction(actionType models.ModerationActionType) int {
	switch actionType {
	case models.ModActionWarn:
		return s.config.WarnLimit
	case models.ModActionMute:
		return s.config.MuteLimit
	case models.ModActionTempBan:
		return s.config.BanLimit
	default:
		return s.config.GeneralLimit
	}
}

// GetModerationLogs gets moderation logs for a server with pagination
func (s *SmartModerationService) GetModerationLogs(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.ModerationLogSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.GetModerationLogsByServerID(ctx, serverID, limit, offset)
}

// GetMemberModerationHistory gets moderation history for a specific member
func (s *SmartModerationService) GetMemberModerationHistory(ctx context.Context, serverID, memberID uuid.UUID, limit, offset int) ([]*models.ModerationLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.GetModerationLogsByMemberID(ctx, serverID, memberID, limit, offset)
}

// ResolveModerationLog marks a moderation log as resolved
func (s *SmartModerationService) ResolveModerationLog(ctx context.Context, logID, resolvedBy uuid.UUID) error {
	return s.repo.ResolveModerationLog(ctx, logID, resolvedBy)
}

// GetDashboardStats gets moderation dashboard statistics
func (s *SmartModerationService) GetDashboardStats(ctx context.Context, serverID uuid.UUID, days int) (*models.ModerationDashboardStats, error) {
	if days <= 0 {
		days = 7
	}

	end := time.Now()
	start := end.AddDate(0, 0, -days)

	return s.repo.GetModerationStats(ctx, serverID, start, end)
}

// ResetMemberViolations resets all violations for a member
func (s *SmartModerationService) ResetMemberViolations(ctx context.Context, serverID, memberID uuid.UUID) error {
	return s.repo.ResetUserViolations(ctx, serverID, memberID)
}

// GetUserViolationSummary gets the violation summary for a user
func (s *SmartModerationService) GetUserViolationSummary(ctx context.Context, serverID, userID uuid.UUID) (*models.UserViolationSummary, error) {
	return s.repo.GetUserViolationSummary(ctx, serverID, userID)
}

// matchRule checks if content matches a keyword rule
func (s *SmartModerationService) matchRule(rule *models.KeywordRule, content string) *models.ViolationDetail {
	lowerContent := strings.ToLower(content)

	if rule.IsRegex {
		re, err := regexp.Compile("(?i)" + rule.Pattern)
		if err != nil {
			return nil
		}
		matches := re.FindStringSubmatch(content)
		if len(matches) > 0 {
			return &models.ViolationDetail{
				MatchedText: matches[0],
			}
		}
	} else {
		lowerPattern := strings.ToLower(rule.Pattern)
		if strings.Contains(lowerContent, lowerPattern) {
			return &models.ViolationDetail{
				MatchedText: rule.Pattern,
			}
		}
	}

	return nil
}

// calculateMLScorePlaceholder is a placeholder for future ML model integration
// Currently returns 0, but could be extended to call an external ML service
func (s *SmartModerationService) calculateMLScorePlaceholder(content string) float64 {
	// Placeholder: simple heuristic based on content characteristics
	// In a real implementation, this would call an ML model API
	
	score := 0.0
	
	// Check for excessive caps (shouting)
	capsRatio := s.calculateCapsRatio(content)
	if capsRatio > 0.7 && len(content) > 10 {
		score += 10.0
	}
	
	// Check for excessive punctuation
	if s.hasExcessivePunctuation(content) {
		score += 5.0
	}
	
	return score
}

func (s *SmartModerationService) calculateCapsRatio(content string) float64 {
	var upper, total int
	for _, r := range content {
		if r >= 'A' && r <= 'Z' {
			upper++
		}
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			total++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(upper) / float64(total)
}

func (s *SmartModerationService) hasExcessivePunctuation(content string) bool {
	var punctCount, total int
	for _, r := range content {
		if r == '!' || r == '?' || r == '.' {
			punctCount++
		}
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			continue
		}
		total++
	}
	if total == 0 {
		return false
	}
	return float64(punctCount)/float64(total) > 0.3
}
