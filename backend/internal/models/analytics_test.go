package models

import (
	"testing"
)

func TestAnalyticsQueryParamsNormalize(t *testing.T) {
	tests := []struct {
		name      string
		input     AnalyticsQueryParams
		wantDays  int
		wantLimit int
	}{
		{
			name:      "defaults for zero values",
			input:     AnalyticsQueryParams{Days: 0, Limit: 0},
			wantDays:  7,
			wantLimit: 10,
		},
		{
			name:      "defaults for negative values",
			input:     AnalyticsQueryParams{Days: -1, Limit: -5},
			wantDays:  7,
			wantLimit: 10,
		},
		{
			name:      "caps days at 90",
			input:     AnalyticsQueryParams{Days: 100, Limit: 10},
			wantDays:  90,
			wantLimit: 10,
		},
		{
			name:      "caps limit at 50",
			input:     AnalyticsQueryParams{Days: 30, Limit: 100},
			wantDays:  30,
			wantLimit: 50,
		},
		{
			name:      "valid values unchanged",
			input:     AnalyticsQueryParams{Days: 30, Limit: 25},
			wantDays:  30,
			wantLimit: 25,
		},
		{
			name:      "boundary values",
			input:     AnalyticsQueryParams{Days: 90, Limit: 50},
			wantDays:  90,
			wantLimit: 50,
		},
		{
			name:      "minimum valid",
			input:     AnalyticsQueryParams{Days: 1, Limit: 1},
			wantDays:  1,
			wantLimit: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Normalize()
			if tt.input.Days != tt.wantDays {
				t.Errorf("Days = %d, want %d", tt.input.Days, tt.wantDays)
			}
			if tt.input.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", tt.input.Limit, tt.wantLimit)
			}
		})
	}
}
