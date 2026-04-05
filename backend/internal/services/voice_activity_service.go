package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// VoiceActivityRepository defines the interface for voice activity data access
type VoiceActivityRepository interface {
	Create(ctx context.Context, activity *models.VoiceActivity) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.VoiceActivity, error)
	GetActiveByChannel(ctx context.Context, channelID uuid.UUID) (*models.VoiceActivity, error)
	EndActivity(ctx context.Context, id uuid.UUID, status models.VoiceActivityStatus) error
	AddParticipant(ctx context.Context, activityID, userID uuid.UUID) error
	RemoveParticipant(ctx context.Context, activityID, userID uuid.UUID) error
	GetParticipants(ctx context.Context, activityID uuid.UUID) ([]models.VoiceActivityParticipantInfo, error)
	GetParticipantCount(ctx context.Context, activityID uuid.UUID) (int, error)
	SaveGameState(ctx context.Context, activityID uuid.UUID, state json.RawMessage) (int, error)
	GetGameState(ctx context.Context, activityID uuid.UUID) (*models.VoiceActivityGameState, error)
	GetActiveActivitiesByServer(ctx context.Context, serverID uuid.UUID) ([]models.VoiceActivity, error)
}

// VoiceActivityService handles voice activity business logic
type VoiceActivityService struct {
	repo     VoiceActivityRepository
	eventBus EventBus
}

// NewVoiceActivityService creates a new voice activity service
func NewVoiceActivityService(repo VoiceActivityRepository, eventBus EventBus) *VoiceActivityService {
	return &VoiceActivityService{
		repo:     repo,
		eventBus: eventBus,
	}
}

// StartActivity starts a new voice activity in a channel
func (s *VoiceActivityService) StartActivity(ctx context.Context, channelID, serverID, creatorID uuid.UUID, req *models.StartActivityRequest) (*models.VoiceActivityWithParticipants, error) {
	// Check for existing active activity in channel
	existing, err := s.repo.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing activity: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("an activity is already active in this channel")
	}

	// Validate activity type
	switch req.ActivityType {
	case models.ActivityPoker, models.ActivityChess, models.ActivityWatchTogether:
		// valid
	default:
		return nil, fmt.Errorf("invalid activity type: %s", req.ActivityType)
	}

	maxParticipants := req.MaxParticipants
	if maxParticipants <= 0 {
		switch req.ActivityType {
		case models.ActivityPoker:
			maxParticipants = 8
		case models.ActivityChess:
			maxParticipants = 2
		case models.ActivityWatchTogether:
			maxParticipants = 50
		}
	}

	activity := &models.VoiceActivity{
		ChannelID:       channelID,
		ServerID:        serverID,
		CreatorID:       creatorID,
		ActivityType:    req.ActivityType,
		Status:          models.ActivityStatusActive,
		MaxParticipants: maxParticipants,
		Metadata:        req.Metadata,
	}

	if err := s.repo.Create(ctx, activity); err != nil {
		return nil, fmt.Errorf("failed to create activity: %w", err)
	}

	// Creator auto-joins
	if err := s.repo.AddParticipant(ctx, activity.ID, creatorID); err != nil {
		return nil, fmt.Errorf("failed to add creator as participant: %w", err)
	}

	// Initialize game state
	initialState, err := s.initGameState(req.ActivityType, creatorID, req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize game state: %w", err)
	}
	if _, err := s.repo.SaveGameState(ctx, activity.ID, initialState); err != nil {
		return nil, fmt.Errorf("failed to save initial game state: %w", err)
	}

	participants, _ := s.repo.GetParticipants(ctx, activity.ID)
	return &models.VoiceActivityWithParticipants{
		VoiceActivity: *activity,
		Participants:  participants,
	}, nil
}

// JoinActivity adds a user to an activity
func (s *VoiceActivityService) JoinActivity(ctx context.Context, activityID, userID uuid.UUID) (*models.VoiceActivityWithParticipants, error) {
	activity, err := s.repo.GetByID(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}
	if activity == nil {
		return nil, fmt.Errorf("activity not found")
	}
	if activity.Status != models.ActivityStatusActive {
		return nil, fmt.Errorf("activity is not active")
	}

	count, err := s.repo.GetParticipantCount(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participant count: %w", err)
	}
	if count >= activity.MaxParticipants {
		return nil, fmt.Errorf("activity is full")
	}

	if err := s.repo.AddParticipant(ctx, activityID, userID); err != nil {
		return nil, fmt.Errorf("failed to join activity: %w", err)
	}

	participants, _ := s.repo.GetParticipants(ctx, activityID)
	return &models.VoiceActivityWithParticipants{
		VoiceActivity: *activity,
		Participants:  participants,
	}, nil
}

// LeaveActivity removes a user from an activity
func (s *VoiceActivityService) LeaveActivity(ctx context.Context, activityID, userID uuid.UUID) error {
	activity, err := s.repo.GetByID(ctx, activityID)
	if err != nil {
		return fmt.Errorf("failed to get activity: %w", err)
	}
	if activity == nil {
		return fmt.Errorf("activity not found")
	}

	if err := s.repo.RemoveParticipant(ctx, activityID, userID); err != nil {
		return fmt.Errorf("failed to leave activity: %w", err)
	}

	// If creator left, or no participants remain, end the activity
	count, _ := s.repo.GetParticipantCount(ctx, activityID)
	if count == 0 || userID == activity.CreatorID {
		_ = s.repo.EndActivity(ctx, activityID, models.ActivityStatusFinished)
	}

	return nil
}

// EndActivity ends an activity
func (s *VoiceActivityService) EndActivity(ctx context.Context, activityID, userID uuid.UUID) error {
	activity, err := s.repo.GetByID(ctx, activityID)
	if err != nil {
		return fmt.Errorf("failed to get activity: %w", err)
	}
	if activity == nil {
		return fmt.Errorf("activity not found")
	}
	if activity.CreatorID != userID {
		return fmt.Errorf("only the creator can end the activity")
	}

	return s.repo.EndActivity(ctx, activityID, models.ActivityStatusFinished)
}

// GetActivity retrieves an activity with its participants
func (s *VoiceActivityService) GetActivity(ctx context.Context, activityID uuid.UUID) (*models.VoiceActivityWithParticipants, error) {
	activity, err := s.repo.GetByID(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}
	if activity == nil {
		return nil, fmt.Errorf("activity not found")
	}

	participants, err := s.repo.GetParticipants(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	return &models.VoiceActivityWithParticipants{
		VoiceActivity: *activity,
		Participants:  participants,
	}, nil
}

// GetChannelActivity retrieves the active activity for a channel
func (s *VoiceActivityService) GetChannelActivity(ctx context.Context, channelID uuid.UUID) (*models.VoiceActivityWithParticipants, error) {
	activity, err := s.repo.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel activity: %w", err)
	}
	if activity == nil {
		return nil, nil
	}

	participants, _ := s.repo.GetParticipants(ctx, activity.ID)
	return &models.VoiceActivityWithParticipants{
		VoiceActivity: *activity,
		Participants:  participants,
	}, nil
}

// GetGameState retrieves the game state for an activity
func (s *VoiceActivityService) GetGameState(ctx context.Context, activityID uuid.UUID) (*models.VoiceActivityGameState, error) {
	return s.repo.GetGameState(ctx, activityID)
}

// ProcessGameMove processes a game move and updates the game state
func (s *VoiceActivityService) ProcessGameMove(ctx context.Context, activityID, userID uuid.UUID, move *models.GameMoveRequest) (*models.VoiceActivityGameState, error) {
	activity, err := s.repo.GetByID(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}
	if activity == nil {
		return nil, fmt.Errorf("activity not found")
	}
	if activity.Status != models.ActivityStatusActive {
		return nil, fmt.Errorf("activity is not active")
	}

	gs, err := s.repo.GetGameState(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game state: %w", err)
	}
	if gs == nil {
		return nil, fmt.Errorf("game state not found")
	}

	// Process the move based on activity type
	var newState json.RawMessage
	switch activity.ActivityType {
	case models.ActivityPoker:
		newState, err = s.processPokerMove(gs.State, userID, move)
	case models.ActivityChess:
		newState, err = s.processChessMove(gs.State, userID, move)
	case models.ActivityWatchTogether:
		newState, err = s.processWatchTogetherAction(gs.State, userID, move)
	default:
		return nil, fmt.Errorf("unsupported activity type")
	}
	if err != nil {
		return nil, err
	}

	version, err := s.repo.SaveGameState(ctx, activityID, newState)
	if err != nil {
		return nil, fmt.Errorf("failed to save game state: %w", err)
	}

	return &models.VoiceActivityGameState{
		ActivityID: activityID,
		State:      newState,
		Version:    version,
	}, nil
}

// GetActiveActivitiesByServer returns all active activities in a server
func (s *VoiceActivityService) GetActiveActivitiesByServer(ctx context.Context, serverID uuid.UUID) ([]models.VoiceActivity, error) {
	return s.repo.GetActiveActivitiesByServer(ctx, serverID)
}

// initGameState creates the initial game state for an activity type
func (s *VoiceActivityService) initGameState(activityType models.VoiceActivityType, creatorID uuid.UUID, metadata json.RawMessage) (json.RawMessage, error) {
	switch activityType {
	case models.ActivityPoker:
		state := models.PokerState{
			Phase:          "waiting",
			Pot:            0,
			CommunityCards: []string{},
			SmallBlind:     10,
			BigBlind:       20,
			Players: []models.PokerPlayer{{
				UserID: creatorID,
				Hand:   []string{},
				Chips:  1000,
			}},
			Round: 0,
		}
		return json.Marshal(state)
	case models.ActivityChess:
		state := models.ChessState{
			Board:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			WhitePlayer: &creatorID,
			CurrentTurn: "white",
			MoveHistory: []string{},
			Status:      "waiting",
		}
		return json.Marshal(state)
	case models.ActivityWatchTogether:
		videoURL := ""
		if metadata != nil {
			var meta map[string]interface{}
			if err := json.Unmarshal(metadata, &meta); err == nil {
				if url, ok := meta["video_url"].(string); ok {
					videoURL = url
				}
			}
		}
		state := models.WatchTogetherState{
			VideoURL:     videoURL,
			IsPlaying:    false,
			CurrentTime:  0,
			PlaybackRate: 1.0,
			UpdatedBy:    &creatorID,
			Queue:        []models.WatchTogetherQueueItem{},
		}
		return json.Marshal(state)
	}
	return json.Marshal(map[string]interface{}{})
}

// processPokerMove processes a poker game action
func (s *VoiceActivityService) processPokerMove(currentState json.RawMessage, userID uuid.UUID, move *models.GameMoveRequest) (json.RawMessage, error) {
	var state models.PokerState
	if err := json.Unmarshal(currentState, &state); err != nil {
		return nil, fmt.Errorf("failed to parse poker state: %w", err)
	}

	switch move.Action {
	case "bet":
		var betData struct {
			Amount int `json:"amount"`
		}
		if err := json.Unmarshal(move.Data, &betData); err != nil {
			return nil, fmt.Errorf("invalid bet data: %w", err)
		}
		for i, p := range state.Players {
			if p.UserID == userID && !p.Folded {
				if betData.Amount > p.Chips {
					return nil, fmt.Errorf("insufficient chips")
				}
				state.Players[i].Chips -= betData.Amount
				state.Players[i].Bet += betData.Amount
				state.Pot += betData.Amount
				break
			}
		}
	case "fold":
		for i, p := range state.Players {
			if p.UserID == userID {
				state.Players[i].Folded = true
				break
			}
		}
	case "check":
		// No-op, advance turn
	case "call":
		// Find the current highest bet
		maxBet := 0
		for _, p := range state.Players {
			if p.Bet > maxBet {
				maxBet = p.Bet
			}
		}
		for i, p := range state.Players {
			if p.UserID == userID && !p.Folded {
				diff := maxBet - p.Bet
				if diff > p.Chips {
					diff = p.Chips
					state.Players[i].AllIn = true
				}
				state.Players[i].Chips -= diff
				state.Players[i].Bet += diff
				state.Pot += diff
				break
			}
		}
	case "start":
		if state.Phase == "waiting" {
			state.Phase = "preflop"
			state.Round++
		}
	default:
		return nil, fmt.Errorf("invalid poker action: %s", move.Action)
	}

	return json.Marshal(state)
}

// processChessMove processes a chess move
func (s *VoiceActivityService) processChessMove(currentState json.RawMessage, userID uuid.UUID, move *models.GameMoveRequest) (json.RawMessage, error) {
	var state models.ChessState
	if err := json.Unmarshal(currentState, &state); err != nil {
		return nil, fmt.Errorf("failed to parse chess state: %w", err)
	}

	switch move.Action {
	case "move":
		// Validate it's the player's turn
		if state.CurrentTurn == "white" && (state.WhitePlayer == nil || *state.WhitePlayer != userID) {
			return nil, fmt.Errorf("not your turn")
		}
		if state.CurrentTurn == "black" && (state.BlackPlayer == nil || *state.BlackPlayer != userID) {
			return nil, fmt.Errorf("not your turn")
		}

		var moveData struct {
			From string `json:"from"`
			To   string `json:"to"`
			FEN  string `json:"fen"` // Client sends updated FEN after move validation
		}
		if err := json.Unmarshal(move.Data, &moveData); err != nil {
			return nil, fmt.Errorf("invalid move data: %w", err)
		}

		notation := moveData.From + moveData.To
		state.MoveHistory = append(state.MoveHistory, notation)
		if moveData.FEN != "" {
			state.Board = moveData.FEN
		}

		// Toggle turn
		if state.CurrentTurn == "white" {
			state.CurrentTurn = "black"
		} else {
			state.CurrentTurn = "white"
		}
	case "join":
		// Second player joins as black
		if state.BlackPlayer == nil && (state.WhitePlayer == nil || *state.WhitePlayer != userID) {
			state.BlackPlayer = &userID
			if state.Status == "waiting" {
				state.Status = "playing"
			}
		}
	case "resign":
		state.Status = "resigned"
		if state.WhitePlayer != nil && *state.WhitePlayer == userID {
			state.Winner = state.BlackPlayer
		} else {
			state.Winner = state.WhitePlayer
		}
	case "checkmate":
		state.Status = "checkmate"
		state.Winner = &userID
	case "draw":
		state.Status = "draw"
	default:
		return nil, fmt.Errorf("invalid chess action: %s", move.Action)
	}

	return json.Marshal(state)
}

// processWatchTogetherAction processes a Watch Together action
func (s *VoiceActivityService) processWatchTogetherAction(currentState json.RawMessage, userID uuid.UUID, move *models.GameMoveRequest) (json.RawMessage, error) {
	var state models.WatchTogetherState
	if err := json.Unmarshal(currentState, &state); err != nil {
		return nil, fmt.Errorf("failed to parse watch together state: %w", err)
	}

	switch move.Action {
	case "play":
		state.IsPlaying = true
		state.UpdatedBy = &userID
	case "pause":
		state.IsPlaying = false
		state.UpdatedBy = &userID
	case "seek":
		var seekData struct {
			Time float64 `json:"time"`
		}
		if err := json.Unmarshal(move.Data, &seekData); err != nil {
			return nil, fmt.Errorf("invalid seek data: %w", err)
		}
		state.CurrentTime = seekData.Time
		state.UpdatedBy = &userID
	case "set_video":
		var videoData struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal(move.Data, &videoData); err != nil {
			return nil, fmt.Errorf("invalid video data: %w", err)
		}
		state.VideoURL = videoData.URL
		state.VideoTitle = videoData.Title
		state.CurrentTime = 0
		state.IsPlaying = false
		state.UpdatedBy = &userID
	case "queue_add":
		var queueData struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal(move.Data, &queueData); err != nil {
			return nil, fmt.Errorf("invalid queue data: %w", err)
		}
		state.Queue = append(state.Queue, models.WatchTogetherQueueItem{
			URL:     queueData.URL,
			Title:   queueData.Title,
			AddedBy: userID,
		})
	case "set_rate":
		var rateData struct {
			Rate float64 `json:"rate"`
		}
		if err := json.Unmarshal(move.Data, &rateData); err != nil {
			return nil, fmt.Errorf("invalid rate data: %w", err)
		}
		state.PlaybackRate = rateData.Rate
		state.UpdatedBy = &userID
	default:
		return nil, fmt.Errorf("invalid watch together action: %s", move.Action)
	}

	return json.Marshal(state)
}
