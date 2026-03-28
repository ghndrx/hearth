package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// strPtr is a helper to get a pointer to a string literal
func strPtr(s string) *string {
	return &s
}

// TestInteractionParsing tests the interaction request parsing
func TestInteractionParsing(t *testing.T) {
	t.Run("parses application command interaction", func(t *testing.T) {
		interactionReq := InteractionRequest{
			ID:        uuid.New(),
			Type:      int(models.InteractionTypeApplicationCommand),
			UserID:    uuid.New().String(),
			ChannelID: uuid.New().String(),
			Token:     "test-command",
			AppID:     uuid.New().String(),
			Data: map[string]interface{}{
				"id":   uuid.New().String(),
				"name": "ping",
				"type": 1,
			},
		}

		body, _ := json.Marshal(interactionReq)

		var parsed InteractionRequest
		err := json.Unmarshal(body, &parsed)
		assert.NoError(t, err)
		assert.Equal(t, interactionReq.ID, parsed.ID)
		assert.Equal(t, "ping", parsed.Data.(map[string]interface{})["name"])
	})

	t.Run("parses interaction with guild ID", func(t *testing.T) {
		serverID := uuid.New().String()
		interactionReq := InteractionRequest{
			ID:        uuid.New(),
			Type:      int(models.InteractionTypeApplicationCommand),
			UserID:    uuid.New().String(),
			ChannelID: uuid.New().String(),
			ServerID:  &serverID,
			Token:     "guild-test",
			AppID:     uuid.New().String(),
		}

		body, _ := json.Marshal(interactionReq)

		var parsed InteractionRequest
		err := json.Unmarshal(body, &parsed)
		assert.NoError(t, err)
		assert.NotNil(t, parsed.ServerID)
		assert.Equal(t, serverID, *parsed.ServerID)
	})
}

// TestInteractionTypes verifies interaction type constants
func TestInteractionTypes(t *testing.T) {
	if int(models.InteractionTypePing) != 1 {
		t.Errorf("InteractionTypePing = %d, want 1", models.InteractionTypePing)
	}
	if int(models.InteractionTypeApplicationCommand) != 2 {
		t.Errorf("InteractionTypeApplicationCommand = %d, want 2", models.InteractionTypeApplicationCommand)
	}
	if int(models.InteractionTypeMessageComponent) != 3 {
		t.Errorf("InteractionTypeMessageComponent = %d, want 3", models.InteractionTypeMessageComponent)
	}
	if int(models.InteractionTypeAutocomplete) != 4 {
		t.Errorf("InteractionTypeAutocomplete = %d, want 4", models.InteractionTypeAutocomplete)
	}
	if int(models.InteractionTypeModalSubmit) != 5 {
		t.Errorf("InteractionTypeModalSubmit = %d, want 5", models.InteractionTypeModalSubmit)
	}
}

// TestCallbackTypes verifies callback type constants
func TestCallbackTypes(t *testing.T) {
	if int(models.CallbackTypePong) != 1 {
		t.Errorf("CallbackTypePong = %d, want 1", models.CallbackTypePong)
	}
	if int(models.CallbackTypeChannelMessage) != 4 {
		t.Errorf("CallbackTypeChannelMessage = %d, want 4", models.CallbackTypeChannelMessage)
	}
	if int(models.CallbackTypeDeferredChannelMessage) != 5 {
		t.Errorf("CallbackTypeDeferredChannelMessage = %d, want 5", models.CallbackTypeDeferredChannelMessage)
	}
	if int(models.CallbackTypeAutocompleteResult) != 8 {
		t.Errorf("CallbackTypeAutocompleteResult = %d, want 8", models.CallbackTypeAutocompleteResult)
	}
}

// TestInteractionRequest validates the InteractionRequest structure
func TestInteractionRequest(t *testing.T) {
	t.Run("required fields", func(t *testing.T) {
		req := InteractionRequest{
			ID:        uuid.New(),
			Type:      int(models.InteractionTypePing),
			Token:     "token",
			ChannelID: uuid.New().String(),
			AppID:     uuid.New().String(),
		}

		assert.NotEqual(t, uuid.Nil, req.ID)
		assert.Equal(t, 1, req.Type)
		assert.NotEmpty(t, req.Token)
	})

	t.Run("optional guild ID", func(t *testing.T) {
		serverID := uuid.New().String()
		req := InteractionRequest{
			ID:        uuid.New(),
			Type:      int(models.InteractionTypeApplicationCommand),
			ServerID:  &serverID,
			Token:     "token",
			ChannelID: uuid.New().String(),
			AppID:     uuid.New().String(),
		}

		assert.NotNil(t, req.ServerID)
	})
}

// TestInteractionResponseJSON verifies response serialization
func TestInteractionResponseJSON(t *testing.T) {
	resp := models.InteractionResponse{
		Type: models.CallbackTypeChannelMessage,
		Data: &models.InteractionCallbackData{
			Content: strPtr("Hello, world!"),
			Flags:   0,
		},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, float64(models.CallbackTypeChannelMessage), parsed["type"])
}

// TestAutocompleteResult verifies autocomplete response format
func TestAutocompleteResult(t *testing.T) {
	resp := models.InteractionResponse{
		Type: models.CallbackTypeAutocompleteResult,
		Data: &models.InteractionCallbackData{
			Choices: []*models.CommandChoice{
				{Name: "Option 1", Value: "opt1"},
				{Name: "Option 2", Value: "opt2"},
			},
		},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed models.InteractionResponse
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, models.CallbackTypeAutocompleteResult, parsed.Type)
	assert.Len(t, parsed.Data.Choices, 2)
}

// TestEphemeralMessage verifies ephemeral message flag
func TestEphemeralMessage(t *testing.T) {
	resp := models.InteractionResponse{
		Type: models.CallbackTypeChannelMessage,
		Data: &models.InteractionCallbackData{
			Content: strPtr("Secret message"),
			Flags:   64, // 1 << 6 = Ephemeral
		},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	assert.Equal(t, float64(64), parsed["data"].(map[string]interface{})["flags"])
}

// TestModalResponse verifies modal response format
func TestModalResponse(t *testing.T) {
	resp := models.InteractionResponse{
		Type: models.CallbackTypeModal,
		Data: &models.InteractionCallbackData{
			Title: strPtr("My Modal"),
		},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed models.InteractionResponse
	json.Unmarshal(data, &parsed)
	assert.Equal(t, models.CallbackTypeModal, parsed.Type)
}

// TestDeferredResponse verifies deferred response format
func TestDeferredResponse(t *testing.T) {
	resp := models.InteractionResponse{
		Type: models.CallbackTypeDeferredChannelMessage,
		Data: &models.InteractionCallbackData{},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed models.InteractionResponse
	json.Unmarshal(data, &parsed)
	assert.Equal(t, models.CallbackTypeDeferredChannelMessage, parsed.Type)
}

// TestComponentInteractionData verifies component interaction data parsing
func TestComponentInteractionData(t *testing.T) {
	data := map[string]interface{}{
		"custom_id":      "button-123",
		"component_type": 2, // Button
	}

	jsonData, _ := json.Marshal(data)

	var parsed map[string]interface{}
	err := json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "button-123", parsed["custom_id"])
}

// TestModalSubmitData verifies modal submit data parsing
func TestModalSubmitData(t *testing.T) {
	data := map[string]interface{}{
		"custom_id": "modal-form-1",
		"components": []interface{}{
			map[string]interface{}{
				"components": []interface{}{
					map[string]interface{}{
						"custom_id": "input-field",
						"value":     "user typed this",
					},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(data)

	var parsed map[string]interface{}
	err := json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "modal-form-1", parsed["custom_id"])

	components := parsed["components"].([]interface{})
	assert.Len(t, components, 1)
}

// TestInteractionEndpointRoute verifies the route configuration
func TestInteractionEndpointRoute(t *testing.T) {
	app := fiber.New()

	// Mock handler that just returns OK
	app.Post("/api/v1/interactions", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestInteractionEndpointWithGuild verifies interaction with guild ID
func TestInteractionEndpointWithGuild(t *testing.T) {
	app := fiber.New()
	serverID := uuid.New().String()

	var capturedServerID string
	app.Post("/api/v1/interactions", func(c *fiber.Ctx) error {
		var req InteractionRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad"})
		}
		if req.ServerID != nil {
			capturedServerID = *req.ServerID
		}
		return c.SendStatus(fiber.StatusOK)
	})

	interactionReq := InteractionRequest{
		ID:        uuid.New(),
		Type:      int(models.InteractionTypeApplicationCommand),
		UserID:    uuid.New().String(),
		ChannelID: uuid.New().String(),
		ServerID:  &serverID,
		Token:     "test",
		AppID:     uuid.New().String(),
	}

	body, _ := json.Marshal(interactionReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, serverID, capturedServerID)
}

// TestInteractionTokenValidation verifies token is captured
func TestInteractionTokenValidation(t *testing.T) {
	app := fiber.New()
	var capturedToken string

	app.Post("/api/v1/interactions", func(c *fiber.Ctx) error {
		var req InteractionRequest
		c.BodyParser(&req)
		capturedToken = req.Token
		return c.SendStatus(fiber.StatusOK)
	})

	token := "unique-token-12345"
	interactionReq := InteractionRequest{
		ID:        uuid.New(),
		Type:      int(models.InteractionTypeApplicationCommand),
		Token:     token,
		ChannelID: uuid.New().String(),
		AppID:     uuid.New().String(),
	}

	body, _ := json.Marshal(interactionReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, token, capturedToken)
}
