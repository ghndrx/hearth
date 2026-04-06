// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// Phase 1: Identity layer — MXID, homeserver config, and Profile API.
//
// Matrix Spec References:
//   - Client-Server API r0.6.1 § 8: User ID Format
//   - Client-Server API r0.6.1 § 11.2: Profile API
package matrixfederation

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

// Common errors
var (
	ErrUserNotFound    = errors.New("matrix: user not found on this homeserver")
	ErrInvalidMXID     = errors.New("matrix: invalid user ID format")
	ErrUserDeactivated = errors.New("matrix: user has been deactivated")
)

// ProfileService defines the interface for fetching user profile data.
type ProfileService interface {
	// GetProfile fetches a user's public profile by their Matrix User ID (MXID).
	// The MXID is expected to be a fully-qualified ID like @alice:hearth.example.com.
	// Returns ErrUserNotFound if the user does not exist on this homeserver.
	GetProfile(ctx context.Context, mxid string) (*UserProfile, error)

	// GetAvatarURL fetches just the avatar URL for a user (optimization).
	GetAvatarURL(ctx context.Context, mxid string) (*string, error)
}

// UserProfile represents a Matrix user's public profile.
// Maps to the Matrix Client-Server API § 11.2 profile response.
//
// https://matrix.org/docs/spec/client_server/r0.6.1#get-matrix-client-r0-profile-userid
type UserProfile struct {
	// User ID (MXID) — always returned even if display_name is not set
	AvatarURL   *string `json:"avatar_url,omitempty"`
	DisplayName *string `json:"displayname,omitempty"`
	// The User ID — this is the MXID
	UserID string `json:"user_id"`
}

// MatrixProfileResponse is the HTTP response shape for GET /_matrix/client/v3/profile/{userId}
type MatrixProfileResponse struct {
	AvatarURL   *string `json:"avatar_url,omitempty"`
	DisplayName *string `json:"displayname,omitempty"`
	UserID      string  `json:"user_id"`
}

// NewMatrixProfileResponse creates a Matrix profile response from a UserProfile.
func NewMatrixProfileResponse(p *UserProfile) MatrixProfileResponse {
	return MatrixProfileResponse{
		AvatarURL:   p.AvatarURL,
		DisplayName: p.DisplayName,
		UserID:      p.UserID,
	}
}

// ProfileHandler handles Matrix Client-Server Profile API endpoints.
//
// Endpoints implemented (Matrix r0.6.1 § 11.2):
//   - GET /_matrix/client/v3/profile/{userId} — get user profile
//   - GET /_matrix/client/v3/profile/{userId}/avatar_url — get avatar URL
//   - GET /_matrix/client/v3/profile/{userId}/displayname — get display name
//
// Note: PUT endpoints for setting profile are out of scope for Phase 1 (identity layer).
type ProfileHandler struct {
	profileService ProfileService
}

// NewProfileHandler creates a new Matrix Profile API handler.
func NewProfileHandler(profileService ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

// decodeUserID URL-decodes a Matrix user ID path parameter.
// Fiber's c.Params does not auto-decode.
func decodeUserID(rawUserID string) (string, error) {
	decoded, err := url.PathUnescape(rawUserID)
	if err != nil {
		return "", fmt.Errorf("matrix: failed to decode userId: %w", err)
	}
	return decoded, nil
}

// GetProfile handles GET /_matrix/client/v3/profile/{userId}
//
// Matrix spec § 11.2.1: Get User Profile
// Returns the profile information (avatar URL and display name) for a user.
// The profile is returned even if the user has not set an avatar or display name,
// but user_id is always present.
//
// Error codes per spec:
//   - M_NOT_FOUND (404): The user was not found
//   - M_USER_DEACTIVATED (404): The user account has been deactivated
func (h *ProfileHandler) GetProfile(c *fiber.Ctx) error {
	rawUserID := c.Params("userId")
	userID, err := decodeUserID(rawUserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "userId is required",
		})
	}
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "userId is required",
		})
	}

	ctx := c.Context()

	profile, err := h.profileService.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrUserDeactivated) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errcode": "M_NOT_FOUND",
				"error":   "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to fetch profile",
		})
	}

	return c.JSON(NewMatrixProfileResponse(profile))
}

// GetAvatarURL handles GET /_matrix/client/v3/profile/{userId}/avatar_url
//
// Matrix spec § 11.2.2: Get User Avatar URL
// Returns the avatar URL for a user. Returns an empty object if no avatar is set.
func (h *ProfileHandler) GetAvatarURL(c *fiber.Ctx) error {
	rawUserID := c.Params("userId")
	userID, err := decodeUserID(rawUserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "userId is required",
		})
	}
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "userId is required",
		})
	}

	ctx := c.Context()

	avatarURL, err := h.profileService.GetAvatarURL(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errcode": "M_NOT_FOUND",
				"error":   "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to fetch avatar URL",
		})
	}

	// Matrix spec: if no avatar is set, return empty object
	// avatar_url is null or omitted if none is set
	return c.JSON(fiber.Map{
		"avatar_url": avatarURL,
	})
}

// GetDisplayName handles GET /_matrix/client/v3/profile/{userId}/displayname
//
// Matrix spec § 11.2.3: Get User Display Name
// Returns the display name for a user. Returns an empty object if none is set.
func (h *ProfileHandler) GetDisplayName(c *fiber.Ctx) error {
	rawUserID := c.Params("userId")
	userID, err := decodeUserID(rawUserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "userId is required",
		})
	}
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "userId is required",
		})
	}

	ctx := c.Context()

	profile, err := h.profileService.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"errcode": "M_NOT_FOUND",
				"error":   "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to fetch display name",
		})
	}

	return c.JSON(fiber.Map{
		"displayname": profile.DisplayName,
	})
}

// SetupProfileRoutes registers all Matrix Profile API routes on a Fiber app.
func SetupProfileRoutes(app *fiber.App, handler *ProfileHandler, prefix string) {
	profile := app.Group(prefix)
	profile.Get("/profile/:userId", handler.GetProfile)
	profile.Get("/profile/:userId/avatar_url", handler.GetAvatarURL)
	profile.Get("/profile/:userId/displayname", handler.GetDisplayName)
}

// MockProfileService is a test implementation of ProfileService.
type MockProfileService struct {
	Profiles map[string]*UserProfile
	Err      error
}

// NewMockProfileService creates a mock profile service with sample data.
func NewMockProfileService() *MockProfileService {
	return &MockProfileService{
		Profiles: make(map[string]*UserProfile),
	}
}

func (m *MockProfileService) GetProfile(ctx context.Context, mxid string) (*UserProfile, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if profile, ok := m.Profiles[mxid]; ok {
		return profile, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUserNotFound, mxid)
}

func (m *MockProfileService) GetAvatarURL(ctx context.Context, mxid string) (*string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if profile, ok := m.Profiles[mxid]; ok {
		return profile.AvatarURL, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUserNotFound, mxid)
}
