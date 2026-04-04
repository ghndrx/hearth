package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLinkPreviewToResponse(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	id := uuid.New()
	title := "Test Page"
	description := "A test description"
	imageURL := "https://example.com/image.png"
	videoURL := "https://example.com/video.mp4"
	siteName := "Example"
	width := 1920
	height := 1080

	preview := &LinkPreview{
		ID:          id,
		URL:         "https://example.com/page",
		Title:       &title,
		Description: &description,
		ImageURL:    &imageURL,
		VideoURL:    &videoURL,
		SiteName:    &siteName,
		Type:        "website",
		Width:       &width,
		Height:      &height,
		ExpiresAt:   &expiresAt,
	}

	resp := preview.ToResponse()

	if resp.ID != id {
		t.Errorf("ID = %v; want %v", resp.ID, id)
	}
	if resp.URL != preview.URL {
		t.Errorf("URL = %v; want %v", resp.URL, preview.URL)
	}
	if resp.Title == nil || *resp.Title != title {
		t.Errorf("Title = %v; want %v", resp.Title, &title)
	}
	if resp.Description == nil || *resp.Description != description {
		t.Errorf("Description = %v; want %v", resp.Description, &description)
	}
	if resp.ImageURL == nil || *resp.ImageURL != imageURL {
		t.Errorf("ImageURL = %v; want %v", resp.ImageURL, &imageURL)
	}
	if resp.VideoURL == nil || *resp.VideoURL != videoURL {
		t.Errorf("VideoURL = %v; want %v", resp.VideoURL, &videoURL)
	}
	if resp.SiteName == nil || *resp.SiteName != siteName {
		t.Errorf("SiteName = %v; want %v", resp.SiteName, &siteName)
	}
	if resp.Type != "website" {
		t.Errorf("Type = %q; want %q", resp.Type, "website")
	}
	if resp.Width == nil || *resp.Width != width {
		t.Errorf("Width = %v; want %v", resp.Width, &width)
	}
	if resp.Height == nil || *resp.Height != height {
		t.Errorf("Height = %v; want %v", resp.Height, &height)
	}
	if resp.ExpiresAt == nil || !resp.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v; want %v", resp.ExpiresAt, &expiresAt)
	}
}

func TestLinkPreviewToResponseNilFields(t *testing.T) {
	preview := &LinkPreview{
		ID:     uuid.New(),
		URL:    "https://example.com/page",
		Type:   "website",
	}

	resp := preview.ToResponse()

	if resp.ID != preview.ID {
		t.Errorf("ID = %v; want %v", resp.ID, preview.ID)
	}
	if resp.URL != preview.URL {
		t.Errorf("URL = %v; want %v", resp.URL, preview.URL)
	}
	if resp.Title != nil {
		t.Errorf("Title = %v; want nil", resp.Title)
	}
	if resp.Description != nil {
		t.Errorf("Description = %v; want nil", resp.Description)
	}
	if resp.ImageURL != nil {
		t.Errorf("ImageURL = %v; want nil", resp.ImageURL)
	}
	if resp.VideoURL != nil {
		t.Errorf("VideoURL = %v; want nil", resp.VideoURL)
	}
	if resp.SiteName != nil {
		t.Errorf("SiteName = %v; want nil", resp.SiteName)
	}
	if resp.Width != nil {
		t.Errorf("Width = %v; want nil", resp.Width)
	}
	if resp.Height != nil {
		t.Errorf("Height = %v; want nil", resp.Height)
	}
	if resp.ExpiresAt != nil {
		t.Errorf("ExpiresAt = %v; want nil", resp.ExpiresAt)
	}
}
