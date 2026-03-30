# Granular Notification Controls

**Feature:** Granular Notification Controls
**Discord Equivalent:** Per-channel notification settings and advanced controls
**Priority:** P1 (Critical for user experience at scale)
**Estimated Complexity:** Medium (10-12 weeks)
**Created:** 2026-03-30

## Overview

Advanced notification system that allows users to customize notification behavior per channel, server, and content type. Essential for managing notification fatigue in active communities while ensuring important messages aren't missed.

## User Value Proposition

- **Notification Control:** Fine-tune which messages trigger notifications
- **Focus Management:** Reduce interruptions during work/focus time
- **Priority Filtering:** Ensure important messages (mentions, DMs) break through
- **Context Awareness:** Different notification rules for different servers/channels

## Discord Equivalent Features

- Per-channel notification overrides (All Messages, Only Mentions, Nothing)
- Server notification settings with channel inheritance
- Keyword notifications (custom words that trigger alerts)
- Notification schedules (quiet hours)
- Mobile vs desktop notification preferences
- Push notification batching and grouping

## Technical Implementation Sketch

### Backend Changes
```go
type NotificationSettings struct {
    UserID      uuid.UUID `json:"user_id" db:"user_id"`
    ServerID    *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
    ChannelID   *uuid.UUID `json:"channel_id,omitempty" db:"channel_id"`

    // Notification levels
    AllMessages    bool `json:"all_messages" db:"all_messages"`
    OnlyMentions   bool `json:"only_mentions" db:"only_mentions"`
    Nothing        bool `json:"nothing" db:"nothing"`

    // Advanced settings
    Keywords       []string `json:"keywords" db:"keywords"`
    MuteUntil      *time.Time `json:"mute_until,omitempty" db:"mute_until"`
    QuietHoursStart *string `json:"quiet_hours_start,omitempty" db:"quiet_hours_start"`
    QuietHoursEnd   *string `json:"quiet_hours_end,omitempty" db:"quiet_hours_end"`

    // Platform preferences
    MobileEnabled  bool `json:"mobile_enabled" db:"mobile_enabled"`
    DesktopEnabled bool `json:"desktop_enabled" db:"desktop_enabled"`
    EmailEnabled   bool `json:"email_enabled" db:"email_enabled"`

    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type NotificationKeyword struct {
    ID       uuid.UUID `json:"id" db:"id"`
    UserID   uuid.UUID `json:"user_id" db:"user_id"`
    Keyword  string    `json:"keyword" db:"keyword"`
    CaseSensitive bool `json:"case_sensitive" db:"case_sensitive"`
    WholeWord     bool `json:"whole_word" db:"whole_word"`
}
```

### Frontend Components
- NotificationSettingsModal with tabbed interface
- Per-channel settings override in channel context menu
- Server-wide notification configuration
- Keyword management interface
- Quiet hours time picker
- Notification preview and testing

### API Endpoints
- `GET/PUT /api/users/notification-settings` - Global settings
- `GET/PUT /api/servers/{id}/notification-settings` - Server overrides
- `GET/PUT /api/channels/{id}/notification-settings` - Channel overrides
- `POST /api/users/notification-keywords` - Keyword management
- `POST /api/notifications/test` - Test notification delivery

## Dependencies

- Enhanced notification delivery system
- Keyword matching algorithm (fuzzy search integration)
- Time zone handling for quiet hours
- Push notification service updates

## Success Metrics

- Reduction in notification opt-outs
- Increase in notification engagement rate
- User retention in high-activity servers
- Decreased support tickets about notifications

## Implementation Phases

1. **Phase 1:** Basic per-channel override system
2. **Phase 2:** Keyword notifications and filtering
3. **Phase 3:** Quiet hours and scheduling
4. **Phase 4:** Advanced batching and grouping