package models

import "testing"

func TestAppCategoryString(t *testing.T) {
	tests := []struct {
		category AppCategory
		expected string
	}{
		{AppCategoryModeration, "moderation"},
		{AppCategoryMusic, "music"},
		{AppCategoryGaming, "gaming"},
		{AppCategoryUtility, "utility"},
		{AppCategoryFun, "fun"},
		{AppCategoryEducation, "education"},
		{AppCategoryRoleplay, "roleplay"},
		{AppCategoryEconomy, "economy"},
		{AppCategory(100), "unknown"}, // invalid category
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.category.String()
			if result != tc.expected {
				t.Errorf("AppCategory(%d).String() = %q; want %q", tc.category, result, tc.expected)
			}
		})
	}
}

func TestParseAppCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected AppCategory
		ok       bool
	}{
		{"moderation", AppCategoryModeration, true},
		{"music", AppCategoryMusic, true},
		{"gaming", AppCategoryGaming, true},
		{"utility", AppCategoryUtility, true},
		{"fun", AppCategoryFun, true},
		{"education", AppCategoryEducation, true},
		{"roleplay", AppCategoryRoleplay, true},
		{"economy", AppCategoryEconomy, true},
		{"unknown", 0, false},
		{"", 0, false},
		{"MODERATION", 0, false}, // case-sensitive
		{"Moderation", 0, false}, // case-sensitive
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, ok := ParseAppCategory(tc.input)
			if ok != tc.ok {
				t.Errorf("ParseAppCategory(%q) ok = %v; want %v", tc.input, ok, tc.ok)
			}
			if result != tc.expected {
				t.Errorf("ParseAppCategory(%q) = %v; want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestAppCategoryNames(t *testing.T) {
	if len(AppCategoryNames) != 8 {
		t.Errorf("len(AppCategoryNames) = %d; want 8", len(AppCategoryNames))
	}

	expected := []string{"moderation", "music", "gaming", "utility", "fun", "education", "roleplay", "economy"}
	for i, name := range AppCategoryNames {
		if name != expected[i] {
			t.Errorf("AppCategoryNames[%d] = %q; want %q", i, name, expected[i])
		}
	}
}

func TestAppStatusString(t *testing.T) {
	tests := []struct {
		status   AppStatus
		expected string
	}{
		{AppStatusPending, "pending"},
		{AppStatusApproved, "approved"},
		{AppStatusRejected, "rejected"},
		{AppStatusSuspended, "suspended"},
		{AppStatus(100), "unknown"}, // invalid status
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.status.String()
			if result != tc.expected {
				t.Errorf("AppStatus(%d).String() = %q; want %q", tc.status, result, tc.expected)
			}
		})
	}
}
