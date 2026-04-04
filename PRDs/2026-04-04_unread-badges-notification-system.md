---
name: Unread Badges & Notification System
description: Real-time unread message badges and notification indicators for channels, servers, and DMs
type: competitive
---

# Unread Badges & Notification System

## Discord Equivalent
Unread Message Badges - Red badges with unread counts, white dots for mentions, notification indicators

## User Value Proposition
Visual notification indicators are essential for message management. Users need immediate visual feedback about new messages, mentions, and activity. Without proper badge systems, users miss important conversations and lose engagement.

**Key Benefits:**
- Clear visual indicators for unread messages
- Mention-specific highlighting (red badges vs white dots)
- Real-time badge updates across all devices
- Server-level and channel-level unread aggregation
- DM notification badges for direct communications
- Customizable notification priorities

## Technical Complexity Estimate
**P0 - High Priority** (6-8 weeks)

**Complexity Factors:**
- Real-time badge sync across devices
- Efficient unread count aggregation
- WebSocket event optimization
- Mobile push notification integration
- Performance for users in many servers
- Read state consistency

## Implementation Sketch

### Backend Models
```go
type ChannelUnreadState struct {
    UserID           uuid.UUID  `json:"user_id" db:"user_id"`
    ChannelID        uuid.UUID  `json:"channel_id" db:"channel_id"`
    LastReadMessageID *uuid.UUID `json:"last_read_message_id" db:"last_read_message_id"`
    UnreadCount      int        `json:"unread_count" db:"unread_count"`
    MentionCount     int        `json:"mention_count" db:"mention_count"`
    LastMessageAt    *time.Time `json:"last_message_at" db:"last_message_at"`
    UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type ServerUnreadSummary struct {
    UserID       uuid.UUID `json:"user_id" db:"user_id"`
    ServerID     uuid.UUID `json:"server_id" db:"server_id"`
    UnreadCount  int       `json:"unread_count" db:"unread_count"`
    MentionCount int       `json:"mention_count" db:"mention_count"`
    HasUnread    bool      `json:"has_unread" db:"has_unread"`
}

type NotificationBadge struct {
    Type         string `json:"type"` // unread, mention, activity
    Count        int    `json:"count"`
    ShowCount    bool   `json:"show_count"`
    Priority     int    `json:"priority"` // 0=none, 1=low, 2=normal, 3=high
}
```

### Core Services
- `UnreadService` - Manage unread state and counts
- `BadgeService` - Calculate and distribute badge states
- `NotificationAggregatorService` - Roll up server/DM badges
- `ReadStateService` - Track read positions

### Real-Time Updates
- WebSocket events for badge changes
- Efficient diff-based badge updates
- Batched badge calculations for performance
- Mobile push badge count sync

### Frontend Components
- `UnreadBadge.svelte` - Notification badge display
- `ServerBadge.svelte` - Server-level unread indicator
- `ChannelBadge.svelte` - Channel unread count
- `DMBadge.svelte` - Direct message indicators
- Enhanced sidebar with badge integration

### Badge Logic
- **White Dot**: New messages, no mention
- **Red Badge with Count**: Direct mentions (@user)
- **Red Badge (no count)**: Role mentions, everyone/here
- **Server Badge**: Aggregated from all channels
- **DM Badge**: Unread direct/group messages

### Performance Optimizations
- Redis-cached badge states
- Lazy badge calculation for inactive users
- Efficient database aggregation queries
- WebSocket event batching
- Background badge reconciliation

## Dependencies
- Enhanced read state tracking system
- Real-time WebSocket optimization
- Mobile push notification service
- Notification preferences integration
- Cross-device state synchronization

## Success Metrics
- Badge accuracy (% of badges showing correct counts)
- Real-time sync latency (time to badge update)
- User engagement with badged content
- Notification interaction rates
- Mobile badge sync reliability