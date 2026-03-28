package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

func (h *Handler) FollowChannel(w http.ResponseWriter, r *http.Request) {
	channelID, _ := uuid.Parse(r.PathValue("channel_id"))

	var req struct {
		FollowerChannelID string `json:"follower_channel_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	followerID, _ := uuid.Parse(req.FollowerChannelID)
	follow := &models.FollowedChannel{
		ChannelID:         channelID,
		FollowerChannelID: followerID,
		CreatedAt:         time.Now(),
	}

	if err := h.db.CreateFollowedChannel(r.Context(), follow); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, follow, http.StatusCreated)
}

func (h *Handler) UnfollowChannel(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("channel_id")

	var req struct {
		FollowerChannelID string `json:"follower_channel_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteFollowedChannel(r.Context(), channelID, req.FollowerChannelID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("channel_id")

	followers, err := h.db.GetFollowedChannels(r.Context(), channelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, followers, http.StatusOK)
}
