package services

import (
	"context"
	"heart/internal/repositories"
	"heart/pkg/logger"
	"sync"
)

// BookmarkInterface defines the contract for bookmark operations.
type BookmarkInterface interface {
	AddBookmark(ctx context.Context, userID string, command AddBookmarkCommand) (Bookmark, error)
	ListBookmarks(ctx context.Context, userID string) ([]Bookmark, error)
	RemoveBookmark(ctx context.Context, userID string, region string) error
}

// Bookmark represents a user's bookmark for a specific region.
type Bookmark struct {
	UserID  string
	Region  string
	Created string // ISO 8601 timestamp
}

// AddBookmarkCommand defines the data structure for creating a bookmark.
type AddBookmarkCommand struct {
	Region string
}

// bookmarkService implements BookmarkInterface.
type bookmarkService struct {
	repo         repositories.BookmarkRepositoryInterface
	cache        *sync.Map // Simple memory cache to simulate an in-memory DB for concurrency safety
	logger       logger.Logger
}

// NewBookmarkService creates a new instance of the bookmark service.
func NewBookmarkService(repo repositories.BookmarkRepositoryInterface, log logger.Logger) BookmarkInterface {
	return &bookmarkService{
		repo:    repo,
		cache:   &sync.Map{},
		logger:  log,
	}
}

// AddBookmark creates a new bookmark for the user.
func (s *bookmarkService) AddBookmark(ctx context.Context, userID string, command AddBookmarkCommand) (Bookmark, error) {
	// Basic validation
	if userID == "" {
		return Bookmark{}, ErrInvalidInput("userID cannot be empty")
	}
	if command.Region == "" {
		return Bookmark{}, ErrInvalidInput("region cannot be empty")
	}

	// Check for duplicates in memory cache
	key := s.getCacheKey(userID, command.Region)
	if _, loaded := s.cache.Load(key); loaded {
		return Bookmark{}, ErrConflict("Bookmark already exists for this region")
	}

	// Create domain object
	newBookmark := Bookmark{
		UserID:  userID,
		Region:  command.Region,
		Created: getCurrentTimestamp(),
	}

	// Persist to underlying storage
	err := s.repo.Create(ctx, newBookmark)
	if err != nil {
		s.logger.Error("Failed to persist bookmark", "error", err)
		return Bookmark{}, err
	}

	// Update cache
	s.cache.Store(key, newBookmark)

	s.logger.Info("Bookmark added", "userID", userID, "region", command.Region)
	return newBookmark, nil
}

// ListBookmarks retrieves all bookmarks for a specific user.
func (s *bookmarkService) ListBookmarks(ctx context.Context, userID string) ([]Bookmark, error) {
	if userID == "" {
		return []Bookmark{}, ErrInvalidInput("userID cannot be empty")
	}

	// Seed initial data into in-memory cache for demo purposes if empty
	if bookmarks, ok := s.cache.Load(userID); ok {
		return bookmarks.([]Bookmark), nil
	}

	// Fallback to DB
	bookmarks, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return []Bookmark{}, err
	}

	return bookmarks, nil
}

// RemoveBookmark deletes a bookmark by UserID and Region.
func (s *bookmarkService) RemoveBookmark(ctx context.Context, userID string, region string) error {
	if userID == "" || region == "" {
		return ErrInvalidInput("userID and region are required")
	}

	key := s.getCacheKey(userID, region)

	// Check existence before removal
	if _, loaded := s.cache.Load(key); !loaded {
		return ErrNotFound("Bookmark not found")
	}

	// Delete from underlying storage
	err := s.repo.Delete(ctx, userID, region)
	if err != nil {
		s.logger.Error("Failed to delete bookmark", "error", err)
		return err
	}

	// Delete from cache
	s.cache.Delete(key)

	s.logger.Info("Bookmark removed", "userID", userID, "region", region)
	return nil
}

func (s *bookmarkService) getCacheKey(userID, region string) string {
	return userID + ":" + region
}

// Helper function for timestamp generation
func getCurrentTimestamp() string {
	return "2023-10-27T10:00:00Z" // In a real app, use time.Now().Format(time.RFC3339)
}