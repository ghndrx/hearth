package handlers

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"hearth/internal/services"
)

// HTTPError is a custom error type that carries HTTP status and response data.
// It is returned by HandleServiceError to signal Fiber's error handler to send
// the appropriate JSON response. Route handlers that call helpers directly still
// get inline response sending via the non-HTTPError helpers below.
type HTTPError struct {
	Status    int
	ErrorType string
	Message   string
	Code      string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// ErrorResponse is the JSON shape returned for API errors.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ---------------------------------------------------------------------------
// Helper functions used directly in route handlers (send response inline)
// ---------------------------------------------------------------------------

func BadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Error:   "bad_request",
		Message: message,
	})
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
		Error:   "unauthorized",
		Message: message,
	})
}

func Forbidden(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
		Error:   "forbidden",
		Message: message,
	})
}

func NotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
		Error:   "not_found",
		Message: message,
	})
}

func Conflict(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
		Error:   "conflict",
		Message: message,
	})
}

func InternalError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Error:   "internal_error",
		Message: message,
	})
}

func NotImplemented(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotImplemented).JSON(ErrorResponse{
		Error:   "not_implemented",
		Message: message,
	})
}

func ServiceUnavailable(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusServiceUnavailable).JSON(ErrorResponse{
		Error:   "service_unavailable",
		Message: message,
	})
}

func TooManyRequests(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusTooManyRequests).JSON(ErrorResponse{
		Error:   "rate_limited",
		Message: message,
	})
}

func ValidationError(c *fiber.Ctx, field, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Error:   "validation_error",
		Message: fmt.Sprintf("%s: %s", field, message),
		Code:    field,
	})
}

func ParseError(c *fiber.Ctx, err error) error {
	return BadRequest(c, "invalid request body: "+err.Error())
}

func InvalidUUID(c *fiber.Ctx, field string) error {
	return BadRequest(c, fmt.Sprintf("invalid %s", field))
}

// ---------------------------------------------------------------------------
// HTTPError constructors for use in HandleServiceError
// These do NOT send a response; they return an *HTTPError that the caller
// returns to Fiber's error handler.
// ---------------------------------------------------------------------------

func httpErr(status int, errType, message string) *HTTPError {
	return &HTTPError{Status: status, ErrorType: errType, Message: message}
}

func httpErrWithCode(status int, errType, message, code string) *HTTPError {
	return &HTTPError{Status: status, ErrorType: errType, Message: message, Code: code}
}

// HandleServiceError maps a service error to an HTTPError.
// The caller MUST return this error to Fiber (not nil) so that Fiber's
// custom error handler serializes it as JSON.
func HandleServiceError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		return httpErr(fiber.StatusUnauthorized, "unauthorized", "invalid email or password")

	case errors.Is(err, services.ErrEmailTaken):
		return httpErr(fiber.StatusConflict, "conflict", "email already registered")

	case errors.Is(err, services.ErrUsernameTaken):
		return httpErr(fiber.StatusConflict, "conflict", "username already taken")

	case errors.Is(err, services.ErrUserNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "user not found")

	case errors.Is(err, services.ErrServerNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "server not found")

	case errors.Is(err, services.ErrNotServerMember):
		return httpErr(fiber.StatusForbidden, "forbidden", "not a member of this server")

	case errors.Is(err, services.ErrNotServerOwner):
		return httpErr(fiber.StatusForbidden, "forbidden", "only the server owner can perform this action")

	case errors.Is(err, services.ErrAlreadyMember):
		return httpErr(fiber.StatusConflict, "conflict", "already a member of this server")

	case errors.Is(err, services.ErrBannedFromServer):
		return httpErr(fiber.StatusForbidden, "forbidden", "you are banned from this server")

	case errors.Is(err, services.ErrChannelNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "channel not found")

	case errors.Is(err, services.ErrNotChannelMember):
		return httpErr(fiber.StatusForbidden, "forbidden", "not a member of this channel")

	case errors.Is(err, services.ErrMessageNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "message not found")

	case errors.Is(err, services.ErrNotMessageAuthor):
		return httpErr(fiber.StatusForbidden, "forbidden", "you can only modify your own messages")

	case errors.Is(err, services.ErrNoPermission):
		return httpErr(fiber.StatusForbidden, "forbidden", "you don't have permission to perform this action")

	case errors.Is(err, services.ErrMissingPermission):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing required permission")

	case errors.Is(err, services.ErrMissingSendMessages):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing SEND_MESSAGES permission")

	case errors.Is(err, services.ErrMissingReadMessages):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing READ_MESSAGE_HISTORY permission")

	case errors.Is(err, services.ErrMissingManageMessages):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing MANAGE_MESSAGES permission")

	case errors.Is(err, services.ErrMissingAddReactions):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing ADD_REACTIONS permission")

	case errors.Is(err, services.ErrMissingManageRoles):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing MANAGE_ROLES permission")

	case errors.Is(err, services.ErrMissingManageChannels):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing MANAGE_CHANNELS permission")

	case errors.Is(err, services.ErrMissingKickMembers):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing KICK_MEMBERS permission")

	case errors.Is(err, services.ErrMissingBanMembers):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing BAN_MEMBERS permission")

	case errors.Is(err, services.ErrMissingCreateInvite):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing CREATE_INVITE permission")

	case errors.Is(err, services.ErrMissingManageServer):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing MANAGE_SERVER permission")

	case errors.Is(err, services.ErrMissingManageWebhooks):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing MANAGE_WEBHOOKS permission")

	case errors.Is(err, services.ErrMissingManageThreads):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing MANAGE_THREADS permission")

	case errors.Is(err, services.ErrMissingAdministrator):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing ADMINISTRATOR permission")

	case errors.Is(err, services.ErrMissingMoveMembers):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing MOVE_MEMBERS permission")

	case errors.Is(err, services.ErrMissingMuteMembers):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing MUTE_MEMBERS permission")

	case errors.Is(err, services.ErrMissingManageEmojis):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing MANAGE_EMOJIS permission")

	case errors.Is(err, services.ErrMissingViewChannels):
		return httpErr(fiber.StatusForbidden, "forbidden", "missing VIEW_CHANNELS permission")

	case errors.Is(err, services.ErrRoleNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "role not found")

	case errors.Is(err, services.ErrCannotDeleteRole):
		return httpErr(fiber.StatusBadRequest, "bad_request", "cannot delete this role")

	case errors.Is(err, services.ErrCannotDeleteDefault):
		return httpErr(fiber.StatusBadRequest, "bad_request", "cannot delete the default role")

	case errors.Is(err, services.ErrRoleHierarchy):
		return httpErr(fiber.StatusForbidden, "forbidden", "cannot modify role with higher position")

	case errors.Is(err, services.ErrCannotManageMember):
		return httpErr(fiber.StatusForbidden, "forbidden", "cannot manage member with equal or higher role")

	case errors.Is(err, services.ErrInviteNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "invite not found")

	case errors.Is(err, services.ErrInviteExpired):
		return httpErr(fiber.StatusBadRequest, "bad_request", "invite has expired")

	case errors.Is(err, services.ErrInviteMaxUses):
		return httpErr(fiber.StatusBadRequest, "bad_request", "invite has reached maximum uses")

	case errors.Is(err, services.ErrWebhookNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "webhook not found")

	case errors.Is(err, services.ErrInvalidWebhookToken):
		return httpErr(fiber.StatusForbidden, "forbidden", "invalid webhook token")

	case errors.Is(err, services.ErrWebhookNameTooLong):
		return httpErr(fiber.StatusBadRequest, "bad_request", "webhook name cannot exceed 80 characters")

	case errors.Is(err, services.ErrTooManyWebhooks):
		return httpErr(fiber.StatusBadRequest, "bad_request", "maximum number of webhooks reached for this channel")

	case errors.Is(err, services.ErrMessageTooLong):
		return httpErr(fiber.StatusBadRequest, "bad_request", "message exceeds maximum length")

	case errors.Is(err, services.ErrRateLimited):
		return httpErr(fiber.StatusTooManyRequests, "rate_limited", "you are sending messages too quickly")

	case errors.Is(err, services.ErrEmptyMessage):
		return httpErr(fiber.StatusBadRequest, "bad_request", "message cannot be empty")

	case errors.Is(err, services.ErrBulkDeleteLimit):
		return httpErr(fiber.StatusBadRequest, "bad_request", "can only delete up to 100 messages at a time")

	case errors.Is(err, services.ErrBulkDeleteTooOld):
		return httpErr(fiber.StatusBadRequest, "bad_request", "messages older than 14 days cannot be bulk deleted")

	case errors.Is(err, services.ErrCannotDeleteDM):
		return httpErr(fiber.StatusBadRequest, "bad_request", "cannot delete DM channel")

	case errors.Is(err, services.ErrNotDMChannel):
		return httpErr(fiber.StatusBadRequest, "bad_request", "channel is not a DM or group DM")

	case errors.Is(err, services.ErrNotGroupDM):
		return httpErr(fiber.StatusBadRequest, "bad_request", "channel is not a group DM")

	case errors.Is(err, services.ErrNotGroupDMOwner):
		return httpErr(fiber.StatusForbidden, "forbidden", "only the group DM owner can perform this action")

	case errors.Is(err, services.ErrAlreadyDMRecipient):
		return httpErr(fiber.StatusConflict, "conflict", "user is already a recipient of this DM")

	case errors.Is(err, services.ErrNotDMRecipient):
		return httpErr(fiber.StatusBadRequest, "bad_request", "user is not a recipient of this DM")

	case errors.Is(err, services.ErrGroupDMFull):
		return httpErr(fiber.StatusBadRequest, "bad_request", "group DM can have at most 10 members")

	case errors.Is(err, services.ErrSelfAction):
		return httpErr(fiber.StatusBadRequest, "bad_request", "cannot perform this action on yourself")

	case errors.Is(err, services.ErrNotificationNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "notification not found")

	case errors.Is(err, services.ErrDigestNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "digest not found")

	case errors.Is(err, services.ErrDigestDisabled):
		return httpErr(fiber.StatusBadRequest, "bad_request", "digest notifications are disabled")

	case errors.Is(err, services.ErrInvalidFrequency):
		return httpErr(fiber.StatusBadRequest, "bad_request", "invalid digest frequency")

	case errors.Is(err, services.ErrInvalidTimezone):
		return httpErr(fiber.StatusBadRequest, "bad_request", "invalid timezone")

	case errors.Is(err, services.ErrAuditLogNotFound):
		return httpErr(fiber.StatusNotFound, "not_found", "audit log entry not found")

	case errors.Is(err, services.ErrRegistrationClosed):
		return httpErr(fiber.StatusForbidden, "forbidden", "registration is currently closed")

	case errors.Is(err, services.ErrInviteRequired):
		return httpErr(fiber.StatusForbidden, "forbidden", "an invite is required to register")

	case errors.Is(err, services.ErrPasswordTooShort):
		return httpErr(fiber.StatusBadRequest, "bad_request", "password must be at least 8 characters")

	case errors.Is(err, services.ErrPasswordTooLong):
		return httpErr(fiber.StatusBadRequest, "bad_request", "password must be at most 72 characters")

	case errors.Is(err, services.ErrPasswordWeak):
		return httpErr(fiber.StatusBadRequest, "bad_request", "password must contain at least one uppercase, lowercase, and number")

	default:
		return httpErr(fiber.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
