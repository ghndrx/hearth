package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestActivityTypes(t *testing.T) {
	if ActivityTypePlaying != 0 {
		t.Errorf("expected ActivityTypePlaying to be 0, got %d", ActivityTypePlaying)
	}
	if ActivityTypeStreaming != 1 {
		t.Errorf("expected ActivityTypeStreaming to be 1, got %d", ActivityTypeStreaming)
	}
	if ActivityTypeListening != 2 {
		t.Errorf("expected ActivityTypeListening to be 2, got %d", ActivityTypeListening)
	}
	if ActivityTypeWatching != 3 {
		t.Errorf("expected ActivityTypeWatching to be 3, got %d", ActivityTypeWatching)
	}
	if ActivityTypeCustom != 4 {
		t.Errorf("expected ActivityTypeCustom to be 4, got %d", ActivityTypeCustom)
	}
	if ActivityTypeCompeting != 5 {
		t.Errorf("expected ActivityTypeCompeting to be 5, got %d", ActivityTypeCompeting)
	}
}

func TestPresenceStruct(t *testing.T) {
	customStatus := "Working on tests"

	presence := Presence{
		UserID:       uuid.New(),
		Status:       StatusOnline,
		CustomStatus: &customStatus,
		Activities:   []Activity{},
		ClientStatus: &ClientStatus{
			Desktop: StatusOnline,
			Mobile:  StatusOffline,
			Web:     StatusOffline,
		},
		UpdatedAt: time.Now(),
	}

	if presence.Status != StatusOnline {
		t.Errorf("expected status 'online', got %s", presence.Status)
	}
	if *presence.CustomStatus != "Working on tests" {
		t.Errorf("expected custom status 'Working on tests', got %s", *presence.CustomStatus)
	}
	if presence.ClientStatus.Desktop != StatusOnline {
		t.Errorf("expected desktop status 'online', got %s", presence.ClientStatus.Desktop)
	}
}

func TestPresenceWithoutCustomStatus(t *testing.T) {
	presence := Presence{
		UserID:       uuid.New(),
		Status:       StatusIdle,
		CustomStatus: nil,
		Activities:   []Activity{},
		UpdatedAt:    time.Now(),
	}

	if presence.CustomStatus != nil {
		t.Error("expected nil custom status")
	}
	if presence.Status != StatusIdle {
		t.Errorf("expected status 'idle', got %s", presence.Status)
	}
}

func TestClientStatus(t *testing.T) {
	cs := ClientStatus{
		Desktop: StatusOnline,
		Mobile:  StatusIdle,
		Web:     StatusDND,
	}

	if cs.Desktop != StatusOnline {
		t.Errorf("expected desktop 'online', got %s", cs.Desktop)
	}
	if cs.Mobile != StatusIdle {
		t.Errorf("expected mobile 'idle', got %s", cs.Mobile)
	}
	if cs.Web != StatusDND {
		t.Errorf("expected web 'dnd', got %s", cs.Web)
	}
}

func TestActivity(t *testing.T) {
	appID := uuid.New()
	now := time.Now()
	end := now.Add(time.Hour)

	activity := Activity{
		Name:          "Coding",
		Type:          ActivityTypePlaying,
		URL:           "https://example.com",
		Details:       "Writing tests",
		State:         "In Progress",
		ApplicationID: &appID,
		Timestamps: &ActivityTime{
			Start: &now,
			End:   &end,
		},
		Assets: &ActivityAssets{
			LargeImage: "code-icon",
			LargeText:  "VS Code",
			SmallImage: "status-icon",
			SmallText:  "Working",
		},
		CreatedAt: now,
	}

	if activity.Name != "Coding" {
		t.Errorf("expected name 'Coding', got %s", activity.Name)
	}
	if activity.Type != ActivityTypePlaying {
		t.Errorf("expected type %d, got %d", ActivityTypePlaying, activity.Type)
	}
	if activity.Details != "Writing tests" {
		t.Errorf("expected details 'Writing tests', got %s", activity.Details)
	}
}

func TestActivityTime(t *testing.T) {
	start := time.Now()
	end := start.Add(2 * time.Hour)

	at := ActivityTime{
		Start: &start,
		End:   &end,
	}

	if at.Start == nil {
		t.Error("expected non-nil start")
	}
	if at.End == nil {
		t.Error("expected non-nil end")
	}
	if at.End.Before(*at.Start) {
		t.Error("end should be after start")
	}
}

func TestActivityTimeStartOnly(t *testing.T) {
	start := time.Now()

	at := ActivityTime{
		Start: &start,
		End:   nil,
	}

	if at.Start == nil {
		t.Error("expected non-nil start")
	}
	if at.End != nil {
		t.Error("expected nil end")
	}
}

func TestActivityAssets(t *testing.T) {
	assets := ActivityAssets{
		LargeImage: "large-image-id",
		LargeText:  "Large Image Description",
		SmallImage: "small-image-id",
		SmallText:  "Small Image Description",
	}

	if assets.LargeImage != "large-image-id" {
		t.Errorf("expected large image 'large-image-id', got %s", assets.LargeImage)
	}
	if assets.SmallText != "Small Image Description" {
		t.Errorf("expected small text 'Small Image Description', got %s", assets.SmallText)
	}
}

func TestVoiceState(t *testing.T) {
	serverID := uuid.New()
	channelID := uuid.New()

	vs := VoiceState{
		UserID:     uuid.New(),
		ServerID:   &serverID,
		ChannelID:  &channelID,
		SessionID:  "session-123",
		Deaf:       false,
		Mute:       false,
		SelfDeaf:   true,
		SelfMute:   true,
		SelfVideo:  false,
		SelfStream: false,
		Suppress:   false,
	}

	if vs.SessionID != "session-123" {
		t.Errorf("expected session ID 'session-123', got %s", vs.SessionID)
	}
	if !vs.SelfDeaf {
		t.Error("expected self deaf to be true")
	}
	if !vs.SelfMute {
		t.Error("expected self mute to be true")
	}
}

func TestVoiceStateWithMember(t *testing.T) {
	member := &Member{
		UserID:   uuid.New(),
		ServerID: uuid.New(),
		Nickname: strPtr("TestMember"),
	}

	serverID := uuid.New()

	vs := VoiceState{
		UserID:    member.UserID,
		ServerID:  &serverID,
		SessionID: "session-456",
		Member:    member,
	}

	if vs.Member == nil {
		t.Error("expected non-nil member")
	}
	if *vs.Member.Nickname != "TestMember" {
		t.Errorf("expected member nickname 'TestMember', got %s", *vs.Member.Nickname)
	}
}

func TestVoiceStateAllFlags(t *testing.T) {
	vs := VoiceState{
		UserID:     uuid.New(),
		SessionID:  "session",
		Deaf:       true,
		Mute:       true,
		SelfDeaf:   true,
		SelfMute:   true,
		SelfVideo:  true,
		SelfStream: true,
		Suppress:   true,
	}

	if !vs.Deaf {
		t.Error("expected deaf to be true")
	}
	if !vs.Mute {
		t.Error("expected mute to be true")
	}
	if !vs.SelfVideo {
		t.Error("expected self video to be true")
	}
	if !vs.SelfStream {
		t.Error("expected self stream to be true")
	}
	if !vs.Suppress {
		t.Error("expected suppress to be true")
	}
}

func TestPresenceWithActivities(t *testing.T) {
	now := time.Now()

	presence := Presence{
		UserID: uuid.New(),
		Status: StatusOnline,
		Activities: []Activity{
			{
				Name:      "Spotify",
				Type:      ActivityTypeListening,
				Details:   "Song Name",
				State:     "Artist Name",
				CreatedAt: now,
			},
			{
				Name:      "Visual Studio Code",
				Type:      ActivityTypePlaying,
				Details:   "Editing main.go",
				CreatedAt: now,
			},
		},
		UpdatedAt: now,
	}

	if len(presence.Activities) != 2 {
		t.Errorf("expected 2 activities, got %d", len(presence.Activities))
	}
	if presence.Activities[0].Name != "Spotify" {
		t.Errorf("expected first activity 'Spotify', got %s", presence.Activities[0].Name)
	}
	if presence.Activities[0].Type != ActivityTypeListening {
		t.Errorf("expected first activity type %d, got %d", ActivityTypeListening, presence.Activities[0].Type)
	}
}

func TestStreamingActivity(t *testing.T) {
	activity := Activity{
		Name: "Playing VALORANT",
		Type: ActivityTypeStreaming,
		URL:  "https://twitch.tv/username",
	}

	if activity.Type != ActivityTypeStreaming {
		t.Errorf("expected type %d, got %d", ActivityTypeStreaming, activity.Type)
	}
	if activity.URL == "" {
		t.Error("streaming activity should have URL")
	}
}

func TestCustomActivity(t *testing.T) {
	activity := Activity{
		Name:  "Custom Status",
		Type:  ActivityTypeCustom,
		State: "😎 Living the dream",
	}

	if activity.Type != ActivityTypeCustom {
		t.Errorf("expected type %d, got %d", ActivityTypeCustom, activity.Type)
	}
	if activity.State != "😎 Living the dream" {
		t.Errorf("expected state '😎 Living the dream', got %s", activity.State)
	}
}
