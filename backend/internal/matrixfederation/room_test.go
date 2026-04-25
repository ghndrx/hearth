package matrixfederation

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/matrix"
)

func TestParseRoomID(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    RoomID
		wantErr bool
	}{
		{
			name: "valid room ID",
			raw:  "!abcdef:hearth.example.com",
			want: RoomID{Localpart: "abcdef", ServerName: "hearth.example.com"},
		},
		{
			name: "valid with complex localpart",
			raw:  "!OeiCDxQZDB_VvI6:hearth.example.com",
			want: RoomID{Localpart: "OeiCDxQZDB_VvI6", ServerName: "hearth.example.com"},
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: true,
		},
		{
			name:    "missing bang",
			raw:     "abcdef:hearth.example.com",
			wantErr: true,
		},
		{
			name:    "missing colon",
			raw:     "!abcdef",
			wantErr: true,
		},
		{
			name:    "empty localpart",
			raw:     "!:hearth.example.com",
			wantErr: true,
		},
		{
			name:    "empty server",
			raw:     "!abcdef:",
			wantErr: true,
		},
		{
			name:    "invalid characters in localpart",
			raw:     "!abc@def:hearth.example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRoomID(tt.raw)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.raw, got.String())
		})
	}
}

func TestParseAlias(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Alias
		wantErr bool
	}{
		{
			name: "valid alias",
			raw:  "#general:hearth.example.com",
			want: Alias{Localpart: "general", ServerName: "hearth.example.com"},
		},
		{
			name: "valid with hyphen",
			raw:  "#dev-chat:hearth.example.com",
			want: Alias{Localpart: "dev-chat", ServerName: "hearth.example.com"},
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: true,
		},
		{
			name:    "missing hash",
			raw:     "general:hearth.example.com",
			wantErr: true,
		},
		{
			name:    "missing colon",
			raw:     "#general",
			wantErr: true,
		},
		{
			name:    "empty localpart",
			raw:     "#:hearth.example.com",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			raw:     "#gen@eral:hearth.example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAlias(tt.raw)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.raw, got.String())
		})
	}
}

func TestGenerateRoomID(t *testing.T) {
	serverName := "hearth.example.com"
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	roomID := GenerateRoomID(channelID, serverName)

	assert.Equal(t, serverName, roomID.ServerName)
	assert.NotEmpty(t, roomID.Localpart)
	assert.True(t, roomID.IsValid())

	// Should be deterministic.
	roomID2 := GenerateRoomID(channelID, serverName)
	assert.Equal(t, roomID, roomID2)

	// Should be parseable.
	parsed, err := ParseRoomID(roomID.String())
	require.NoError(t, err)
	assert.Equal(t, roomID, parsed)
}

func TestParseRoomIDLocalpart(t *testing.T) {
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	roomID := GenerateRoomID(channelID, "hearth.example.com")
	parsedID, err := ParseRoomIDLocalpart(roomID.Localpart)
	require.NoError(t, err)
	assert.Equal(t, channelID, parsedID)
}

func TestInMemoryRoomAliasStore_CreateMapping(t *testing.T) {
	store := NewInMemoryRoomAliasStore()
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	aliases := []Alias{
		{Localpart: "general", ServerName: "hearth.example.com"},
		{Localpart: "main", ServerName: "hearth.example.com"},
	}

	// Create initial mapping.
	err := store.CreateMapping(ctx, roomID, channelID, aliases)
	require.NoError(t, err)

	// Verify we can retrieve by room ID.
	gotChanID, gotAliases, err := store.GetByRoomID(ctx, roomID)
	require.NoError(t, err)
	assert.Equal(t, channelID, gotChanID)
	assert.Len(t, gotAliases, 2)

	// Verify we can retrieve by alias.
	for _, alias := range aliases {
		gotRoomID, gotChanID, err := store.GetByAlias(ctx, alias)
		require.NoError(t, err)
		assert.Equal(t, roomID, gotRoomID)
		assert.Equal(t, channelID, gotChanID)
	}

	// Verify we can retrieve by channel ID.
	gotRoomID, gotAliases, err := store.GetByChannelID(ctx, channelID)
	require.NoError(t, err)
	assert.Equal(t, roomID, gotRoomID)
	assert.Len(t, gotAliases, 2)

	// Duplicate should fail.
	err = store.CreateMapping(ctx, roomID, uuid.New(), nil)
	assert.Error(t, err)

	// Same alias should fail for different room.
	err = store.CreateMapping(ctx, RoomID{Localpart: "other", ServerName: "hearth.example.com"}, uuid.New(), aliases[:1])
	assert.Error(t, err)
}

func TestInMemoryRoomAliasStore_GetByRoomID_NotFound(t *testing.T) {
	store := NewInMemoryRoomAliasStore()
	ctx := context.Background()

	_, _, err := store.GetByRoomID(ctx, RoomID{Localpart: "nonexistent", ServerName: "hearth.example.com"})
	assert.ErrorIs(t, err, ErrRoomNotFound)
}

func TestInMemoryRoomAliasStore_GetByAlias_NotFound(t *testing.T) {
	store := NewInMemoryRoomAliasStore()
	ctx := context.Background()

	_, _, err := store.GetByAlias(ctx, Alias{Localpart: "nonexistent", ServerName: "hearth.example.com"})
	assert.ErrorIs(t, err, ErrAliasNotFound)
}

func TestInMemoryRoomAliasStore_AddAlias(t *testing.T) {
	store := NewInMemoryRoomAliasStore()
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	err := store.CreateMapping(ctx, roomID, channelID, nil)
	require.NoError(t, err)

	newAlias := Alias{Localpart: "new-alias", ServerName: "hearth.example.com"}
	err = store.AddAlias(ctx, roomID, newAlias)
	require.NoError(t, err)

	// Verify alias was added.
	gotRoomID, gotChanID, err := store.GetByAlias(ctx, newAlias)
	require.NoError(t, err)
	assert.Equal(t, roomID, gotRoomID)
	assert.Equal(t, channelID, gotChanID)

	// Adding to nonexistent room should fail.
	err = store.AddAlias(ctx, RoomID{Localpart: "other", ServerName: "hearth.example.com"}, newAlias)
	assert.ErrorIs(t, err, ErrRoomNotFound)
}

func TestInMemoryRoomAliasStore_RemoveAlias(t *testing.T) {
	store := NewInMemoryRoomAliasStore()
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	alias := Alias{Localpart: "general", ServerName: "hearth.example.com"}

	err := store.CreateMapping(ctx, roomID, channelID, []Alias{alias})
	require.NoError(t, err)

	// Remove alias.
	err = store.RemoveAlias(ctx, alias)
	require.NoError(t, err)

	// Should no longer be found.
	_, _, err = store.GetByAlias(ctx, alias)
	assert.ErrorIs(t, err, ErrAliasNotFound)

	// Removing nonexistent alias should fail.
	err = store.RemoveAlias(ctx, Alias{Localpart: "nonexistent", ServerName: "hearth.example.com"})
	assert.ErrorIs(t, err, ErrAliasNotFound)
}

func TestInMemoryRoomAliasStore_RemoveMapping(t *testing.T) {
	store := NewInMemoryRoomAliasStore()
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	alias := Alias{Localpart: "general", ServerName: "hearth.example.com"}

	err := store.CreateMapping(ctx, roomID, channelID, []Alias{alias})
	require.NoError(t, err)

	// Remove mapping.
	err = store.RemoveMapping(ctx, roomID)
	require.NoError(t, err)

	// Should no longer be found by room ID.
	_, _, err = store.GetByRoomID(ctx, roomID)
	assert.ErrorIs(t, err, ErrRoomNotFound)

	// Should no longer be found by alias.
	_, _, err = store.GetByAlias(ctx, alias)
	assert.ErrorIs(t, err, ErrAliasNotFound)

	// Should no longer be found by channel ID.
	_, _, err = store.GetByChannelID(ctx, channelID)
	assert.ErrorIs(t, err, ErrRoomNotFound)

	// Removing again should fail.
	err = store.RemoveMapping(ctx, roomID)
	assert.ErrorIs(t, err, ErrRoomNotFound)
}

func TestInMemoryRoomAliasStore_ListAliases(t *testing.T) {
	store := NewInMemoryRoomAliasStore()
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	aliases := []Alias{
		{Localpart: "general", ServerName: "hearth.example.com"},
		{Localpart: "main", ServerName: "hearth.example.com"},
	}

	err := store.CreateMapping(ctx, roomID, channelID, aliases)
	require.NoError(t, err)

	got, err := store.ListAliases(ctx, roomID)
	require.NoError(t, err)
	assert.Len(t, got, 2)

	// Nonexistent room.
	_, err = store.ListAliases(ctx, RoomID{Localpart: "other", ServerName: "hearth.example.com"})
	assert.ErrorIs(t, err, ErrRoomNotFound)
}

func TestRoomPowerLevelsContent(t *testing.T) {
	creator := "@alice:hearth.example.com"
	pl := NewRoomPowerLevelsContent(creator)

	assert.Equal(t, int64(100), pl.Users[creator])
	assert.Equal(t, int64(50), pl.Ban)
	assert.Equal(t, int64(0), pl.Invite)
	assert.Equal(t, int64(100), pl.Events["m.room.power_levels"])
	assert.Equal(t, int64(50), pl.Events["m.room.name"])
}

func TestRoomDirectoryHandler_ResolveAlias(t *testing.T) {
	store := NewInMemoryRoomAliasStore()
	ctx := context.Background()

	roomID := RoomID{Localpart: "abc123", ServerName: "hearth.example.com"}
	channelID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	alias := Alias{Localpart: "general", ServerName: "hearth.example.com"}

	err := store.CreateMapping(ctx, roomID, channelID, []Alias{alias})
	require.NoError(t, err)

	cfg := &matrix.HomeserverConfig{
		ServerName: "hearth.example.com",
	}

	handler := NewRoomDirectoryHandler(store, cfg)

	// Test resolve via store.
	gotRoomID, gotChanID, err := handler.store.GetByAlias(ctx, alias)
	require.NoError(t, err)
	assert.Equal(t, roomID, gotRoomID)
	assert.Equal(t, channelID, gotChanID)
}

func TestRoomID_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		roomID RoomID
		want   bool
	}{
		{"valid", RoomID{Localpart: "abc", ServerName: "example.com"}, true},
		{"empty localpart", RoomID{Localpart: "", ServerName: "example.com"}, false},
		{"empty server", RoomID{Localpart: "abc", ServerName: ""}, false},
		{"both empty", RoomID{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.roomID.IsValid())
		})
	}
}

func TestAlias_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		alias Alias
		want  bool
	}{
		{"valid", Alias{Localpart: "general", ServerName: "example.com"}, true},
		{"empty localpart", Alias{Localpart: "", ServerName: "example.com"}, false},
		{"empty server", Alias{Localpart: "general", ServerName: ""}, false},
		{"both empty", Alias{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.alias.IsValid())
		})
	}
}
