package matrixfederation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func ptr(s string) *string { return &s }

func deref(s *string, empty string) string {
	if s == nil {
		return empty
	}
	return *s
}

func TestGetProfileHandler(t *testing.T) {
	tests := []struct {
		name            string
		userID          string
		mockProfile     *UserProfile
		mockErr         error
		wantStatus      int
		wantErrcode     string
		wantDisplayName *string
	}{
		{
			name:        "user with avatar and display name",
			userID:      "@alice:hearth.example.com",
			mockProfile: &UserProfile{UserID: "@alice:hearth.example.com", AvatarURL: ptr("https://hearth.example.com/avatar.png"), DisplayName: ptr("Alice")},
			wantStatus:  200,
		},
		{
			name:        "user with no avatar or display name",
			userID:      "@bob:hearth.example.com",
			mockProfile: &UserProfile{UserID: "@bob:hearth.example.com", AvatarURL: nil, DisplayName: nil},
			wantStatus:  200,
		},
		{
			name:        "user not found",
			userID:      "@ghost:hearth.example.com",
			mockErr:     ErrUserNotFound,
			wantStatus:  404,
			wantErrcode: "M_NOT_FOUND",
		},
		{
			name:        "user deactivated",
			userID:      "@deactivated:hearth.example.com",
			mockErr:     ErrUserDeactivated,
			wantStatus:  404,
			wantErrcode: "M_NOT_FOUND",
		},
		{
			name:        "internal error",
			userID:      "@error:hearth.example.com",
			mockErr:     errors.New("database connection failed"),
			wantStatus:  500,
			wantErrcode: "M_UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockProfileService()
			if tt.mockProfile != nil {
				mockSvc.Profiles[tt.mockProfile.UserID] = tt.mockProfile
			}
			mockSvc.Err = tt.mockErr

			handler := NewProfileHandler(mockSvc)
			app := fiber.New()
			app.Get("/profile/:userId", handler.GetProfile)

			req, err := http.NewRequest("GET", "/profile/"+url.QueryEscape(tt.userID), nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			if tt.wantErrcode != "" {
				var body map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if body["errcode"] != tt.wantErrcode {
					t.Errorf("errcode = %q, want %q", body["errcode"], tt.wantErrcode)
				}
			}

			if tt.wantStatus == 200 && tt.mockProfile != nil {
				var body MatrixProfileResponse
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode profile response: %v", err)
				}
				if body.UserID != tt.mockProfile.UserID {
					t.Errorf("UserID = %q, want %q", body.UserID, tt.mockProfile.UserID)
				}
			}
		})
	}
}

func TestGetAvatarURLHandler(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		mockProfile *UserProfile
		mockErr     error
		wantStatus  int
		wantErrcode string
	}{
		{
			name:        "user with avatar",
			userID:      "@alice:hearth.example.com",
			mockProfile: &UserProfile{UserID: "@alice:hearth.example.com", AvatarURL: ptr("https://example.com/avatar.png")},
			wantStatus:  200,
		},
		{
			name:        "user with no avatar",
			userID:      "@bob:hearth.example.com",
			mockProfile: &UserProfile{UserID: "@bob:hearth.example.com", AvatarURL: nil},
			wantStatus:  200,
		},
		{
			name:        "user not found",
			userID:      "@ghost:hearth.example.com",
			mockErr:     ErrUserNotFound,
			wantStatus:  404,
			wantErrcode: "M_NOT_FOUND",
		},
		{
			name:        "internal error",
			userID:      "@error:hearth.example.com",
			mockErr:     errors.New("database failure"),
			wantStatus:  500,
			wantErrcode: "M_UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockProfileService()
			if tt.mockProfile != nil {
				mockSvc.Profiles[tt.mockProfile.UserID] = tt.mockProfile
			}
			mockSvc.Err = tt.mockErr

			handler := NewProfileHandler(mockSvc)
			app := fiber.New()
			app.Get("/profile/:userId/avatar_url", handler.GetAvatarURL)

			req, err := http.NewRequest("GET", "/profile/"+url.QueryEscape(tt.userID)+"/avatar_url", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			if tt.wantErrcode != "" {
				var body map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if body["errcode"] != tt.wantErrcode {
					t.Errorf("errcode = %q, want %q", body["errcode"], tt.wantErrcode)
				}
			}
		})
	}
}

func TestGetDisplayNameHandler(t *testing.T) {
	tests := []struct {
		name            string
		userID          string
		mockProfile     *UserProfile
		mockErr         error
		wantStatus      int
		wantErrcode     string
		wantDisplayName *string
	}{
		{
			name:        "user with display name",
			userID:      "@alice:hearth.example.com",
			mockProfile: &UserProfile{UserID: "@alice:hearth.example.com", DisplayName: ptr("Alice")},
			wantStatus:  200,
		},
		{
			name:        "user with no display name",
			userID:      "@bob:hearth.example.com",
			mockProfile: &UserProfile{UserID: "@bob:hearth.example.com", DisplayName: nil},
			wantStatus:  200,
		},
		{
			name:        "user not found",
			userID:      "@ghost:hearth.example.com",
			mockErr:     ErrUserNotFound,
			wantStatus:  404,
			wantErrcode: "M_NOT_FOUND",
		},
		{
			name:        "internal error",
			userID:      "@error:hearth.example.com",
			mockErr:     errors.New("database failure"),
			wantStatus:  500,
			wantErrcode: "M_UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockProfileService()
			if tt.mockProfile != nil {
				mockSvc.Profiles[tt.mockProfile.UserID] = tt.mockProfile
			}
			mockSvc.Err = tt.mockErr

			handler := NewProfileHandler(mockSvc)
			app := fiber.New()
			app.Get("/profile/:userId/displayname", handler.GetDisplayName)

			req, err := http.NewRequest("GET", "/profile/"+url.QueryEscape(tt.userID)+"/displayname", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			if tt.wantErrcode != "" {
				var body map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if body["errcode"] != tt.wantErrcode {
					t.Errorf("errcode = %q, want %q", body["errcode"], tt.wantErrcode)
				}
			}
		})
	}
}

func TestMockProfileService(t *testing.T) {
	svc := NewMockProfileService()
	ctx := context.Background()

	_, err := svc.GetProfile(ctx, "@alice:hearth.example.com")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}

	svc.Profiles["@alice:hearth.example.com"] = &UserProfile{
		UserID:      "@alice:hearth.example.com",
		DisplayName: ptr("Alice"),
	}

	profile, err := svc.GetProfile(ctx, "@alice:hearth.example.com")
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if profile.DisplayName == nil || *profile.DisplayName != "Alice" {
		t.Errorf("DisplayName = %v, want 'Alice'", deref(profile.DisplayName, "<nil>"))
	}

	avatarURL, err := svc.GetAvatarURL(ctx, "@alice:hearth.example.com")
	if err != nil {
		t.Fatalf("GetAvatarURL failed: %v", err)
	}
	if avatarURL != nil {
		t.Errorf("GetAvatarURL = %v, want nil", *avatarURL)
	}
}

func TestNewMatrixProfileResponse(t *testing.T) {
	profile := &UserProfile{UserID: "@alice:hearth.example.com", AvatarURL: ptr("https://example.com/avatar.png"), DisplayName: ptr("Alice")}

	resp := NewMatrixProfileResponse(profile)
	if resp.AvatarURL == nil || *resp.AvatarURL != "https://example.com/avatar.png" {
		t.Errorf("AvatarURL = %v", resp.AvatarURL)
	}
	if resp.DisplayName == nil || *resp.DisplayName != "Alice" {
		t.Errorf("DisplayName = %v", resp.DisplayName)
	}
	if resp.UserID != "@alice:hearth.example.com" {
		t.Errorf("UserID = %s", resp.UserID)
	}
}

func TestSetupProfileRoutes(t *testing.T) {
	mockSvc := NewMockProfileService()
	handler := NewProfileHandler(mockSvc)

	// Pre-populate mock with a known user so handler returns 200
	mockSvc.Profiles["@alice:hearth.example.com"] = &UserProfile{
		UserID:      "@alice:hearth.example.com",
		AvatarURL:   ptr("https://hearth.example.com/avatar.png"),
		DisplayName: ptr("Alice"),
	}

	// Register routes on a fresh app
	app := fiber.New()
	SetupProfileRoutes(app, handler, "/_matrix/client/v3")

	// Verify each route is reachable and returns 200 for a known user
	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/_matrix/client/v3/profile/%40alice%3Ahearth.example.com"},
		{"GET", "/_matrix/client/v3/profile/%40alice%3Ahearth.example.com/avatar_url"},
		{"GET", "/_matrix/client/v3/profile/%40alice%3Ahearth.example.com/displayname"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed for %s %s: %v", tt.method, tt.path, err)
			}
			defer resp.Body.Close()
			// Route should be matched and return 200 for known user
			if resp.StatusCode != fiber.StatusOK {
				t.Errorf("route %s %s returned %d, want 200", tt.method, tt.path, resp.StatusCode)
			}
		})
	}
}

func TestDecodeUserID(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"encoded @", "%40alice:hearth.example.com", "@alice:hearth.example.com", false},
		{"encoded colon", "@alice%3Ahearth.example.com", "@alice:hearth.example.com", false},
		{"plain", "@alice:hearth.example.com", "@alice:hearth.example.com", false},
		{"encoded full", url.QueryEscape("@alice:hearth.example.com"), "@alice:hearth.example.com", false},
		{"invalid percent", "%ZZ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeUserID(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeUserID(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("decodeUserID(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestUserProfileJSON(t *testing.T) {
	// Verify JSON field tags match Matrix spec
	profile := &UserProfile{
		AvatarURL:   ptr("mxc://hearth.example.com/avatar123"),
		DisplayName: ptr("Alice"),
		UserID:      "@alice:hearth.example.com",
	}

	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// displayname field must be snake_case per Matrix spec
	if !strings.Contains(string(data), `"displayname"`) {
		t.Errorf("expected displayname field in JSON, got: %s", string(data))
	}
	if !strings.Contains(string(data), `"avatar_url"`) {
		t.Errorf("expected avatar_url field in JSON, got: %s", string(data))
	}
	if !strings.Contains(string(data), `"user_id"`) {
		t.Errorf("expected user_id field in JSON, got: %s", string(data))
	}
}

func TestUserNotFoundDoesNotLeakInfo(t *testing.T) {
	mockSvc := NewMockProfileService()
	mockSvc.Err = errors.New("sql: connection refused")

	handler := NewProfileHandler(mockSvc)
	app := fiber.New()
	app.Get("/profile/:userId", handler.GetProfile)

	req, err := http.NewRequest("GET", "/profile/"+url.QueryEscape("@alice:hearth.example.com"), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 500, not leak "connection refused"
	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	// Should be M_UNKNOWN, not the raw error message
	if body["errcode"] != "M_UNKNOWN" {
		t.Errorf("errcode = %q, want M_UNKNOWN", body["errcode"])
	}
	// Error message should not contain the raw error
	if strings.Contains(strings.ToLower(body["error"]), "connection refused") {
		t.Errorf("error message leaks internal details: %s", body["error"])
	}
}
