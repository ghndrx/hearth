package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateFrequency(t *testing.T) {
	tests := []struct {
		freq     DigestFrequency
		expected bool
	}{
		{DigestFrequencyHourly, true},
		{DigestFrequencyDaily, true},
		{DigestFrequencyWeekly, true},
		{"invalid", false},
		{"", false},
		{"monthly", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.freq), func(t *testing.T) {
			if got := ValidateFrequency(tt.freq); got != tt.expected {
				t.Errorf("ValidateFrequency(%q) = %v, want %v", tt.freq, got, tt.expected)
			}
		})
	}
}

func TestValidateAggregationMode(t *testing.T) {
	tests := []struct {
		mode     DigestAggregationMode
		expected bool
	}{
		{DigestAggregationChannel, true},
		{DigestAggregationServer, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := ValidateAggregationMode(tt.mode); got != tt.expected {
				t.Errorf("ValidateAggregationMode(%q) = %v, want %v", tt.mode, got, tt.expected)
			}
		})
	}
}

func TestValidateDigestMode(t *testing.T) {
	tests := []struct {
		mode     DigestMode
		expected bool
	}{
		{DigestModeInherit, true},
		{DigestModeInclude, true},
		{DigestModeExclude, true},
		{DigestModeImmediate, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := ValidateDigestMode(tt.mode); got != tt.expected {
				t.Errorf("ValidateDigestMode(%q) = %v, want %v", tt.mode, got, tt.expected)
			}
		})
	}
}

func TestDefaultDigestPreferences(t *testing.T) {
	userID := uuid.New()
	prefs := DefaultDigestPreferences(userID)

	if prefs.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, prefs.UserID)
	}
	if prefs.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if prefs.Enabled {
		t.Error("expected Enabled to be false by default")
	}
	if prefs.Frequency != DigestFrequencyDaily {
		t.Errorf("expected Frequency 'daily', got %s", prefs.Frequency)
	}
	if prefs.PreferredHour != 9 {
		t.Errorf("expected PreferredHour 9, got %d", prefs.PreferredHour)
	}
	if prefs.PreferredDay != 1 {
		t.Errorf("expected PreferredDay 1, got %d", prefs.PreferredDay)
	}
	if prefs.AggregationMode != DigestAggregationServer {
		t.Errorf("expected AggregationMode 'server', got %s", prefs.AggregationMode)
	}
	if prefs.MaxMessagesPerSource != 50 {
		t.Errorf("expected MaxMessagesPerSource 50, got %d", prefs.MaxMessagesPerSource)
	}
	if !prefs.MutedChannelsOnly {
		t.Error("expected MutedChannelsOnly true")
	}
	if prefs.Timezone != "UTC" {
		t.Errorf("expected Timezone 'UTC', got %s", prefs.Timezone)
	}
}

func TestDigestContentToJSON(t *testing.T) {
	content := &DigestContent{
		Period: DigestPeriodInfo{
			Start:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:       time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Frequency: DigestFrequencyDaily,
		},
		TotalStats: DigestStats{
			MessageCount: 42,
			MentionCount: 3,
		},
		Servers: []DigestServerSummary{
			{
				ServerID:   uuid.New(),
				ServerName: "Test Server",
				Stats:      DigestStats{MessageCount: 42, MentionCount: 3},
			},
		},
	}

	jsonStr, err := content.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}
	if jsonStr == "" {
		t.Fatal("ToJSON() returned empty string")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("ToJSON() produced invalid JSON: %v", err)
	}
}

func TestParseDigestContent(t *testing.T) {
	original := &DigestContent{
		Period: DigestPeriodInfo{
			Start:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:       time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Frequency: DigestFrequencyWeekly,
		},
		TotalStats: DigestStats{
			MessageCount: 100,
			MentionCount: 5,
		},
	}

	jsonStr, _ := original.ToJSON()

	parsed, err := ParseDigestContent(jsonStr)
	if err != nil {
		t.Fatalf("ParseDigestContent() error: %v", err)
	}
	if parsed.TotalStats.MessageCount != 100 {
		t.Errorf("expected MessageCount 100, got %d", parsed.TotalStats.MessageCount)
	}
	if parsed.TotalStats.MentionCount != 5 {
		t.Errorf("expected MentionCount 5, got %d", parsed.TotalStats.MentionCount)
	}
	if parsed.Period.Frequency != DigestFrequencyWeekly {
		t.Errorf("expected Frequency 'weekly', got %s", parsed.Period.Frequency)
	}
}

func TestParseDigestContentInvalid(t *testing.T) {
	_, err := ParseDigestContent("not valid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseDigestContentEmpty(t *testing.T) {
	parsed, err := ParseDigestContent("{}")
	if err != nil {
		t.Fatalf("ParseDigestContent() error: %v", err)
	}
	if parsed.TotalStats.MessageCount != 0 {
		t.Errorf("expected 0 messages, got %d", parsed.TotalStats.MessageCount)
	}
}

func TestDigestContentRoundTrip(t *testing.T) {
	serverID := uuid.New()
	channelID := uuid.New()
	msgID := uuid.New()
	authorID := uuid.New()

	original := &DigestContent{
		Period: DigestPeriodInfo{
			Start:     time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			End:       time.Date(2024, 6, 2, 0, 0, 0, 0, time.UTC),
			Frequency: DigestFrequencyHourly,
		},
		Servers: []DigestServerSummary{
			{
				ServerID:   serverID,
				ServerName: "My Server",
				Channels: []DigestChannelSummary{
					{
						ChannelID:   channelID,
						ChannelName: "general",
						Messages: []DigestMessageSummary{
							{
								MessageID:  &msgID,
								AuthorID:   &authorID,
								AuthorName: "alice",
								Content:    "hello",
								IsMention:  true,
								CreatedAt:  time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
							},
						},
						Stats: DigestStats{MessageCount: 1, MentionCount: 1},
					},
				},
				Stats: DigestStats{MessageCount: 1, MentionCount: 1},
			},
		},
		TotalStats: DigestStats{MessageCount: 1, MentionCount: 1},
	}

	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	parsed, err := ParseDigestContent(jsonStr)
	if err != nil {
		t.Fatalf("ParseDigestContent() error: %v", err)
	}

	if len(parsed.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(parsed.Servers))
	}
	if parsed.Servers[0].ServerName != "My Server" {
		t.Errorf("expected server name 'My Server', got %s", parsed.Servers[0].ServerName)
	}
	if len(parsed.Servers[0].Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(parsed.Servers[0].Channels))
	}
	if parsed.Servers[0].Channels[0].Messages[0].AuthorName != "alice" {
		t.Errorf("expected author 'alice', got %s", parsed.Servers[0].Channels[0].Messages[0].AuthorName)
	}
	if !parsed.Servers[0].Channels[0].Messages[0].IsMention {
		t.Error("expected IsMention true")
	}
}
