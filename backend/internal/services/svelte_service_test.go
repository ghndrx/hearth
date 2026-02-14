package services

import (
	"context"
	"testing"

	"github.com/hearth-distro/dsadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDatastoreAdapter is a generated mock implementation of dsadapter.DatastoreAdapter.
// Provide this in your project dependencies, or define a mock struct manually.
// (For this example, assume this is imported or provided by the test harness).

func TestSvelteService_LaunchSprint_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockDS := new(MockDatastoreAdapter) // Assuming the interface is imported from dsadapter
	
	expectedSprint := SprintUpdate{
		SprintName:  "Viking Conquest",
		StartTime:   1672531200,
		EndTime:     1672617600,
		Participants: []Participant{
			{UserID: "user_1", Username: "Thor"},
			{UserID: "user_2", Username: "Loki"},
		},
	}

	// Setup expectations
	mockDS.On("StoreSprintConfig", mock.Anything, expectedSprint).Return(nil)
	mockDS.On("NotifyDiscordAdapter", mock.Anything, "sprint_launch", mock.AnythingOfType("SprintUpdate")).Return(nil)

	// Act
	service := NewSvelteService(mockDS)
	err := service.LaunchSprint(ctx, expectedSprint)

	// Assert
	assert.NoError(t, err)
	mockDS.AssertExpectations(t)
}

func TestSvelteService_LaunchSprint_ValidationError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockDS := new(MockDatastoreAdapter)
	
	invalidSprint := SprintUpdate{
		SprintName: "",
	}

	// Act
	service := NewSvelteService(mockDS)
	err := service.LaunchSprint(ctx, invalidSprint)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
	mockDS.AssertNotCalled(t, "StoreSprintConfig")
}

func TestSvelteService_LaunchSprint_DatabaseFailure(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockDS := new(MockDatastoreAdapter)
	
	sprint := SprintUpdate{
		SprintName: "Test",
		Participants: []Participant{{UserID: "u1"}},
	}

	mockDS.On("StoreSprintConfig", mock.Anything, sprint).Return(assert.AnError) // Simulate DB Write failure
	
	// Act
	service := NewSvelteService(mockDS)
	err := service.LaunchSprint(ctx, sprint)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	mockDS.AssertExpectations(t)
}