package services

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      string
	Title     string
	Body      string
	Read      bool
	CreatedAt time.Time
}

type NotificationService struct {
	mu            sync.RWMutex
	notifications map[uuid.UUID][]*Notification
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		notifications: make(map[uuid.UUID][]*Notification),
	}
}

func (s *NotificationService) Create(ctx context.Context, userID uuid.UUID, notifType, title, body string) (*Notification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := &Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
	}
	s.notifications[userID] = append(s.notifications[userID], n)
	return n, nil
}

func (s *NotificationService) GetUnread(ctx context.Context, userID uuid.UUID) ([]*Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var unread []*Notification
	for _, n := range s.notifications[userID] {
		if !n.Read {
			unread = append(unread, n)
		}
	}
	return unread, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, notifID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, notifs := range s.notifications {
		for _, n := range notifs {
			if n.ID == notifID {
				n.Read = true
				return nil
			}
		}
	}
	return nil
}
