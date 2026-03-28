package handlers

import (
	"net/http"
	"hearth/internal/models"
	"time"
)

func (h *Handler) CreateReminder(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	
	var req models.CreateReminderRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	remindAt, err := time.Parse(time.RFC3339, req.RemindAt)
	if err != nil {
		http.Error(w, "invalid remind_at format", http.StatusBadRequest)
		return
	}
	
	reminder := &models.Reminder{
		ID:        generateID(),
		UserID:    user.ID,
		MessageID: req.MessageID,
		ChannelID: req.ChannelID,
		RemindAt:  remindAt,
		CreatedAt: time.Now(),
	}
	
	if err := h.db.CreateReminder(r.Context(), reminder); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	jsonResponse(w, reminder, http.StatusCreated)
}

func (h *Handler) GetReminders(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	
	reminders, err := h.db.GetUserReminders(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	jsonResponse(w, reminders, http.StatusOK)
}

func (h *Handler) DeleteReminder(w http.ResponseWriter, r *http.Request) {
	reminderID := r.PathValue("id")
	user := getUser(r)
	
	if err := h.db.DeleteReminder(r.Context(), reminderID, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}
