package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockPollRepository is a mock implementation of PollRepository for testing.
type MockPollRepository struct {
	mock.Mock
}

func (m *MockPollRepository) Create(ctx context.Context, poll *models.Poll) error {
	args := m.Called(ctx, poll)
	return args.Error(0)
}

func (m *MockPollRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Poll, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Poll), args.Error(1)
}

func (m *MockPollRepository) Update(ctx context.Context, poll *models.Poll) error {
	args := m.Called(ctx, poll)
	return args.Error(0)
}

func (m *MockPollRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPollRepository) GetByGuildID(ctx context.Context, guildID uuid.UUID) ([]*models.Poll, error) {
	args := m.Called(ctx, guildID)
	return args.Get(0).([]*models.Poll), args.Error(1)
}

func (m *MockPollRepository) VoteForOption(ctx context.Context, pollID, optionID uuid.UUID) error {
	args := m.Called(ctx, pollID, optionID)
	return args.Error(0)
}

func (m *MockPollRepository) CheckUserVote(ctx context.Context, pollID, userID uuid.UUID) (*models.PollOptionVote, error) {
	args := m.Called(ctx, pollID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PollOptionVote), args.Error(1)
}

func setupMockRepo() *MockPollRepository {
	return new(MockPollRepository)
}

func setupTestPoll(id uuid.UUID) *models.Poll {
	return &models.Poll{
		ID:        id,
		ChannelID: uuid.New(),
		CreatorID: uuid.New(),
		Question:  "Test Poll",
		Options: []models.PollOption{
			{
				ID:   uuid.New(),
				Text: "Yes",
			},
		},
	}
}

func TestPollService_CreatePoll(t *testing.T) {
	ctx := context.Background()
	repo := setupMockRepo()
	service := NewPollService(repo)
	poll := setupTestPoll(uuid.Nil)

	repo.On("Create", ctx, mock.AnythingOfType("*models.Poll")).Return(nil)

	err := service.CreatePoll(ctx, poll)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestPollService_GetGuildPolls(t *testing.T) {
	ctx := context.Background()
	repo := setupMockRepo()
	service := NewPollService(repo)
	guildID := uuid.New()

	expectedPolls := []*models.Poll{
		setupTestPoll(uuid.New()),
	}

	repo.On("GetByGuildID", ctx, guildID).Return(expectedPolls, nil)

	polls, err := service.GetGuildPolls(ctx, guildID)

	assert.NoError(t, err)
	assert.Equal(t, expectedPolls, polls)
	repo.AssertExpectations(t)
}

func TestPollService_Vote(t *testing.T) {
	ctx := context.Background()
	repo := setupMockRepo()
	service := NewPollService(repo)
	pollID := uuid.New()
	optionID := uuid.New()
	userID := uuid.New()

	// Scenario 1: Successful Vote
	repo.On("CheckUserVote", ctx, pollID, userID).Return(nil, nil)
	repo.On("VoteForOption", ctx, pollID, optionID).Return(nil)

	err := service.Vote(ctx, pollID, optionID, userID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)

	// Scenario 2: User Already Voted
	repo.ExpectedCalls = nil
	repo.On("CheckUserVote", ctx, pollID, userID).Return(&models.PollOptionVote{}, nil) // Mock existing vote
	repo.On("VoteForOption", ctx, pollID, optionID).Return(nil)                         // Should not be called

	err = service.Vote(ctx, pollID, optionID, userID)
	assert.Error(t, err)
	assert.Equal(t, "user has already voted on this poll", err.Error())
	repo.AssertNotCalled(t, "VoteForOption")
}
