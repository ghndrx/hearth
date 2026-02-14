package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// Define specific errors for the service
var (
	ErrMemberNotFound = errors.New("member not found")
)

// MemberSidebarRepository defines the data access methods required for the Member sidebar.
// Note: This uses an interface to avoid importing postgres directly.
type MemberSidebarRepository interface {
	// GetMembersByServer fetches all members for a given server ID.
	GetMembersByServer(ctx context.Context, serverID uuid.UUID) ([]*models.Member, error)
	// GetMemberPresence retrieves the current status of a user (e.g., Online, Idle).
	GetMemberPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error)
	// GetServerRoles fetches role definitions to help with sorting/grouping.
	GetServerRoles(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error)
}

// MemberSidebarService handles business logic for the member list sidebar.
// It aggregates members, enriches them with presence data, and groups them by role.
type MemberSidebarService struct {
	repo MemberSidebarRepository
}

// NewMemberSidebarService creates a new instance of the service.
func NewMemberSidebarService(repo MemberSidebarRepository) *MemberSidebarService {
	return &MemberSidebarService{
		repo: repo,
	}
}

// MemberView represents the data structure sent to the frontend sidebar.
// It combines Member data with their live Presence status.
type MemberView struct {
	Member   *models.Member
	Presence *models.Presence
}

// RoleGroup represents a section in the sidebar (e.g., "Admins", "Members").
type RoleGroup struct {
	RoleName string
	Members  []*MemberView
}

// GetServerSidebar organizes members by role for a specific server.
// This corresponds to "Group members by role" from Task F2.
func (s *MemberSidebarService) GetServerSidebar(ctx context.Context, serverID uuid.UUID) ([]*RoleGroup, error) {
	if serverID == uuid.Nil {
		return nil, ErrInvalidInput
	}

	// Fetch raw members
	members, err := s.repo.GetMembersByServer(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch members: %w", err)
	}

	// Fetch roles to determine sort order/grouping
	roles, err := s.repo.GetServerRoles(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roles: %w", err)
	}

	// Create a map for Role ID -> Role Name
	roleMap := make(map[uuid.UUID]string)
	for _, r := range roles {
		roleMap[r.ID] = r.Name
	}

	// Prepare map for grouping: Role Name -> Members
	groupsMap := make(map[string][]*MemberView)

	// Iterate over members to attach presence and group them
	for _, m := range members {
		// Fetch Presence (Task F2 Requirement: "show presence dots")
		presence, err := s.repo.GetMemberPresence(ctx, m.UserID)
		if err != nil {
			// Log error but don't fail entire request if one presence is missing
			// Default to offline/unknown
			presence = &models.Presence{Status: models.StatusOffline}
		}

		view := &MemberView{
			Member:   m,
			Presence: presence,
		}

		// Determine group name based on member's roles
		// Use the first role name found, or default to "Member"
		roleName := "Member"
		if len(m.Roles) > 0 {
			if name, ok := roleMap[m.Roles[0]]; ok {
				roleName = name
			}
		}

		groupsMap[roleName] = append(groupsMap[roleName], view)
	}

	// Convert map to ordered slice (Sorting roles by priority or creation would happen here)
	// For this example, we just dump the map into the slice structure.
	var result []*RoleGroup
	for name, views := range groupsMap {
		result = append(result, &RoleGroup{
			RoleName: name,
			Members:  views,
		})
	}

	return result, nil
}

// GetMemberProfile retrieves detailed info for a specific user interaction.
// This corresponds to "click for profile" from Task F2.
func (s *MemberSidebarService) GetMemberProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidInput
	}

	// In a real scenario, the repo would have a GetUser method.
	// For this task, we assume the presence lookup logic validates existence.
	_, err := s.repo.GetMemberPresence(ctx, userID)
	if err != nil {
		return nil, ErrMemberNotFound
	}

	// Placeholder for actual profile retrieval logic
	return &models.User{
		ID:       userID,
		Username: "MockUser",
	}, nil
}
