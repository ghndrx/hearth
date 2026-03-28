package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestAutoModTriggerValueAndScan(t *testing.T) {
	trigger := AutoModTrigger{
		Keywords:     []string{"bad", "word"},
		MentionLimit: 5,
		Whitelist:    []string{"allowed"},
	}

	val, err := trigger.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var scanned AutoModTrigger
	err = scanned.Scan(val.([]byte))
	if err != nil {
		t.Fatalf("unexpected error scanning: %v", err)
	}
	if len(scanned.Keywords) != 2 || scanned.Keywords[0] != "bad" {
		t.Errorf("unexpected keywords: %v", scanned.Keywords)
	}
	if scanned.MentionLimit != 5 {
		t.Errorf("expected mention limit 5, got %d", scanned.MentionLimit)
	}

	// Scan nil
	err = scanned.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected error scanning nil: %v", err)
	}

	// Scan non-byte type
	err = scanned.Scan("not bytes")
	if err != nil {
		t.Fatalf("unexpected error scanning non-bytes: %v", err)
	}
}

func TestAutoModActionMetadataValueAndScan(t *testing.T) {
	channelID := uuid.New()
	alertMsg := "Alert!"
	meta := AutoModActionMetadata{
		ChannelID:    &channelID,
		AlertMessage: &alertMsg,
	}

	val, err := meta.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var scanned AutoModActionMetadata
	err = scanned.Scan(val.([]byte))
	if err != nil {
		t.Fatalf("unexpected error scanning: %v", err)
	}
	if scanned.ChannelID == nil || *scanned.ChannelID != channelID {
		t.Errorf("unexpected channel ID: %v", scanned.ChannelID)
	}
	if scanned.AlertMessage == nil || *scanned.AlertMessage != "Alert!" {
		t.Errorf("unexpected alert message: %v", scanned.AlertMessage)
	}

	// Scan nil
	err = scanned.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected error scanning nil: %v", err)
	}

	// Scan non-byte type
	err = scanned.Scan(123)
	if err != nil {
		t.Fatalf("unexpected error scanning non-bytes: %v", err)
	}
}

func TestAutoModActionsValueAndScan(t *testing.T) {
	actions := AutoModActions{
		{Type: ActionBlockMessage},
		{Type: ActionFlagToModerators},
	}

	val, err := actions.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var scanned AutoModActions
	err = scanned.Scan(val.([]byte))
	if err != nil {
		t.Fatalf("unexpected error scanning: %v", err)
	}
	if len(scanned) != 2 {
		t.Errorf("expected 2 actions, got %d", len(scanned))
	}
	if scanned[0].Type != ActionBlockMessage {
		t.Errorf("expected ActionBlockMessage, got %d", scanned[0].Type)
	}

	// Scan nil
	err = scanned.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected error scanning nil: %v", err)
	}

	// Scan non-byte type
	err = scanned.Scan("not bytes")
	if err != nil {
		t.Fatalf("unexpected error scanning non-bytes: %v", err)
	}
}

func TestAutoModTriggerJSON(t *testing.T) {
	trigger := AutoModTrigger{
		Keywords:      []string{"spam"},
		RegexPatterns: []string{`\bfree\b`},
		MLCategories:  []string{"toxicity"},
	}

	data, err := json.Marshal(trigger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed AutoModTrigger
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed.RegexPatterns) != 1 || parsed.RegexPatterns[0] != `\bfree\b` {
		t.Errorf("unexpected regex patterns: %v", parsed.RegexPatterns)
	}
}
