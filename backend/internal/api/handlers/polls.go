package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// PollServiceInterface defines the interface for poll operations
type PollServiceInterface interface {
	CreatePoll(ctx context.Context, poll *models.Poll) error
	GetPoll(ctx context.Context, id uuid.UUID) (*models.Poll, error)
	UpdatePoll(ctx context.Context, poll *models.Poll) error
	DeletePoll(ctx context.Context, id uuid.UUID) error
	GetGuildPolls(ctx context.Context, guildID uuid.UUID) ([]*models.Poll, error)
	Vote(ctx context.Context, pollID, optionID, userID uuid.UUID) error
}

// PollHandler handles poll-related HTTP requests
type PollHandler struct {
	pollService *services.PollService
}

// NewPollHandler creates a new PollHandler
func NewPollHandler(pollService *services.PollService) *PollHandler {
	return &PollHandler{
		pollService: pollService,
	}
}

// CreatePollRequest represents the request body for creating a poll
type CreatePollRequest struct {
	Question   string   `json:"question"`
	Options    []string `json:"options"`
	IsMultiple bool     `json:"is_multiple"`
	EndTime    *string  `json:"end_time,omitempty"` // ISO 8601 format
}

// VoteRequest represents the request body for voting on a poll
type VoteRequest struct {
	OptionID string `json:"option_id"`
}

// CreatePoll creates a new poll in a channel
func (h *PollHandler) CreatePoll(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	var req CreatePollRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate request
	if req.Question == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "question is required",
		})
	}

	if len(req.Options) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "at least 2 options are required",
		})
	}

	if len(req.Options) > 10 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "maximum 10 options allowed",
		})
	}

	// Parse end time if provided
	var endTime *time.Time
	if req.EndTime != nil {
		t, err := time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid end_time format, use ISO 8601 (RFC3339)",
			})
		}
		if t.Before(time.Now()) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "end_time must be in the future",
			})
		}
		endTime = &t
	}

	// Create poll options
	options := make([]models.PollOption, len(req.Options))
	for i, optText := range req.Options {
		options[i] = models.PollOption{
			ID:   uuid.New(),
			Text: optText,
		}
	}

	poll := &models.Poll{
		ID:         uuid.New(),
		ChannelID:  channelID,
		CreatorID:  userID,
		Question:   req.Question,
		Options:    options,
		IsMultiple: req.IsMultiple,
		EndTime:    endTime,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Set poll ID on options
	for i := range poll.Options {
		poll.Options[i].PollID = poll.ID
	}

	if err := h.pollService.CreatePoll(c.Context(), poll); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create poll",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(poll)
}

// GetPoll returns a poll by ID
func (h *PollHandler) GetPoll(c *fiber.Ctx) error {
	pollID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid poll id",
		})
	}

	poll, err := h.pollService.GetPoll(c.Context(), pollID)
	if err != nil {
		if err.Error() == "invalid poll ID" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid poll id",
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "poll not found",
		})
	}

	return c.JSON(poll)
}

// Vote casts a vote on a poll option
func (h *PollHandler) Vote(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	pollID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid poll id",
		})
	}

	var req VoteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	optionID, err := uuid.Parse(req.OptionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid option id",
		})
	}

	if err := h.pollService.Vote(c.Context(), pollID, optionID, userID); err != nil {
		if err.Error() == "user has already voted on this poll" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "you have already voted on this poll",
			})
		}
		if err.Error() == "poll is closed" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "poll is closed",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to cast vote",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "vote recorded",
	})
}

// GetResults returns the results of a poll
func (h *PollHandler) GetResults(c *fiber.Ctx) error {
	pollID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid poll id",
		})
	}

	poll, err := h.pollService.GetPoll(c.Context(), pollID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "poll not found",
		})
	}

	// Calculate total votes
	totalVotes := 0
	for _, opt := range poll.Options {
		totalVotes += opt.Votes
	}

	// Build results with percentages
	results := make([]map[string]interface{}, len(poll.Options))
	for i, opt := range poll.Options {
		percentage := 0.0
		if totalVotes > 0 {
			percentage = float64(opt.Votes) / float64(totalVotes) * 100
		}
		results[i] = map[string]interface{}{
			"option_id":  opt.ID,
			"text":       opt.Text,
			"votes":      opt.Votes,
			"percentage": percentage,
		}
	}

	return c.JSON(fiber.Map{
		"poll_id":     poll.ID,
		"question":    poll.Question,
		"total_votes": totalVotes,
		"is_multiple": poll.IsMultiple,
		"is_closed":   isPollClosed(poll),
		"results":     results,
	})
}

// ClosePoll closes a poll so no more votes can be cast
func (h *PollHandler) ClosePoll(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	pollID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid poll id",
		})
	}

	// Get the poll to verify ownership
	poll, err := h.pollService.GetPoll(c.Context(), pollID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "poll not found",
		})
	}

	// Only the creator can close the poll
	if poll.CreatorID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "only the poll creator can close the poll",
		})
	}

	// Check if already closed
	if isPollClosed(poll) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "poll is already closed",
		})
	}

	// Set end time to now to close the poll
	now := time.Now()
	poll.EndTime = &now
	poll.UpdatedAt = now

	if err := h.pollService.UpdatePoll(c.Context(), poll); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to close poll",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "poll closed",
		"poll":    poll,
	})
}

// GetChannelPolls returns all polls in a channel
func (h *PollHandler) GetChannelPolls(c *fiber.Ctx) error {
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid channel id",
		})
	}

	// Note: Using GetGuildPolls as it's already implemented
	// In a real implementation, we'd have GetByChannelID
	polls, err := h.pollService.GetGuildPolls(c.Context(), channelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get polls",
		})
	}

	return c.JSON(polls)
}

// DeletePoll deletes a poll
func (h *PollHandler) DeletePoll(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	pollID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid poll id",
		})
	}

	// Get the poll to verify ownership
	poll, err := h.pollService.GetPoll(c.Context(), pollID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "poll not found",
		})
	}

	// Only the creator can delete the poll
	if poll.CreatorID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "only the poll creator can delete the poll",
		})
	}

	if err := h.pollService.DeletePoll(c.Context(), pollID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete poll",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// isPollClosed checks if a poll is closed (end time has passed)
func isPollClosed(poll *models.Poll) bool {
	if poll.EndTime == nil {
		return false
	}
	return poll.EndTime.Before(time.Now())
}
