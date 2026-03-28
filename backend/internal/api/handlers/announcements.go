package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// AnnouncementHandler handles announcement channel endpoints
type AnnouncementHandler struct {
	announcementService *services.AnnouncementService
}

// NewAnnouncementHandler creates a new announcement handler
func NewAnnouncementHandler(announcementService *services.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: announcementService,
	}
}

// FollowChannelResponse represents a follow response
type FollowChannelResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ChannelID       string `json:"channel_id"`
	GuildID         string `json:"guild_id"`
	Token           string `json:"token"`
	Type            int    `json:"type"`
	SourceChannelID string `json:"source_channel_id,omitempty"`
	SourceGuildID   string `json:"source_guild_id,omitempty"`
}

// FollowChannel adds a follower to an announcement channel
// @Summary Follow announcement channel
// @Description Follow an announcement channel to receive crossposted messages in a target channel
// @Tags Announcements
// @Accept json
// @Produce json
// @Param channelID path string true "Announcement Channel ID"
// @Param body body struct{WebhookChannelID string `json:"webhook_channel_id"`} true "Target channel to receive messages"
// @Success 201 {object} FollowChannelResponse "Successfully followed channel"
// @Failure 400 {object} fiber.Map "Invalid channel ID or not an announcement channel"
// @Failure 403 {object} fiber.Map "Not a server member or missing permissions"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 409 {object} fiber.Map "Already following this channel"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/followers [post]
func (h *AnnouncementHandler) FollowChannel(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	sourceChannelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req struct {
		WebhookChannelID string `json:"webhook_channel_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.WebhookChannelID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "webhook_channel_id is required",
		})
	}

	targetChannelID, err := uuid.Parse(req.WebhookChannelID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid webhook_channel_id",
		})
	}

	webhook, err := h.announcementService.FollowChannel(c.Context(), sourceChannelID, targetChannelID, userID)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotAnnouncementChannel:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not an announcement channel",
			})
		case services.ErrCannotFollowOwnChannel:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cannot follow your own announcement channel",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_WEBHOOKS permission",
			})
		default:
			if err.Error() == "already following this channel" {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "already following this channel",
				})
			}
			log.Printf("Error following channel: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to follow channel",
			})
		}
	}

	resp := FollowChannelResponse{
		ID:        webhook.ID.String(),
		Name:      webhook.Name,
		ChannelID: webhook.ChannelID.String(),
		Token:     webhook.Token,
		Type:      int(webhook.Type),
	}
	if webhook.ServerID != nil {
		resp.GuildID = webhook.ServerID.String()
	}
	if webhook.SourceChannelID != nil {
		resp.SourceChannelID = webhook.SourceChannelID.String()
	}
	if webhook.SourceServerID != nil {
		resp.SourceGuildID = webhook.SourceServerID.String()
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// UnfollowChannel removes a follower from an announcement channel
// @Summary Unfollow announcement channel
// @Description Remove a follower webhook from an announcement channel
// @Tags Announcements
// @Param channelID path string true "Announcement Channel ID"
// @Param webhookID path string true "Follower Webhook ID"
// @Success 204 "Successfully unfollowed channel"
// @Failure 400 {object} fiber.Map "Invalid channel ID or not an announcement channel"
// @Failure 403 {object} fiber.Map "Not a server member or missing permissions"
// @Failure 404 {object} fiber.Map "Channel or webhook not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/followers/{webhookID} [delete]
func (h *AnnouncementHandler) UnfollowChannel(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	sourceChannelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	followerWebhookID, err := uuid.Parse(c.Params("webhookID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid webhook id",
		})
	}

	err = h.announcementService.UnfollowChannel(c.Context(), sourceChannelID, followerWebhookID, userID)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotAnnouncementChannel:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not an announcement channel",
			})
		case services.ErrWebhookNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "webhook not found",
			})
		case services.ErrNotFollower:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "webhook is not a follower of this channel",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_WEBHOOKS permission",
			})
		default:
			log.Printf("Error unfollowing channel: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to unfollow channel",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetFollowers returns all followers of an announcement channel
// @Summary Get channel followers
// @Description Get all follower webhooks for an announcement channel
// @Tags Announcements
// @Produce json
// @Param channelID path string true "Announcement Channel ID"
// @Success 200 {array} FollowChannelResponse "List of followers"
// @Failure 400 {object} fiber.Map "Invalid channel ID or not an announcement channel"
// @Failure 404 {object} fiber.Map "Channel not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/followers [get]
func (h *AnnouncementHandler) GetFollowers(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	followers, err := h.announcementService.GetFollowers(c.Context(), channelID)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotAnnouncementChannel:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not an announcement channel",
			})
		default:
			log.Printf("Error getting followers: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get followers",
			})
		}
	}

	response := make([]FollowChannelResponse, len(followers))
	for i, f := range followers {
		response[i] = FollowChannelResponse{
			ID:        f.ID.String(),
			Name:      f.Name,
			ChannelID: f.ChannelID.String(),
			Type:      int(f.Type),
		}
		if f.ServerID != nil {
			response[i].GuildID = f.ServerID.String()
		}
		if f.SourceChannelID != nil {
			response[i].SourceChannelID = f.SourceChannelID.String()
		}
		if f.SourceServerID != nil {
			response[i].SourceGuildID = f.SourceServerID.String()
		}
	}

	return c.JSON(response)
}

// CrosspostMessage publishes a message to all followers of an announcement channel
// @Summary Crosspost message
// @Description Publish a message from an announcement channel to all servers that follow it
// @Tags Announcements
// @Produce json
// @Param channelID path string true "Announcement Channel ID"
// @Param messageID path string true "Message ID to crosspost"
// @Success 200 {array} models.Message "Crossposted messages"
// @Failure 400 {object} fiber.Map "Invalid channel ID or not an announcement channel"
// @Failure 403 {object} fiber.Map "Not a server member or missing permissions"
// @Failure 404 {object} fiber.Map "Channel or message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /channels/{channelID}/messages/{messageID}/crosspost [post]
func (h *AnnouncementHandler) CrosspostMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("channelID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	messageID, err := uuid.Parse(c.Params("messageID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid message id",
		})
	}

	messages, err := h.announcementService.CrosspostMessage(c.Context(), channelID, messageID, userID)
	if err != nil {
		switch err {
		case services.ErrChannelNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "channel not found",
			})
		case services.ErrNotAnnouncementChannel:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel is not an announcement channel",
			})
		case services.ErrMessageNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "message not found",
			})
		case services.ErrNotServerMember:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		case services.ErrNoPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_MESSAGES permission",
			})
		default:
			log.Printf("Error crossposting message: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to crosspost message",
			})
		}
	}

	return c.JSON(messages)
}
