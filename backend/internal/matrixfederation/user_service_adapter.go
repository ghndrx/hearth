package matrixfederation

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"hearth/internal/matrix"
	"hearth/internal/models"
)

// UserGetter defines the minimal interface needed from the Hearth user service
// to adapt it to the Matrix ProfileService interface.
type UserGetter interface {
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
}

// UserServiceAdapter wraps a Hearth UserGetter to implement the Matrix
// ProfileService interface for Phase 1 identity layer.
type UserServiceAdapter struct {
	userService   UserGetter
	homeserverCfg *matrix.HomeserverConfig
}

// NewUserServiceAdapter creates an adapter that bridges Hearth's user service
// to the Matrix Profile API.
func NewUserServiceAdapter(userService UserGetter, homeserverCfg *matrix.HomeserverConfig) *UserServiceAdapter {
	return &UserServiceAdapter{
		userService:   userService,
		homeserverCfg: homeserverCfg,
	}
}

// GetProfile implements ProfileService.
// It parses the MXID, resolves the localpart to a Hearth user, and returns
// the public profile.
//
// For a userID like @alice:hearth.example.com, we parse out alice as the localpart
// and hearth.example.com as the server name. If the server name matches our
// configured homeserver, we look up the user by username (alice). If it belongs
// to a remote server, we return ErrUserNotFound (Phase 1 does not support
// remote server lookups — those come in Phase 3).
func (a *UserServiceAdapter) GetProfile(ctx context.Context, userID string) (*UserProfile, error) {
	mxid, err := matrix.ParseMXID(userID)
	if err != nil {
		return nil, ErrInvalidMXID
	}

	// Phase 1: we only handle local users
	if !a.homeserverCfg.IsLocalMXID(mxid) {
		// Remote user — not yet supported (Phase 3 federation)
		return nil, ErrUserNotFound
	}

	// Look up user by their localpart (username)
	user, err := a.userService.GetUserByUsername(ctx, mxid.Localpart)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Check if user is deactivated
	if user.Flags&models.UserFlagDeletedUser != 0 {
		return nil, ErrUserDeactivated
	}

	return &UserProfile{
		UserID:      mxid.String(),
		AvatarURL:   user.AvatarURL,
		DisplayName: user.DisplayName,
	}, nil
}

// GetAvatarURL implements ProfileService.
// Returns just the avatar URL for a user.
func (a *UserServiceAdapter) GetAvatarURL(ctx context.Context, userID string) (*string, error) {
	profile, err := a.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return profile.AvatarURL, nil
}
