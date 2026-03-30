---
name: Granular Notification Controls
description: Advanced per-channel notification settings, keyword alerts, quiet hours, and notification scheduling
type: feature
priority: P0
---

# Granular Notification Controls

## Discord Equivalent
Discord's comprehensive notification system including:
- Per-channel notification overrides (All Messages, @mentions only, Nothing)
- Keyword notifications (get pinged on specific words)
- Quiet Hours (Do Not Disturb scheduling)
- Mobile vs Desktop notification preferences
- @everyone/@here suppress options per server

## User Value Proposition
- **User Experience**: Prevents notification fatigue while ensuring important messages aren't missed
- **Retention**: Users stay engaged longer when notifications are relevant, not overwhelming
- **Professional Use**: Makes Hearth viable for work environments with quiet hours
- **Accessibility**: Custom keywords help users with different communication needs
- **Platform Growth**: Essential feature parity for Discord users considering migration

## Technical Complexity: P0 (High)

### Core Features

#### 1. Per-Channel Notification Overrides
- **All Messages**: Every message triggers notification
- **@mentions Only**: Only mentions of user/roles trigger notifications
- **Nothing**: Channel is completely muted
- **Inherit**: Use server-level default setting
- Server-level defaults with per-channel overrides

#### 2. Keyword Notifications
- Custom keyword lists per user (global + per-server)
- Case-insensitive matching with word boundaries
- Exclude channels/servers from keyword monitoring
- Rate limiting to prevent spam (max 1/minute per keyword)

#### 3. Quiet Hours & Scheduling
- Daily quiet hours (e.g., 11 PM - 8 AM)
- Different schedules for weekdays vs weekends
- Platform-specific rules (mobile vs desktop)
- Emergency override for critical mentions

#### 4. Notification Routing
- Separate mobile/desktop notification preferences
- Email digest options for missed mentions
- Push notification batching during active usage
- Smart suppression (don't notify if user is active in channel)

### Implementation Sketch
```
/notifications/
├── engine/
│   ├── matcher.go        # Keyword/mention detection
│   ├── scheduler.go      # Quiet hours & timing
│   ├── router.go         # Platform routing (mobile/desktop/email)
│   └── suppression.go    # Smart suppression logic
├── storage/
│   ├── user_settings.go  # Per-user preferences
│   ├── channel_overrides.go # Per-channel settings
│   └── keywords.go       # Keyword management
├── delivery/
│   ├── push.go          # Mobile push notifications
│   ├── websocket.go     # Real-time desktop notifications
│   └── email.go         # Email digest system
└── frontend/
    ├── NotificationSettings.svelte
    ├── ChannelNotificationMenu.svelte
    ├── KeywordManager.svelte
    └── QuietHoursScheduler.svelte
```

### Database Schema Updates
```sql
-- Extend user_channel_settings table
ALTER TABLE user_channel_settings ADD COLUMN notification_level INTEGER DEFAULT 0;
-- 0: inherit, 1: all messages, 2: mentions only, 3: nothing

-- User notification preferences
CREATE TABLE user_notification_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    weekday_schedule BOOLEAN DEFAULT TRUE,
    weekend_schedule BOOLEAN DEFAULT TRUE,
    mobile_notifications BOOLEAN DEFAULT TRUE,
    desktop_notifications BOOLEAN DEFAULT TRUE,
    email_digest BOOLEAN DEFAULT FALSE,
    suppress_when_active BOOLEAN DEFAULT TRUE
);

-- Keywords table
CREATE TABLE notification_keywords (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    server_id UUID REFERENCES servers(id), -- NULL for global keywords
    keyword VARCHAR(50) NOT NULL,
    case_sensitive BOOLEAN DEFAULT FALSE,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Key Technical Challenges
- **Performance**: Keyword matching across thousands of messages/second
- **Mobile Battery**: Efficient push notification batching
- **Timezone Handling**: Quiet hours across different user timezones
- **Real-time Updates**: Settings changes reflected immediately
- **Spam Prevention**: Rate limiting without missing legitimate notifications

## Dependencies
- Existing notification system (✓ basic version exists)
- Push notification infrastructure (needs enhancement)
- User channel settings table (✓ exists, needs columns)
- Timezone handling in backend
- Mobile app notification permission management

## Success Metrics
- **Engagement**: Notification open rate >25% (vs <15% without controls)
- **Retention**: Reduce notification-related churn by 30%
- **User Satisfaction**: >80% of users customize at least one notification setting
- **Support Reduction**: Fewer "too many notifications" support tickets

## Migration Strategy
1. **Phase 1**: Per-channel notification overrides (basic)
2. **Phase 2**: Quiet hours and scheduling
3. **Phase 3**: Keyword notifications
4. **Phase 4**: Advanced routing and batching

## Competitive Analysis
- **Discord**: Industry standard, very comprehensive
- **Slack**: Enterprise-focused, excellent keyword system
- **Telegram**: Strong quiet hours, basic channel controls
- **Hearth Opportunity**: AI-powered smart notification priority ranking