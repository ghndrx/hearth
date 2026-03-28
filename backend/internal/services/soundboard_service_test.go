package services

import (
	"context"
	"mime/multipart"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

func TestSoundboardService_ValidateSoundUpload(t *testing.T) {
	service := &SoundboardService{}

	testCases := []struct {
		name      string
		size      int64
		mimeType  string
		expectErr error
	}{
		{
			name:      "valid MP3",
			size:      100 * 1024, // 100KB
			mimeType:  "audio/mpeg",
			expectErr: nil,
		},
		{
			name:      "valid OGG",
			size:      200 * 1024, // 200KB
			mimeType:  "audio/ogg",
			expectErr: nil,
		},
		{
			name:      "valid WAV",
			size:      300 * 1024, // 300KB
			mimeType:  "audio/wav",
			expectErr: nil,
		},
		{
			name:      "valid OPUS",
			size:      150 * 1024, // 150KB
			mimeType:  "audio/opus",
			expectErr: nil,
		},
		{
			name:      "file too large",
			size:      600 * 1024, // 600KB
			mimeType:  "audio/mpeg",
			expectErr: ErrSoundboardTooLarge,
		},
		{
			name:      "invalid format",
			size:      100 * 1024,
			mimeType:  "audio/flac",
			expectErr: ErrSoundboardFormat,
		},
		{
			name:      "not audio",
			size:      100 * 1024,
			mimeType:  "image/png",
			expectErr: ErrSoundboardFormat,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file := &multipart.FileHeader{
				Filename: "test.mp3",
				Header:   make(map[string][]string),
				Size:     tc.size,
			}
			file.Header.Set("Content-Type", tc.mimeType)

			err := service.ValidateSoundUpload(file)
			if tc.expectErr != nil {
				assert.Equal(t, tc.expectErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSoundboardService_Create(t *testing.T) {
	service := NewSoundboardService(nil)

	serverID := uuid.New()
	userID := uuid.New()

	t.Run("creates sound with valid data", func(t *testing.T) {
		file := &multipart.FileHeader{
			Filename: "test.mp3",
			Header:   make(map[string][]string),
			Size:     100 * 1024,
		}
		file.Header.Set("Content-Type", "audio/mpeg")

		sound, err := service.Create(context.Background(), &serverID, "Test Sound", "🔊", 0.8, file, userID)

		assert.NoError(t, err)
		assert.NotNil(t, sound)
		assert.Equal(t, "Test Sound", sound.Name)
		assert.Equal(t, "🔊", sound.EmojiName)
		assert.Equal(t, 0.8, sound.Volume)
		assert.Equal(t, serverID, *sound.ServerID)
		assert.Equal(t, userID, sound.CreatorID)
		assert.True(t, sound.Available)
		assert.NotEqual(t, uuid.Nil, sound.ID)
	})

	t.Run("creates sound with default volume", func(t *testing.T) {
		file := &multipart.FileHeader{
			Filename: "test.mp3",
			Header:   make(map[string][]string),
			Size:     100 * 1024,
		}
		file.Header.Set("Content-Type", "audio/mpeg")

		sound, err := service.Create(context.Background(), &serverID, "Test Sound 2", "", 0, file, userID)

		assert.NoError(t, err)
		assert.Equal(t, 1.0, sound.Volume) // Default volume
	})

	t.Run("creates global sound with nil serverID", func(t *testing.T) {
		file := &multipart.FileHeader{
			Filename: "default.mp3",
			Header:   make(map[string][]string),
			Size:     50 * 1024,
		}
		file.Header.Set("Content-Type", "audio/mpeg")

		sound, err := service.Create(context.Background(), nil, "Default Sound", "🎵", 1.0, file, userID)

		assert.NoError(t, err)
		assert.Nil(t, sound.ServerID)
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		file := &multipart.FileHeader{
			Filename: "test.mp3",
			Header:   make(map[string][]string),
			Size:     100 * 1024,
		}
		file.Header.Set("Content-Type", "audio/mpeg")

		sound, err := service.Create(context.Background(), &serverID, "", "", 1.0, file, userID)

		assert.Error(t, err)
		assert.Nil(t, sound)
		assert.Equal(t, ErrSoundboardNameRequired, err)
	})

	t.Run("returns error for name too long", func(t *testing.T) {
		file := &multipart.FileHeader{
			Filename: "test.mp3",
			Header:   make(map[string][]string),
			Size:     100 * 1024,
		}
		file.Header.Set("Content-Type", "audio/mpeg")

		longName := ""
		for i := 0; i < 101; i++ {
			longName += "a"
		}

		sound, err := service.Create(context.Background(), &serverID, longName, "", 1.0, file, userID)

		assert.Error(t, err)
		assert.Nil(t, sound)
		assert.Equal(t, ErrSoundboardNameTooLong, err)
	})
}

func TestSoundboardService_Get(t *testing.T) {
	service := NewSoundboardService(nil)
	ctx := context.Background()

	sound := &models.SoundboardSound{
		ID:         uuid.New(),
		ServerID:   nil,
		Name:       "Test Sound",
		EmojiName:  "🔊",
		Volume:     0.8,
		AudioURL:   "/test.mp3",
		DurationMs: 1000,
		Available:  true,
		CreatorID:  uuid.New(),
		CreatedAt:  time.Now(),
	}
	service.Add_Test(sound)

	t.Run("returns sound by ID", func(t *testing.T) {
		result, err := service.Get(ctx, sound.ID)

		assert.NoError(t, err)
		assert.Equal(t, sound.ID, result.ID)
		assert.Equal(t, sound.Name, result.Name)
	})

	t.Run("returns error for non-existent ID", func(t *testing.T) {
		result, err := service.Get(ctx, uuid.New())

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrSoundboardSoundNotFound, err)
	})
}

func TestSoundboardService_GetByServer(t *testing.T) {
	service := NewSoundboardService(nil)
	ctx := context.Background()

	serverID1 := uuid.New()
	serverID2 := uuid.New()
	userID := uuid.New()

	// Add sounds for server1
	for i := 0; i < 3; i++ {
		s := &models.SoundboardSound{
			ID:         uuid.New(),
			ServerID:   &serverID1,
			Name:       "Sound1",
			Volume:     1.0,
			AudioURL:   "/test1.mp3",
			DurationMs: 1000,
			Available:  true,
			CreatorID:  userID,
			CreatedAt:  time.Now(),
		}
		service.Add_Test(s)
	}

	// Add sounds for server2
	for i := 0; i < 2; i++ {
		s := &models.SoundboardSound{
			ID:         uuid.New(),
			ServerID:   &serverID2,
			Name:       "Sound2",
			Volume:     1.0,
			AudioURL:   "/test2.mp3",
			DurationMs: 1000,
			Available:  true,
			CreatorID:  userID,
			CreatedAt:  time.Now(),
		}
		service.Add_Test(s)
	}

	t.Run("returns sounds for server", func(t *testing.T) {
		sounds, err := service.GetByServer(ctx, serverID1)

		assert.NoError(t, err)
		assert.Len(t, sounds, 3)
	})

	t.Run("returns sounds for different server", func(t *testing.T) {
		sounds, err := service.GetByServer(ctx, serverID2)

		assert.NoError(t, err)
		assert.Len(t, sounds, 2)
	})

	t.Run("returns empty for server with no sounds", func(t *testing.T) {
		sounds, err := service.GetByServer(ctx, uuid.New())

		assert.NoError(t, err)
		assert.Len(t, sounds, 0)
	})
}

func TestSoundboardService_Update(t *testing.T) {
	service := NewSoundboardService(nil)
	ctx := context.Background()

	sound := &models.SoundboardSound{
		ID:         uuid.New(),
		ServerID:   nil,
		Name:       "Original Name",
		EmojiName:  "🔊",
		Volume:     0.5,
		AudioURL:   "/test.mp3",
		DurationMs: 1000,
		Available:  true,
		CreatorID:  uuid.New(),
		CreatedAt:  time.Now(),
	}
	service.Add_Test(sound)

	t.Run("updates name", func(t *testing.T) {
		newName := "Updated Name"
		updated, err := service.Update(ctx, sound.ID, newName, "", nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, newName, updated.Name)
	})

	t.Run("updates emoji", func(t *testing.T) {
		newEmoji := "🎵"
		updated, err := service.Update(ctx, sound.ID, "", newEmoji, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, newEmoji, updated.EmojiName)
	})

	t.Run("updates volume", func(t *testing.T) {
		newVolume := 0.7
		updated, err := service.Update(ctx, sound.ID, "", "", &newVolume, nil)

		assert.NoError(t, err)
		assert.Equal(t, newVolume, updated.Volume)
	})

	t.Run("updates availability", func(t *testing.T) {
		newAvailable := false
		updated, err := service.Update(ctx, sound.ID, "", "", nil, &newAvailable)

		assert.NoError(t, err)
		assert.Equal(t, newAvailable, updated.Available)
	})

	t.Run("returns error for non-existent sound", func(t *testing.T) {
		updated, err := service.Update(ctx, uuid.New(), "New Name", "", nil, nil)

		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Equal(t, ErrSoundboardSoundNotFound, err)
	})
}

func TestSoundboardService_Delete(t *testing.T) {
	service := NewSoundboardService(nil)
	ctx := context.Background()

	sound := &models.SoundboardSound{
		ID:         uuid.New(),
		ServerID:   nil,
		Name:       "To Delete",
		Volume:     1.0,
		AudioURL:   "/test.mp3",
		DurationMs: 1000,
		Available:  true,
		CreatorID:  uuid.New(),
		CreatedAt:  time.Now(),
	}
	service.Add_Test(sound)

	t.Run("deletes existing sound", func(t *testing.T) {
		err := service.Delete(ctx, sound.ID)

		assert.NoError(t, err)

		// Verify deleted
		result, getErr := service.Get(ctx, sound.ID)
		assert.Error(t, getErr)
		assert.Nil(t, result)
	})

	t.Run("returns error for non-existent sound", func(t *testing.T) {
		err := service.Delete(ctx, uuid.New())

		assert.Error(t, err)
		assert.Equal(t, ErrSoundboardSoundNotFound, err)
	})
}

func TestSoundboardService_Search(t *testing.T) {
	service := NewSoundboardService(nil)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	// Add test sounds
	sounds := []*models.SoundboardSound{
		{ID: uuid.New(), ServerID: &serverID, Name: "Drum Beat", EmojiName: "🥁", Volume: 1.0, AudioURL: "/drum.mp3", DurationMs: 500, Available: true, CreatorID: userID, CreatedAt: time.Now()},
		{ID: uuid.New(), ServerID: &serverID, Name: "Cymbal Crash", EmojiName: "🔔", Volume: 0.8, AudioURL: "/cymbal.mp3", DurationMs: 1000, Available: true, CreatorID: userID, CreatedAt: time.Now()},
		{ID: uuid.New(), ServerID: &serverID, Name: "Bass Drop", EmojiName: "🎸", Volume: 1.0, AudioURL: "/bass.mp3", DurationMs: 2000, Available: false, CreatorID: userID, CreatedAt: time.Now()},
		{ID: uuid.New(), ServerID: nil, Name: "Airhorn", EmojiName: "📢", Volume: 1.0, AudioURL: "/airhorn.mp3", DurationMs: 500, Available: true, CreatorID: userID, CreatedAt: time.Now()},
	}
	for _, s := range sounds {
		service.Add_Test(s)
	}

	t.Run("search by name", func(t *testing.T) {
		results, err := service.Search(ctx, "drum", &serverID)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Drum Beat", results[0].Name)
	})

	t.Run("search by emoji name", func(t *testing.T) {
		results, err := service.Search(ctx, "crash", &serverID)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Cymbal Crash", results[0].Name)
	})

	t.Run("search is case insensitive", func(t *testing.T) {
		results, err := service.Search(ctx, "DRUM", nil)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Drum Beat", results[0].Name)
	})

	t.Run("unavailable sounds are excluded", func(t *testing.T) {
		results, err := service.Search(ctx, "bass", &serverID)

		assert.NoError(t, err)
		assert.Len(t, results, 0) // Bass Drop is not available
	})

	t.Run("search with nil serverID returns global sounds", func(t *testing.T) {
		results, err := service.Search(ctx, "airhorn", nil)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Airhorn", results[0].Name)
	})
}

func TestSoundboardService_GetAvailable(t *testing.T) {
	service := NewSoundboardService(nil)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	// Add test sounds
	sounds := []*models.SoundboardSound{
		{ID: uuid.New(), ServerID: nil, Name: "Global Sound", Volume: 1.0, AudioURL: "/global.mp3", DurationMs: 500, Available: true, CreatorID: userID, CreatedAt: time.Now()},
		{ID: uuid.New(), ServerID: &serverID, Name: "Server Sound", Volume: 1.0, AudioURL: "/server.mp3", DurationMs: 500, Available: true, CreatorID: userID, CreatedAt: time.Now()},
		{ID: uuid.New(), ServerID: &serverID, Name: "Unavailable Sound", Volume: 1.0, AudioURL: "/unavail.mp3", DurationMs: 500, Available: false, CreatorID: userID, CreatedAt: time.Now()},
	}
	for _, s := range sounds {
		service.Add_Test(s)
	}

	t.Run("returns available sounds for server", func(t *testing.T) {
		results, err := service.GetAvailable(ctx, &serverID)

		assert.NoError(t, err)
		assert.Len(t, results, 2) // Global + Server (unavailable excluded)
	})

	t.Run("returns only global sounds when serverID is nil", func(t *testing.T) {
		results, err := service.GetAvailable(ctx, nil)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Global Sound", results[0].Name)
	})
}
