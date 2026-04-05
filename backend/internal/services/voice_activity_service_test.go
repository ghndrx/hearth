package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"hearth/internal/models"
)

func TestVoiceActivityService_InitGameState(t *testing.T) {
	svc := &VoiceActivityService{}
	creatorID := uuid.New()

	t.Run("poker initial state", func(t *testing.T) {
		state, err := svc.initGameState(models.ActivityPoker, creatorID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var poker models.PokerState
		if err := json.Unmarshal(state, &poker); err != nil {
			t.Fatalf("failed to unmarshal poker state: %v", err)
		}

		if poker.Phase != "waiting" {
			t.Errorf("expected phase 'waiting', got %q", poker.Phase)
		}
		if poker.Pot != 0 {
			t.Errorf("expected pot 0, got %d", poker.Pot)
		}
		if poker.SmallBlind != 10 {
			t.Errorf("expected small blind 10, got %d", poker.SmallBlind)
		}
		if poker.BigBlind != 20 {
			t.Errorf("expected big blind 20, got %d", poker.BigBlind)
		}
		if len(poker.Players) != 1 {
			t.Fatalf("expected 1 player, got %d", len(poker.Players))
		}
		if poker.Players[0].UserID != creatorID {
			t.Errorf("expected creator as first player")
		}
		if poker.Players[0].Chips != 1000 {
			t.Errorf("expected 1000 chips, got %d", poker.Players[0].Chips)
		}
	})

	t.Run("chess initial state", func(t *testing.T) {
		state, err := svc.initGameState(models.ActivityChess, creatorID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var chess models.ChessState
		if err := json.Unmarshal(state, &chess); err != nil {
			t.Fatalf("failed to unmarshal chess state: %v", err)
		}

		if chess.Status != "waiting" {
			t.Errorf("expected status 'waiting', got %q", chess.Status)
		}
		if chess.CurrentTurn != "white" {
			t.Errorf("expected current turn 'white', got %q", chess.CurrentTurn)
		}
		if chess.WhitePlayer == nil || *chess.WhitePlayer != creatorID {
			t.Errorf("expected creator as white player")
		}
		if chess.BlackPlayer != nil {
			t.Errorf("expected no black player yet")
		}
		if chess.Board != "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" {
			t.Errorf("unexpected initial FEN: %s", chess.Board)
		}
	})

	t.Run("watch_together initial state", func(t *testing.T) {
		metadata, _ := json.Marshal(map[string]interface{}{
			"video_url": "https://www.youtube.com/watch?v=test123",
		})
		state, err := svc.initGameState(models.ActivityWatchTogether, creatorID, metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var wt models.WatchTogetherState
		if err := json.Unmarshal(state, &wt); err != nil {
			t.Fatalf("failed to unmarshal watch together state: %v", err)
		}

		if wt.VideoURL != "https://www.youtube.com/watch?v=test123" {
			t.Errorf("expected video URL from metadata, got %q", wt.VideoURL)
		}
		if wt.IsPlaying {
			t.Errorf("expected is_playing false initially")
		}
		if wt.PlaybackRate != 1.0 {
			t.Errorf("expected playback rate 1.0, got %f", wt.PlaybackRate)
		}
	})
}

func TestVoiceActivityService_ProcessPokerMove(t *testing.T) {
	svc := &VoiceActivityService{}
	player1 := uuid.New()
	player2 := uuid.New()

	initialState, _ := json.Marshal(models.PokerState{
		Phase:      "preflop",
		Pot:        30,
		SmallBlind: 10,
		BigBlind:   20,
		Players: []models.PokerPlayer{
			{UserID: player1, Chips: 980, Bet: 20, Hand: []string{"AH", "KS"}},
			{UserID: player2, Chips: 990, Bet: 10, Hand: []string{"QH", "JD"}},
		},
	})

	t.Run("bet action", func(t *testing.T) {
		moveData, _ := json.Marshal(map[string]int{"amount": 50})
		move := &models.GameMoveRequest{Action: "bet", Data: moveData}
		result, err := svc.processPokerMove(initialState, player1, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.PokerState
		json.Unmarshal(result, &state)
		if state.Players[0].Chips != 930 {
			t.Errorf("expected 930 chips after bet, got %d", state.Players[0].Chips)
		}
		if state.Pot != 80 {
			t.Errorf("expected pot 80 after bet, got %d", state.Pot)
		}
	})

	t.Run("fold action", func(t *testing.T) {
		move := &models.GameMoveRequest{Action: "fold"}
		result, err := svc.processPokerMove(initialState, player2, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.PokerState
		json.Unmarshal(result, &state)
		if !state.Players[1].Folded {
			t.Errorf("expected player2 to be folded")
		}
	})

	t.Run("call action", func(t *testing.T) {
		move := &models.GameMoveRequest{Action: "call"}
		result, err := svc.processPokerMove(initialState, player2, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.PokerState
		json.Unmarshal(result, &state)
		// player2 had bet 10, max bet is 20, so should call 10 more
		if state.Players[1].Bet != 20 {
			t.Errorf("expected bet 20 after call, got %d", state.Players[1].Bet)
		}
		if state.Players[1].Chips != 980 {
			t.Errorf("expected 980 chips after call, got %d", state.Players[1].Chips)
		}
	})

	t.Run("invalid action", func(t *testing.T) {
		move := &models.GameMoveRequest{Action: "bluff"}
		_, err := svc.processPokerMove(initialState, player1, move)
		if err == nil {
			t.Errorf("expected error for invalid action")
		}
	})
}

func TestVoiceActivityService_ProcessChessMove(t *testing.T) {
	svc := &VoiceActivityService{}
	white := uuid.New()
	black := uuid.New()

	initialState, _ := json.Marshal(models.ChessState{
		Board:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		WhitePlayer: &white,
		BlackPlayer: &black,
		CurrentTurn: "white",
		MoveHistory: []string{},
		Status:      "playing",
	})

	t.Run("valid move by white", func(t *testing.T) {
		moveData, _ := json.Marshal(map[string]string{"from": "e2", "to": "e4", "fen": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"})
		move := &models.GameMoveRequest{Action: "move", Data: moveData}
		result, err := svc.processChessMove(initialState, white, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.ChessState
		json.Unmarshal(result, &state)
		if state.CurrentTurn != "black" {
			t.Errorf("expected turn to switch to black, got %q", state.CurrentTurn)
		}
		if len(state.MoveHistory) != 1 {
			t.Errorf("expected 1 move in history, got %d", len(state.MoveHistory))
		}
	})

	t.Run("wrong player turn", func(t *testing.T) {
		moveData, _ := json.Marshal(map[string]string{"from": "e7", "to": "e5", "fen": ""})
		move := &models.GameMoveRequest{Action: "move", Data: moveData}
		_, err := svc.processChessMove(initialState, black, move)
		if err == nil {
			t.Errorf("expected error for wrong turn")
		}
	})

	t.Run("join as black", func(t *testing.T) {
		noBlackState, _ := json.Marshal(models.ChessState{
			Board:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			WhitePlayer: &white,
			CurrentTurn: "white",
			MoveHistory: []string{},
			Status:      "waiting",
		})
		newPlayer := uuid.New()
		move := &models.GameMoveRequest{Action: "join"}
		result, err := svc.processChessMove(noBlackState, newPlayer, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.ChessState
		json.Unmarshal(result, &state)
		if state.BlackPlayer == nil || *state.BlackPlayer != newPlayer {
			t.Errorf("expected new player as black")
		}
		if state.Status != "playing" {
			t.Errorf("expected status 'playing' after both players joined, got %q", state.Status)
		}
	})

	t.Run("resign", func(t *testing.T) {
		move := &models.GameMoveRequest{Action: "resign"}
		result, err := svc.processChessMove(initialState, white, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.ChessState
		json.Unmarshal(result, &state)
		if state.Status != "resigned" {
			t.Errorf("expected status 'resigned', got %q", state.Status)
		}
		if state.Winner == nil || *state.Winner != black {
			t.Errorf("expected black to win after white resigns")
		}
	})
}

func TestVoiceActivityService_ProcessWatchTogetherAction(t *testing.T) {
	svc := &VoiceActivityService{}
	user1 := uuid.New()
	_ = context.Background()

	initialState, _ := json.Marshal(models.WatchTogetherState{
		VideoURL:     "https://www.youtube.com/watch?v=abc",
		IsPlaying:    false,
		CurrentTime:  0,
		PlaybackRate: 1.0,
		Queue:        []models.WatchTogetherQueueItem{},
	})

	t.Run("play", func(t *testing.T) {
		move := &models.GameMoveRequest{Action: "play"}
		result, err := svc.processWatchTogetherAction(initialState, user1, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.WatchTogetherState
		json.Unmarshal(result, &state)
		if !state.IsPlaying {
			t.Errorf("expected is_playing true")
		}
		if state.UpdatedBy == nil || *state.UpdatedBy != user1 {
			t.Errorf("expected updated_by to be user1")
		}
	})

	t.Run("pause", func(t *testing.T) {
		playingState, _ := json.Marshal(models.WatchTogetherState{
			VideoURL:  "https://www.youtube.com/watch?v=abc",
			IsPlaying: true,
		})
		move := &models.GameMoveRequest{Action: "pause"}
		result, err := svc.processWatchTogetherAction(playingState, user1, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.WatchTogetherState
		json.Unmarshal(result, &state)
		if state.IsPlaying {
			t.Errorf("expected is_playing false after pause")
		}
	})

	t.Run("seek", func(t *testing.T) {
		seekData, _ := json.Marshal(map[string]float64{"time": 120.5})
		move := &models.GameMoveRequest{Action: "seek", Data: seekData}
		result, err := svc.processWatchTogetherAction(initialState, user1, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.WatchTogetherState
		json.Unmarshal(result, &state)
		if state.CurrentTime != 120.5 {
			t.Errorf("expected current_time 120.5, got %f", state.CurrentTime)
		}
	})

	t.Run("set_video", func(t *testing.T) {
		videoData, _ := json.Marshal(map[string]string{
			"url":   "https://www.youtube.com/watch?v=new",
			"title": "New Video",
		})
		move := &models.GameMoveRequest{Action: "set_video", Data: videoData}
		result, err := svc.processWatchTogetherAction(initialState, user1, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.WatchTogetherState
		json.Unmarshal(result, &state)
		if state.VideoURL != "https://www.youtube.com/watch?v=new" {
			t.Errorf("unexpected video URL: %s", state.VideoURL)
		}
		if state.VideoTitle != "New Video" {
			t.Errorf("unexpected title: %s", state.VideoTitle)
		}
		if state.CurrentTime != 0 {
			t.Errorf("expected current_time reset to 0")
		}
	})

	t.Run("queue_add", func(t *testing.T) {
		queueData, _ := json.Marshal(map[string]string{
			"url":   "https://www.youtube.com/watch?v=queued",
			"title": "Queued Video",
		})
		move := &models.GameMoveRequest{Action: "queue_add", Data: queueData}
		result, err := svc.processWatchTogetherAction(initialState, user1, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.WatchTogetherState
		json.Unmarshal(result, &state)
		if len(state.Queue) != 1 {
			t.Fatalf("expected 1 item in queue, got %d", len(state.Queue))
		}
		if state.Queue[0].URL != "https://www.youtube.com/watch?v=queued" {
			t.Errorf("unexpected queued URL: %s", state.Queue[0].URL)
		}
	})

	t.Run("set_rate", func(t *testing.T) {
		rateData, _ := json.Marshal(map[string]float64{"rate": 1.5})
		move := &models.GameMoveRequest{Action: "set_rate", Data: rateData}
		result, err := svc.processWatchTogetherAction(initialState, user1, move)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var state models.WatchTogetherState
		json.Unmarshal(result, &state)
		if state.PlaybackRate != 1.5 {
			t.Errorf("expected rate 1.5, got %f", state.PlaybackRate)
		}
	})
}
