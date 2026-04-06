package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrEmbedNotFound    = errors.New("embed not found")
	ErrInvalidEmbedData = errors.New("invalid embed data")
)

// EmbedRepositoryInterface defines methods needed from EmbedRepository
type EmbedRepositoryInterface interface {
	CreateTemplate(ctx context.Context, template *models.EmbedTemplate) error
	GetTemplateByID(ctx context.Context, id uuid.UUID) (*models.EmbedTemplate, error)
	GetTemplatesByUserID(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error)
	UpdateTemplate(ctx context.Context, template *models.EmbedTemplate) error
	DeleteTemplate(ctx context.Context, id, userID uuid.UUID) error
}

// EmbedService handles embed operations
type EmbedService struct {
	repo       EmbedRepositoryInterface
	httpClient *http.Client
}

// NewEmbedService creates a new embed service
func NewEmbedService(repo EmbedRepositoryInterface) *EmbedService {
	return &EmbedService{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
	}
}

// CreateTemplate creates a new embed template
func (s *EmbedService) CreateTemplate(ctx context.Context, userID uuid.UUID, req *models.CreateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
	if req.Name == "" {
		return nil, ErrInvalidEmbedData
	}

	template := &models.EmbedTemplate{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Title:       req.Title,
		Description: req.Description,
		URL:         req.URL,
		Color:       req.Color,
		AuthorName:  req.AuthorName,
		AuthorURL:   req.AuthorURL,
		AuthorIcon:  req.AuthorIcon,
		FooterText:  req.FooterText,
		FooterIcon:  req.FooterIcon,
		ImageURL:    req.ImageURL,
		ThumbnailURL: req.ThumbnailURL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// GetTemplates retrieves all embed templates for a user
func (s *EmbedService) GetTemplates(ctx context.Context, userID uuid.UUID) ([]models.EmbedTemplate, error) {
	templates, err := s.repo.GetTemplatesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}
	if templates == nil {
		return []models.EmbedTemplate{}, nil
	}
	return templates, nil
}

// GetTemplate retrieves a specific template
func (s *EmbedService) GetTemplate(ctx context.Context, userID, templateID uuid.UUID) (*models.EmbedTemplate, error) {
	template, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	if template == nil {
		return nil, ErrTemplateNotFound
	}
	if template.UserID != userID {
		return nil, ErrTemplateNotFound
	}
	return template, nil
}

// UpdateTemplate updates an embed template
func (s *EmbedService) UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req *models.UpdateEmbedTemplateRequest) (*models.EmbedTemplate, error) {
	template, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	if template == nil {
		return nil, ErrTemplateNotFound
	}
	if template.UserID != userID {
		return nil, ErrTemplateNotFound
	}

	// Update fields if provided
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Title != nil {
		template.Title = req.Title
	}
	if req.Description != nil {
		template.Description = req.Description
	}
	if req.URL != nil {
		template.URL = req.URL
	}
	if req.Color != nil {
		template.Color = req.Color
	}
	if req.AuthorName != nil {
		template.AuthorName = req.AuthorName
	}
	if req.AuthorURL != nil {
		template.AuthorURL = req.AuthorURL
	}
	if req.AuthorIcon != nil {
		template.AuthorIcon = req.AuthorIcon
	}
	if req.FooterText != nil {
		template.FooterText = req.FooterText
	}
	if req.FooterIcon != nil {
		template.FooterIcon = req.FooterIcon
	}
	if req.ImageURL != nil {
		template.ImageURL = req.ImageURL
	}
	if req.ThumbnailURL != nil {
		template.ThumbnailURL = req.ThumbnailURL
	}
	template.UpdatedAt = time.Now()

	if err := s.repo.UpdateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return template, nil
}

// DeleteTemplate deletes an embed template
func (s *EmbedService) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	if err := s.repo.DeleteTemplate(ctx, templateID, userID); err != nil {
		return ErrTemplateNotFound
	}
	return nil
}

// FetchURLMetadata fetches OpenGraph/metadata for a URL and returns a link preview response
func (s *EmbedService) FetchURLMetadata(ctx context.Context, rawURL string) (*models.LinkPreviewResponse, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, ErrInvalidURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, ErrInvalidURL
	}
	req.Header.Set("User-Agent", "HearthEmbedPreview/1.0 (+https://hearth.chat)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, ErrUnreachableURL
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, ErrUnreachableURL
	}

	// Limit body read
	limited := io.LimitReader(resp.Body, 5*1024*1024) // 5MB
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, ErrUnreachableURL
	}

	htmlContent := string(body)

	preview := &models.LinkPreviewResponse{
		ID:    uuid.New(),
		URL:   rawURL,
		Type:  "website",
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "video") {
		preview.Type = "video"
	} else if strings.Contains(contentType, "audio") {
		preview.Type = "audio"
	} else if strings.Contains(contentType, "image") {
		preview.Type = "image"
	}

	// Extract title from <title> tag as fallback
	if preview.Title == nil {
		if titleMatch := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`).FindStringSubmatch(htmlContent); len(titleMatch) > 1 {
			title := titleMatch[1]
			preview.Title = &title
		}
	}

	return preview, nil
}

// TemplateToEmbedData converts a template to EmbedData for the frontend
func (s *EmbedService) TemplateToEmbedData(template *models.EmbedTemplate) map[string]interface{} {
	data := make(map[string]interface{})

	if template.Title != nil {
		data["title"] = *template.Title
	}
	if template.Description != nil {
		data["description"] = *template.Description
	}
	if template.URL != nil {
		data["url"] = *template.URL
	}
	if template.Color != nil {
		data["color"] = *template.Color
	}
	if template.AuthorName != nil || template.AuthorURL != nil || template.AuthorIcon != nil {
		author := make(map[string]interface{})
		if template.AuthorName != nil {
			author["name"] = *template.AuthorName
		}
		if template.AuthorURL != nil {
			author["url"] = *template.AuthorURL
		}
		if template.AuthorIcon != nil {
			author["icon"] = *template.AuthorIcon
		}
		data["author"] = author
	}
	if template.FooterText != nil || template.FooterIcon != nil {
		footer := make(map[string]interface{})
		if template.FooterText != nil {
			footer["text"] = *template.FooterText
		}
		if template.FooterIcon != nil {
			footer["icon"] = *template.FooterIcon
		}
		data["footer"] = footer
	}
	if template.ImageURL != nil {
		data["image_url"] = *template.ImageURL
	}
	if template.ThumbnailURL != nil {
		data["thumbnail_url"] = *template.ThumbnailURL
	}

	return data
}
