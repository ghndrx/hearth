package handlers

import (
	"net/http"
	"hearth/internal/models"
)

func (h *Handler) GetSounds(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("server_id")
	
	sounds, err := h.db.GetSounds(r.Context(), serverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	jsonResponse(w, sounds, http.StatusOK)
}

func (h *Handler) CreateSound(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("server_id")
	user := getUser(r)
	
	var req struct {
		Name     string `json:"name"`
		AudioURL string `json:"audio_url"`
		Emoji    string `json:"emoji"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	sound := &models.Sound{
		ID:        generateID(),
		ServerID:  serverID,
		Name:      req.Name,
		AudioURL: req.AudioURL,
		Emoji:    req.Emoji,
		CreatedBy: user.ID,
	}
	
	if err := h.db.CreateSound(r.Context(), sound); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	jsonResponse(w, sound, http.StatusCreated)
}

func (h *Handler) DeleteSound(w http.ResponseWriter, r *http.Request) {
	soundID := r.PathValue("id")
	
	if err := h.db.DeleteSound(r.Context(), soundID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PlaySound(w http.ResponseWriter, r *http.Request) {
	soundID := r.PathValue("id")
	
	sound, err := h.db.GetSound(r.Context(), soundID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	// Broadcast to channel that this sound was triggered
	h.wsHub.BroadcastToChannel(sound.ServerID, map[string]interface{}{
		"type":      "sound_play",
		"sound_id":  sound.ID,
		"audio_url": sound.AudioURL,
		"emoji":     sound.Emoji,
	})
	
	w.WriteHeader(http.StatusNoContent)
}
