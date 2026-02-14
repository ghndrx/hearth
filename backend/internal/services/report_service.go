package services

import (
	"context"
	"time"
)

// ReportReason represents the category of the report (e.g., Spam, Harassment).
type ReportReason string

const (
	ReasonSpam      ReportReason = "spam"
	ReasonHarassment ReportReason = "harassment"
	ReasonInappropriateContent ReportReason = "inappropriate_content"
	ReasonOther ReportReason = "other"
)

// ReportStatus represents the current state of the report workflow.
type ReportStatus string

const (
	StatusOpen ReportStatus = "open"
	StatusPendingAdmin ReportStatus = "pending_admin_review"
	StatusResolved ReportStatus = "resolved"
	StatusDismissed ReportStatus = "dismissed"
)

// Report represents a user report within the application.
type Report struct {
	ID           string
	ChannelID    string         `json:"channel_id"`
	AuthorID     string         `json:"author_id"`
	ReporteeID   string         `json:"reportee_id"`
	Reason       ReportReason   `json:"reason"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Status       ReportStatus   `json:"status"`
	ResolvedBy   string         `json:"resolved_by,omitempty"`
	ResolvedAt   *time.Time     `json:"resolved_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ReportRepository defines the contract for data persistence.
type ReportRepository interface {
	CreateReport(ctx context.Context, r *Report) error
	GetReport(ctx context.Context, id string) (*Report, error)
	GetReports(ctx context.Context, filter ReportFilter) ([]*Report, error)
}

// ReportFilter is used to query reports.
type ReportFilter struct {
	Status     ReportStatus
	AuthorID   string
	CreatedAt  time.Time
	Limit      int
}

// ReportService handles the business logic for reports.
type ReportService struct {
	repo ReportRepository
}

// NewReportService initializes a new ReportService.
func NewReportService(repo ReportRepository) *ReportService {
	return &ReportService{
		repo: repo,
	}
}

// ReportInput defines the fields required to create a new report.
type ReportInput struct {
	ChannelID    string
	AuthorID     string
	ReporteeID   string
	Reason       ReportReason
	Title        string
	Description  string
}

// CreateReport creates a new report entry.
func (s *ReportService) CreateReport(ctx context.Context, input ReportInput) (*Report, error) {
	now := time.Now()

	report := &Report{
		ID:           generateUUID(), // In a real implementation, use ID generator
		ChannelID:    input.ChannelID,
		AuthorID:     input.AuthorID,
		ReporteeID:   input.ReporteeID,
		Reason:       input.Reason,
		Title:        input.Title,
		Description:  input.Description,
		Status:       StatusOpen,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateReport(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

// GetReports retrieves reports based on filters.
func (s *ReportService) GetReports(ctx context.Context, filter ReportFilter) ([]*Report, error) {
	return s.repo.GetReports(ctx, filter)
}

// UpdateStatus updates the status of a report by an admin.
func (s *ReportService) UpdateStatus(ctx context.Context, reportID string, status ReportStatus, adminID string) (*Report, error) {
	report, err := s.repo.GetReport(ctx, reportID)
	if err != nil {
		return nil, err
	}

	if report.Status == status {
		return report, nil // No change needed
	}

	now := time.Now()
	newResolvedAt := now

	// If closed, set resolved timestamp.
	// If reopened, handle logic accordingly (omitted for brevity, keeping simple resolution).
	if status == StatusResolved || status == StatusDismissed {
		report.Status = status
		report.ResolvedBy = &adminID
		report.ResolvedAt = &newResolvedAt
	} else {
		// Update only status and timestamp
		report.Status = status
	}

	report.UpdatedAt = now

	// Note: This example does not implement a UpdateReport method in the custom Repo interface
	// to keep the example files clean. In prod, ensure a generic repository method exists.
	// We'll just mock the repository update for the purposes of the interface contract:
	
	// pseudo code: err = s.repo.UpdateReport(ctx, report)

	return report, nil
}

// generateUUID is a helper for unique ID generation (simplified).
func generateUUID() string {
	return fmt.Sprintf("rep-%d", time.Now().UnixNano())
}