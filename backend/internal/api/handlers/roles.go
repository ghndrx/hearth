package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// RoleHandlers handles role-related HTTP requests
type RoleHandlers struct {
	roleService *services.RoleService
}

// NewRoleHandlers creates new role handlers
func NewRoleHandlers(roleService *services.RoleService) *RoleHandlers {
	return &RoleHandlers{roleService: roleService}
}

// CreateRoleRequest represents a role creation request
type CreateRoleRequest struct {
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Permissions int64  `json:"permissions"`
	Hoist       bool   `json:"hoist"`
	Mentionable bool   `json:"mentionable"`
}

// UpdateRoleRequest represents a role update request
type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty"`
	Color       *int    `json:"color,omitempty"`
	Permissions *int64  `json:"permissions,omitempty"`
	Hoist       *bool   `json:"hoist,omitempty"`
	Mentionable *bool   `json:"mentionable,omitempty"`
}

// RoleResponse represents a role in API responses
type RoleResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Color       int       `json:"color"`
	Position    int       `json:"position"`
	Permissions string    `json:"permissions"` // String for large numbers
	Hoist       bool      `json:"hoist"`
	Mentionable bool      `json:"mentionable"`
	Managed     bool      `json:"managed"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateRole creates a new role in a server
func (h *RoleHandlers) CreateRole(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	serverID, err := uuid.Parse(chi.URLParam(r, "serverID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid server ID")
		return
	}

	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		req.Name = "new role"
	}

	role, err := h.roleService.CreateRole(r.Context(), serverID, userID, req.Name, req.Color, req.Permissions)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toRoleResponse(role))
}

// GetServerRoles gets all roles in a server
func (h *RoleHandlers) GetServerRoles(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	serverID, err := uuid.Parse(chi.URLParam(r, "serverID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid server ID")
		return
	}

	roles, err := h.roleService.GetServerRoles(r.Context(), serverID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := make([]RoleResponse, len(roles))
	for i, role := range roles {
		response[i] = toRoleResponse(role)
	}

	respondJSON(w, http.StatusOK, response)
}

// UpdateRole updates a role
func (h *RoleHandlers) UpdateRole(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := &models.RoleUpdate{
		Name:        req.Name,
		Color:       req.Color,
		Permissions: req.Permissions,
		Hoist:       req.Hoist,
		Mentionable: req.Mentionable,
	}

	role, err := h.roleService.UpdateRole(r.Context(), roleID, userID, updates)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toRoleResponse(role))
}

// DeleteRole deletes a role
func (h *RoleHandlers) DeleteRole(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	if err := h.roleService.DeleteRole(r.Context(), roleID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateRolePositions updates role positions
func (h *RoleHandlers) UpdateRolePositions(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	serverID, err := uuid.Parse(chi.URLParam(r, "serverID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid server ID")
		return
	}

	var positions []struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&positions); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	posMap := make(map[uuid.UUID]int)
	for _, p := range positions {
		id, err := uuid.Parse(p.ID)
		if err != nil {
			continue
		}
		posMap[id] = p.Position
	}

	if err := h.roleService.UpdateRolePositions(r.Context(), serverID, userID, posMap); err != nil {
		handleServiceError(w, err)
		return
	}

	// Return updated roles
	roles, _ := h.roleService.GetServerRoles(r.Context(), serverID, userID)
	response := make([]RoleResponse, len(roles))
	for i, role := range roles {
		response[i] = toRoleResponse(role)
	}

	respondJSON(w, http.StatusOK, response)
}

// Member role management

// AddMemberRole adds a role to a member
func (h *RoleHandlers) AddMemberRole(w http.ResponseWriter, r *http.Request) {
	requesterID := getUserID(r)
	serverID, err := uuid.Parse(chi.URLParam(r, "serverID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid server ID")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	if err := h.roleService.AddRoleToMember(r.Context(), serverID, userID, roleID, requesterID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveMemberRole removes a role from a member
func (h *RoleHandlers) RemoveMemberRole(w http.ResponseWriter, r *http.Request) {
	requesterID := getUserID(r)
	serverID, err := uuid.Parse(chi.URLParam(r, "serverID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid server ID")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid role ID")
		return
	}

	if err := h.roleService.RemoveRoleFromMember(r.Context(), serverID, userID, roleID, requesterID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper

func toRoleResponse(role *models.Role) RoleResponse {
	return RoleResponse{
		ID:          role.ID.String(),
		Name:        role.Name,
		Color:       role.Color,
		Position:    role.Position,
		Permissions: formatPermissions(role.Permissions),
		Hoist:       role.Hoist,
		Mentionable: role.Mentionable,
		Managed:     role.Managed,
		CreatedAt:   role.CreatedAt,
	}
}

func formatPermissions(perms int64) string {
	// Return as string for JavaScript BigInt compatibility
	return string(rune(perms))
}
