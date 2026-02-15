package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"hearth/internal/models"
)

// MockPollService mocks the PollService for testing
type MockPollService struct {
	mock.Mock
}

func (m *MockPollService) CreatePoll(ctx context.Context, poll *models.Poll) error {
	args := m.Called(ctx, poll)
	return args.Error(0)
}

func (m *MockPollService) GetPoll(ctx context.Context, id uuid.UUID) (*models.Poll, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Poll), args.Error(1)
}

func (m *MockPollService) UpdatePoll(ctx context.Context, poll *models.Poll) error {
	args := m.Called(ctx, poll)
	return args.Error(0)
}

func (m *MockPollService) DeletePoll(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPollService) GetGuildPolls(ctx context.Context, guildID uuid.UUID) ([]*models.Poll, error) {
	args := m.Called(ctx, guildID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Poll), args.Error(1)
}

func (m *MockPollService) Vote(ctx context.Context, pollID, optionID, userID uuid.UUID) error {
	args := m.Called(ctx, pollID, optionID, userID)
	return args.Error(0)
}

func createTestPoll(userID uuid.UUID) *models.Poll {
	pollID := uuid.New()
	return &models.Poll{
		ID:        pollID,
		ChannelID: uuid.New(),
		CreatorID: userID,
		Question:  "What is your favorite color?",
		Options: []models.PollOption{
			{ID: uuid.New(), PollID: pollID, Text: "Red", Votes: 5},
			{ID: uuid.New(), PollID: pollID, Text: "Blue", Votes: 3},
			{ID: uuid.New(), PollID: pollID, Text: "Green", Votes: 2},
		},
		IsMultiple: false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// Helper to set up test app with routes
func setupPollTestApp(pollService *MockPollService, userID uuid.UUID) *fiber.App {
	app := fiber.New()

	// Add middleware to set userID in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	// Create poll endpoint
	app.Post("/channels/:id/polls", func(c *fiber.Ctx) error {
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

		for i := range poll.Options {
			poll.Options[i].PollID = poll.ID
		}

		if err := pollService.CreatePoll(c.Context(), poll); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create poll",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(poll)
	})

	// Get poll endpoint
	app.Get("/polls/:id", func(c *fiber.Ctx) error {
		pollID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid poll id",
			})
		}

		poll, err := pollService.GetPoll(c.Context(), pollID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "poll not found",
			})
		}

		return c.JSON(poll)
	})

	// Vote endpoint
	app.Post("/polls/:id/vote", func(c *fiber.Ctx) error {
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

		if err := pollService.Vote(c.Context(), pollID, optionID, userID); err != nil {
			if err.Error() == "user has already voted on this poll" {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "you have already voted on this poll",
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
	})

	// Get results endpoint
	app.Get("/polls/:id/results", func(c *fiber.Ctx) error {
		pollID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid poll id",
			})
		}

		poll, err := pollService.GetPoll(c.Context(), pollID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "poll not found",
			})
		}

		totalVotes := 0
		for _, opt := range poll.Options {
			totalVotes += opt.Votes
		}

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

		isClosed := false
		if poll.EndTime != nil && poll.EndTime.Before(time.Now()) {
			isClosed = true
		}

		return c.JSON(fiber.Map{
			"poll_id":     poll.ID,
			"question":    poll.Question,
			"total_votes": totalVotes,
			"is_multiple": poll.IsMultiple,
			"is_closed":   isClosed,
			"results":     results,
		})
	})

	// Close poll endpoint
	app.Post("/polls/:id/close", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		pollID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid poll id",
			})
		}

		poll, err := pollService.GetPoll(c.Context(), pollID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "poll not found",
			})
		}

		if poll.CreatorID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "only the poll creator can close the poll",
			})
		}

		if poll.EndTime != nil && poll.EndTime.Before(time.Now()) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "poll is already closed",
			})
		}

		now := time.Now()
		poll.EndTime = &now
		poll.UpdatedAt = now

		if err := pollService.UpdatePoll(c.Context(), poll); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to close poll",
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": "poll closed",
			"poll":    poll,
		})
	})

	// Delete poll endpoint
	app.Delete("/polls/:id", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		pollID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid poll id",
			})
		}

		poll, err := pollService.GetPoll(c.Context(), pollID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "poll not found",
			})
		}

		if poll.CreatorID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "only the poll creator can delete the poll",
			})
		}

		if err := pollService.DeletePoll(c.Context(), pollID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete poll",
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// Get channel polls endpoint
	app.Get("/channels/:id/polls", func(c *fiber.Ctx) error {
		channelID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid channel id",
			})
		}

		polls, err := pollService.GetGuildPolls(c.Context(), channelID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get polls",
			})
		}

		return c.JSON(polls)
	})

	return app
}

func TestPollHandler_CreatePoll(t *testing.T) {
	channelID := uuid.New()

	t.Run("successful poll creation", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		pollService.On("CreatePoll", mock.Anything, mock.AnythingOfType("*models.Poll")).Return(nil)

		body := CreatePollRequest{
			Question:   "What is your favorite color?",
			Options:    []string{"Red", "Blue", "Green"},
			IsMultiple: false,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/polls", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result models.Poll
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "What is your favorite color?", result.Question)
		assert.Len(t, result.Options, 3)

		pollService.AssertExpectations(t)
	})

	t.Run("missing question", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		body := CreatePollRequest{
			Question: "",
			Options:  []string{"Red", "Blue"},
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/polls", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "question is required", result["error"])
	})

	t.Run("too few options", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		body := CreatePollRequest{
			Question: "What is your favorite color?",
			Options:  []string{"Red"},
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/polls", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "at least 2 options are required", result["error"])
	})

	t.Run("too many options", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		body := CreatePollRequest{
			Question: "What is your favorite color?",
			Options:  []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"},
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/polls", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "maximum 10 options allowed", result["error"])
	})

	t.Run("invalid channel id", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		body := CreatePollRequest{
			Question: "What is your favorite color?",
			Options:  []string{"Red", "Blue"},
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/channels/invalid-uuid/polls", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "invalid channel id", result["error"])
	})

	t.Run("with valid end time", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		pollService.On("CreatePoll", mock.Anything, mock.AnythingOfType("*models.Poll")).Return(nil)

		futureTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		body := CreatePollRequest{
			Question: "What is your favorite color?",
			Options:  []string{"Red", "Blue"},
			EndTime:  &futureTime,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/polls", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		pollService.AssertExpectations(t)
	})

	t.Run("with past end time", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		pastTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		body := CreatePollRequest{
			Question: "What is your favorite color?",
			Options:  []string{"Red", "Blue"},
			EndTime:  &pastTime,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/channels/"+channelID.String()+"/polls", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "end_time must be in the future", result["error"])
	})
}

func TestPollHandler_GetPoll(t *testing.T) {
	t.Run("successful get poll", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(userID)

		pollService.On("GetPoll", mock.Anything, poll.ID).Return(poll, nil)

		req := httptest.NewRequest(http.MethodGet, "/polls/"+poll.ID.String(), nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result models.Poll
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, poll.ID, result.ID)
		assert.Equal(t, poll.Question, result.Question)

		pollService.AssertExpectations(t)
	})

	t.Run("poll not found", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		pollID := uuid.New()

		pollService.On("GetPoll", mock.Anything, pollID).Return(nil, assert.AnError)

		req := httptest.NewRequest(http.MethodGet, "/polls/"+pollID.String(), nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "poll not found", result["error"])

		pollService.AssertExpectations(t)
	})

	t.Run("invalid poll id", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		req := httptest.NewRequest(http.MethodGet, "/polls/invalid-uuid", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "invalid poll id", result["error"])
	})
}

func TestPollHandler_Vote(t *testing.T) {
	t.Run("successful vote", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(userID)
		optionID := poll.Options[0].ID

		pollService.On("Vote", mock.Anything, poll.ID, optionID, userID).Return(nil)

		body := VoteRequest{OptionID: optionID.String()}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/polls/"+poll.ID.String()+"/vote", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, true, result["success"])
		assert.Equal(t, "vote recorded", result["message"])

		pollService.AssertExpectations(t)
	})

	t.Run("already voted", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(userID)
		optionID := poll.Options[0].ID

		pollService.On("Vote", mock.Anything, poll.ID, optionID, userID).
			Return(errorWithMessage("user has already voted on this poll"))

		body := VoteRequest{OptionID: optionID.String()}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/polls/"+poll.ID.String()+"/vote", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "you have already voted on this poll", result["error"])

		pollService.AssertExpectations(t)
	})

	t.Run("invalid poll id", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		body := VoteRequest{OptionID: uuid.New().String()}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/polls/invalid-uuid/vote", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "invalid poll id", result["error"])
	})

	t.Run("invalid option id", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(userID)

		body := VoteRequest{OptionID: "invalid-uuid"}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/polls/"+poll.ID.String()+"/vote", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "invalid option id", result["error"])
	})
}

func TestPollHandler_GetResults(t *testing.T) {
	t.Run("successful get results", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(userID)

		pollService.On("GetPoll", mock.Anything, poll.ID).Return(poll, nil)

		req := httptest.NewRequest(http.MethodGet, "/polls/"+poll.ID.String()+"/results", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, poll.ID.String(), result["poll_id"])
		assert.Equal(t, poll.Question, result["question"])
		assert.Equal(t, float64(10), result["total_votes"]) // 5 + 3 + 2
		assert.Equal(t, false, result["is_closed"])

		results := result["results"].([]interface{})
		assert.Len(t, results, 3)

		// First option (Red) has 5 votes out of 10 = 50%
		firstResult := results[0].(map[string]interface{})
		assert.Equal(t, "Red", firstResult["text"])
		assert.Equal(t, float64(5), firstResult["votes"])
		assert.Equal(t, float64(50), firstResult["percentage"])

		pollService.AssertExpectations(t)
	})

	t.Run("poll not found", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		pollID := uuid.New()

		pollService.On("GetPoll", mock.Anything, pollID).Return(nil, assert.AnError)

		req := httptest.NewRequest(http.MethodGet, "/polls/"+pollID.String()+"/results", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		pollService.AssertExpectations(t)
	})

	t.Run("closed poll shows is_closed true", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(userID)
		pastTime := time.Now().Add(-1 * time.Hour)
		poll.EndTime = &pastTime

		pollService.On("GetPoll", mock.Anything, poll.ID).Return(poll, nil)

		req := httptest.NewRequest(http.MethodGet, "/polls/"+poll.ID.String()+"/results", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, true, result["is_closed"])

		pollService.AssertExpectations(t)
	})
}

func TestPollHandler_ClosePoll(t *testing.T) {
	t.Run("successful close poll", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(userID)

		pollService.On("GetPoll", mock.Anything, poll.ID).Return(poll, nil)
		pollService.On("UpdatePoll", mock.Anything, mock.AnythingOfType("*models.Poll")).Return(nil)

		req := httptest.NewRequest(http.MethodPost, "/polls/"+poll.ID.String()+"/close", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, true, result["success"])
		assert.Equal(t, "poll closed", result["message"])

		pollService.AssertExpectations(t)
	})

	t.Run("not poll creator", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(uuid.New()) // Different creator

		pollService.On("GetPoll", mock.Anything, poll.ID).Return(poll, nil)

		req := httptest.NewRequest(http.MethodPost, "/polls/"+poll.ID.String()+"/close", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "only the poll creator can close the poll", result["error"])

		pollService.AssertExpectations(t)
	})

	t.Run("poll already closed", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(userID)
		pastTime := time.Now().Add(-1 * time.Hour)
		poll.EndTime = &pastTime

		pollService.On("GetPoll", mock.Anything, poll.ID).Return(poll, nil)

		req := httptest.NewRequest(http.MethodPost, "/polls/"+poll.ID.String()+"/close", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "poll is already closed", result["error"])

		pollService.AssertExpectations(t)
	})

	t.Run("poll not found", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		pollID := uuid.New()

		pollService.On("GetPoll", mock.Anything, pollID).Return(nil, assert.AnError)

		req := httptest.NewRequest(http.MethodPost, "/polls/"+pollID.String()+"/close", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		pollService.AssertExpectations(t)
	})
}

func TestPollHandler_DeletePoll(t *testing.T) {
	t.Run("successful delete poll", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(userID)

		pollService.On("GetPoll", mock.Anything, poll.ID).Return(poll, nil)
		pollService.On("DeletePoll", mock.Anything, poll.ID).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/polls/"+poll.ID.String(), nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		pollService.AssertExpectations(t)
	})

	t.Run("not poll creator", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		poll := createTestPoll(uuid.New()) // Different creator

		pollService.On("GetPoll", mock.Anything, poll.ID).Return(poll, nil)

		req := httptest.NewRequest(http.MethodDelete, "/polls/"+poll.ID.String(), nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "only the poll creator can delete the poll", result["error"])

		pollService.AssertExpectations(t)
	})

	t.Run("poll not found", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		pollID := uuid.New()

		pollService.On("GetPoll", mock.Anything, pollID).Return(nil, assert.AnError)

		req := httptest.NewRequest(http.MethodDelete, "/polls/"+pollID.String(), nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		pollService.AssertExpectations(t)
	})
}

func TestPollHandler_GetChannelPolls(t *testing.T) {
	t.Run("successful get channel polls", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		channelID := uuid.New()
		polls := []*models.Poll{
			createTestPoll(userID),
			createTestPoll(userID),
		}

		pollService.On("GetGuildPolls", mock.Anything, channelID).Return(polls, nil)

		req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/polls", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result []*models.Poll
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Len(t, result, 2)

		pollService.AssertExpectations(t)
	})

	t.Run("empty channel polls", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)
		channelID := uuid.New()
		polls := []*models.Poll{}

		pollService.On("GetGuildPolls", mock.Anything, channelID).Return(polls, nil)

		req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/polls", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result []*models.Poll
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Len(t, result, 0)

		pollService.AssertExpectations(t)
	})

	t.Run("invalid channel id", func(t *testing.T) {
		pollService := new(MockPollService)
		userID := uuid.New()
		app := setupPollTestApp(pollService, userID)

		req := httptest.NewRequest(http.MethodGet, "/channels/invalid-uuid/polls", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "invalid channel id", result["error"])
	})
}

// Helper for creating errors with specific messages
type customError struct {
	msg string
}

func (e customError) Error() string {
	return e.msg
}

func errorWithMessage(msg string) error {
	return customError{msg: msg}
}
