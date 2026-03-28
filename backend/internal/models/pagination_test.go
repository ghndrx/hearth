package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMemberCursor_EncodeDecode(t *testing.T) {
	tests := []struct {
		name    string
		cursor  *MemberCursor
		wantErr bool
	}{
		{
			name: "valid cursor",
			cursor: &MemberCursor{
				JoinedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				UserID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			},
			wantErr: false,
		},
		{
			name:    "nil cursor encodes to empty string",
			cursor:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := tt.cursor.Encode()

			if tt.cursor == nil {
				assert.Empty(t, encoded)
				return
			}

			assert.NotEmpty(t, encoded)

			// Decode
			decoded, err := DecodeMemberCursor(encoded)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, decoded)
			assert.Equal(t, tt.cursor.UserID, decoded.UserID)
			assert.True(t, tt.cursor.JoinedAt.Equal(decoded.JoinedAt))
		})
	}
}

func TestDecodeMemberCursor_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		cursor  string
		wantNil bool
		wantErr bool
	}{
		{
			name:    "empty cursor returns nil",
			cursor:  "",
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "invalid base64",
			cursor:  "not-valid-base64!!!",
			wantNil: false,
			wantErr: true,
		},
		{
			name:    "valid base64 but invalid JSON",
			cursor:  "bm90LWpzb24=", // "not-json" in base64
			wantNil: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := DecodeMemberCursor(tt.cursor)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, decoded)
			}
		})
	}
}

func TestPresenceCursor_EncodeDecode(t *testing.T) {
	userID := uuid.New()
	cursor := &PresenceCursor{UserID: userID}

	encoded := cursor.Encode()
	assert.NotEmpty(t, encoded)

	decoded, err := DecodePresenceCursor(encoded)
	assert.NoError(t, err)
	assert.NotNil(t, decoded)
	assert.Equal(t, userID, decoded.UserID)
}

func TestDecodePresenceCursor_Empty(t *testing.T) {
	decoded, err := DecodePresenceCursor("")
	assert.NoError(t, err)
	assert.Nil(t, decoded)
}

func TestNormalizeLimit(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"negative returns default", -1, DefaultPageLimit},
		{"zero returns default", 0, DefaultPageLimit},
		{"valid limit unchanged", 100, 100},
		{"over max returns max", 2000, MaxPageLimit},
		{"exact max returns max", MaxPageLimit, MaxPageLimit},
		{"exact default returns default", DefaultPageLimit, DefaultPageLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeLimit(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMemberCursor_RoundTrip(t *testing.T) {
	// Test that multiple encode/decode cycles produce identical results
	original := &MemberCursor{
		JoinedAt: time.Now().UTC().Truncate(time.Millisecond), // Truncate for comparison
		UserID:   uuid.New(),
	}

	encoded1 := original.Encode()
	decoded1, _ := DecodeMemberCursor(encoded1)

	encoded2 := decoded1.Encode()
	decoded2, _ := DecodeMemberCursor(encoded2)

	assert.Equal(t, original.UserID, decoded2.UserID)
	assert.True(t, original.JoinedAt.Equal(decoded2.JoinedAt))
}
