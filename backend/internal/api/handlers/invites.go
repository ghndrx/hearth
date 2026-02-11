package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hearth/internal/services"
)

// InviteHandlers handles invite-related HTTP requests
type InviteHandlers struct {
	inviteService *services.InviteService
}

// NewInviteHandlers creates new invite handlers
func NewInviteHandlers(inviteService *services.InviteService) *InviteHandlers {
	return &InviteHandlers{inviteService: inviteService}
}

// CreateInviteRequest represents an invite creation request
type CreateInviteRequest struct {
	MaxAge    int  `json:"max_age"`    // Seconds, 0 = never expires
	MaxUses   int  `json:"max_uses"`   // 0 = unlimited
	Temporary bool `json:"temporary"`
}

// InviteResponse represents an invite in API responses
type InviteResponse struct {
	Code      string          `json:"code"`
	ServerID  string          `json:"guild_id"`
	ChannelID string          `json:"channel_id"`
	CreatorID string          `json:"inviter_id"`
	MaxUses   int             `json:"max_uses"`
	Uses      int             `json:"uses"`
	ExpiresAt *time.Time      `json:"expires_at,omitempty"`
	Temporary bool            `json:"temporary"`
	CreatedAt time.Time       `json:"created_at"`
	Server    *ServerResponse `json:"guild,omitempty"`
}

// CreateInvite creates a new invite for a channel
func (h *InviteHandlers) CreateInvite(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	var req CreateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get channel to find server ID
	// TODO: Get channel from service to get server ID
	serverID := uuid.Nil // Placeholder

	var maxAge time.Duration
	if req.MaxAge > 0 {
		maxAge = time.Duration(req.MaxAge) * time.Second
	}

	invite, err := h.inviteService.CreateInvite(r.Context(), &services.CreateInviteRequest{
		ServerID:  serverID,
		ChannelID: channelID,
		CreatorID: userID,
		MaxAge:    maxAge,
		MaxUses:   req.MaxUses,
		Temporary: req.Temporary,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toInviteResponse(invite))
}

// GetInvite retrieves an invite by code
func (h *InviteHandlers) GetInvite(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "Invalid invite code")
		return
	}

	invite, err := h.inviteService.GetInvite(r.Context(), code)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toInviteResponse(invite))
}

// UseInvite joins a server using an invite
func (h *InviteHandlers) UseInvite(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	code := chi.URLParam(r, "code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "Invalid invite code")
		return
	}

	server, err := h.inviteService.UseInvite(r.Context(), code, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toServerResponse(server))
}

// DeleteInvite deletes an invite
func (h *InviteHandlers) DeleteInvite(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	code := chi.URLParam(r, "code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "Invalid invite code")
		return
	}

	if err := h.inviteService.DeleteInvite(r.Context(), code, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetServerInvites gets all invites for a server
func (h *InviteHandlers) GetServerInvites(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	serverID, err := uuid.Parse(chi.URLParam(r, "serverID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid server ID")
		return
	}

	invites, err := h.inviteService.GetServerInvites(r.Context(), serverID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := make([]InviteResponse, len(invites))
	for i, invite := range invites {
		response[i] = toInviteResponse(invite)
	}

	respondJSON(w, http.StatusOK, response)
}

// Ban handlers

// BanRequest represents a ban request
type BanRequest struct {
	Reason string `json:"reason"`
}

// BanResponse represents a ban in API responses
type BanResponse struct {
	User   *UserResponse `json:"user"`
	Reason string        `json:"reason,omitempty"`
}

// BanMember bans a user from a server
func (h *InviteHandlers) BanMember(w http.ResponseWriter, r *http.Request) {
	moderatorID := getUserID(r)
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

	var req BanRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // Reason is optional

	if err := h.inviteService.BanMember(r.Context(), serverID, userID, moderatorID, req.Reason); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UnbanMember removes a ban from a server
func (h *InviteHandlers) UnbanMember(w http.ResponseWriter, r *http.Request) {
	moderatorID := getUserID(r)
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

	if err := h.inviteService.UnbanMember(r.Context(), serverID, userID, moderatorID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetServerBans gets all bans for a server
func (h *InviteHandlers) GetServerBans(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	serverID, err := uuid.Parse(chi.URLParam(r, "serverID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid server ID")
		return
	}

	bans, err := h.inviteService.GetServerBans(r.Context(), serverID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := make([]BanResponse, len(bans))
	for i, ban := range bans {
		response[i] = BanResponse{
			Reason: ban.Reason,
			// TODO: Populate user
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// Helper functions

func toInviteResponse(invite interface{}) InviteResponse {
	// TODO: Proper conversion from model
	return InviteResponse{}
}
