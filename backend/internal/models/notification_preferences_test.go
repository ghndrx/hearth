package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestDefaultChannelNotificationPreference(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	pref := DefaultChannelNotificationPreference(userID, channelID, serverID)

	if pref.UserID != userID {
		t.Errorf("UserID = %v; want %v", pref.UserID, userID)
	}
	if pref.ChannelID != channelID {
		t.Errorf("ChannelID = %v; want %v", pref.ChannelID, channelID)
	}
	if pref.ServerID != serverID {
		t.Errorf("ServerID = %v; want %v", pref.ServerID, serverID)
	}
	if !pref.EnableMentions {
		t.Error("EnableMentions should be true")
	}
	if !pref.EnableMessages {
		t.Error("EnableMessages should be true")
	}
	if !pref.EnableReactions {
		t.Error("EnableReactions should be true")
	}
	if !pref.EnableThreads {
		t.Error("EnableThreads should be true")
	}
	if !pref.EnablePins {
		t.Error("EnablePins should be true")
	}
	if !pref.EnableVoiceActivity {
		t.Error("EnableVoiceActivity should be true")
	}
	if pref.DeliveryMode != DeliveryBatched {
		t.Errorf("DeliveryMode = %v; want DeliveryBatched", pref.DeliveryMode)
	}
	if pref.Muted {
		t.Error("Muted should be false")
	}
	if pref.ID != uuid.Nil {
		t.Error("ID should be uuid.Nil")
	}
}

func TestDefaultServerNotificationPreference(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()

	pref := DefaultServerNotificationPreference(userID, serverID)

	if pref.UserID != userID {
		t.Errorf("UserID = %v; want %v", pref.UserID, userID)
	}
	if pref.ServerID != serverID {
		t.Errorf("ServerID = %v; want %v", pref.ServerID, serverID)
	}
	if !pref.EnableMentions {
		t.Error("EnableMentions should be true")
	}
	if !pref.EnableMessages {
		t.Error("EnableMessages should be true")
	}
	if !pref.EnableReactions {
		t.Error("EnableReactions should be true")
	}
	if !pref.EnableThreads {
		t.Error("EnableThreads should be true")
	}
	if pref.Muted {
		t.Error("Muted should be false")
	}
	if pref.ID != uuid.Nil {
		t.Error("ID should be uuid.Nil")
	}
}

func TestContentFilterDataValueAndScan(t *testing.T) {
	original := ContentFilterData{
		Keywords:       []string{"bad", "worst"},
		RegexPatterns:  []string{`\b(spam|scam)\b`},
		Whitelist:     []string{"good"},
		ThresholdValue: 0.75,
		AlertChannelID: strPtr("123456"),
	}

	// Test Value()
	val, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	// Test Scan()
	var scanned ContentFilterData
	err = scanned.Scan(val)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(scanned.Keywords) != len(original.Keywords) {
		t.Errorf("Keywords length = %d; want %d", len(scanned.Keywords), len(original.Keywords))
	}
	for i, k := range scanned.Keywords {
		if k != original.Keywords[i] {
			t.Errorf("Keywords[%d] = %q; want %q", i, k, original.Keywords[i])
		}
	}
	if len(scanned.RegexPatterns) != len(original.RegexPatterns) {
		t.Errorf("RegexPatterns length = %d; want %d", len(scanned.RegexPatterns), len(original.RegexPatterns))
	}
	if scanned.ThresholdValue != original.ThresholdValue {
		t.Errorf("ThresholdValue = %v; want %v", scanned.ThresholdValue, original.ThresholdValue)
	}
	if scanned.AlertChannelID == nil || *scanned.AlertChannelID != *original.AlertChannelID {
		t.Errorf("AlertChannelID = %v; want %v", scanned.AlertChannelID, original.AlertChannelID)
	}
}

func TestContentFilterDataScanNil(t *testing.T) {
	var data ContentFilterData
	err := data.Scan(nil)
	if err != nil {
		t.Errorf("Scan(nil) error = %v; want nil", err)
	}
}

func TestContentFilterDataScanInvalidType(t *testing.T) {
	var data ContentFilterData
	err := data.Scan("not a byte slice")
	if err != nil {
		t.Errorf("Scan(invalid type) error = %v; want nil", err)
	}
}

func TestContentFilterDataScanInvalidJSON(t *testing.T) {
	var data ContentFilterData
	err := data.Scan([]byte("not valid json"))
	if err == nil {
		t.Error("Scan(invalid JSON) error = nil; want error")
	}
}

func TestContentFilterDataValueEmpty(t *testing.T) {
	data := ContentFilterData{}
	val, err := data.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	bytes, ok := val.([]byte)
	if !ok {
		t.Fatalf("Value() type = %T; want []byte", val)
	}
	var roundTrip ContentFilterData
	if err := json.Unmarshal(bytes, &roundTrip); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if len(roundTrip.Keywords) != 0 {
		t.Errorf("Keywords length = %d; want 0", len(roundTrip.Keywords))
	}
}
