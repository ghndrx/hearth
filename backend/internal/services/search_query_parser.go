package services

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SearchQueryToken represents a parsed token from the search query
type SearchQueryToken struct {
	Type  string // "text", "from", "in", "has", "before", "after", "mentions", "pinned"
	Value string
	Raw   string // Original token text for display
}

// ParsedSearchQuery contains the parsed search query components
type ParsedSearchQuery struct {
	Tokens    []SearchQueryToken
	FreeText  string
	AuthorID  *uuid.UUID
	ChannelID *uuid.UUID
	Before    *time.Time
	After     *time.Time
	Has       []string // "attachment", "embed", "image", "video", "file", "link", "reaction"
	Pinned    *bool
	Mentions  []uuid.UUID
}

// filterPatterns defines regex patterns for search filters
var filterPatterns = map[string]*regexp.Regexp{
	"from":     regexp.MustCompile(`(?i)from:\s*<?@?([^\s>]+)>?`),
	"in":       regexp.MustCompile(`(?i)in:\s*<?#?([^\s>]+)>?`),
	"has":      regexp.MustCompile(`(?i)has:\s*(\w+)`),
	"before":   regexp.MustCompile(`(?i)before:\s*(\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})?)?)`),
	"after":    regexp.MustCompile(`(?i)after:\s*(\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})?)?)`),
	"mentions": regexp.MustCompile(`(?i)mentions:\s*<?@?([^\s>]+)>?`),
	"pinned":   regexp.MustCompile(`(?i)pinned:\s*(true|false|yes|no)`),
}

// ParseSearchQueryString parses a search query string and extracts filters
// Supports Discord-like syntax: from:@user in:#channel has:attachment before:2024-01-01
func ParseSearchQueryString(query string) *ParsedSearchQuery {
	result := &ParsedSearchQuery{
		Tokens: make([]SearchQueryToken, 0),
		Has:    make([]string, 0),
	}

	if query == "" {
		return result
	}

	// Track what we've matched to remove from the query
	remainingQuery := query

	// Parse "from:" filter
	if matches := filterPatterns["from"].FindAllStringSubmatch(query, -1); len(matches) > 0 {
		for _, match := range matches {
			result.Tokens = append(result.Tokens, SearchQueryToken{
				Type:  "from",
				Value: match[1],
				Raw:   match[0],
			})
			remainingQuery = strings.Replace(remainingQuery, match[0], "", 1)
		}
	}

	// Parse "in:" filter
	if matches := filterPatterns["in"].FindAllStringSubmatch(query, -1); len(matches) > 0 {
		for _, match := range matches {
			result.Tokens = append(result.Tokens, SearchQueryToken{
				Type:  "in",
				Value: match[1],
				Raw:   match[0],
			})
			remainingQuery = strings.Replace(remainingQuery, match[0], "", 1)
		}
	}

	// Parse "has:" filter
	if matches := filterPatterns["has"].FindAllStringSubmatch(query, -1); len(matches) > 0 {
		for _, match := range matches {
			hasValue := strings.ToLower(match[1])
			// Normalize has values
			switch hasValue {
			case "file", "attachment", "attachments":
				hasValue = "attachment"
			case "image", "images":
				hasValue = "image"
			case "video", "videos":
				hasValue = "video"
			case "link", "links", "url":
				hasValue = "link"
			case "embed", "embeds":
				hasValue = "embed"
			case "reaction", "reactions":
				hasValue = "reaction"
			}
			result.Has = append(result.Has, hasValue)
			result.Tokens = append(result.Tokens, SearchQueryToken{
				Type:  "has",
				Value: hasValue,
				Raw:   match[0],
			})
			remainingQuery = strings.Replace(remainingQuery, match[0], "", 1)
		}
	}

	// Parse "before:" filter
	if matches := filterPatterns["before"].FindStringSubmatch(query); len(matches) > 0 {
		if t, err := parseFlexibleDate(matches[1]); err == nil {
			result.Before = &t
			result.Tokens = append(result.Tokens, SearchQueryToken{
				Type:  "before",
				Value: matches[1],
				Raw:   matches[0],
			})
			remainingQuery = strings.Replace(remainingQuery, matches[0], "", 1)
		}
	}

	// Parse "after:" filter
	if matches := filterPatterns["after"].FindStringSubmatch(query); len(matches) > 0 {
		if t, err := parseFlexibleDate(matches[1]); err == nil {
			result.After = &t
			result.Tokens = append(result.Tokens, SearchQueryToken{
				Type:  "after",
				Value: matches[1],
				Raw:   matches[0],
			})
			remainingQuery = strings.Replace(remainingQuery, matches[0], "", 1)
		}
	}

	// Parse "mentions:" filter
	if matches := filterPatterns["mentions"].FindAllStringSubmatch(query, -1); len(matches) > 0 {
		for _, match := range matches {
			result.Tokens = append(result.Tokens, SearchQueryToken{
				Type:  "mentions",
				Value: match[1],
				Raw:   match[0],
			})
			remainingQuery = strings.Replace(remainingQuery, match[0], "", 1)
		}
	}

	// Parse "pinned:" filter
	if matches := filterPatterns["pinned"].FindStringSubmatch(query); len(matches) > 0 {
		val := strings.ToLower(matches[1])
		pinned := val == "true" || val == "yes"
		result.Pinned = &pinned
		result.Tokens = append(result.Tokens, SearchQueryToken{
			Type:  "pinned",
			Value: matches[1],
			Raw:   matches[0],
		})
		remainingQuery = strings.Replace(remainingQuery, matches[0], "", 1)
	}

	// Clean up and set remaining free text
	result.FreeText = strings.TrimSpace(remainingQuery)

	return result
}

// parseFlexibleDate parses various date formats
func parseFlexibleDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, &time.ParseError{Value: s}
}

// ValidHasValues returns the list of valid values for has: filter
func ValidHasValues() []string {
	return []string{"attachment", "image", "video", "file", "link", "embed", "reaction"}
}

// ResolveFromFilter resolves a "from:" filter value to a user ID
// This is called by the service layer which has access to user lookup
func (p *ParsedSearchQuery) SetAuthorID(id uuid.UUID) {
	p.AuthorID = &id
}

// SetChannelID sets the channel ID for "in:" filter
func (p *ParsedSearchQuery) SetChannelID(id uuid.UUID) {
	p.ChannelID = &id
}

// AddMention adds a mention user ID
func (p *ParsedSearchQuery) AddMention(id uuid.UUID) {
	p.Mentions = append(p.Mentions, id)
}

// ToSearchMessageOptions converts parsed query to SearchMessageOptions
func (p *ParsedSearchQuery) ToSearchMessageOptions() SearchMessageOptions {
	opts := SearchMessageOptions{
		Query:    p.FreeText,
		AuthorID: p.AuthorID,
		Before:   p.Before,
		After:    p.After,
		Pinned:   p.Pinned,
		Mentions: p.Mentions,
	}

	if p.ChannelID != nil {
		opts.ChannelID = p.ChannelID
	}

	// Handle "has" filters
	for _, has := range p.Has {
		switch has {
		case "attachment", "file", "image", "video":
			val := true
			opts.HasAttachments = &val
		case "embed":
			val := true
			opts.HasEmbeds = &val
		case "reaction":
			val := true
			opts.HasReactions = &val
		}
	}

	return opts
}
