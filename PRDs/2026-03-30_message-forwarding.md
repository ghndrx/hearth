# Message Forwarding System

**Feature:** Message Forwarding
**Discord Equivalent:** Forward Message functionality
**Priority:** P0 (Critical messaging feature)
**Estimated Complexity:** Medium (8-10 weeks)
**Created:** 2026-03-30

## Overview

Message forwarding allows users to share messages across channels and servers while maintaining attribution and context. This is a core Discord feature that enables content sharing and reduces repetitive reposting.

## User Value Proposition

- **Content Sharing:** Easily share valuable messages across different channels/servers
- **Context Preservation:** Forward with original author attribution and timestamp
- **Cross-Community:** Bridge conversations between different servers
- **Moderation:** Helps staff share important information across mod channels

## Discord Equivalent Features

- Right-click message → Forward
- Select destination channels/servers
- Add optional comment when forwarding
- Maintain original author attribution
- Forward with embeds and attachments intact

## Technical Implementation Sketch

### Backend Changes
```go
// Add forwarded message tracking
type ForwardedMessage struct {
    ID              uuid.UUID `json:"id" db:"id"`
    OriginalMessageID uuid.UUID `json:"original_message_id" db:"original_message_id"`
    ForwardedByID   uuid.UUID `json:"forwarded_by_id" db:"forwarded_by_id"`
    DestinationChannelID uuid.UUID `json:"destination_channel_id" db:"destination_channel_id"`
    Comment         string    `json:"comment" db:"comment"`
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
}
```

### Frontend Components
- ForwardMessageModal component
- Message context menu integration
- Channel/server selector
- Comment input field

### API Endpoints
- `POST /api/messages/{id}/forward` - Forward message
- `GET /api/messages/{id}/forwards` - Get forward history

## Dependencies

- Message permissions system (check user can read original)
- Channel permissions (check user can post to destination)
- Attachment handling (ensure attachments are accessible)

## Success Metrics

- Forward usage rate (% of active users who forward messages)
- Cross-server engagement increase
- Reduction in duplicate posting