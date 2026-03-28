package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

func (h *Handler) GetPinnedMessages(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("channel_id")

	pins, err := h.db.GetPinnedMessages(r.Context(), channelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, pins, http.StatusOK)
}

func (h *Handler) PinMessage(w http.ResponseWriter, r *http.Request) {
	channelID, _ := uuid.Parse(r.PathValue("channel_id"))
	messageID, _ := uuid.Parse(r.PathValue("message_id"))

	user := getUser(r)
	userID, _ := uuid.Parse(user.ID)

	pin := &models.Pin{
		MessageID: messageID,
		ChannelID: channelID,
		PinnedBy:  userID,
		PinnedAt:  time.Now(),
	}

	if err := h.db.CreatePin(r.Context(), pin); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, pin, http.StatusCreated)
}

func (h *Handler) UnpinMessage(w http.ResponseWriter, r *http.Request) {
	messageID := r.PathValue("message_id")

	if err := h.db.DeletePin(r.Context(), messageID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
