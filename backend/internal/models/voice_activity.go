package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// VoiceActivityType represents the type of voice activity
type VoiceActivityType string

const (
	ActivityPoker        VoiceActivityType = "poker"
	ActivityChess        VoiceActivityType = "chess"
	ActivityWatchTogether VoiceActivityType = "watch_together"
)

// VoiceActivityStatus represents the status of a voice activity
type VoiceActivityStatus string

const (
	ActivityStatusActive    VoiceActivityStatus = "active"
	ActivityStatusFinished  VoiceActivityStatus = "finished"
	ActivityStatusCancelled VoiceActivityStatus = "cancelled"
)

// VoiceActivity represents a voice channel activity session
type VoiceActivity struct {
	ID              uuid.UUID           `json:"id" db:"id"`
	ChannelID       uuid.UUID           `json:"channel_id" db:"channel_id"`
	ServerID        uuid.UUID           `json:"server_id" db:"server_id"`
	CreatorID       uuid.UUID           `json:"creator_id" db:"creator_id"`
	ActivityType    VoiceActivityType   `json:"activity_type" db:"activity_type"`
	Status          VoiceActivityStatus `json:"status" db:"status"`
	MaxParticipants int                 `json:"max_participants" db:"max_participants"`
	Metadata        json.RawMessage     `json:"metadata" db:"metadata"`
	CreatedAt       time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at" db:"updated_at"`
	EndedAt         *time.Time          `json:"ended_at,omitempty" db:"ended_at"`
}

// VoiceActivityParticipant represents a participant in a voice activity
type VoiceActivityParticipant struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	ActivityID uuid.UUID  `json:"activity_id" db:"activity_id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	JoinedAt   time.Time  `json:"joined_at" db:"joined_at"`
	LeftAt     *time.Time `json:"left_at,omitempty" db:"left_at"`
}

// VoiceActivityGameState represents the game state for an activity
type VoiceActivityGameState struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	ActivityID uuid.UUID       `json:"activity_id" db:"activity_id"`
	State      json.RawMessage `json:"state" db:"state"`
	Version    int             `json:"version" db:"version"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}

// StartActivityRequest is the request to start a voice activity
type StartActivityRequest struct {
	ActivityType    VoiceActivityType `json:"activity_type" validate:"required"`
	MaxParticipants int              `json:"max_participants,omitempty"`
	Metadata        json.RawMessage  `json:"metadata,omitempty"`
}

// GameMoveRequest is the request to make a game move
type GameMoveRequest struct {
	Action string          `json:"action" validate:"required"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// VoiceActivityWithParticipants is a view combining activity with its participants
type VoiceActivityWithParticipants struct {
	VoiceActivity
	Participants []VoiceActivityParticipantInfo `json:"participants"`
}

// VoiceActivityParticipantInfo is participant info with user details
type VoiceActivityParticipantInfo struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName *string   `json:"display_name,omitempty"`
	Avatar      *string   `json:"avatar,omitempty"`
	JoinedAt    time.Time `json:"joined_at"`
}

// PokerState represents the state of a poker game
type PokerState struct {
	Phase         string            `json:"phase"` // waiting, dealing, preflop, flop, turn, river, showdown
	Pot           int               `json:"pot"`
	CommunityCards []string         `json:"community_cards"`
	CurrentTurn   *uuid.UUID        `json:"current_turn,omitempty"`
	DealerIndex   int               `json:"dealer_index"`
	SmallBlind    int               `json:"small_blind"`
	BigBlind      int               `json:"big_blind"`
	Players       []PokerPlayer     `json:"players"`
	Round         int               `json:"round"`
}

// PokerPlayer represents a player in a poker game
type PokerPlayer struct {
	UserID   uuid.UUID `json:"user_id"`
	Hand     []string  `json:"hand,omitempty"` // only visible to the player
	Chips    int       `json:"chips"`
	Bet      int       `json:"bet"`
	Folded   bool      `json:"folded"`
	AllIn    bool      `json:"all_in"`
	IsDealer bool      `json:"is_dealer"`
}

// ChessState represents the state of a chess game
type ChessState struct {
	Board       string     `json:"board"` // FEN notation
	WhitePlayer *uuid.UUID `json:"white_player,omitempty"`
	BlackPlayer *uuid.UUID `json:"black_player,omitempty"`
	CurrentTurn string     `json:"current_turn"` // "white" or "black"
	MoveHistory []string   `json:"move_history"` // algebraic notation
	Status      string     `json:"status"`       // playing, checkmate, stalemate, draw, resigned
	Winner      *uuid.UUID `json:"winner,omitempty"`
}

// WatchTogetherState represents the state of a Watch Together session
type WatchTogetherState struct {
	VideoURL    string     `json:"video_url"`
	VideoTitle  string     `json:"video_title,omitempty"`
	IsPlaying   bool       `json:"is_playing"`
	CurrentTime float64    `json:"current_time"` // seconds
	PlaybackRate float64   `json:"playback_rate"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
	Queue       []WatchTogetherQueueItem `json:"queue,omitempty"`
}

// WatchTogetherQueueItem represents a queued video
type WatchTogetherQueueItem struct {
	URL      string    `json:"url"`
	Title    string    `json:"title,omitempty"`
	AddedBy  uuid.UUID `json:"added_by"`
	AddedAt  time.Time `json:"added_at"`
}
