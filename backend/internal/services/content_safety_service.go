package services

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// ContentSafetyRepository defines the repository interface for content safety
type ContentSafetyRepository interface {
	// Content Filters
	CreateContentFilter(ctx context.Context, filter *models.ContentFilter) error
	GetContentFilterByID(ctx context.Context, id uuid.UUID) (*models.ContentFilter, error)
	GetContentFiltersByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error)
	GetContentFiltersByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.ContentFilter, error)
	GetContentFiltersForMessage(ctx context.Context, serverID, channelID uuid.UUID) ([]*models.ContentFilter, error)
	UpdateContentFilter(ctx context.Context, filter *models.ContentFilter) error
	DeleteContentFilter(ctx context.Context, id uuid.UUID) error
	GetEnabledFiltersByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error)

	// Age Verification
	CreateAgeVerification(ctx context.Context, settings *models.AgeVerificationSetting) error
	GetAgeVerificationByServerID(ctx context.Context, serverID uuid.UUID) (*models.AgeVerificationSetting, error)
	GetAgeVerificationForChannel(ctx context.Context, serverID, channelID uuid.UUID) (*models.AgeVerificationSetting, error)
	UpdateAgeVerification(ctx context.Context, settings *models.AgeVerificationSetting) error
	DeleteAgeVerification(ctx context.Context, id uuid.UUID) error

	// User Content Preferences
	CreateUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error
	GetUserContentPreference(ctx context.Context, userID uuid.UUID) (*models.UserContentPreference, error)
	UpdateUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error
	UpsertUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error
}

// ContentSafetyService handles content safety operations
type ContentSafetyService struct {
	repo ContentSafetyRepository
}

// NewContentSafetyService creates a new content safety service
func NewContentSafetyService(repo ContentSafetyRepository) *ContentSafetyService {
	return &ContentSafetyService{repo: repo}
}

// Content Filter methods

// CreateContentFilter creates a new content filter
func (s *ContentSafetyService) CreateContentFilter(ctx context.Context, serverID, createdBy uuid.UUID, req *models.CreateContentFilterRequest) (*models.ContentFilter, error) {
	// Parse channel ID if provided
	var channelID *uuid.UUID
	if req.ChannelID != nil && *req.ChannelID != "" {
		parsed, err := uuid.Parse(*req.ChannelID)
		if err != nil {
			return nil, ErrInvalidInput
		}
		channelID = &parsed
	}

	// Parse exempt roles
	var exemptRoles []uuid.UUID
	for _, roleStr := range req.ExemptRoles {
		parsed, err := uuid.Parse(roleStr)
		if err != nil {
			continue
		}
		exemptRoles = append(exemptRoles, parsed)
	}

	// Set defaults
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	filter := &models.ContentFilter{
		ServerID:    serverID,
		ChannelID:   channelID,
		Type:        req.Type,
		Name:        req.Name,
		Enabled:     enabled,
		Threshold:   req.Threshold,
		Action:      req.Action,
		FilterData:  req.FilterData,
		ExemptRoles: exemptRoles,
		CreatedBy:   createdBy,
	}

	if err := s.repo.CreateContentFilter(ctx, filter); err != nil {
		return nil, err
	}

	return filter, nil
}

// GetContentFilter gets a content filter by ID
func (s *ContentSafetyService) GetContentFilter(ctx context.Context, id uuid.UUID) (*models.ContentFilter, error) {
	return s.repo.GetContentFilterByID(ctx, id)
}

// GetServerContentFilters gets all content filters for a server
func (s *ContentSafetyService) GetServerContentFilters(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error) {
	return s.repo.GetContentFiltersByServerID(ctx, serverID)
}

// GetChannelContentFilters gets all content filters for a channel
func (s *ContentSafetyService) GetChannelContentFilters(ctx context.Context, channelID uuid.UUID) ([]*models.ContentFilter, error) {
	return s.repo.GetContentFiltersByChannelID(ctx, channelID)
}

// UpdateContentFilter updates an existing content filter
func (s *ContentSafetyService) UpdateContentFilter(ctx context.Context, id uuid.UUID, req *models.UpdateContentFilterRequest) (*models.ContentFilter, error) {
	filter, err := s.repo.GetContentFilterByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if filter == nil {
		return nil, ErrServerNotFound
	}

	if req.Name != nil {
		filter.Name = *req.Name
	}
	if req.Enabled != nil {
		filter.Enabled = *req.Enabled
	}
	if req.Threshold != nil {
		filter.Threshold = *req.Threshold
	}
	if req.Action != nil {
		filter.Action = *req.Action
	}
	if req.FilterData != nil {
		filter.FilterData = *req.FilterData
	}
	if req.ExemptRoles != nil {
		var exemptRoles []uuid.UUID
		for _, roleStr := range *req.ExemptRoles {
			parsed, err := uuid.Parse(roleStr)
			if err != nil {
				continue
			}
			exemptRoles = append(exemptRoles, parsed)
		}
		filter.ExemptRoles = exemptRoles
	}

	if err := s.repo.UpdateContentFilter(ctx, filter); err != nil {
		return nil, err
	}

	return filter, nil
}

// DeleteContentFilter deletes a content filter
func (s *ContentSafetyService) DeleteContentFilter(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteContentFilter(ctx, id)
}

// Age Verification methods

// CreateAgeVerification creates age verification settings
func (s *ContentSafetyService) CreateAgeVerification(ctx context.Context, serverID, createdBy uuid.UUID, req *models.CreateAgeVerificationRequest) (*models.AgeVerificationSetting, error) {
	// Parse channel ID if provided
	var channelID *uuid.UUID
	if req.ChannelID != nil && *req.ChannelID != "" {
		parsed, err := uuid.Parse(*req.ChannelID)
		if err != nil {
			return nil, ErrInvalidInput
		}
		channelID = &parsed
	}

	settings := &models.AgeVerificationSetting{
		ServerID:         serverID,
		ChannelID:        channelID,
		Enabled:          req.Enabled,
		RequiredAge:      req.RequiredAge,
		VerificationType: req.VerificationType,
		CreatedBy:        createdBy,
	}

	if err := s.repo.CreateAgeVerification(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// GetAgeVerification gets age verification settings for a server
func (s *ContentSafetyService) GetAgeVerification(ctx context.Context, serverID uuid.UUID) (*models.AgeVerificationSetting, error) {
	return s.repo.GetAgeVerificationByServerID(ctx, serverID)
}

// GetChannelAgeVerification gets age verification settings for a channel
func (s *ContentSafetyService) GetChannelAgeVerification(ctx context.Context, serverID, channelID uuid.UUID) (*models.AgeVerificationSetting, error) {
	return s.repo.GetAgeVerificationForChannel(ctx, serverID, channelID)
}

// UpdateAgeVerification updates age verification settings
func (s *ContentSafetyService) UpdateAgeVerification(ctx context.Context, serverID uuid.UUID, req *models.UpdateAgeVerificationRequest) (*models.AgeVerificationSetting, error) {
	settings, err := s.repo.GetAgeVerificationByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, ErrServerNotFound
	}

	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.RequiredAge != nil {
		settings.RequiredAge = *req.RequiredAge
	}
	if req.VerificationType != nil {
		settings.VerificationType = *req.VerificationType
	}

	if err := s.repo.UpdateAgeVerification(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// DeleteAgeVerification deletes age verification settings
func (s *ContentSafetyService) DeleteAgeVerification(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteAgeVerification(ctx, id)
}

// User Content Preference methods

// GetUserContentPreference gets user content preferences
func (s *ContentSafetyService) GetUserContentPreference(ctx context.Context, userID uuid.UUID) (*models.UserContentPreference, error) {
	prefs, err := s.repo.GetUserContentPreference(ctx, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		// Return default preferences
		return &models.UserContentPreference{
			UserID:                userID,
			NSFWFilterLevel:        models.NSFWThresholdMedium,
			HideNSFWContent:        true,
			HideExplicitContent:   false,
			AutoCollapseNSFW:      false,
			AllowAgeVerifiedChannels: true,
		}, nil
	}
	return prefs, nil
}

// UpdateUserContentPreference updates user content preferences
func (s *ContentSafetyService) UpdateUserContentPreference(ctx context.Context, userID uuid.UUID, req *models.UpdateUserContentPreferenceRequest) (*models.UserContentPreference, error) {
	prefs, err := s.repo.GetUserContentPreference(ctx, userID)
	if err != nil {
		return nil, err
	}

	if prefs == nil {
		// Create new preferences with defaults
		prefs = &models.UserContentPreference{
			UserID:                userID,
			NSFWFilterLevel:        models.NSFWThresholdMedium,
			HideNSFWContent:        true,
			HideExplicitContent:    false,
			AutoCollapseNSFW:      false,
			AllowAgeVerifiedChannels: true,
		}
	}

	if req.NSFWFilterLevel != nil {
		prefs.NSFWFilterLevel = *req.NSFWFilterLevel
	}
	if req.HideNSFWContent != nil {
		prefs.HideNSFWContent = *req.HideNSFWContent
	}
	if req.HideExplicitContent != nil {
		prefs.HideExplicitContent = *req.HideExplicitContent
	}
	if req.AutoCollapseNSFW != nil {
		prefs.AutoCollapseNSFW = *req.AutoCollapseNSFW
	}
	if req.AllowAgeVerifiedChannels != nil {
		prefs.AllowAgeVerifiedChannels = *req.AllowAgeVerifiedChannels
	}
	if req.TrustedServers != nil {
		var servers []uuid.UUID
		for _, serverStr := range *req.TrustedServers {
			parsed, err := uuid.Parse(serverStr)
			if err != nil {
				continue
			}
			servers = append(servers, parsed)
		}
		prefs.TrustedServers = servers
	}

	if err := s.repo.UpsertUserContentPreference(ctx, prefs); err != nil {
		return nil, err
	}

	return prefs, nil
}

// GetServerSafetySettings gets comprehensive safety settings for a server
func (s *ContentSafetyService) GetServerSafetySettings(ctx context.Context, serverID uuid.UUID) (*models.ContentSafetySettings, error) {
	filters, err := s.repo.GetContentFiltersByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	ageVerification, err := s.repo.GetAgeVerificationByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Determine server default threshold from filters
	defaultThreshold := models.NSFWThresholdNone
	for _, f := range filters {
		if f.Type == models.FilterTypeNSFW && f.Threshold > defaultThreshold {
			defaultThreshold = f.Threshold
		}
	}

	return &models.ContentSafetySettings{
		ServerID:               serverID,
		Filters:                filters,
		AgeVerification:       ageVerification,
		ServerDefaultThreshold: defaultThreshold,
	}, nil
}

// ScanContent scans content against enabled filters
func (s *ContentSafetyService) ScanContent(ctx context.Context, serverID, channelID, userID uuid.UUID, userRoles []uuid.UUID, content string) (*models.ContentScanResult, error) {
	filters, err := s.repo.GetContentFiltersForMessage(ctx, serverID, channelID)
	if err != nil {
		return nil, err
	}

	result := &models.ContentScanResult{
		Passed: true,
		Flags:  []models.ContentFlag{},
	}

	for _, filter := range filters {
		// Check if user is exempt
		if s.isUserExempt(userRoles, filter.ExemptRoles) {
			continue
		}

		// Check if filter applies based on threshold
		if filter.Threshold == models.NSFWThresholdNone && filter.Type == models.FilterTypeNSFW {
			continue
		}

		matchResult := s.matchFilter(filter, content)
		if matchResult.Matched {
			result.Passed = false
			result.Flags = append(result.Flags, matchResult.Flags...)
			result.ActionTaken = filter.Action
			result.FilterName = filter.Name
			result.MatchedRule = &filter.ID

			// If action is block or higher, don't continue checking
			if filter.Action >= models.FilterActionBlock {
				break
			}
		}
	}

	return result, nil
}

// ShouldBlockContent determines if content should be blocked
func (s *ContentSafetyService) ShouldBlockContent(ctx context.Context, serverID, channelID, userID uuid.UUID, userRoles []uuid.UUID, content string) (bool, *models.ContentScanResult, error) {
	result, err := s.ScanContent(ctx, serverID, channelID, userID, userRoles, content)
	if err != nil {
		return false, nil, err
	}

	if !result.Passed && result.ActionTaken >= models.FilterActionBlock {
		return true, result, nil
	}

	return false, result, nil
}

func (s *ContentSafetyService) isUserExempt(userRoles, exemptRoles []uuid.UUID) bool {
	for _, userRole := range userRoles {
		for _, exemptRole := range exemptRoles {
			if userRole == exemptRole {
				return true
			}
		}
	}
	return false
}

type matchResult struct {
	Matched bool
	Flags   []models.ContentFlag
}

func (s *ContentSafetyService) matchFilter(filter *models.ContentFilter, content string) *matchResult {
	switch filter.Type {
	case models.FilterTypeNSFW:
		return s.matchNSFWFilter(filter, content)
	case models.FilterTypeViolence:
		return s.matchViolenceFilter(filter, content)
	case models.FilterTypeHateSpeech:
		return s.matchHateSpeechFilter(filter, content)
	case models.FilterTypeHarassment:
		return s.matchHarassmentFilter(filter, content)
	case models.FilterTypeSpam:
		return s.matchSpamFilter(filter, content)
	case models.FilterTypeCustomKeyword:
		return s.matchKeywordFilter(filter, content)
	default:
		return &matchResult{Matched: false}
	}
}

func (s *ContentSafetyService) matchNSFWFilter(filter *models.ContentFilter, content string) *matchResult {
	// This is a simplified implementation
	// In production, this would integrate with ML-based NSFW detection
	lowerContent := strings.ToLower(content)

	// Check against keywords
	for _, keyword := range filter.FilterData.Keywords {
		if strings.Contains(lowerContent, strings.ToLower(keyword)) {
			return &matchResult{
				Matched: true,
				Flags: []models.ContentFlag{
					{
						Type:     models.FilterTypeNSFW,
						Severity: 8,
						Detail:   "NSFW keyword detected",
						Keyword:  &keyword,
					},
				},
			}
		}
	}

	return &matchResult{Matched: false}
}

func (s *ContentSafetyService) matchViolenceFilter(filter *models.ContentFilter, content string) *matchResult {
	lowerContent := strings.ToLower(content)

	for _, keyword := range filter.FilterData.Keywords {
		if strings.Contains(lowerContent, strings.ToLower(keyword)) {
			return &matchResult{
				Matched: true,
				Flags: []models.ContentFlag{
					{
						Type:     models.FilterTypeViolence,
						Severity: 7,
						Detail:   "Violence-related keyword detected",
						Keyword:  &keyword,
					},
				},
			}
		}
	}

	return &matchResult{Matched: false}
}

func (s *ContentSafetyService) matchHateSpeechFilter(filter *models.ContentFilter, content string) *matchResult {
	lowerContent := strings.ToLower(content)

	for _, keyword := range filter.FilterData.Keywords {
		if strings.Contains(lowerContent, strings.ToLower(keyword)) {
			return &matchResult{
				Matched: true,
				Flags: []models.ContentFlag{
					{
						Type:     models.FilterTypeHateSpeech,
						Severity: 9,
						Detail:   "Hate speech keyword detected",
						Keyword:  &keyword,
					},
				},
			}
		}
	}

	return &matchResult{Matched: false}
}

func (s *ContentSafetyService) matchHarassmentFilter(filter *models.ContentFilter, content string) *matchResult {
	lowerContent := strings.ToLower(content)

	for _, keyword := range filter.FilterData.Keywords {
		if strings.Contains(lowerContent, strings.ToLower(keyword)) {
			return &matchResult{
				Matched: true,
				Flags: []models.ContentFlag{
					{
						Type:     models.FilterTypeHarassment,
						Severity: 6,
						Detail:   "Harassment keyword detected",
						Keyword:  &keyword,
					},
				},
			}
		}
	}

	return &matchResult{Matched: false}
}

func (s *ContentSafetyService) matchSpamFilter(filter *models.ContentFilter, content string) *matchResult {
	// Simple spam detection: repeated characters, excessive caps, etc.
	lowerContent := strings.ToLower(content)

	// Check whitelist first
	for _, wl := range filter.FilterData.Whitelist {
		if strings.Contains(lowerContent, strings.ToLower(wl)) {
			return &matchResult{Matched: false}
		}
	}

	// Check for repeated characters
	repeated := regexp.MustCompile(`(.)\1{5,}`)
	if repeated.MatchString(content) {
		return &matchResult{
			Matched: true,
			Flags: []models.ContentFlag{
				{
					Type:     models.FilterTypeSpam,
					Severity: 5,
					Detail:   "Repeated character pattern detected",
				},
			},
		}
	}

	return &matchResult{Matched: false}
}

func (s *ContentSafetyService) matchKeywordFilter(filter *models.ContentFilter, content string) *matchResult {
	lowerContent := strings.ToLower(content)

	// Check whitelist
	for _, wl := range filter.FilterData.Whitelist {
		if strings.Contains(lowerContent, strings.ToLower(wl)) {
			return &matchResult{Matched: false}
		}
	}

	// Check keywords
	for _, keyword := range filter.FilterData.Keywords {
		if strings.Contains(lowerContent, strings.ToLower(keyword)) {
			return &matchResult{
				Matched: true,
				Flags: []models.ContentFlag{
					{
						Type:     models.FilterTypeCustomKeyword,
						Severity: 5,
						Detail:   "Custom keyword match",
						Keyword:  &keyword,
					},
				},
			}
		}
	}

	// Check regex patterns
	for _, pattern := range filter.FilterData.RegexPatterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			continue
		}
		if re.MatchString(content) {
			return &matchResult{
				Matched: true,
				Flags: []models.ContentFlag{
					{
						Type:     models.FilterTypeCustomKeyword,
						Severity: 6,
						Detail:   "Regex pattern match: " + pattern,
					},
				},
			}
		}
	}

	return &matchResult{Matched: false}
}
