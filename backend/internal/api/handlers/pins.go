package handlers

import (
	"net/http"
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
	channelID := r.PathValue("channel_id")
	messageID := r.PathValue("message_id")
	
	user := getUser(r)
	
	pin := &models.Pin{
		ID:        generateID(),
		MessageID: messageID,
		ChannelID: channelID,
		UserID:    user.ID,
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
