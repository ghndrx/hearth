package services

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Report struct {
	ID         uuid.UUID
	ReporterID uuid.UUID
	TargetID   uuid.UUID
	TargetType string
	Reason     string
	Status     string
	CreatedAt  time.Time
}

type ReportService struct {
	mu      sync.RWMutex
	reports []Report
}

func NewReportService() *ReportService {
	return &ReportService{}
}

func (s *ReportService) Create(ctx context.Context, reporterID, targetID uuid.UUID, targetType, reason string) (*Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := Report{
		ID:         uuid.New(),
		ReporterID: reporterID,
		TargetID:   targetID,
		TargetType: targetType,
		Reason:     reason,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	s.reports = append(s.reports, r)
	return &r, nil
}

func (s *ReportService) UpdateStatus(ctx context.Context, reportID uuid.UUID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.reports {
		if s.reports[i].ID == reportID {
			s.reports[i].Status = status
			return nil
		}
	}
	return nil
}

func (s *ReportService) GetPending(ctx context.Context) ([]Report, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pending []Report
	for _, r := range s.reports {
		if r.Status == "pending" {
			pending = append(pending, r)
		}
	}
	return pending, nil
}
