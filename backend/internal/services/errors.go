package services

import "errors"

// Shared errors used across multiple services
var (
	// Auth errors
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrRegistrationClosed = errors.New("registration is currently closed")
	ErrInviteRequired     = errors.New("an invite is required to register")

	// Channel errors
	ErrChannelNotFound  = errors.New("channel not found")
	ErrNotChannelMember = errors.New("not a member of this channel")
	ErrCannotDeleteDM   = errors.New("cannot delete DM channel")

	// Message errors
	ErrMessageNotFound  = errors.New("message not found")
	ErrNotMessageAuthor = errors.New("not message author")
	ErrNoPermission     = errors.New("no permission to send messages")
	ErrMessageTooLong   = errors.New("message exceeds maximum length")
	ErrRateLimited      = errors.New("you are sending messages too quickly")
	ErrEmptyMessage     = errors.New("message cannot be empty")

	// Server errors
	ErrServerNotFound   = errors.New("server not found")
	ErrNotServerMember  = errors.New("not a server member")
	ErrNotServerOwner   = errors.New("not the server owner")
	ErrAlreadyMember    = errors.New("already a member of this server")
	ErrBannedFromServer = errors.New("you are banned from this server")

	// Invite errors
	ErrInviteNotFound = errors.New("invite not found")
	ErrInviteExpired  = errors.New("invite has expired")
	ErrInviteMaxUses  = errors.New("invite has reached maximum uses")

	// Role errors
	ErrRoleNotFound        = errors.New("role not found")
	ErrCannotDeleteRole    = errors.New("cannot delete this role")
	ErrCannotDeleteDefault = errors.New("cannot delete the default role")
	ErrRoleHierarchy       = errors.New("cannot modify role with higher position")

	// User errors
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already taken")
	ErrSelfAction    = errors.New("cannot perform this action on yourself")

	// Cache errors
	ErrCacheNotFound = errors.New("key not found in cache")
)
