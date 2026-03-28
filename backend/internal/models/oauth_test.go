package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOAuthProviderToResponse(t *testing.T) {
	username := "testuser"
	displayName := "Test User"
	avatarURL := "https://example.com/avatar.png"
	accessToken := "secret-access-token"
	refreshToken := "secret-refresh-token"
	expiresAt := time.Now().Add(time.Hour)

	provider := &OAuthProvider{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		Provider:       "github",
		ProviderUserID: "12345",
		Email:          "test@example.com",
		Username:       &username,
		DisplayName:    &displayName,
		AvatarURL:      &avatarURL,
		AccessToken:    &accessToken,
		RefreshToken:   &refreshToken,
		TokenExpiresAt: &expiresAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	resp := provider.ToResponse()

	if resp.ID != provider.ID {
		t.Errorf("expected ID %v, got %v", provider.ID, resp.ID)
	}
	if resp.Provider != "github" {
		t.Errorf("expected Provider 'github', got %s", resp.Provider)
	}
	if resp.ProviderUserID != "12345" {
		t.Errorf("expected ProviderUserID '12345', got %s", resp.ProviderUserID)
	}
	if resp.Email != "test@example.com" {
		t.Errorf("expected Email 'test@example.com', got %s", resp.Email)
	}
	if resp.Username == nil || *resp.Username != username {
		t.Errorf("expected Username %s, got %v", username, resp.Username)
	}
	if resp.DisplayName == nil || *resp.DisplayName != displayName {
		t.Errorf("expected DisplayName %s, got %v", displayName, resp.DisplayName)
	}
	if resp.AvatarURL == nil || *resp.AvatarURL != avatarURL {
		t.Errorf("expected AvatarURL %s, got %v", avatarURL, resp.AvatarURL)
	}
}

func TestOAuthProviderToResponseNilOptionals(t *testing.T) {
	provider := &OAuthProvider{
		ID:             uuid.New(),
		Provider:       "google",
		ProviderUserID: "67890",
		Email:          "user@gmail.com",
		Username:       nil,
		DisplayName:    nil,
		AvatarURL:      nil,
		CreatedAt:      time.Now(),
	}

	resp := provider.ToResponse()

	if resp.Username != nil {
		t.Error("expected nil Username")
	}
	if resp.DisplayName != nil {
		t.Error("expected nil DisplayName")
	}
	if resp.AvatarURL != nil {
		t.Error("expected nil AvatarURL")
	}
}

func TestOAuthAppToResponse(t *testing.T) {
	desc := "Test app"
	iconURL := "https://example.com/icon.png"
	homepageURL := "https://example.com"

	app := &OAuthApp{
		ID:           uuid.New(),
		OwnerID:      uuid.New(),
		Name:         "Test App",
		Description:  &desc,
		ClientID:     "client-123",
		ClientSecret: "hashed-secret",
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"read", "write"},
		IconURL:      &iconURL,
		HomepageURL:  &homepageURL,
		IsPublic:     true,
		IsVerified:   false,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	resp := app.ToResponse()

	if resp.ID != app.ID {
		t.Errorf("expected ID %v, got %v", app.ID, resp.ID)
	}
	if resp.Name != "Test App" {
		t.Errorf("expected Name 'Test App', got %s", resp.Name)
	}
	if resp.ClientID != "client-123" {
		t.Errorf("expected ClientID 'client-123', got %s", resp.ClientID)
	}
	if resp.Description == nil || *resp.Description != desc {
		t.Error("expected Description to be set")
	}
	if len(resp.RedirectURIs) != 1 || resp.RedirectURIs[0] != "https://example.com/callback" {
		t.Errorf("unexpected RedirectURIs: %v", resp.RedirectURIs)
	}
	if len(resp.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(resp.Scopes))
	}
	if !resp.IsPublic {
		t.Error("expected IsPublic true")
	}
	if resp.IsVerified {
		t.Error("expected IsVerified false")
	}
	if resp.IconURL == nil || *resp.IconURL != iconURL {
		t.Error("expected IconURL to be set")
	}
	if resp.HomepageURL == nil || *resp.HomepageURL != homepageURL {
		t.Error("expected HomepageURL to be set")
	}
}

func TestOAuthAppToResponseNilOptionals(t *testing.T) {
	app := &OAuthApp{
		ID:           uuid.New(),
		Name:         "Minimal App",
		ClientID:     "client-min",
		RedirectURIs: []string{},
		Scopes:       []string{},
		CreatedAt:    time.Now(),
	}

	resp := app.ToResponse()

	if resp.Description != nil {
		t.Error("expected nil Description")
	}
	if resp.IconURL != nil {
		t.Error("expected nil IconURL")
	}
	if resp.HomepageURL != nil {
		t.Error("expected nil HomepageURL")
	}
}

func TestValidScopes(t *testing.T) {
	expectedScopes := []OAuthScope{
		OAuthScopeRead, OAuthScopeWrite, OAuthScopeAdmin,
		OAuthScopeOpenID, OAuthScopeProfile, OAuthScopeEmail,
		OAuthScopeServers, OAuthScopeMessages,
	}

	for _, scope := range expectedScopes {
		if !ValidScopes[scope] {
			t.Errorf("expected scope %s to be valid", scope)
		}
	}

	if ValidScopes["invalid_scope"] {
		t.Error("expected 'invalid_scope' to be invalid")
	}
}

func TestScopeDescriptions(t *testing.T) {
	for scope := range ValidScopes {
		desc, ok := ScopeDescriptions[scope]
		if !ok {
			t.Errorf("missing description for scope %s", scope)
		}
		if desc == "" {
			t.Errorf("empty description for scope %s", scope)
		}
	}
}
