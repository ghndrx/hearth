package services

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// SearchSuggestion represents a search suggestion
type SearchSuggestion struct {
	Type        string `json:"type"`        // "user", "channel", "filter", "recent"
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Value       string `json:"value"` // The value to insert in search
}

// SearchSuggestionsRequest contains parameters for fetching suggestions
type SearchSuggestionsRequest struct {
	Query     string
	ServerID  *uuid.UUID
	Limit     int
	UserID    uuid.UUID
}

// SearchSuggestionsResult contains categorized suggestions
type SearchSuggestionsResult struct {
	Users    []SearchSuggestion `json:"users,omitempty"`
	Channels []SearchSuggestion `json:"channels,omitempty"`
	Filters  []SearchSuggestion `json:"filters,omitempty"`
}

// filterSuggestions are the available search filters
var filterSuggestions = []SearchSuggestion{
	{Type: "filter", Name: "from:", Description: "Messages from a user", Value: "from:"},
	{Type: "filter", Name: "in:", Description: "Messages in a channel", Value: "in:"},
	{Type: "filter", Name: "has:attachment", Description: "Messages with attachments", Value: "has:attachment"},
	{Type: "filter", Name: "has:image", Description: "Messages with images", Value: "has:image"},
	{Type: "filter", Name: "has:video", Description: "Messages with videos", Value: "has:video"},
	{Type: "filter", Name: "has:link", Description: "Messages with links", Value: "has:link"},
	{Type: "filter", Name: "has:embed", Description: "Messages with embeds", Value: "has:embed"},
	{Type: "filter", Name: "has:reaction", Description: "Messages with reactions", Value: "has:reaction"},
	{Type: "filter", Name: "before:", Description: "Messages before a date", Value: "before:"},
	{Type: "filter", Name: "after:", Description: "Messages after a date", Value: "after:"},
	{Type: "filter", Name: "pinned:true", Description: "Pinned messages only", Value: "pinned:true"},
	{Type: "filter", Name: "mentions:", Description: "Messages mentioning a user", Value: "mentions:"},
}

// GetSearchSuggestions returns suggestions based on the current query
func (s *SearchService) GetSearchSuggestions(ctx context.Context, req SearchSuggestionsRequest) (*SearchSuggestionsResult, error) {
	result := &SearchSuggestionsResult{
		Users:    make([]SearchSuggestion, 0),
		Channels: make([]SearchSuggestion, 0),
		Filters:  make([]SearchSuggestion, 0),
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}

	query := strings.TrimSpace(req.Query)

	// Check if user is typing a filter
	lastToken := getLastToken(query)

	// If typing "from:user" or similar, suggest users
	if strings.HasPrefix(strings.ToLower(lastToken), "from:") {
		userQuery := strings.TrimPrefix(strings.ToLower(lastToken), "from:")
		if len(userQuery) > 0 {
			users, err := s.searchRepo.SearchUsers(ctx, userQuery, req.ServerID, req.Limit)
			if err == nil {
				for _, u := range users {
					displayName := u.Username
					result.Users = append(result.Users, SearchSuggestion{
						Type:        "user",
						ID:          u.ID.String(),
						Name:        displayName,
						Description: "@" + u.Username,
						Icon:        getAvatarURLFromString(u.AvatarURL),
						Value:       "from:" + u.Username,
					})
				}
			}
		}
		return result, nil
	}

	// If typing "in:channel", suggest channels
	if strings.HasPrefix(strings.ToLower(lastToken), "in:") {
		channelQuery := strings.TrimPrefix(strings.ToLower(lastToken), "in:")
		if len(channelQuery) > 0 {
			channels, err := s.searchRepo.SearchChannels(ctx, channelQuery, req.ServerID, req.Limit)
			if err == nil {
				for _, ch := range channels {
					result.Channels = append(result.Channels, SearchSuggestion{
						Type:        "channel",
						ID:          ch.ID.String(),
						Name:        ch.Name,
						Description: getChannelTypeDescription(ch.Type),
						Value:       "in:" + ch.Name,
					})
				}
			}
		}
		return result, nil
	}

	// If typing "mentions:user", suggest users
	if strings.HasPrefix(strings.ToLower(lastToken), "mentions:") {
		userQuery := strings.TrimPrefix(strings.ToLower(lastToken), "mentions:")
		if len(userQuery) > 0 {
			users, err := s.searchRepo.SearchUsers(ctx, userQuery, req.ServerID, req.Limit)
			if err == nil {
				for _, u := range users {
					displayName := u.Username
					result.Users = append(result.Users, SearchSuggestion{
						Type:        "user",
						ID:          u.ID.String(),
						Name:        displayName,
						Description: "@" + u.Username,
						Icon:        getAvatarURLFromString(u.AvatarURL),
						Value:       "mentions:" + u.Username,
					})
				}
			}
		}
		return result, nil
	}

	// If query is empty or matches filter prefixes, suggest filters
	lowerQuery := strings.ToLower(query)
	for _, filter := range filterSuggestions {
		lowerName := strings.ToLower(filter.Name)
		if query == "" || strings.HasPrefix(lowerName, lowerQuery) || strings.Contains(lowerName, lowerQuery) {
			result.Filters = append(result.Filters, filter)
			if len(result.Filters) >= req.Limit {
				break
			}
		}
	}

	// Also suggest users and channels if we have a general query
	if len(query) >= 2 {
		// Suggest users
		users, err := s.searchRepo.SearchUsers(ctx, query, req.ServerID, req.Limit)
		if err == nil {
			for _, u := range users {
				displayName := u.Username
				result.Users = append(result.Users, SearchSuggestion{
					Type:        "user",
					ID:          u.ID.String(),
					Name:        displayName,
					Description: "@" + u.Username,
					Icon:        getAvatarURLFromString(u.AvatarURL),
					Value:       "from:" + u.Username,
				})
			}
		}

		// Suggest channels (if in a server context)
		if req.ServerID != nil {
			channels, err := s.searchRepo.SearchChannels(ctx, query, req.ServerID, req.Limit)
			if err == nil {
				for _, ch := range channels {
					result.Channels = append(result.Channels, SearchSuggestion{
						Type:        "channel",
						ID:          ch.ID.String(),
						Name:        ch.Name,
						Description: getChannelTypeDescription(ch.Type),
						Value:       "in:" + ch.Name,
					})
				}
			}
		}
	}

	return result, nil
}

// getLastToken extracts the last token from a query string
func getLastToken(query string) string {
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return ""
	}
	return tokens[len(tokens)-1]
}

// getAvatarURLFromString returns avatar URL from a pointer string
func getAvatarURLFromString(avatarURL *string) string {
	if avatarURL == nil || *avatarURL == "" {
		return ""
	}
	return *avatarURL
}

// getChannelTypeDescription returns a human-readable description for channel type
func getChannelTypeDescription(channelType models.ChannelType) string {
	switch channelType {
	case models.ChannelTypeText:
		return "Text Channel"
	case models.ChannelTypeVoice:
		return "Voice Channel"
	case models.ChannelTypeCategory:
		return "Category"
	case models.ChannelTypeDM:
		return "Direct Message"
	case models.ChannelTypeGroupDM:
		return "Group DM"
	case models.ChannelTypeAnnouncement:
		return "Announcement Channel"
	default:
		return "Channel"
	}
}
