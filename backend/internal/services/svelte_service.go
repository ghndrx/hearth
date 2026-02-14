package services

import (
	"context"
	"errors"

	"github.com/hearth-distro/dsadapter"
)

// SprintUpdate represents the payload sent to Discord webhooks or adapters.
type SprintUpdate struct {
	SprintName string `json:"sprintName"`
	StartTime  int64  `json:"startTime"`
	EndTime    int64  `json:"endTime"`
	Participants []Participant `json:"participants"`
}

type Participant struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

// ISvelteService defines the contract for managing the Svelte mapping service.
type ISvelteService interface {
	// LaunchSprint triggers a process where the Hearth UI sends configuration
	// data to the Discord service to sync the current sprint.
	LaunchSprint(ctx context.Context, sprint SprintUpdate) error
}

// SvelteService implements the business logic for "svelte" integration.
// In this context, it manages the synchronization state between the HTML/JS UI (Svelte)
// and the underlying backend services.
type SvelteService struct {
	ds dsadapter.DatastoreAdapter
}

// NewSvelteService initializes a new Svelte interaction handler.
func NewSvelteService(ds dsadapter.DatastoreAdapter) ISvelteService {
	return &SvelteService{
		ds: ds,
	}
}

// LaunchSprint validates the payload and updates the datastore/configuration.
// In a real scenario, this might trigger a webhook to Discord Bot APIs.
func (s *SvelteService) LaunchSprint(ctx context.Context, sprint SprintUpdate) error {
	// 1. Validation Logic (Business Rules)
	if sprint.SprintName == "" {
		return errors.New("sprint name cannot be empty")
	}
	if len(sprint.Participants) == 0 {
		// Warning or Error depending on requirements. 
		// For now, we assume a sprint requires participants.
		return errors.New("sprint must have at least one participant")
	}

	// 2. Data Persistence / State Management
	// Here we save the 'active configuration' allowing the Svelte client 
	// to remain stateless and query the DB for the latest config.
	err := s.ds.StoreSprintConfig(ctx, sprint)
	if err != nil {
		return err
	}

	// 3. Notify Discord Adapter (Post-Commit)
	// This simulates the service notifying the Discord side that the config changed.
	err = s.ds.NotifyDiscordAdapter(ctx, "sprint_launch", sprint)
	if err != nil {
		// Log error but return success to the frontend so the UI doesn't break
		// In production, levels like logrus/slog would be used here.
		return err
	}

	return nil
}