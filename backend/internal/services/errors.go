package services

import "errors"

// Shared errors used across multiple services
var (
	// Auth errors
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrRegistrationClosed = errors.New("registration is currently closed")
	ErrInviteRequired     = errors.New("an invite is required to register")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password must be at most 72 characters")
	ErrPasswordWeak       = errors.New("password must contain at least one uppercase, lowercase, and number")
	ErrMFARequired        = errors.New("MFA code required")
	ErrInvalidMFACode     = errors.New("invalid MFA code")
	ErrMFAAlreadyEnabled  = errors.New("MFA is already enabled")
	ErrMFANotEnabled      = errors.New("MFA is not enabled")

	// Channel errors
	ErrChannelNotFound         = errors.New("channel not found")
	ErrNotChannelMember        = errors.New("not a member of this channel")
	ErrCannotDeleteDM          = errors.New("cannot delete DM channel")
	ErrChannelTypeNotSupported = errors.New("channel type not supported for this operation")

	// Message errors
	ErrMessageNotFound  = errors.New("message not found")
	ErrNotMessageAuthor = errors.New("not message author")
	ErrNoPermission     = errors.New("no permission to send messages")
	ErrMessageTooLong   = errors.New("message exceeds maximum length")
	ErrRateLimited      = errors.New("you are sending messages too quickly")
	ErrEmptyMessage     = errors.New("message cannot be empty")
	ErrBulkDeleteLimit  = errors.New("can only delete up to 100 messages at a time")
	ErrBulkDeleteTooOld = errors.New("messages older than 14 days cannot be bulk deleted")

	// Permission errors
	ErrMissingPermission     = errors.New("missing required permission")
	ErrMissingSendMessages   = errors.New("missing SEND_MESSAGES permission")
	ErrMissingReadMessages   = errors.New("missing READ_MESSAGE_HISTORY permission")
	ErrMissingManageMessages = errors.New("missing MANAGE_MESSAGES permission")
	ErrMissingAddReactions   = errors.New("missing ADD_REACTIONS permission")
	ErrMissingManageRoles    = errors.New("missing MANAGE_ROLES permission")
	ErrMissingManageChannels = errors.New("missing MANAGE_CHANNELS permission")
	ErrMissingKickMembers    = errors.New("missing KICK_MEMBERS permission")
	ErrMissingBanMembers     = errors.New("missing BAN_MEMBERS permission")
	ErrMissingCreateInvite   = errors.New("missing CREATE_INVITE permission")
	ErrMissingManageServer   = errors.New("missing MANAGE_SERVER permission")
	ErrMissingManageWebhooks = errors.New("missing MANAGE_WEBHOOKS permission")
	ErrMissingManageThreads  = errors.New("missing MANAGE_THREADS permission")
	ErrMissingAdministrator  = errors.New("missing ADMINISTRATOR permission")
	ErrMissingMoveMembers    = errors.New("missing MOVE_MEMBERS permission")
	ErrMissingMuteMembers    = errors.New("missing MUTE_MEMBERS permission")
	ErrMissingManageEmojis   = errors.New("missing MANAGE_EMOJIS permission")
	ErrMissingViewChannels   = errors.New("missing VIEW_CHANNELS permission")

	// Server errors
	ErrServerNotFound   = errors.New("server not found")
	ErrNotServerMember  = errors.New("not a server member")
	ErrNotServerOwner   = errors.New("not the server owner")
	ErrAlreadyMember    = errors.New("already a member of this server")
	ErrBannedFromServer = errors.New("you are banned from this server")

	// Invite errors
	ErrInviteNotFound    = errors.New("invite not found")
	ErrInviteExpired     = errors.New("invite has expired")
	ErrInviteMaxUses     = errors.New("invite has reached maximum uses")
	ErrInviteRateLimited = errors.New("too many invites created recently")
	ErrVanityCodeTaken   = errors.New("vanity code is already taken")
	ErrVanityCodeInvalid = errors.New("vanity code must be 3-32 characters and contain only letters, numbers, hyphens, and underscores")

	// Role errors
	ErrRoleNotFound        = errors.New("role not found")
	ErrCannotDeleteRole    = errors.New("cannot delete this role")
	ErrCannotDeleteDefault = errors.New("cannot delete the default role")
	ErrRoleHierarchy       = errors.New("cannot modify role with higher position")
	ErrCannotManageMember  = errors.New("cannot manage member with equal or higher role")

	// User errors
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already taken")
	ErrSelfAction    = errors.New("cannot perform this action on yourself")

	// Webhook errors
	ErrWebhookNotFound     = errors.New("webhook not found")
	ErrInvalidWebhookToken = errors.New("invalid webhook token")
	ErrWebhookNameTooLong  = errors.New("webhook name cannot exceed 80 characters")
	ErrTooManyWebhooks     = errors.New("maximum number of webhooks reached for this channel")
	ErrWebhookRateLimited  = errors.New("webhook is rate limited, please try again later")
	ErrTooManyEmbeds       = errors.New("maximum 10 embeds allowed")

	// DM errors
	ErrNotDMChannel              = errors.New("channel is not a DM or group DM")
	ErrNotGroupDM                = errors.New("channel is not a group DM")
	ErrNotGroupDMOwner           = errors.New("only the group DM owner can perform this action")
	ErrAlreadyDMRecipient        = errors.New("user is already a recipient of this DM")
	ErrNotDMRecipient            = errors.New("user is not a recipient of this DM")
	ErrGroupDMFull               = errors.New("group DM can have at most 50 members")
	ErrCannotTransferToNonMember = errors.New("cannot transfer ownership to a non-member")


	// Generic permission errors
	ErrForbidden = errors.New("forbidden")

	// Notification errors
	ErrNotificationNotFound = errors.New("notification not found")

	// Digest errors
	ErrDigestNotFound   = errors.New("digest not found")
	ErrDigestDisabled   = errors.New("digest notifications are disabled")
	ErrInvalidFrequency = errors.New("invalid digest frequency")
	ErrInvalidTimezone  = errors.New("invalid timezone")

	// Audit log errors
	ErrAuditLogNotFound = errors.New("audit log entry not found")

	// Poll errors
	ErrPollClosed   = errors.New("poll is closed")
	ErrPollNotFound = errors.New("poll not found")

	// Moderation errors
	ErrModerationRuleNotFound = errors.New("moderation rule not found")
	ErrModerationRateLimited  = errors.New("rate limit exceeded for moderation actions")

	// Generic errors
	ErrInvalidInput = errors.New("invalid input")
)
