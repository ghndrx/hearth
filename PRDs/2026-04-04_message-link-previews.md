---
name: Message Link Previews & Unfurling
description: Automatic link preview generation with OpenGraph metadata extraction for rich message content
type: competitive
---

# Message Link Previews & Unfurling

## Discord Equivalent
Automatic Link Previews - Rich previews for URLs with images, titles, descriptions, and metadata

## User Value Proposition
Modern messaging requires rich link previews. Users expect shared links to show meaningful previews rather than bare URLs. This is a core messaging feature that significantly impacts user experience and content engagement.

**Key Benefits:**
- Rich content sharing with visual previews
- Automatic metadata extraction from websites
- Image/video thumbnails for media links
- Improved message engagement and readability
- Support for specialized platforms (YouTube, Twitter, GitHub, etc.)

## Technical Complexity Estimate
**P0 - High Priority** (8-10 weeks)

**Complexity Factors:**
- OpenGraph/oEmbed metadata extraction
- Image/video thumbnail generation
- Caching layer for performance
- Privacy controls and domain blocking
- Rate limiting for external requests
- Security considerations (malicious links)

## Implementation Sketch

### Backend Models
```go
type LinkPreview struct {
    ID          uuid.UUID  `json:"id" db:"id"`
    URL         string     `json:"url" db:"url"`
    Title       *string    `json:"title,omitempty" db:"title"`
    Description *string    `json:"description,omitempty" db:"description"`
    ImageURL    *string    `json:"image_url,omitempty" db:"image_url"`
    VideoURL    *string    `json:"video_url,omitempty" db:"video_url"`
    SiteName    *string    `json:"site_name,omitempty" db:"site_name"`
    Type        string     `json:"type" db:"type"` // website, video, image, rich
    Width       *int       `json:"width,omitempty" db:"width"`
    Height      *int       `json:"height,omitempty" db:"height"`
    ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type MessageLinkPreview struct {
    MessageID     uuid.UUID `json:"message_id" db:"message_id"`
    LinkPreviewID uuid.UUID `json:"link_preview_id" db:"link_preview_id"`
}
```

### Core Services
- `LinkPreviewService` - Extract and cache metadata
- `UnfurlService` - Parse URLs from message content
- `ImageProxyService` - Proxy external images for security
- `DomainFilterService` - Block malicious/unwanted domains

### API Integration
- OpenGraph metadata extraction
- oEmbed support for major platforms
- YouTube, Twitter, GitHub specific handlers
- Image/video thumbnail generation
- CDN integration for cached previews

### Frontend Components
- `LinkPreviewCard.svelte` - Rich preview display
- `PreviewSettings.svelte` - User privacy controls
- Enhanced `MessageContent.svelte` with preview support
- `LinkPreviewSkeleton.svelte` - Loading state

### Privacy & Security
- User setting to disable link previews
- Domain allowlist/blocklist management
- Rate limiting for unfurl requests
- Image proxy to prevent IP leakage
- Malicious link detection and warnings

## Dependencies
- Image CDN/proxy infrastructure
- Redis caching for preview metadata
- Background job processing for unfurling
- Content safety scanning for preview images
- Mobile app preview support

## Success Metrics
- Link preview generation rate (% of messages with links that get previews)
- Preview interaction rate (clicks on previews vs raw links)
- Time-to-preview (latency for preview generation)
- User satisfaction with rich content sharing