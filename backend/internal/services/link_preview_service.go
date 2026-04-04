package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrLinkPreviewNotFound = errors.New("link preview not found")
	ErrInvalidURL         = errors.New("invalid URL")
	ErrUnreachableURL     = errors.New("URL is unreachable")

	defaultPreviewTTL = 24 * time.Hour
)

// LinkPreviewService extracts and caches OpenGraph metadata for URLs
type LinkPreviewService struct {
	mu          sync.RWMutex
	cache       map[string]*cachedPreview
	httpClient  *http.Client
	maxBodySize int64 // max bytes to read when fetching
}

type cachedPreview struct {
	preview *models.LinkPreview
	expiry  time.Time
}

// NewLinkPreviewService creates a new link preview service
func NewLinkPreviewService() *LinkPreviewService {
	return &LinkPreviewService{
		cache: make(map[string]*cachedPreview),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
		maxBodySize: 5 * 1024 * 1024, // 5MB
	}
}

// GetPreview fetches or retrieves from cache a link preview for a URL
func (s *LinkPreviewService) GetPreview(ctx context.Context, rawURL string) (*models.LinkPreview, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, ErrInvalidURL
	}

	// Check cache
	cacheKey := parsedURL.String()
	s.mu.RLock()
	if cached, ok := s.cache[cacheKey]; ok && time.Now().Before(cached.expiry) {
		s.mu.RUnlock()
		return cached.preview, nil
	}
	s.mu.RUnlock()

	// Fetch and extract
	preview, err := s.extractPreview(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	// Cache it
	s.mu.Lock()
	s.cache[cacheKey] = &cachedPreview{
		preview: preview,
		expiry:  time.Now().Add(defaultPreviewTTL),
	}
	s.mu.Unlock()

	return preview, nil
}

// extractPreview fetches a URL and extracts OpenGraph metadata
func (s *LinkPreviewService) extractPreview(ctx context.Context, rawURL string) (*models.LinkPreview, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, ErrInvalidURL
	}
	req.Header.Set("User-Agent", "HearthLinkPreview/1.0 (+https://hearth.chat)")
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
	limited := io.LimitReader(resp.Body, s.maxBodySize)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, ErrUnreachableURL
	}

	preview := &models.LinkPreview{
		ID:        uuid.New(),
		URL:       rawURL,
		Type:      "website",
		CreatedAt: time.Now(),
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "video") {
		preview.Type = "video"
	} else if strings.Contains(contentType, "audio") {
		preview.Type = "audio"
	} else if strings.Contains(contentType, "image") {
		preview.Type = "image"
	}

	// Extract OpenGraph tags
	s.extractOGTags(string(body), preview)

	// Set expiry
	ttl := defaultPreviewTTL
	exp := time.Now().Add(ttl)
	preview.ExpiresAt = &exp

	return preview, nil
}

var ogTagRegex = regexp.MustCompile(`<meta[^>]+(?:property|name)=["'](?:og:)?([^"']+)["'][^>]+content=["']([^"']+)["']`)
var ogTagRegex2 = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+(?:property|name)=["'](?:og:)?([^"']+)["']`)

func (s *LinkPreviewService) extractOGTags(htmlContent string, preview *models.LinkPreview) {
	// Extract og: tags first (they take precedence)
	for _, re := range []*regexp.Regexp{ogTagRegex, ogTagRegex2} {
		matches := re.FindAllStringSubmatch(htmlContent, -1)
		for _, match := range matches {
			var key, value string
			if re == ogTagRegex {
				key, value = match[1], match[2]
			} else {
				value, key = match[1], match[2]
			}
			value = decodeHTMLEntities(strings.TrimSpace(value))

			switch key {
			case "title", "og:title":
				if preview.Title == nil || *preview.Title == "" {
					preview.Title = &value
				}
			case "description", "og:description":
				preview.Description = &value
			case "image", "og:image":
				preview.ImageURL = &value
			case "video", "og:video":
				preview.VideoURL = &value
			case "site_name", "og:site_name":
				preview.SiteName = &value
			case "type", "og:type":
				preview.Type = value
			case "width", "og:width":
				if w := parseInt(value); w > 0 {
					preview.Width = &w
				}
			case "height", "og:height":
				if h := parseInt(value); h > 0 {
					preview.Height = &h
				}
			}
		}
	}

	// Fallback to <title> tag if og:title wasn't set
	if preview.Title == nil {
		if titleMatch := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`).FindStringSubmatch(htmlContent); len(titleMatch) > 1 {
			title := decodeHTMLEntities(strings.TrimSpace(titleMatch[1]))
			preview.Title = &title
		}
	}
}

func decodeHTMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&apos;", "'")
	// Decode numeric entities
	s = decodeNumEntity(s, `&#(\d+);`)
	s = decodeNumEntity(s, `&#x([0-9a-fA-F]+);`)
	return s
}

func decodeNumEntity(s, pattern string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		var num int
		if strings.HasPrefix(match, "&#x") {
			hex := match[3 : len(match)-1]
			fmt.Sscanf(hex, "%x", &num)
		} else {
			fmt.Sscanf(match[2:len(match)-1], "%d", &num)
		}
		if num > 0 && num < 0x10FFFF {
			return string(rune(num))
		}
		return match
	})
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

// ClearCache invalidates all cached previews
func (s *LinkPreviewService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*cachedPreview)
}

// InvalidateURL removes a specific URL from the cache
func (s *LinkPreviewService) InvalidateURL(rawURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, rawURL)
}

// PurgeExpired removes expired entries from cache
func (s *LinkPreviewService) PurgeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.cache {
		if now.After(v.expiry) {
			delete(s.cache, k)
		}
	}
}

// IsEmptyBody checks if reader has no content (for tests)
func IsEmptyBody(body []byte) bool {
	return len(bytes.TrimSpace(body)) == 0
}
