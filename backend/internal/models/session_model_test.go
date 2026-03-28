package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSessionToResponse(t *testing.T) {
	now := time.Now()
	lastUsed := now.Add(-1 * time.Hour)
	deviceName := "Test Device"
	browser := "Chrome"
	os := "Windows"
	browserVersion := "120"
	osVersion := "11"
	locationCity := "New York"
	locationCountry := "US"

	session := &Session{
		ID:              uuid.New(),
		DeviceName:      &deviceName,
		DeviceType:      DeviceTypeDesktop,
		Browser:         &browser,
		BrowserVersion:  &browserVersion,
		OS:              &os,
		OSVersion:       &osVersion,
		LocationCity:    &locationCity,
		LocationCountry: &locationCountry,
		IsCurrent:       true,
		LastUsed:        &lastUsed,
		CreatedAt:       now,
	}

	resp := session.ToResponse()

	if resp.ID != session.ID {
		t.Errorf("expected ID %v, got %v", session.ID, resp.ID)
	}
	if resp.DeviceName != deviceName {
		t.Errorf("expected DeviceName %s, got %s", deviceName, resp.DeviceName)
	}
	if resp.DeviceType != DeviceTypeDesktop {
		t.Errorf("expected DeviceType %s, got %s", DeviceTypeDesktop, resp.DeviceType)
	}
	if resp.Browser != browser {
		t.Errorf("expected Browser %s, got %s", browser, resp.Browser)
	}
	if resp.BrowserVersion != browserVersion {
		t.Errorf("expected BrowserVersion %s, got %s", browserVersion, resp.BrowserVersion)
	}
	if resp.OS != os {
		t.Errorf("expected OS %s, got %s", os, resp.OS)
	}
	if resp.OSVersion != osVersion {
		t.Errorf("expected OSVersion %s, got %s", osVersion, resp.OSVersion)
	}
	if resp.LocationCity != "New York" {
		t.Errorf("expected LocationCity New York, got %s", resp.LocationCity)
	}
	if resp.LocationCountry != "US" {
		t.Errorf("expected LocationCountry US, got %s", resp.LocationCountry)
	}
	if !resp.IsCurrent {
		t.Error("expected IsCurrent true")
	}
	if resp.LastUsed == nil || !resp.LastUsed.Equal(lastUsed) {
		t.Errorf("expected LastUsed %v, got %v", lastUsed, resp.LastUsed)
	}
}

func TestSessionToResponseUnknownDevice(t *testing.T) {
	session := &Session{
		ID:         uuid.New(),
		DeviceName: nil,
		DeviceType: DeviceTypeMobile,
		IsCurrent:  false,
		CreatedAt:  time.Now(),
	}

	resp := session.ToResponse()

	// Should default to "Unknown Device"
	if resp.DeviceName != "Unknown Device" {
		t.Errorf("expected DeviceName 'Unknown Device', got %s", resp.DeviceName)
	}
}

func TestSessionToResponseWithBrowserAndOS(t *testing.T) {
	browser := "Firefox"
	os := "macOS"
	session := &Session{
		ID:         uuid.New(),
		DeviceName: nil,
		Browser:    &browser,
		OS:         &os,
		DeviceType: DeviceTypeDesktop,
		IsCurrent:  true,
		CreatedAt:  time.Now(),
	}

	resp := session.ToResponse()

	expected := browser + " on " + os
	if resp.DeviceName != expected {
		t.Errorf("expected DeviceName %q, got %q", expected, resp.DeviceName)
	}
}

func TestRefreshTokenIsValid(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	now := time.Now()

	tests := []struct {
		name      string
		used      bool
		revoked   bool
		expiresAt time.Time
		expected  bool
	}{
		{"valid token", false, false, future, true},
		{"used token", true, false, future, false},
		{"revoked token", false, true, future, false},
		{"expired token", false, false, now.Add(-1 * time.Second), false},
		{"used and revoked", true, true, future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &RefreshToken{
				ID:        uuid.New(),
				Used:      tt.used,
				Revoked:   tt.revoked,
				ExpiresAt: tt.expiresAt,
			}
			if got := rt.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHashToken(t *testing.T) {
	token := "test-token-12345"
	hash := HashToken(token)

	// SHA-256 produces 64 hex characters
	if len(hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash))
	}

	// Same token should produce same hash
	hash2 := HashToken(token)
	if hash != hash2 {
		t.Errorf("expected same hash for same token, got %s and %s", hash, hash2)
	}

	// Different token should produce different hash
	hash3 := HashToken("different-token")
	if hash == hash3 {
		t.Errorf("expected different hash for different token")
	}
}

func TestGenerateTokenFamily(t *testing.T) {
	id1 := GenerateTokenFamily()
	id2 := GenerateTokenFamily()

	if id1 == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if id1 == id2 {
		t.Error("expected different UUIDs for each call")
	}
}

func TestWebSocketConnectionSend(t *testing.T) {
	conn := &WebSocketConnection{
		Out:  make(chan []byte, 256),
		Done: make(chan struct{}),
	}

	data := json.RawMessage(`{"type":"TEST"}`)
	conn.Send(data)

	select {
	case msg := <-conn.Out:
		if string(msg) != `{"type":"TEST"}` {
			t.Errorf("expected %q, got %q", `{"type":"TEST"}`, string(msg))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected message to be sent")
	}
}

func TestWebSocketConnectionSendDropOnFullChannel(t *testing.T) {
	conn := &WebSocketConnection{
		Out:  make(chan []byte, 1), // Buffer of 1
		Done: make(chan struct{}),
	}

	// Fill the channel
	data1 := json.RawMessage(`{"type":"FIRST"}`)
	conn.Send(data1)

	// This should not block (non-blocking send)
	data2 := json.RawMessage(`{"type":"SECOND"}`)
	conn.Send(data2) // Should drop silently

	// Only one message should be in the channel
	select {
	case msg := <-conn.Out:
		if string(msg) != `{"type":"FIRST"}` {
			t.Errorf("expected FIRST, got %s", string(msg))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected FIRST message")
	}

	// Drain and verify no second message
	select {
	case msg := <-conn.Out:
		t.Errorf("unexpected second message: %s", string(msg))
	case <-time.After(50 * time.Millisecond):
		// Expected - no second message
	}
}

func TestWebSocketConnectionClose(t *testing.T) {
	conn := &WebSocketConnection{
		Out:  make(chan []byte, 256),
		Done: make(chan struct{}),
	}

	err := conn.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Done should be closed
	select {
	case <-conn.Done:
		// Expected
	default:
		t.Error("expected Done channel to be closed")
	}
}

func TestNewWebSocketConnection(t *testing.T) {
	conn := NewWebSocketConnection(nil)

	if conn.Out == nil {
		t.Error("expected Out channel to be initialized")
	}
	if conn.Done == nil {
		t.Error("expected Done channel to be initialized")
	}
	if cap(conn.Out) != 256 {
		t.Errorf("expected Out channel capacity 256, got %d", cap(conn.Out))
	}
}

func TestStatusNow(t *testing.T) {
	before := time.Now().Add(-1 * time.Millisecond)
	now := Now()
	after := time.Now().Add(1 * time.Millisecond)

	if now.Before(before) || now.After(after) {
		t.Errorf("Now() = %v, expected time within [ %v, %v ]", now, before, after)
	}
}
