package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserToPublic(t *testing.T) {
	avatarURL := "https://example.com/avatar.png"
	bannerURL := "https://example.com/banner.png"
	bio := "Test bio"
	customStatus := "Testing"

	user := &User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		Username:      "testuser",
		Discriminator: "1234",
		PasswordHash:  "secret-hash",
		AvatarURL:     &avatarURL,
		BannerURL:     &bannerURL,
		Bio:           &bio,
		Status:        StatusOnline,
		CustomStatus:  &customStatus,
		MFAEnabled:    true,
		MFASecret:     nil,
		Verified:      true,
		Flags:         UserFlagStaff | UserFlagPremium,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	public := user.ToPublic()

	// Verify public fields are copied
	if public.ID != user.ID {
		t.Errorf("expected ID %v, got %v", user.ID, public.ID)
	}
	if public.Username != user.Username {
		t.Errorf("expected Username %s, got %s", user.Username, public.Username)
	}
	if public.Discriminator != user.Discriminator {
		t.Errorf("expected Discriminator %s, got %s", user.Discriminator, public.Discriminator)
	}
	if *public.AvatarURL != *user.AvatarURL {
		t.Errorf("expected AvatarURL %s, got %s", *user.AvatarURL, *public.AvatarURL)
	}
	if *public.BannerURL != *user.BannerURL {
		t.Errorf("expected BannerURL %s, got %s", *user.BannerURL, *public.BannerURL)
	}
	if *public.Bio != *user.Bio {
		t.Errorf("expected Bio %s, got %s", *user.Bio, *public.Bio)
	}
	if public.Status != user.Status {
		t.Errorf("expected Status %s, got %s", user.Status, public.Status)
	}
	if *public.CustomStatus != *user.CustomStatus {
		t.Errorf("expected CustomStatus %s, got %s", *user.CustomStatus, *public.CustomStatus)
	}
	if public.Flags != user.Flags {
		t.Errorf("expected Flags %d, got %d", user.Flags, public.Flags)
	}
}

func TestUserTag(t *testing.T) {
	user := &User{
		Username:      "testuser",
		Discriminator: "1234",
	}

	tag := user.Tag()
	expected := "testuser#1234"

	if tag != expected {
		t.Errorf("expected tag %s, got %s", expected, tag)
	}
}

func TestPresenceStatus(t *testing.T) {
	statuses := []PresenceStatus{
		StatusOnline,
		StatusIdle,
		StatusDND,
		StatusInvisible,
		StatusOffline,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("presence status constant is empty")
		}
	}

	// Verify expected values
	if StatusOnline != "online" {
		t.Errorf("expected 'online', got '%s'", StatusOnline)
	}
	if StatusIdle != "idle" {
		t.Errorf("expected 'idle', got '%s'", StatusIdle)
	}
	if StatusDND != "dnd" {
		t.Errorf("expected 'dnd', got '%s'", StatusDND)
	}
	if StatusInvisible != "invisible" {
		t.Errorf("expected 'invisible', got '%s'", StatusInvisible)
	}
	if StatusOffline != "offline" {
		t.Errorf("expected 'offline', got '%s'", StatusOffline)
	}
}

func TestUserFlags(t *testing.T) {
	// Test flag bit positions
	if UserFlagStaff != 1<<0 {
		t.Errorf("UserFlagStaff should be 1<<0")
	}
	if UserFlagPartner != 1<<1 {
		t.Errorf("UserFlagPartner should be 1<<1")
	}
	if UserFlagBugHunter != 1<<2 {
		t.Errorf("UserFlagBugHunter should be 1<<2")
	}
	if UserFlagPremium != 1<<3 {
		t.Errorf("UserFlagPremium should be 1<<3")
	}
	if UserFlagSystemBot != 1<<4 {
		t.Errorf("UserFlagSystemBot should be 1<<4")
	}
	if UserFlagDeletedUser != 1<<5 {
		t.Errorf("UserFlagDeletedUser should be 1<<5")
	}

	// Test combining flags
	combined := UserFlagStaff | UserFlagPremium
	if combined&UserFlagStaff == 0 {
		t.Error("combined flags should include Staff")
	}
	if combined&UserFlagPremium == 0 {
		t.Error("combined flags should include Premium")
	}
	if combined&UserFlagPartner != 0 {
		t.Error("combined flags should not include Partner")
	}
}

func TestUserToPublicWithNilFields(t *testing.T) {
	user := &User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		Username:      "testuser",
		Discriminator: "1234",
		PasswordHash:  "secret",
		AvatarURL:     nil,
		BannerURL:     nil,
		Bio:           nil,
		Status:        StatusOffline,
		CustomStatus:  nil,
		Flags:         0,
	}

	public := user.ToPublic()

	if public.AvatarURL != nil {
		t.Error("expected nil AvatarURL")
	}
	if public.BannerURL != nil {
		t.Error("expected nil BannerURL")
	}
	if public.Bio != nil {
		t.Error("expected nil Bio")
	}
	if public.CustomStatus != nil {
		t.Error("expected nil CustomStatus")
	}
}
