package services

import (
	"testing"
	"time"
)

func TestParseSearchQueryString(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantText string
		wantFrom string
		wantIn   string
		wantHas  []string
		wantPinned *bool
	}{
		{
			name:     "simple text query",
			query:    "hello world",
			wantText: "hello world",
		},
		{
			name:     "from filter",
			query:    "from:johndoe hello",
			wantText: "hello",
			wantFrom: "johndoe",
		},
		{
			name:     "from filter with @ symbol",
			query:    "from:@johndoe test message",
			wantText: "test message",
			wantFrom: "johndoe",
		},
		{
			name:     "in filter",
			query:    "in:general my search",
			wantText: "my search",
			wantIn:   "general",
		},
		{
			name:     "in filter with # symbol",
			query:    "in:#announcements important",
			wantText: "important",
			wantIn:   "announcements",
		},
		{
			name:     "has attachment filter",
			query:    "has:attachment project files",
			wantText: "project files",
			wantHas:  []string{"attachment"},
		},
		{
			name:     "has image filter",
			query:    "has:image cat pictures",
			wantText: "cat pictures",
			wantHas:  []string{"image"},
		},
		{
			name:     "has link filter",
			query:    "has:link useful resources",
			wantText: "useful resources",
			wantHas:  []string{"link"},
		},
		{
			name:     "multiple has filters",
			query:    "has:attachment has:image photos",
			wantText: "photos",
			wantHas:  []string{"attachment", "image"},
		},
		{
			name:     "pinned true filter",
			query:    "pinned:true important",
			wantText: "important",
			wantPinned: boolPtr(true),
		},
		{
			name:     "pinned false filter",
			query:    "pinned:false regular",
			wantText: "regular",
			wantPinned: boolPtr(false),
		},
		{
			name:     "combined filters",
			query:    "from:@alice in:#general has:attachment budget report",
			wantText: "budget report",
			wantFrom: "alice",
			wantIn:   "general",
			wantHas:  []string{"attachment"},
		},
		{
			name:     "case insensitive filters",
			query:    "FROM:Bob IN:#Help HAS:image screenshot",
			wantText: "screenshot",
			wantFrom: "Bob",
			wantIn:   "Help",
			wantHas:  []string{"image"},
		},
		{
			name:     "empty query",
			query:    "",
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSearchQueryString(tt.query)

			if result.FreeText != tt.wantText {
				t.Errorf("FreeText = %q, want %q", result.FreeText, tt.wantText)
			}

			// Check from filter
			fromToken := findToken(result.Tokens, "from")
			if tt.wantFrom != "" {
				if fromToken == nil {
					t.Errorf("Expected from token with value %q, got none", tt.wantFrom)
				} else if fromToken.Value != tt.wantFrom {
					t.Errorf("from token value = %q, want %q", fromToken.Value, tt.wantFrom)
				}
			}

			// Check in filter
			inToken := findToken(result.Tokens, "in")
			if tt.wantIn != "" {
				if inToken == nil {
					t.Errorf("Expected in token with value %q, got none", tt.wantIn)
				} else if inToken.Value != tt.wantIn {
					t.Errorf("in token value = %q, want %q", inToken.Value, tt.wantIn)
				}
			}

			// Check has filters
			if len(tt.wantHas) > 0 {
				if len(result.Has) != len(tt.wantHas) {
					t.Errorf("Has count = %d, want %d", len(result.Has), len(tt.wantHas))
				}
				for i, want := range tt.wantHas {
					if i < len(result.Has) && result.Has[i] != want {
						t.Errorf("Has[%d] = %q, want %q", i, result.Has[i], want)
					}
				}
			}

			// Check pinned filter
			if tt.wantPinned != nil {
				if result.Pinned == nil {
					t.Errorf("Expected Pinned = %v, got nil", *tt.wantPinned)
				} else if *result.Pinned != *tt.wantPinned {
					t.Errorf("Pinned = %v, want %v", *result.Pinned, *tt.wantPinned)
				}
			}
		})
	}
}

func TestParseSearchQueryStringWithDates(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantBefore string
		wantAfter  string
	}{
		{
			name:       "before date",
			query:      "before:2024-01-15 old messages",
			wantBefore: "2024-01-15",
		},
		{
			name:      "after date",
			query:     "after:2024-06-01 recent posts",
			wantAfter: "2024-06-01",
		},
		{
			name:       "date range",
			query:      "after:2024-01-01 before:2024-12-31 this year",
			wantBefore: "2024-12-31",
			wantAfter:  "2024-01-01",
		},
		{
			name:       "ISO8601 datetime",
			query:      "before:2024-06-15T14:30:00Z meeting notes",
			wantBefore: "2024-06-15T14:30:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSearchQueryString(tt.query)

			if tt.wantBefore != "" {
				if result.Before == nil {
					t.Errorf("Expected Before date, got nil")
				} else {
					// Verify date was parsed
					expectedBefore, _ := parseFlexibleDate(tt.wantBefore)
					if !result.Before.Equal(expectedBefore) {
						t.Errorf("Before = %v, want %v", *result.Before, expectedBefore)
					}
				}
			}

			if tt.wantAfter != "" {
				if result.After == nil {
					t.Errorf("Expected After date, got nil")
				} else {
					expectedAfter, _ := parseFlexibleDate(tt.wantAfter)
					if !result.After.Equal(expectedAfter) {
						t.Errorf("After = %v, want %v", *result.After, expectedAfter)
					}
				}
			}
		})
	}
}

func TestToSearchMessageOptions(t *testing.T) {
	t.Run("converts parsed query to options", func(t *testing.T) {
		parsed := &ParsedSearchQuery{
			FreeText: "test search",
			Has:      []string{"attachment", "embed"},
			Pinned:   boolPtr(true),
		}
		
		before := time.Now().Add(-24 * time.Hour)
		after := time.Now().Add(-7 * 24 * time.Hour)
		parsed.Before = &before
		parsed.After = &after

		opts := parsed.ToSearchMessageOptions()

		if opts.Query != "test search" {
			t.Errorf("Query = %q, want %q", opts.Query, "test search")
		}

		if opts.HasAttachments == nil || !*opts.HasAttachments {
			t.Error("Expected HasAttachments to be true")
		}

		if opts.HasEmbeds == nil || !*opts.HasEmbeds {
			t.Error("Expected HasEmbeds to be true")
		}

		if opts.Pinned == nil || !*opts.Pinned {
			t.Error("Expected Pinned to be true")
		}

		if opts.Before == nil || !opts.Before.Equal(before) {
			t.Error("Before date not set correctly")
		}

		if opts.After == nil || !opts.After.Equal(after) {
			t.Error("After date not set correctly")
		}
	})
}

func TestValidHasValues(t *testing.T) {
	values := ValidHasValues()
	
	expected := []string{"attachment", "image", "video", "file", "link", "embed", "reaction"}
	
	if len(values) != len(expected) {
		t.Errorf("ValidHasValues() returned %d values, want %d", len(values), len(expected))
	}
	
	for i, v := range expected {
		if i < len(values) && values[i] != v {
			t.Errorf("ValidHasValues()[%d] = %q, want %q", i, values[i], v)
		}
	}
}

// Helper functions
func findToken(tokens []SearchQueryToken, tokenType string) *SearchQueryToken {
	for _, token := range tokens {
		if token.Type == tokenType {
			return &token
		}
	}
	return nil
}

func boolPtr(b bool) *bool {
	return &b
}
