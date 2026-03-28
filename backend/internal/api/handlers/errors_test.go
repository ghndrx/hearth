package handlers

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/services"
)

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		name       string
		handler    func(c *fiber.Ctx) error
		wantStatus int
		wantError  string
		wantMsg    string
	}{
		{"BadRequest", func(c *fiber.Ctx) error { return BadRequest(c, "bad") }, 400, "bad_request", "bad"},
		{"Unauthorized", func(c *fiber.Ctx) error { return Unauthorized(c, "no auth") }, 401, "unauthorized", "no auth"},
		{"Forbidden", func(c *fiber.Ctx) error { return Forbidden(c, "denied") }, 403, "forbidden", "denied"},
		{"NotFound", func(c *fiber.Ctx) error { return NotFound(c, "missing") }, 404, "not_found", "missing"},
		{"Conflict", func(c *fiber.Ctx) error { return Conflict(c, "dup") }, 409, "conflict", "dup"},
		{"InternalError", func(c *fiber.Ctx) error { return InternalError(c, "oops") }, 500, "internal_error", "oops"},
		{"NotImplemented", func(c *fiber.Ctx) error { return NotImplemented(c, "todo") }, 501, "not_implemented", "todo"},
		{"ServiceUnavailable", func(c *fiber.Ctx) error { return ServiceUnavailable(c, "down") }, 503, "service_unavailable", "down"},
		{"TooManyRequests", func(c *fiber.Ctx) error { return TooManyRequests(c, "slow down") }, 429, "rate_limited", "slow down"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", tc.handler)

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			var result ErrorResponse
			json.NewDecoder(resp.Body).Decode(&result)
			assert.Equal(t, tc.wantError, result.Error)
			assert.Equal(t, tc.wantMsg, result.Message)
		})
	}
}

func TestValidationError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return ValidationError(c, "email", "is required")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result ErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "validation_error", result.Error)
	assert.Equal(t, "email: is required", result.Message)
	assert.Equal(t, "email", result.Code)
}

func TestParseError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return ParseError(c, fiber.NewError(400, "unexpected EOF"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result ErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result.Message, "invalid request body")
}

func TestInvalidUUID(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return InvalidUUID(c, "server_id")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result ErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid server_id", result.Message)
}

func TestHandleServiceError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"nil error", nil, 200, ""},
		{"ErrInvalidCredentials", services.ErrInvalidCredentials, 401, "invalid email or password"},
		{"ErrEmailTaken", services.ErrEmailTaken, 409, "email already registered"},
		{"ErrUsernameTaken", services.ErrUsernameTaken, 409, "username already taken"},
		{"ErrUserNotFound", services.ErrUserNotFound, 404, "user not found"},
		{"ErrServerNotFound", services.ErrServerNotFound, 404, "server not found"},
		{"ErrNotServerMember", services.ErrNotServerMember, 403, "not a member of this server"},
		{"ErrNotServerOwner", services.ErrNotServerOwner, 403, "only the server owner can perform this action"},
		{"ErrAlreadyMember", services.ErrAlreadyMember, 409, "already a member of this server"},
		{"ErrBannedFromServer", services.ErrBannedFromServer, 403, "you are banned from this server"},
		{"ErrChannelNotFound", services.ErrChannelNotFound, 404, "channel not found"},
		{"ErrNotChannelMember", services.ErrNotChannelMember, 403, "not a member of this channel"},
		{"ErrMessageNotFound", services.ErrMessageNotFound, 404, "message not found"},
		{"ErrNotMessageAuthor", services.ErrNotMessageAuthor, 403, "you can only modify your own messages"},
		{"ErrNoPermission", services.ErrNoPermission, 403, "you don't have permission to perform this action"},
		{"ErrMissingPermission", services.ErrMissingPermission, 403, "missing required permission"},
		{"ErrMissingSendMessages", services.ErrMissingSendMessages, 403, "missing SEND_MESSAGES permission"},
		{"ErrMissingReadMessages", services.ErrMissingReadMessages, 403, "missing READ_MESSAGE_HISTORY permission"},
		{"ErrMissingManageMessages", services.ErrMissingManageMessages, 403, "missing MANAGE_MESSAGES permission"},
		{"ErrMissingAddReactions", services.ErrMissingAddReactions, 403, "missing ADD_REACTIONS permission"},
		{"ErrMissingManageRoles", services.ErrMissingManageRoles, 403, "missing MANAGE_ROLES permission"},
		{"ErrMissingManageChannels", services.ErrMissingManageChannels, 403, "missing MANAGE_CHANNELS permission"},
		{"ErrMissingKickMembers", services.ErrMissingKickMembers, 403, "missing KICK_MEMBERS permission"},
		{"ErrMissingBanMembers", services.ErrMissingBanMembers, 403, "missing BAN_MEMBERS permission"},
		{"ErrMissingCreateInvite", services.ErrMissingCreateInvite, 403, "missing CREATE_INVITE permission"},
		{"ErrMissingManageServer", services.ErrMissingManageServer, 403, "missing MANAGE_SERVER permission"},
		{"ErrMissingManageWebhooks", services.ErrMissingManageWebhooks, 403, "missing MANAGE_WEBHOOKS permission"},
		{"ErrMissingManageThreads", services.ErrMissingManageThreads, 403, "missing MANAGE_THREADS permission"},
		{"ErrMissingAdministrator", services.ErrMissingAdministrator, 403, "missing ADMINISTRATOR permission"},
		{"ErrMissingMoveMembers", services.ErrMissingMoveMembers, 403, "missing MOVE_MEMBERS permission"},
		{"ErrMissingMuteMembers", services.ErrMissingMuteMembers, 403, "missing MUTE_MEMBERS permission"},
		{"ErrMissingManageEmojis", services.ErrMissingManageEmojis, 403, "missing MANAGE_EMOJIS permission"},
		{"ErrMissingViewChannels", services.ErrMissingViewChannels, 403, "missing VIEW_CHANNELS permission"},
		{"ErrRoleNotFound", services.ErrRoleNotFound, 404, "role not found"},
		{"ErrCannotDeleteRole", services.ErrCannotDeleteRole, 400, "cannot delete this role"},
		{"ErrCannotDeleteDefault", services.ErrCannotDeleteDefault, 400, "cannot delete the default role"},
		{"ErrRoleHierarchy", services.ErrRoleHierarchy, 403, "cannot modify role with higher position"},
		{"ErrCannotManageMember", services.ErrCannotManageMember, 403, "cannot manage member with equal or higher role"},
		{"ErrInviteNotFound", services.ErrInviteNotFound, 404, "invite not found"},
		{"ErrInviteExpired", services.ErrInviteExpired, 400, "invite has expired"},
		{"ErrInviteMaxUses", services.ErrInviteMaxUses, 400, "invite has reached maximum uses"},
		{"ErrWebhookNotFound", services.ErrWebhookNotFound, 404, "webhook not found"},
		{"ErrInvalidWebhookToken", services.ErrInvalidWebhookToken, 403, "invalid webhook token"},
		{"ErrWebhookNameTooLong", services.ErrWebhookNameTooLong, 400, "webhook name cannot exceed 80 characters"},
		{"ErrTooManyWebhooks", services.ErrTooManyWebhooks, 400, "maximum number of webhooks reached for this channel"},
		{"ErrMessageTooLong", services.ErrMessageTooLong, 400, "message exceeds maximum length"},
		{"ErrRateLimited", services.ErrRateLimited, 429, "you are sending messages too quickly"},
		{"ErrEmptyMessage", services.ErrEmptyMessage, 400, "message cannot be empty"},
		{"ErrBulkDeleteLimit", services.ErrBulkDeleteLimit, 400, "can only delete up to 100 messages at a time"},
		{"ErrBulkDeleteTooOld", services.ErrBulkDeleteTooOld, 400, "messages older than 14 days cannot be bulk deleted"},
		{"ErrCannotDeleteDM", services.ErrCannotDeleteDM, 400, "cannot delete DM channel"},
		{"ErrSelfAction", services.ErrSelfAction, 400, "cannot perform this action on yourself"},
		{"ErrNotificationNotFound", services.ErrNotificationNotFound, 404, "notification not found"},
		{"ErrDigestNotFound", services.ErrDigestNotFound, 404, "digest not found"},
		{"ErrDigestDisabled", services.ErrDigestDisabled, 400, "digest notifications are disabled"},
		{"ErrInvalidFrequency", services.ErrInvalidFrequency, 400, "invalid digest frequency"},
		{"ErrInvalidTimezone", services.ErrInvalidTimezone, 400, "invalid timezone"},
		{"ErrAuditLogNotFound", services.ErrAuditLogNotFound, 404, "audit log entry not found"},
		{"ErrRegistrationClosed", services.ErrRegistrationClosed, 403, "registration is currently closed"},
		{"ErrInviteRequired", services.ErrInviteRequired, 403, "an invite is required to register"},
		{"ErrPasswordTooShort", services.ErrPasswordTooShort, 400, "password must be at least 8 characters"},
		{"ErrPasswordTooLong", services.ErrPasswordTooLong, 400, "password must be at most 72 characters"},
		{"ErrPasswordWeak", services.ErrPasswordWeak, 400, "password must contain at least one uppercase, lowercase, and number"},
		{"unknown error", fiber.NewError(500, "unknown"), 500, "an unexpected error occurred"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: func(c *fiber.Ctx, err error) error {
					var httpErr *HTTPError
					if errors.As(err, &httpErr) {
						return c.Status(httpErr.Status).JSON(ErrorResponse{
							Error:   httpErr.ErrorType,
							Message: httpErr.Message,
							Code:    httpErr.Code,
						})
					}
					code := fiber.StatusInternalServerError
					if e, ok := err.(*fiber.Error); ok {
						code = e.Code
					}
					return c.Status(code).JSON(ErrorResponse{
						Error:   "internal_error",
						Message: err.Error(),
					})
				},
			})
			app.Get("/test", func(c *fiber.Ctx) error {
				if tc.err == nil {
					return c.JSON(fiber.Map{"ok": true})
				}
				return HandleServiceError(c, tc.err)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.wantMsg != "" {
				var result ErrorResponse
				json.NewDecoder(resp.Body).Decode(&result)
				assert.Equal(t, tc.wantMsg, result.Message)
			}
		})
	}
}
