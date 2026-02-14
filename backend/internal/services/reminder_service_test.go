package services

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"hearth/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockReminderRepository implements ReminderRepository for testing.
type MockReminderRepository struct {
	mock.Mock
}

func (m *MockReminderRepository) Create(ctx context.Context, reminder models.Reminder) error {
	args := m.Called(ctx, reminder)
	return args.Error(0)
}

func (m *MockReminderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Reminder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reminder), args.Error(1)
}

func (m *MockReminderRepository) Update(ctx context.Context, reminder models.Reminder) error {
	args := m.Called(ctx, reminder)
	return args.Error(0)
}

func (m *MockReminderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReminderRepository) GetRemindersByChannel(ctx context.Context, channelID uuid.UUID) ([]models.Reminder, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Reminder), args.Error(1)
}

func TestReminderService_Create(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockReminderRepository)
	service := NewReminderService(mockRepo)

	channelID := uuid.New()
	userID := uuid.New()
	content := "Meeting at 3pm"

	expectedID := uuid.New()

	mockRepo.On("Create", ctx, mock.MatchedBy(func(r models.Reminder) bool {
		return r.ChannelID == channelID &&
			r.UserID == userID &&
			r.Content == content
	})).Return(nil).Once()

	reminder, err := service.Create(ctx, channelID, userID, content)

	mockRepo.AssertExpectations(t)

	assert.NoError(t, err)
	assert.NotNil(t, reminder)
	assert.Equal(t, expectedID, reminder.ID)
	assert.Equal(t, channelID, reminder.ChannelID)
	assert.Equal(t, userID, reminder.UserID)
	assert.Equal(t, content, reminder.Content)
}

func TestReminderService_Create_Repository_Failure(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockReminderRepository)
	service := NewReminderService(mockRepo)

	mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("db connection failed")).Once()

	_, err := service.Create(ctx, uuid.New(), uuid.New(), "test")

	mockRepo.AssertExpectations(t)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create reminder")
}

func TestReminderService_Get(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockReminderRepository)
	service := NewReminderService(mockRepo)

	reminderID := uuid.New()
	Date := "2023-10-27T09:41:12Z"
	expectedNode := uuid.New()
	content := "Water the plant"
	channel := uuid.New()
	user := uuid.New()

	reminder := &models.Reminder{
		ID:        reminderID,
		UserID:    user,
		ChannelID: channel,
		Content:   content,
	}

	mockRepo.On("GetByID", ctx, reminderID).Return(reminder, nil).Once()

	result, err := service.Get(ctx, reminderID)

	mockRepo.AssertExpectations(t)

	assert.NoError(t, err)
	assert.Equal(t, reminder, result)
}

func TestReminderService_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockReminderRepository)
	service := NewReminderService(mockRepo)

	mockRepo.On("GetByID", ctx, uuid.New()).Return(nil, errors.New("record not found")).Once()

	_, err := service.Get(ctx, uuid.New())

	mockRepo.AssertExpectations(t)
	assert.Error(t, err)
}

func TestReminderService_Get_EmptyID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockReminderRepository)
	service := NewReminderService(mockRepo)

	_, err := service.Get(ctx, uuid.Nil)

	assert.Error(t, err)
	assert.Equal(t, "reminder ID cannot be empty", err.Error())
	// Ensure repository was not called
	mockRepo.AssertNotCalled(t, "GetByID")
}