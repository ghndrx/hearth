package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockReportRepository is a concrete implementation of ReportRepository for testing.
type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) CreateReport(ctx context.Context, report *Report) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockReportRepository) GetReport(ctx context.Context, id string) (*Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Report), args.Error(1)
}

func (m *MockReportRepository) GetReports(ctx context.Context, filter ReportFilter) ([]*Report, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*Report), args.Error(1)
}

// In-memory database for tests
var testDB = make(map[string]*Report)

func init() {
	// Populate with a default report
	now := time.Now()
	testDB["rep-1"] = &Report{
		ID:         "rep-1",
		Status:     StatusOpen,
		Reason:     ReasonSpam,
		AuthorID:   "user-1",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestCreateReport(t *testing.T) {
	service := NewReportService(&MockReportRepository{})
	
	ctx := context.Background()
	input := ReportInput{
		ChannelID:   "ch-123",
		AuthorID:    "user-2",
		ReporteeID:  "user-3",
		Reason:      ReasonHarassment,
		Title:       "Inappropriate DMs",
		Description: "User is sending insults.",
	}

	// Expect the repo to be called once with the correct args
	mockRepo := service.repo.(*MockReportRepository)
	mockRepo.On("CreateReport", ctx, mock.MatchedBy(func(r *Report) bool {
		return r.ChannelID == input.ChannelID && 
		       r.Reason == ReasonHarassment && 
		       r.Status == StatusOpen
	})).Return(nil)

	report, err := service.CreateReport(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, input.ChannelID, report.ChannelID)
	assert.Equal(t, StatusOpen, report.Status)
	mockRepo.AssertExpectations(t)
}

func TestGetReports(t *testing.T) {
	service := NewReportService(&MockReportRepository{})
	ctx := context.Background()

	// Define filter to get open reports
	filter := ReportFilter{
		Status: StatusOpen,
	}

	mockRepo := service.repo.(*MockReportRepository)
	mockRepo.On("GetReports", ctx, filter).Return([]*Report{
		{ID: "rep-1", Status: StatusOpen},
		{ID: "rep-4", Status: StatusOpen},
	}, nil)

	reports, err := service.GetReports(ctx, filter)

	assert.NoError(t, err)
	assert.Len(t, reports, 2)
	mockRepo.AssertExpectations(t)
}

func TestUpdateStatus(t *testing.T) {
	service := NewReportService(&MockReportRepository{})
	ctx := context.Background()
	adminID := "admin-1"

	mockRepo := service.repo.(*MockReportRepository)
	
	// Get existing report
	existingReport := &Report{
		ID:      "rep-1",
		Status:  StatusPendingAdminReview,
		AuthorID: "user-1",
	}
	mockRepo.On("GetReport", ctx, "rep-1").Return(existingReport, nil)

	// Update status
	status := StatusResolved
	newResolvedAt := time.Now()
	existingReport.Status = StatusResolved
	existingReport.ResolvedBy = &adminID
	existingReport.ResolvedAt = &newResolvedAt
	existingReport.UpdatedAt = newResolvedAt

	mockRepo.On("GetReports", ctx, ReportFilter{}).Return([]*Report{existingReport}, nil)

	report, err := service.UpdateStatus(ctx, "rep-1", status, adminID)

	assert.NoError(t, err)
	assert.Equal(t, StatusResolved, report.Status)
	assert.NotNil(t, report.ResolvedAt)
	assert.Equal(t, adminID, *report.ResolvedBy)
	mockRepo.AssertExpectations(t)
}