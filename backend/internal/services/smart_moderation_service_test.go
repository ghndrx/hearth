package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// MockSmartModerationRepository is a mock implementation for testing
type MockSmartModerationRepository struct {
	settings          *models.ModerationSettings
	rules             []*models.KeywordRule
	logs              []*models.ModerationLog
	violationSummary  *models.UserViolationSummary
	rateLimitCount    int
	rateLimitWindow   time.Time
}

func NewMockSmartModerationRepository() *MockSmartModerationRepository {
	return &MockSmartModerationRepository{
		rules:    make([]*models.KeywordRule, 0),
		logs:     make([]*models.ModerationLog, 0),
	}
}

func (m *MockSmartModerationRepository) CreateModerationSettings(ctx context.Context, settings *models.ModerationSettings) error {
	settings.ID = uuid.New()
	settings.CreatedAt = time.Now()
	settings.UpdatedAt = time.Now()
	m.settings = settings
	return nil
}

func (m *MockSmartModerationRepository) GetModerationSettings(ctx context.Context, serverID uuid.UUID) (*models.ModerationSettings, error) {
	return m.settings, nil
}

func (m *MockSmartModerationRepository) UpdateModerationSettings(ctx context.Context, settings *models.ModerationSettings) error {
	settings.UpdatedAt = time.Now()
	m.settings = settings
	return nil
}

func (m *MockSmartModerationRepository) DeleteModerationSettings(ctx context.Context, serverID uuid.UUID) error {
	m.settings = nil
	return nil
}

func (m *MockSmartModerationRepository) CreateKeywordRule(ctx context.Context, rule *models.KeywordRule) error {
	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	m.rules = append(m.rules, rule)
	return nil
}

func (m *MockSmartModerationRepository) GetKeywordRuleByID(ctx context.Context, id uuid.UUID) (*models.KeywordRule, error) {
	for _, r := range m.rules {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}

func (m *MockSmartModerationRepository) GetKeywordRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.KeywordRule, error) {
	return m.rules, nil
}

func (m *MockSmartModerationRepository) GetEnabledKeywordRulesByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.KeywordRule, error) {
	var enabled []*models.KeywordRule
	for _, r := range m.rules {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	return enabled, nil
}

func (m *MockSmartModerationRepository) GetKeywordRulesByCategory(ctx context.Context, serverID uuid.UUID, category models.ToxicityCategory) ([]*models.KeywordRule, error) {
	var result []*models.KeywordRule
	for _, r := range m.rules {
		if r.Category == category && r.Enabled {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MockSmartModerationRepository) UpdateKeywordRule(ctx context.Context, rule *models.KeywordRule) error {
	rule.UpdatedAt = time.Now()
	for i, r := range m.rules {
		if r.ID == rule.ID {
			m.rules[i] = rule
			break
		}
	}
	return nil
}

func (m *MockSmartModerationRepository) DeleteKeywordRule(ctx context.Context, id uuid.UUID) error {
	for i, r := range m.rules {
		if r.ID == id {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockSmartModerationRepository) CreateModerationLog(ctx context.Context, log *models.ModerationLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	m.logs = append(m.logs, log)
	return nil
}

func (m *MockSmartModerationRepository) GetModerationLogByID(ctx context.Context, id uuid.UUID) (*models.ModerationLog, error) {
	for _, l := range m.logs {
		if l.ID == id {
			return l, nil
		}
	}
	return nil, nil
}

func (m *MockSmartModerationRepository) GetModerationLogsByServerID(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.ModerationLogSummary, error) {
	var result []*models.ModerationLogSummary
	for _, l := range m.logs {
		if l.ServerID == serverID {
			result = append(result, &models.ModerationLogSummary{
				ID:         l.ID,
				ServerID:   l.ServerID,
				MemberID:   l.MemberID,
				ActionType: l.ActionType,
				Reason:     l.Reason,
				Resolved:   l.Resolved,
				CreatedAt:  l.CreatedAt,
			})
		}
	}
	return result, nil
}

func (m *MockSmartModerationRepository) GetModerationLogsByMemberID(ctx context.Context, serverID, memberID uuid.UUID, limit, offset int) ([]*models.ModerationLog, error) {
	var result []*models.ModerationLog
	for _, l := range m.logs {
		if l.ServerID == serverID && l.MemberID == memberID {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *MockSmartModerationRepository) GetUnresolvedModerationLogs(ctx context.Context, serverID uuid.UUID) ([]*models.ModerationLog, error) {
	var result []*models.ModerationLog
	for _, l := range m.logs {
		if l.ServerID == serverID && !l.Resolved {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *MockSmartModerationRepository) ResolveModerationLog(ctx context.Context, logID, resolvedBy uuid.UUID) error {
	for _, l := range m.logs {
		if l.ID == logID {
			l.Resolved = true
			l.ResolvedBy = &resolvedBy
			now := time.Now()
			l.ResolvedAt = &now
			break
		}
	}
	return nil
}

func (m *MockSmartModerationRepository) UpdateModerationLog(ctx context.Context, log *models.ModerationLog) error {
	for i, l := range m.logs {
		if l.ID == log.ID {
			m.logs[i] = log
			break
		}
	}
	return nil
}

func (m *MockSmartModerationRepository) GetModerationLogsByDateRange(ctx context.Context, serverID uuid.UUID, start, end time.Time) ([]*models.ModerationLog, error) {
	var result []*models.ModerationLog
	for _, l := range m.logs {
		if l.ServerID == serverID && l.CreatedAt.After(start) && l.CreatedAt.Before(end) {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *MockSmartModerationRepository) GetUserViolationSummary(ctx context.Context, serverID, userID uuid.UUID) (*models.UserViolationSummary, error) {
	return m.violationSummary, nil
}

func (m *MockSmartModerationRepository) UpdateUserViolationSummary(ctx context.Context, summary *models.UserViolationSummary) error {
	m.violationSummary = summary
	return nil
}

func (m *MockSmartModerationRepository) IncrementViolation(ctx context.Context, serverID, userID uuid.UUID, score float64, actionType models.ModerationActionType) error {
	if m.violationSummary == nil {
		m.violationSummary = &models.UserViolationSummary{
			UserID:   userID,
			ServerID: serverID,
		}
	}
	m.violationSummary.ViolationCount++
	m.violationSummary.TotalScore += score
	now := time.Now()
	m.violationSummary.LastViolationAt = &now
	return nil
}

func (m *MockSmartModerationRepository) ResetUserViolations(ctx context.Context, serverID, userID uuid.UUID) error {
	if m.violationSummary != nil {
		m.violationSummary.ViolationCount = 0
		m.violationSummary.WarnCount = 0
		m.violationSummary.MuteCount = 0
		m.violationSummary.TempBanCount = 0
		m.violationSummary.TotalScore = 0
		m.violationSummary.LastViolationAt = nil
	}
	return nil
}

func (m *MockSmartModerationRepository) GetTopOffenders(ctx context.Context, serverID uuid.UUID, limit int) ([]*models.UserViolationSummary, error) {
	if m.violationSummary != nil {
		return []*models.UserViolationSummary{m.violationSummary}, nil
	}
	return nil, nil
}

func (m *MockSmartModerationRepository) GetModerationStats(ctx context.Context, serverID uuid.UUID, start, end time.Time) (*models.ModerationDashboardStats, error) {
	return &models.ModerationDashboardStats{}, nil
}

func (m *MockSmartModerationRepository) GetDailyModerationCounts(ctx context.Context, serverID uuid.UUID, start, end time.Time) ([]*models.DailyModerationCount, error) {
	return nil, nil
}

func (m *MockSmartModerationRepository) GetRateLimitWindow(ctx context.Context, serverID, moderatorID uuid.UUID, actionType models.ModerationActionType) (int, time.Time, error) {
	return m.rateLimitCount, m.rateLimitWindow, nil
}

func (m *MockSmartModerationRepository) IncrementRateLimit(ctx context.Context, serverID, moderatorID uuid.UUID, actionType models.ModerationActionType) error {
	m.rateLimitCount++
	return nil
}

func TestSmartModerationService_GetOrCreateSettings(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()

	// Test creating settings
	settings, err := svc.GetOrCreateSettings(ctx, serverID)
	if err != nil {
		t.Fatalf("GetOrCreateSettings failed: %v", err)
	}

	if settings.ServerID != serverID {
		t.Errorf("Expected ServerID %v, got %v", serverID, settings.ServerID)
	}

	if !settings.Enabled {
		t.Error("Expected settings to be enabled by default")
	}

	if settings.SensitivityLevel != models.SensitivityMedium {
		t.Errorf("Expected SensitivityMedium, got %v", settings.SensitivityLevel)
	}
}

func TestSmartModerationService_CreateKeywordRule(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	req := &models.CreateKeywordRuleRequest{
		Name:        "Test Rule",
		Pattern:     "badword",
		IsRegex:     false,
		Sensitivity: 2,
		Category:    models.ToxicityProfanity,
		Action:      models.ModActionWarn,
	}

	rule, err := svc.CreateKeywordRule(ctx, serverID, userID, req)
	if err != nil {
		t.Fatalf("CreateKeywordRule failed: %v", err)
	}

	if rule.Name != "Test Rule" {
		t.Errorf("Expected Name 'Test Rule', got %v", rule.Name)
	}

	if rule.Pattern != "badword" {
		t.Errorf("Expected Pattern 'badword', got %v", rule.Pattern)
	}

	if rule.ServerID != serverID {
		t.Errorf("Expected ServerID %v, got %v", serverID, rule.ServerID)
	}

	if rule.ID == uuid.Nil {
		t.Error("Expected rule ID to be set")
	}
}

func TestSmartModerationService_AnalyzeContent(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create a keyword rule
	_, err := svc.CreateKeywordRule(ctx, serverID, userID, &models.CreateKeywordRuleRequest{
		Name:        "Profanity Filter",
		Pattern:     "badword",
		IsRegex:     false,
		Sensitivity: 2,
		Category:    models.ToxicityProfanity,
		Action:      models.ModActionWarn,
	})
	if err != nil {
		t.Fatalf("CreateKeywordRule failed: %v", err)
	}

	// Test analyzing content with matching keyword
	result, err := svc.AnalyzeContent(ctx, &models.AnalyzeContentRequest{
		ServerID: serverID,
		Content:  "This contains a badword in it",
	})
	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	if len(result.Violations) == 0 {
		t.Error("Expected at least one violation")
	}

	if result.TotalScore <= 0 {
		t.Error("Expected positive total score")
	}
}

func TestSmartModerationService_AnalyzeContent_NoMatch(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()

	// Create a keyword rule
	_, err := svc.CreateKeywordRule(ctx, serverID, uuid.New(), &models.CreateKeywordRuleRequest{
		Name:        "Profanity Filter",
		Pattern:     "badword",
		IsRegex:     false,
		Sensitivity: 2,
		Category:    models.ToxicityProfanity,
		Action:      models.ModActionWarn,
	})
	if err != nil {
		t.Fatalf("CreateKeywordRule failed: %v", err)
	}

	// Test analyzing clean content
	result, err := svc.AnalyzeContent(ctx, &models.AnalyzeContentRequest{
		ServerID: serverID,
		Content:  "This is clean content",
	})
	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	if len(result.Violations) != 0 {
		t.Errorf("Expected no violations, got %d", len(result.Violations))
	}
}

func TestSmartModerationService_AnalyzeContent_Regex(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create a regex rule
	_, err := svc.CreateKeywordRule(ctx, serverID, userID, &models.CreateKeywordRuleRequest{
		Name:        "Phone Number Filter",
		Pattern:     `\d{3}-\d{3}-\d{4}`,
		IsRegex:     true,
		Sensitivity: 2,
		Category:    models.ToxicityPersonalInfo,
		Action:      models.ModActionFlagForReview,
	})
	if err != nil {
		t.Fatalf("CreateKeywordRule failed: %v", err)
	}

	// Test analyzing content with phone number
	result, err := svc.AnalyzeContent(ctx, &models.AnalyzeContentRequest{
		ServerID: serverID,
		Content:  "Call me at 555-123-4567",
	})
	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	if len(result.Violations) == 0 {
		t.Error("Expected violation for phone number pattern")
	}
}

func TestSmartModerationService_AnalyzeContent_Disabled(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create settings with moderation disabled
	_, err := svc.UpdateSettings(ctx, serverID, &models.UpdateModerationSettingsRequest{
		Enabled: func() *bool { v := false; return &v }(),
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Create a keyword rule
	_, err = svc.CreateKeywordRule(ctx, serverID, userID, &models.CreateKeywordRuleRequest{
		Name:        "Profanity Filter",
		Pattern:     "badword",
		IsRegex:     false,
		Sensitivity: 2,
		Category:    models.ToxicityProfanity,
		Action:      models.ModActionWarn,
	})
	if err != nil {
		t.Fatalf("CreateKeywordRule failed: %v", err)
	}

	// Test analyzing content - should return no violations when disabled
	result, err := svc.AnalyzeContent(ctx, &models.AnalyzeContentRequest{
		ServerID: serverID,
		Content:  "This contains a badword in it",
	})
	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	if len(result.Violations) != 0 {
		t.Errorf("Expected no violations when disabled, got %d", len(result.Violations))
	}
}

func TestSmartModerationService_UpdateSettings(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()

	// Create default settings first
	_, err := svc.GetOrCreateSettings(ctx, serverID)
	if err != nil {
		t.Fatalf("GetOrCreateSettings failed: %v", err)
	}

	// Update settings
	newThreshold := 75
	enabled := false
	updated, err := svc.UpdateSettings(ctx, serverID, &models.UpdateModerationSettingsRequest{
		ViolationThreshold: &newThreshold,
		Enabled:           &enabled,
	})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	if updated.ViolationThreshold != 75 {
		t.Errorf("Expected ViolationThreshold 75, got %d", updated.ViolationThreshold)
	}

	if updated.Enabled {
		t.Error("Expected Enabled to be false")
	}
}

func TestSmartModerationService_CheckRateLimit(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	moderatorID := uuid.New()

	// Should allow when no rate limit exists
	allowed, err := svc.CheckRateLimit(ctx, serverID, moderatorID, models.ModActionWarn)
	if err != nil {
		t.Fatalf("CheckRateLimit failed: %v", err)
	}
	if !allowed {
		t.Error("Expected rate limit to allow first action")
	}

	// Increment rate limit
	err = repo.IncrementRateLimit(ctx, serverID, moderatorID, models.ModActionWarn)
	if err != nil {
		t.Fatalf("IncrementRateLimit failed: %v", err)
	}

	// Should still be allowed (under limit)
	allowed, err = svc.CheckRateLimit(ctx, serverID, moderatorID, models.ModActionWarn)
	if err != nil {
		t.Fatalf("CheckRateLimit failed: %v", err)
	}
	if !allowed {
		t.Error("Expected rate limit to allow (under limit)")
	}
}

func TestSmartModerationService_TakeModerationAction(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	moderatorID := uuid.New()
	memberID := uuid.New()

	// Take moderation action
	log, err := svc.TakeModerationAction(ctx, serverID, moderatorID, &models.ModerationActionRequest{
		MemberID: memberID,
		Action:  models.ModActionWarn,
		Reason:  "Test warning",
	})
	if err != nil {
		t.Fatalf("TakeModerationAction failed: %v", err)
	}

	if log.MemberID != memberID {
		t.Errorf("Expected MemberID %v, got %v", memberID, log.MemberID)
	}

	if log.ActionType != models.ModActionWarn {
		t.Errorf("Expected ActionType ModActionWarn, got %v", log.ActionType)
	}

	if log.Reason != "Test warning" {
		t.Errorf("Expected Reason 'Test warning', got %v", log.Reason)
	}
}

func TestSmartModerationService_ResolveModerationLog(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	moderatorID := uuid.New()
	memberID := uuid.New()

	// Create a moderation log
	log, err := svc.TakeModerationAction(ctx, serverID, moderatorID, &models.ModerationActionRequest{
		MemberID: memberID,
		Action:  models.ModActionWarn,
		Reason:  "Test warning",
	})
	if err != nil {
		t.Fatalf("TakeModerationAction failed: %v", err)
	}

	if log.Resolved {
		t.Error("Expected log to be unresolved initially")
	}

	// Resolve the log
	err = svc.ResolveModerationLog(ctx, log.ID, moderatorID)
	if err != nil {
		t.Fatalf("ResolveModerationLog failed: %v", err)
	}

	// Verify the log is resolved
	resolvedLog, err := repo.GetModerationLogByID(ctx, log.ID)
	if err != nil {
		t.Fatalf("GetModerationLogByID failed: %v", err)
	}

	if !resolvedLog.Resolved {
		t.Error("Expected log to be resolved")
	}

	if resolvedLog.ResolvedBy == nil || *resolvedLog.ResolvedBy != moderatorID {
		t.Error("Expected ResolvedBy to be set to moderatorID")
	}
}

func TestSmartModerationService_ResetMemberViolations(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()
	moderatorID := uuid.New()
	memberID := uuid.New()

	// Create some violations
	for i := 0; i < 3; i++ {
		_, err := svc.TakeModerationAction(ctx, serverID, moderatorID, &models.ModerationActionRequest{
			MemberID: memberID,
			Action:  models.ModActionWarn,
			Reason:  "Test warning",
		})
		if err != nil {
			t.Fatalf("TakeModerationAction failed: %v", err)
		}
	}

	// Check violation summary
	summary, err := svc.GetUserViolationSummary(ctx, serverID, memberID)
	if err != nil {
		t.Fatalf("GetUserViolationSummary failed: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected violation summary to exist")
	}

	if summary.ViolationCount != 3 {
		t.Errorf("Expected ViolationCount 3, got %d", summary.ViolationCount)
	}

	// Reset violations
	err = svc.ResetMemberViolations(ctx, serverID, memberID)
	if err != nil {
		t.Fatalf("ResetMemberViolations failed: %v", err)
	}

	// Verify reset
	summary, err = svc.GetUserViolationSummary(ctx, serverID, memberID)
	if err != nil {
		t.Fatalf("GetUserViolationSummary failed: %v", err)
	}

	if summary.ViolationCount != 0 {
		t.Errorf("Expected ViolationCount 0 after reset, got %d", summary.ViolationCount)
	}
}

func TestSmartModerationService_GetDashboardStats(t *testing.T) {
	repo := NewMockSmartModerationRepository()
	svc := NewSmartModerationService(repo)
	ctx := context.Background()
	serverID := uuid.New()

	stats, err := svc.GetDashboardStats(ctx, serverID, 7)
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
}
