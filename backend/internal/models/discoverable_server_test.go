package models

import (
	"testing"
)

func TestAllDiscoveryCategories(t *testing.T) {
	cats := AllDiscoveryCategories()
	if len(cats) != 9 {
		t.Errorf("len(AllDiscoveryCategories()) = %d; want 9", len(cats))
	}

	expected := []ServerDiscoveryCategory{
		DiscoveryCategoryGaming,
		DiscoveryCategoryTechnology,
		DiscoveryCategoryArt,
		DiscoveryCategoryMusic,
		DiscoveryCategorySports,
		DiscoveryCategoryEducation,
		DiscoveryCategoryEntertainment,
		DiscoveryCategoryCommunity,
		DiscoveryCategoryOther,
	}

	for i, cat := range cats {
		if cat != expected[i] {
			t.Errorf("AllDiscoveryCategories()[%d] = %q; want %q", i, cat, expected[i])
		}
	}
}

func TestIsValidCategory(t *testing.T) {
	tests := []struct {
		category string
		expected bool
	}{
		{"gaming", true},
		{"technology", true},
		{"art", true},
		{"music", true},
		{"sports", true},
		{"education", true},
		{"entertainment", true},
		{"community", true},
		{"other", true},
		{"invalid", false},
		{"", false},
		{"GAMING", false}, // case-sensitive
		{"Gaming", false}, // case-sensitive
	}

	for _, tc := range tests {
		t.Run(tc.category, func(t *testing.T) {
			result := IsValidCategory(tc.category)
			if result != tc.expected {
				t.Errorf("IsValidCategory(%q) = %v; want %v", tc.category, result, tc.expected)
			}
		})
	}
}

func TestNormalizeDiscoverFilters(t *testing.T) {
	tests := []struct {
		name          string
		inputPage     int
		inputLimit    int
		expectedPage  int
		expectedLimit int
	}{
		{"zero values", 0, 0, 1, 20},
		{"negative page", -1, 10, 1, 10},
		{"negative limit", 1, -5, 1, 20},
		{"limit over 100", 1, 200, 1, 100},
		{"valid values", 5, 50, 5, 50},
		{"limit exactly 100", 1, 100, 1, 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &DiscoverFilters{Page: tc.inputPage, Limit: tc.inputLimit}
			NormalizeDiscoverFilters(f)
			if f.Page != tc.expectedPage {
				t.Errorf("Page = %d; want %d", f.Page, tc.expectedPage)
			}
			if f.Limit != tc.expectedLimit {
				t.Errorf("Limit = %d; want %d", f.Limit, tc.expectedLimit)
			}
		})
	}
}
