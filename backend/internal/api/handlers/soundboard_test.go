package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
)

// setupSoundboardTestApp creates a test Fiber app with soundboard routes
func setupSoundboardTestApp(handler *SoundboardHandler) *fiber.App {
	app := fiber.New()

	if handler != nil {
		// Default sounds
		app.Get("/soundboard/defaults", handler.ListDefaultSounds)

		// Server sounds
		app.Get("/servers/:id/soundboard", handler.ListServerSounds)

		// Individual sound operations
		app.Get("/soundboard/:soundId", handler.GetSound)
		app.Post("/servers/:id/soundboard", handler.CreateSound)
		app.Patch("/soundboard/:soundId", handler.ModifySound)
		app.Delete("/soundboard/:soundId", handler.DeleteSound)
	}

	return app
}

func TestSoundboardHandler_ListDefaultSounds_Success(t *testing.T) {
	svc := services.NewSoundboardService(nil)

	// Add a default sound
	defaultSound := &models.SoundboardSound{
		ID:         uuid.New(),
		ServerID:   nil, // default sound
		Name:       "Airhorn",
		EmojiName:  "📢",
		Volume:     1.0,
		AudioURL:   "/soundboard/airhorn.mp3",
		DurationMs: 500,
		Available:  true,
		CreatorID:  uuid.New(),
	}
	svc.Add_Test(defaultSound)

	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	req := httptest.NewRequest("GET", "/soundboard/defaults", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var sounds []*models.SoundboardSound
	err = json.NewDecoder(resp.Body).Decode(&sounds)
	require.NoError(t, err)
	assert.Len(t, sounds, 1)
	assert.Equal(t, "Airhorn", sounds[0].Name)
}

func TestSoundboardHandler_ListDefaultSounds_Empty(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	req := httptest.NewRequest("GET", "/soundboard/defaults", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var sounds []*models.SoundboardSound
	err = json.NewDecoder(resp.Body).Decode(&sounds)
	require.NoError(t, err)
	assert.Len(t, sounds, 0)
}

func TestSoundboardHandler_ListServerSounds_Success(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	serverID := uuid.New()

	// Add a server-specific sound
	serverSound := &models.SoundboardSound{
		ID:         uuid.New(),
		ServerID:   &serverID,
		Name:       "Tada",
		EmojiName:  "🎉",
		Volume:     0.8,
		AudioURL:   "/soundboard/tada.mp3",
		DurationMs: 1000,
		Available:  true,
		CreatorID:  uuid.New(),
	}
	svc.Add_Test(serverSound)

	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/soundboard", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var sounds []*models.SoundboardSound
	err = json.NewDecoder(resp.Body).Decode(&sounds)
	require.NoError(t, err)
	assert.Len(t, sounds, 1)
	assert.Equal(t, "Tada", sounds[0].Name)
}

func TestSoundboardHandler_ListServerSounds_InvalidServerID(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	req := httptest.NewRequest("GET", "/servers/not-a-uuid/soundboard", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var errResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errResp)
	assert.Equal(t, "invalid server ID", errResp["error"])
}

func TestSoundboardHandler_GetSound_Success(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	soundID := uuid.New()

	sound := &models.SoundboardSound{
		ID:         soundID,
		ServerID:   nil,
		Name:       "DrumRoll",
		EmojiName:  "🥁",
		Volume:     1.0,
		AudioURL:   "/soundboard/drumroll.mp3",
		DurationMs: 2000,
		Available:  true,
		CreatorID:  uuid.New(),
	}
	svc.Add_Test(sound)

	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	req := httptest.NewRequest("GET", "/soundboard/"+soundID.String(), nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.SoundboardSound
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "DrumRoll", result.Name)
}

func TestSoundboardHandler_GetSound_InvalidSoundID(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	req := httptest.NewRequest("GET", "/soundboard/not-a-uuid", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var errResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errResp)
	assert.Equal(t, "invalid sound ID", errResp["error"])
}

func TestSoundboardHandler_GetSound_NotFound(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	nonExistentID := uuid.New()
	req := httptest.NewRequest("GET", "/soundboard/"+nonExistentID.String(), nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var errResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errResp)
	assert.Equal(t, "sound not found", errResp["error"])
}

func TestSoundboardHandler_ModifySound_Success(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	soundID := uuid.New()

	sound := &models.SoundboardSound{
		ID:         soundID,
		ServerID:   nil,
		Name:       "OriginalName",
		EmojiName:  "🎵",
		Volume:     0.5,
		AudioURL:   "/soundboard/original.mp3",
		DurationMs: 500,
		Available:  true,
		CreatorID:  uuid.New(),
	}
	svc.Add_Test(sound)

	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	newName := "NewName"
	newVolume := 0.9
	reqBody := map[string]interface{}{
		"name":   newName,
		"volume": newVolume,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/soundboard/"+soundID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.SoundboardSound
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "NewName", result.Name)
	assert.Equal(t, 0.9, result.Volume)
}

func TestSoundboardHandler_ModifySound_InvalidSoundID(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	reqBody := map[string]interface{}{"name": "NewName"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/soundboard/not-a-uuid", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSoundboardHandler_ModifySound_InvalidBody(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	soundID := uuid.New()
	req := httptest.NewRequest("PATCH", "/soundboard/"+soundID.String(), bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSoundboardHandler_DeleteSound_Success(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	soundID := uuid.New()

	sound := &models.SoundboardSound{
		ID:         soundID,
		ServerID:   nil,
		Name:       "ToDelete",
		EmojiName:  "🗑️",
		Volume:     1.0,
		AudioURL:   "/soundboard/delete.mp3",
		DurationMs: 100,
		Available:  true,
		CreatorID:  uuid.New(),
	}
	svc.Add_Test(sound)

	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	req := httptest.NewRequest("DELETE", "/soundboard/"+soundID.String(), nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)

	// Verify sound is deleted
	_, err = svc.Get(context.Background(), soundID)
	assert.Error(t, err)
}

func TestSoundboardHandler_DeleteSound_InvalidSoundID(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	req := httptest.NewRequest("DELETE", "/soundboard/not-a-uuid", nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var errResp map[string]string
	json.NewDecoder(resp.Body).Decode(&errResp)
	assert.Equal(t, "invalid sound ID", errResp["error"])
}

func TestSoundboardHandler_DeleteSound_NotFound(t *testing.T) {
	svc := services.NewSoundboardService(nil)
	handler := NewSoundboardHandler(svc, nil, nil)
	app := setupSoundboardTestApp(handler)
	defer app.Shutdown()

	nonExistentID := uuid.New()
	req := httptest.NewRequest("DELETE", "/soundboard/"+nonExistentID.String(), nil)
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// Test validateAudioURL function
func TestValidateAudioURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid mp3",
			url:     "https://example.com/sounds/airhorn.mp3",
			wantErr: false,
		},
		{
			name:    "valid wav",
			url:     "https://example.com/sounds/drum.wav",
			wantErr: false,
		},
		{
			name:    "valid ogg",
			url:     "https://example.com/sounds/bell.ogg",
			wantErr: false,
		},
		{
			name:    "valid m4a",
			url:     "https://example.com/sounds/ping.m4a",
			wantErr: false,
		},
		{
			name:    "valid aac",
			url:     "https://example.com/sounds/beep.aac",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			url:     "ftp://example.com/sounds/airhorn.mp3",
			wantErr: true,
		},
		{
			name:    "invalid extension",
			url:     "https://example.com/sounds/video.mp4",
			wantErr: true,
		},
		{
			name:    "no extension",
			url:     "https://example.com/sounds/airhorn",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAudioURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
