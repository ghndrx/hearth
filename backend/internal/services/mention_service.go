package services

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// MentionType represents different types of mentions
type MentionType string

const (
	MentionTypeUser     MentionType = "user"
	MentionTypeRole     MentionType = "role"
	MentionTypeEveryone MentionType = "everyone"
	MentionTypeHere     MentionType = "here"
)

// ParsedMention represents a parsed mention from message content
type ParsedMention struct {
	Type     MentionType
	ID       *uuid.UUID // nil for @everyone/@here
	Username string     // For display purposes
	Raw      string     // The original matched text
	Start    int        // Position in string
	End      int        // End position in string
}

// MentionParseResult contains all parsed mentions from a message
type MentionParseResult struct {
	UserMentions    []uuid.UUID
	RoleMentions    []uuid.UUID
	MentionEveryone bool
	MentionHere     bool
	AllMentions     []ParsedMention
}

// MentionService handles mention parsing and notification creation
type MentionService struct {
	userRepo         UserRepository
	roleRepo         RoleRepository
	memberRepo       ServerRepository
	notificationRepo NotificationRepository
	readStateRepo    ReadStateRepository
	mentionRepo      MentionRepository
	eventBus         EventBus
}

// NewMentionService creates a new mention service
func NewMentionService(
	userRepo UserRepository,
	roleRepo RoleRepository,
	memberRepo ServerRepository,
	notificationRepo NotificationRepository,
	readStateRepo ReadStateRepository,
	eventBus EventBus,
) *MentionService {
	return &MentionService{
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		memberRepo:       memberRepo,
		notificationRepo: notificationRepo,
		readStateRepo:    readStateRepo,
		eventBus:         eventBus,
	}
}

// NewMentionServiceWithRepo creates a new mention service with mention repository
func NewMentionServiceWithRepo(
	userRepo UserRepository,
	roleRepo RoleRepository,
	memberRepo ServerRepository,
	notificationRepo NotificationRepository,
	readStateRepo ReadStateRepository,
	mentionRepo MentionRepository,
	eventBus EventBus,
) *MentionService {
	return &MentionService{
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		memberRepo:       memberRepo,
		notificationRepo: notificationRepo,
		readStateRepo:    readStateRepo,
		mentionRepo:      mentionRepo,
		eventBus:         eventBus,
	}
}

// SetMentionRepo sets the mention repository (allows adding after creation)
func (s *MentionService) SetMentionRepo(repo MentionRepository) {
	s.mentionRepo = repo
}

var (
	// Matches @username (alphanumeric + underscore, 2-32 chars)
	userMentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]{2,32})`)
	// Matches <@user_id> format (Discord-style)
	userIDMentionRegex = regexp.MustCompile(`<@!?([a-f0-9-]{36})>`)
	// Matches <@&role_id> format (Discord-style role mention)
	roleMentionRegex = regexp.MustCompile(`<@&([a-f0-9-]{36})>`)
	// Matches @everyone
	everyoneMentionRegex = regexp.MustCompile(`@everyone\b`)
	// Matches @here
	hereMentionRegex = regexp.MustCompile(`@here\b`)
)

// ParseMentions extracts all mentions from message content
func (s *MentionService) ParseMentions(ctx context.Context, content string, serverID *uuid.UUID) (*MentionParseResult, error) {
	result := &MentionParseResult{
		UserMentions: make([]uuid.UUID, 0),
		RoleMentions: make([]uuid.UUID, 0),
		AllMentions:  make([]ParsedMention, 0),
	}

	seenUsers := make(map[uuid.UUID]bool)
	seenRoles := make(map[uuid.UUID]bool)

	// Parse @everyone
	if everyoneMentionRegex.MatchString(content) {
		result.MentionEveryone = true
		matches := everyoneMentionRegex.FindAllStringIndex(content, -1)
		for _, match := range matches {
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:  MentionTypeEveryone,
				Raw:   "@everyone",
				Start: match[0],
				End:   match[1],
			})
		}
	}

	// Parse @here
	if hereMentionRegex.MatchString(content) {
		result.MentionHere = true
		matches := hereMentionRegex.FindAllStringIndex(content, -1)
		for _, match := range matches {
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:  MentionTypeHere,
				Raw:   "@here",
				Start: match[0],
				End:   match[1],
			})
		}
	}

	// Parse <@user_id> format
	userIDMatches := userIDMentionRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range userIDMatches {
		idStr := content[match[2]:match[3]]
		if id, err := uuid.Parse(idStr); err == nil {
			if !seenUsers[id] {
				seenUsers[id] = true
				result.UserMentions = append(result.UserMentions, id)
			}
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:  MentionTypeUser,
				ID:    &id,
				Raw:   content[match[0]:match[1]],
				Start: match[0],
				End:   match[1],
			})
		}
	}

	// Parse <@&role_id> format
	roleMatches := roleMentionRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range roleMatches {
		idStr := content[match[2]:match[3]]
		if id, err := uuid.Parse(idStr); err == nil {
			if !seenRoles[id] {
				seenRoles[id] = true
				result.RoleMentions = append(result.RoleMentions, id)
			}
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:  MentionTypeRole,
				ID:    &id,
				Raw:   content[match[0]:match[1]],
				Start: match[0],
				End:   match[1],
			})
		}
	}

	// Parse @username format (resolve to user IDs)
	userMatches := userMentionRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range userMatches {
		username := content[match[2]:match[3]]
		rawMention := content[match[0]:match[1]]

		// Skip if it's @everyone or @here
		if username == "everyone" || username == "here" {
			continue
		}

		// Try to resolve username to user
		user, err := s.userRepo.GetByUsername(ctx, username)
		if err == nil && user != nil {
			if !seenUsers[user.ID] {
				seenUsers[user.ID] = true
				result.UserMentions = append(result.UserMentions, user.ID)
			}
			id := user.ID
			result.AllMentions = append(result.AllMentions, ParsedMention{
				Type:     MentionTypeUser,
				ID:       &id,
				Username: username,
				Raw:      rawMention,
				Start:    match[0],
				End:      match[1],
			})
		}
	}

	return result, nil
}

// ParseMentionsSimple is a simpler version that just extracts user IDs
// Used by the message service for the Mentions field
func ParseMentionsSimple(content string) []uuid.UUID {
	mentions := make([]uuid.UUID, 0)
	seen := make(map[uuid.UUID]bool)

	// Parse <@user_id> format
	matches := userIDMentionRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			if id, err := uuid.Parse(match[1]); err == nil {
				if !seen[id] {
					seen[id] = true
					mentions = append(mentions, id)
				}
			}
		}
	}

	return mentions
}

// ProcessMessageMentions processes mentions in a message and creates notifications
func (s *MentionService) ProcessMessageMentions(
	ctx context.Context,
	message *models.Message,
	author *models.User,
	serverID *uuid.UUID,
) error {
	if message.EncryptedContent != "" {
		// Skip encrypted messages - can't parse mentions
		return nil
	}

	result, err := s.ParseMentions(ctx, message.Content, serverID)
	if err != nil {
		return err
	}

	// Update message with parsed mentions
	message.Mentions = result.UserMentions
	message.MentionRoles = result.RoleMentions
	message.MentionEveryone = result.MentionEveryone

	// Create notifications and mention records for user mentions
	for _, userID := range result.UserMentions {
		if userID == message.AuthorID {
			continue // Don't notify yourself
		}
		if err := s.createMentionNotification(ctx, message, author, userID, serverID); err != nil {
			continue // Log but don't fail
		}
		// Create mention record
		if s.mentionRepo != nil {
			mention := &models.Mention{
				UserID:      userID,
				MessageID:   message.ID,
				MentionedBy: message.AuthorID,
				ChannelID:   message.ChannelID,
				GuildID:     serverID,
				MentionType: models.MentionKindUser,
			}
			_ = s.mentionRepo.Create(ctx, mention)
		}
		// Increment mention count for read state
		if err := s.readStateRepo.IncrementMentionCount(ctx, userID, message.ChannelID); err != nil {
			continue
		}
	}

	// Handle @everyone/@here mentions in servers
	if serverID != nil && (result.MentionEveryone || result.MentionHere) {
		mentionType := models.MentionKindEveryone
		if result.MentionHere {
			mentionType = models.MentionKindHere
		}
		members, err := s.getAllMembersWithPagination(ctx, *serverID)
		if err == nil {
			for _, member := range members {
				if member.UserID == message.AuthorID {
					continue
				}
				// For @here, we'd check online status - simplified for now
				if err := s.createMentionNotification(ctx, message, author, member.UserID, serverID); err != nil {
					continue
				}
				// Create mention record
				if s.mentionRepo != nil {
					mention := &models.Mention{
						UserID:      member.UserID,
						MessageID:   message.ID,
						MentionedBy: message.AuthorID,
						ChannelID:   message.ChannelID,
						GuildID:     serverID,
						MentionType: mentionType,
					}
					_ = s.mentionRepo.Create(ctx, mention)
				}
				if err := s.readStateRepo.IncrementMentionCount(ctx, member.UserID, message.ChannelID); err != nil {
					continue
				}
			}
		}
	}

	// Handle role mentions
	if serverID != nil && len(result.RoleMentions) > 0 {
		for _, roleID := range result.RoleMentions {
			// Get members with this role
			membersWithRole, err := s.memberRepo.GetMembersWithRole(ctx, *serverID, roleID)
			if err != nil {
				continue
			}
			for _, member := range membersWithRole {
				if member.UserID == message.AuthorID {
					continue
				}
				if err := s.createMentionNotification(ctx, message, author, member.UserID, serverID); err != nil {
					continue
				}
				// Create mention record for role mention
				if s.mentionRepo != nil {
					mention := &models.Mention{
						UserID:          member.UserID,
						MessageID:       message.ID,
						MentionedBy:     message.AuthorID,
						ChannelID:       message.ChannelID,
						GuildID:         serverID,
						MentionType:     models.MentionKindRole,
						MentionedRoleID: &roleID,
					}
					_ = s.mentionRepo.Create(ctx, mention)
				}
				if err := s.readStateRepo.IncrementMentionCount(ctx, member.UserID, message.ChannelID); err != nil {
					continue
				}
			}
		}
	}

	return nil
}

// createMentionNotification creates a notification for a mention
func (s *MentionService) createMentionNotification(
	ctx context.Context,
	message *models.Message,
	author *models.User,
	recipientID uuid.UUID,
	serverID *uuid.UUID,
) error {
	// Truncate content for notification body
	body := message.Content
	if len(body) > 200 {
		body = body[:197] + "..."
	}

	notification := &models.Notification{
		UserID:    recipientID,
		Type:      models.NotificationTypeMention,
		Title:     author.Username + " mentioned you",
		Body:      body,
		ActorID:   &message.AuthorID,
		ServerID:  serverID,
		ChannelID: &message.ChannelID,
		MessageID: &message.ID,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return err
	}

	// Emit event for real-time delivery
	s.eventBus.Publish("notification.created", &NotificationCreatedEvent{
		Notification: notification,
	})

	return nil
}

// ProcessReplyMention creates a notification when someone replies to a message
func (s *MentionService) ProcessReplyMention(
	ctx context.Context,
	message *models.Message,
	author *models.User,
	replyToAuthorID uuid.UUID,
	serverID *uuid.UUID,
) error {
	if replyToAuthorID == message.AuthorID {
		return nil // Don't notify yourself
	}

	body := message.Content
	if len(body) > 200 {
		body = body[:197] + "..."
	}

	notification := &models.Notification{
		UserID:    replyToAuthorID,
		Type:      models.NotificationTypeReply,
		Title:     author.Username + " replied to you",
		Body:      body,
		ActorID:   &message.AuthorID,
		ServerID:  serverID,
		ChannelID: &message.ChannelID,
		MessageID: &message.ID,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return err
	}

	// Emit event for real-time delivery
	s.eventBus.Publish("notification.created", &NotificationCreatedEvent{
		Notification: notification,
	})

	// Increment mention count
	return s.readStateRepo.IncrementMentionCount(ctx, replyToAuthorID, message.ChannelID)
}

// FormatMentionContent converts @username mentions to <@user_id> format for storage
func (s *MentionService) FormatMentionContent(ctx context.Context, content string) string {
	// Find all @username mentions and replace with <@user_id>
	result := userMentionRegex.ReplaceAllStringFunc(content, func(match string) string {
		username := strings.TrimPrefix(match, "@")
		if username == "everyone" || username == "here" {
			return match // Keep as-is
		}
		user, err := s.userRepo.GetByUsername(ctx, username)
		if err == nil && user != nil {
			return "<@" + user.ID.String() + ">"
		}
		return match // Keep original if user not found
	})
	return result
}

// RenderMentionContent converts <@user_id> format back to displayable format
func RenderMentionContent(content string, usernames map[uuid.UUID]string) string {
	result := userIDMentionRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract ID from <@id> or <@!id>
		idStr := strings.TrimPrefix(match, "<@")
		idStr = strings.TrimPrefix(idStr, "!")
		idStr = strings.TrimSuffix(idStr, ">")

		if id, err := uuid.Parse(idStr); err == nil {
			if username, ok := usernames[id]; ok {
				return "@" + username
			}
		}
		return match
	})
	return result
}

func (s *MentionService) getAllMembersWithPagination(ctx context.Context, serverID uuid.UUID) ([]*models.Member, error) {
	const batchSize = 100
	var allMembers []*models.Member
	var cursor *models.MemberCursor

	for {
		result, err := s.memberRepo.GetMembersPaginated(ctx, serverID, cursor, batchSize)
		if err != nil {
			return nil, err
		}

		allMembers = append(allMembers, result.Members...)

		if !result.HasMore {
			break
		}

		nextCursor, err := models.DecodeMemberCursor(result.NextCursor)
		if err != nil {
			return nil, err
		}
		cursor = nextCursor
	}

	return allMembers, nil
}
