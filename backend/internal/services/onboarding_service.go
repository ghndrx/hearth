package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrWelcomeScreenNotFound  = errors.New("welcome screen not found")
	ErrScreeningNotFound      = errors.New("screening not found")
	ErrNotServerModerator     = errors.New("not a server moderator")
	ErrScreeningNotPending    = errors.New("screening is not pending")
	ErrScreeningAlreadyExists = errors.New("screening already submitted")
)

// WelcomeRepository defines the interface for welcome screen data access
type WelcomeRepository interface {
	GetWelcomeScreen(ctx context.Context, serverID uuid.UUID) (*WelcomeScreenRepo, error)
	UpsertWelcomeScreen(ctx context.Context, ws *WelcomeScreenRepo) error
	GetRules(ctx context.Context, serverID uuid.UUID) ([]RuleRepo, error)
	UpsertRules(ctx context.Context, serverID uuid.UUID, rules []RuleRepo) error
	GetQuestions(ctx context.Context, serverID uuid.UUID) ([]QuestionRepo, error)
	UpsertQuestions(ctx context.Context, serverID uuid.UUID, questions []QuestionRepo) error
	GetMemberScreening(ctx context.Context, userID, serverID uuid.UUID) (*ScreeningRepo, error)
	UpsertMemberScreening(ctx context.Context, screening *ScreeningRepo) error
	GetPendingScreenings(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]ScreeningRepo, error)
	UpdateScreeningStatus(ctx context.Context, userID, serverID uuid.UUID, status string) error
}

// Repository-level types (for the repository interface)
type WelcomeScreenRepo struct {
	ID              uuid.UUID
	ServerID        uuid.UUID
	Enabled         bool
	Title           string
	Description     string
	WelcomeChannels []string
	UpdatedAt       string
	CreatedAt       string
}

type RuleRepo struct {
	ID          uuid.UUID
	ServerID    uuid.UUID
	RuleOrder   int
	Title       string
	Description string
	CreatedAt   string
}

type QuestionRepo struct {
	ID            uuid.UUID
	ServerID      uuid.UUID
	QuestionOrder int
	Question      string
	Required      bool
	QuestionType  string
	Options       []string
	CreatedAt     string
}

type ScreeningRepo struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ServerID  uuid.UUID
	Answers   json.RawMessage
	RulesRead bool
	Status    string
	CreatedAt string
	UpdatedAt string
}

// WelcomeService handles welcome screen business logic
type WelcomeService struct {
	repo        WelcomeRepository
	serverRepo  ServerRepository
	permService *PermissionService
}

// NewWelcomeService creates a new welcome service
func NewWelcomeService(repo WelcomeRepository, serverRepo ServerRepository, permService *PermissionService) *WelcomeService {
	return &WelcomeService{
		repo:        repo,
		serverRepo:  serverRepo,
		permService: permService,
	}
}

// GetWelcomeScreen retrieves the welcome screen config for a server
func (s *WelcomeService) GetWelcomeScreen(ctx context.Context, serverID uuid.UUID) (*models.WelcomeScreenConfig, error) {
	ws, err := s.repo.GetWelcomeScreen(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		// Return empty config if no welcome screen exists
		return &models.WelcomeScreenConfig{
			WelcomeScreen: models.WelcomeScreen{
				ID:          uuid.Nil,
				ServerID:    serverID,
				Enabled:     false,
				Title:       "Welcome to the server",
				Description: "",
			},
			Rules:     []models.Rule{},
			Questions: []models.ScreeningQuestion{},
		}, nil
	}

	// Get rules
	rulesRepo, err := s.repo.GetRules(ctx, serverID)
	if err != nil {
		return nil, err
	}
	rules := make([]models.Rule, len(rulesRepo))
	for i, r := range rulesRepo {
		rules[i] = models.Rule{
			ID:          r.ID.String(),
			Order:       r.RuleOrder,
			Title:       r.Title,
			Description: r.Description,
		}
	}

	// Get questions
	questionsRepo, err := s.repo.GetQuestions(ctx, serverID)
	if err != nil {
		return nil, err
	}
	questions := make([]models.ScreeningQuestion, len(questionsRepo))
	for i, q := range questionsRepo {
		questions[i] = models.ScreeningQuestion{
			ID:       q.ID.String(),
			Order:    q.QuestionOrder,
			Question: q.Question,
			Required: q.Required,
			Type:     q.QuestionType,
			Options:  q.Options,
		}
	}

	return &models.WelcomeScreenConfig{
		WelcomeScreen: models.WelcomeScreen{
			ID:              ws.ID,
			ServerID:        ws.ServerID,
			Enabled:         ws.Enabled,
			Title:           ws.Title,
			Description:     ws.Description,
			WelcomeChannels: ws.WelcomeChannels,
			UpdatedAt:       parseTime(ws.UpdatedAt),
			CreatedAt:       parseTime(ws.CreatedAt),
		},
		Rules:     rules,
		Questions: questions,
	}, nil
}

func parseTime(s string) (t time.Time) {
	t, _ = time.Parse(time.RFC3339, s)
	return
}

// UpdateWelcomeScreen updates the welcome screen configuration
func (s *WelcomeService) UpdateWelcomeScreen(ctx context.Context, serverID, requesterID uuid.UUID, req *models.UpdateWelcomeScreenRequest) (*models.WelcomeScreenConfig, error) {
	// Check if requester is server owner
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrServerNotFound
	}
	if server.OwnerID != requesterID {
		// Check for manage server permission
		hasPerm, err := s.permService.HasPermission(ctx, serverID, requesterID, models.PermManageServer)
		if err != nil {
			return nil, err
		}
		if !hasPerm {
			return nil, ErrNotServerOwner
		}
	}

	// Get or create welcome screen
	ws, err := s.repo.GetWelcomeScreen(ctx, serverID)
	if err != nil {
		return nil, err
	}

	if ws == nil {
		ws = &WelcomeScreenRepo{
			ID:       uuid.New(),
			ServerID: serverID,
		}
	}

	// Apply updates
	if req.Enabled != nil {
		ws.Enabled = *req.Enabled
	}
	if req.Title != nil {
		ws.Title = *req.Title
	}
	if req.Description != nil {
		ws.Description = *req.Description
	}
	if req.WelcomeChannels != nil {
		ws.WelcomeChannels = req.WelcomeChannels
	}

	// Save welcome screen
	if err := s.repo.UpsertWelcomeScreen(ctx, ws); err != nil {
		return nil, err
	}

	// Update rules if provided
	if req.Rules != nil {
		rulesRepo := make([]RuleRepo, len(req.Rules))
		for i, r := range req.Rules {
			id, _ := uuid.Parse(r.ID)
			if id == uuid.Nil {
				id = uuid.New()
			}
			rulesRepo[i] = RuleRepo{
				ID:          id,
				ServerID:    serverID,
				RuleOrder:   r.Order,
				Title:       r.Title,
				Description: r.Description,
			}
		}
		if err := s.repo.UpsertRules(ctx, serverID, rulesRepo); err != nil {
			return nil, err
		}
	}

	// Update questions if provided
	if req.Questions != nil {
		questionsRepo := make([]QuestionRepo, len(req.Questions))
		for i, q := range req.Questions {
			id, _ := uuid.Parse(q.ID)
			if id == uuid.Nil {
				id = uuid.New()
			}
			questionsRepo[i] = QuestionRepo{
				ID:            id,
				ServerID:      serverID,
				QuestionOrder: q.Order,
				Question:      q.Question,
				Required:      q.Required,
				QuestionType:  q.Type,
				Options:       q.Options,
			}
		}
		if err := s.repo.UpsertQuestions(ctx, serverID, questionsRepo); err != nil {
			return nil, err
		}
	}

	// Return updated config
	return s.GetWelcomeScreen(ctx, serverID)
}

// SubmitScreening submits a member's screening answers
func (s *WelcomeService) SubmitScreening(ctx context.Context, userID, serverID uuid.UUID, req *models.SubmitScreeningRequest) (*models.MemberScreening, error) {
	// Check if screening already exists
	existing, err := s.repo.GetMemberScreening(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status != models.ScreeningStatusPending {
		return nil, ErrScreeningAlreadyExists
	}

	// Marshal answers
	answersJSON, err := json.Marshal(req.Answers)
	if err != nil {
		return nil, err
	}

	screening := &ScreeningRepo{
		ID:        uuid.New(),
		UserID:    userID,
		ServerID:  serverID,
		Answers:   answersJSON,
		RulesRead: req.RulesRead,
		Status:    models.ScreeningStatusPending,
	}

	if err := s.repo.UpsertMemberScreening(ctx, screening); err != nil {
		return nil, err
	}

	var answers []models.ScreeningAnswer
	json.Unmarshal(answersJSON, &answers)

	return &models.MemberScreening{
		ID:        screening.ID,
		UserID:    screening.UserID,
		ServerID:  screening.ServerID,
		Answers:   answers,
		RulesRead: screening.RulesRead,
		Status:    screening.Status,
	}, nil
}

// GetMemberScreening retrieves a member's screening status
func (s *WelcomeService) GetMemberScreening(ctx context.Context, userID, serverID uuid.UUID) (*models.MemberScreening, error) {
	screening, err := s.repo.GetMemberScreening(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if screening == nil {
		return nil, nil
	}

	var answers []models.ScreeningAnswer
	if err := json.Unmarshal(screening.Answers, &answers); err != nil {
		return nil, err
	}

	return &models.MemberScreening{
		ID:        screening.ID,
		UserID:    screening.UserID,
		ServerID:  screening.ServerID,
		Answers:   answers,
		RulesRead: screening.RulesRead,
		Status:    screening.Status,
		CreatedAt: parseTime(screening.CreatedAt),
		UpdatedAt: parseTime(screening.UpdatedAt),
	}, nil
}

// GetPendingScreenings retrieves pending screenings for a server (moderators only)
func (s *WelcomeService) GetPendingScreenings(ctx context.Context, serverID, requesterID uuid.UUID, limit, offset int) ([]*models.MemberScreening, error) {
	// Check if requester is server owner or has manage server permission
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrServerNotFound
	}

	hasPerm, err := s.permService.HasPermission(ctx, serverID, requesterID, models.PermManageServer)
	if err != nil {
		return nil, err
	}
	if server.OwnerID != requesterID && !hasPerm {
		return nil, ErrNotServerModerator
	}

	screeningsRepo, err := s.repo.GetPendingScreenings(ctx, serverID, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]*models.MemberScreening, len(screeningsRepo))
	for i, sr := range screeningsRepo {
		var answers []models.ScreeningAnswer
		if err := json.Unmarshal(sr.Answers, &answers); err != nil {
			return nil, err
		}
		result[i] = &models.MemberScreening{
			ID:        sr.ID,
			UserID:    sr.UserID,
			ServerID:  sr.ServerID,
			Answers:   answers,
			RulesRead: sr.RulesRead,
			Status:    sr.Status,
			CreatedAt: parseTime(sr.CreatedAt),
			UpdatedAt: parseTime(sr.UpdatedAt),
		}
	}

	return result, nil
}

// ApproveScreening approves a member's screening
func (s *WelcomeService) ApproveScreening(ctx context.Context, userID, serverID, moderatorID uuid.UUID) error {
	// Check permissions
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if server == nil {
		return ErrServerNotFound
	}

	hasPerm, err := s.permService.HasPermission(ctx, serverID, moderatorID, models.PermManageServer)
	if err != nil {
		return err
	}
	if server.OwnerID != moderatorID && !hasPerm {
		return ErrNotServerModerator
	}

	// Update screening status
	if err := s.repo.UpdateScreeningStatus(ctx, userID, serverID, models.ScreeningStatusApproved); err != nil {
		return err
	}

	// Update member's pending status to false (they're now approved)
	member, err := s.serverRepo.GetMember(ctx, serverID, userID)
	if err != nil {
		return err
	}
	if member != nil {
		member.Pending = false
		if err := s.serverRepo.UpdateMember(ctx, member); err != nil {
			return err
		}
	}

	return nil
}

// RejectScreening rejects a member's screening and removes them from the server
func (s *WelcomeService) RejectScreening(ctx context.Context, userID, serverID, moderatorID uuid.UUID, reason string) error {
	// Check permissions
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if server == nil {
		return ErrServerNotFound
	}

	hasPerm, err := s.permService.HasPermission(ctx, serverID, moderatorID, models.PermManageServer)
	if err != nil {
		return err
	}
	if server.OwnerID != moderatorID && !hasPerm {
		return ErrNotServerModerator
	}

	// Update screening status
	if err := s.repo.UpdateScreeningStatus(ctx, userID, serverID, models.ScreeningStatusRejected); err != nil {
		return err
	}

	// Remove member from server
	if err := s.serverRepo.RemoveMember(ctx, serverID, userID); err != nil {
		return err
	}

	return nil
}
