---
feature: Native Mobile Push Notification System
discord_equivalent: Mobile App → Notifications (complete iOS/Android native push)
priority: P0
complexity: High
estimated_effort: 6-8 weeks
---

# Native Mobile Push Notification System

## Overview

Mobile push notifications are the #1 driver of user engagement and retention for social platforms. Discord's mobile engagement dominates because of intelligent, timely push notifications. Without native mobile push, Hearth cannot compete for mobile-first users or achieve sustainable daily/weekly active user growth. This is a **critical user retention and growth blocker**.

## Discord Feature Parity

### Push Notification Types
- **Direct Messages**: Immediate push for all DMs with sender name/message preview
- **Mentions**: @username mentions in channels with channel/server context
- **Replies**: Thread replies and message replies with context
- **Server Events**: Friend requests, server invites, role pings
- **Live Events**: Go Live notifications, stage channel announcements
- **Custom Keywords**: User-defined keyword alerts across servers

### Smart Delivery Features
- **Intelligent Batching**: Group related notifications to reduce spam
- **Priority Scoring**: VIP users, important servers, urgent keywords get immediate delivery
- **Quiet Hours**: User-configurable do not disturb times
- **Granular Control**: Per-server, per-channel notification settings
- **Platform Sync**: Dismiss on one device removes from all devices

## User Value Proposition

- **User Retention**: 3x higher 7-day retention with push notifications enabled
- **Engagement**: 40% higher daily active usage with smart notifications
- **Mobile-First Users**: Critical for younger demographics who expect instant mobile responses
- **Community Building**: Real-time notifications drive conversation momentum
- **Business Communication**: Professional teams need reliable mobile alerts

## Technical Implementation

### Mobile Platform Integration
```typescript
// iOS (Swift/React Native)
import UserNotifications

class HearthNotificationService: UNNotificationServiceExtension {
    func didReceive(request: UNNotificationRequest,
                   withContentHandler: (UNNotificationContent) -> Void) {
        // Rich notification processing
        let content = request.content.mutableCopy()
        content.title = enrichTitle(from: request.userInfo)
        content.body = enrichBody(from: request.userInfo)
        content.sound = customSound(for: request.userInfo["type"])
        withContentHandler(content)
    }
}

// Android (Kotlin/React Native)
class HearthFirebaseService : FirebaseMessagingService() {
    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        val notification = buildRichNotification(remoteMessage.data)
        NotificationManagerCompat.from(this).notify(
            generateNotificationId(),
            notification
        )
    }
}
```

### Backend Push Infrastructure
```go
// Push notification delivery system
type PushDeliveryService struct {
    APNSClient     *apns.Client      // iOS delivery
    FCMClient      *fcm.Client       // Android delivery
    WebPushClient  *webpush.Client   // Browser delivery
    Templates      *TemplateEngine   // Notification templates
    Analytics      *AnalyticsTracker // Delivery tracking
}

type PushNotification struct {
    UserID          uuid.UUID            `json:"user_id"`
    Type            NotificationType     `json:"type"`
    Priority        Priority             `json:"priority"`
    Title           string               `json:"title"`
    Body            string               `json:"body"`
    Sound           string               `json:"sound,omitempty"`
    Badge           int                  `json:"badge,omitempty"`
    Data            map[string]string    `json:"data"`
    ScheduledFor    *time.Time           `json:"scheduled_for,omitempty"`
    ExpiresAt       time.Time            `json:"expires_at"`
    Platforms       []Platform           `json:"platforms"` // iOS, Android, Web
}

// Smart delivery logic
type DeliveryStrategy struct {
    ImmediateTypes  []NotificationType // DMs, mentions - always immediate
    BatchedTypes    []NotificationType // Reactions, server events - can batch
    QuietHours      QuietHoursConfig   // Per-user do not disturb
    FrequencyCap    FrequencyConfig    // Max notifications per hour/day
    UserPreferences UserNotificationSettings
}
```

### Intelligent Notification Scoring
```go
type NotificationScorer struct {
    RelationshipWeight map[RelationshipType]float64 // Friend=1.0, Server member=0.3
    ChannelImportance  map[uuid.UUID]float64        // Per-channel user importance
    KeywordMatches     []string                      // User-defined keywords
    TimeDecayFactor    float64                       // Recent activity boost
    EngagementHistory  UserEngagementProfile         // Historical interaction patterns
}

func (s *NotificationScorer) CalculatePriority(notification *PushNotification, user *User) Priority {
    score := 0.0

    // Relationship scoring
    score += s.RelationshipWeight[notification.SenderRelationship] * 0.4

    // Channel importance
    score += s.ChannelImportance[notification.ChannelID] * 0.3

    // Keyword matching
    if s.MatchesKeywords(notification.Content) {
        score += 0.8
    }

    // Time-based relevance
    score *= s.TimeDecayFactor

    // Convert score to priority
    switch {
    case score >= 0.8: return PriorityUrgent
    case score >= 0.6: return PriorityHigh
    case score >= 0.4: return PriorityNormal
    default: return PriorityLow
    }
}
```

### User Notification Preferences
```go
type UserNotificationSettings struct {
    UserID            uuid.UUID                 `json:"user_id"`
    GlobalEnabled     bool                      `json:"global_enabled"`
    QuietHours        QuietHoursConfig          `json:"quiet_hours"`
    MaxPerHour        int                       `json:"max_per_hour"`
    Keywords          []string                  `json:"keywords"`
    ServerSettings    map[uuid.UUID]ServerPushSettings `json:"server_settings"`
    ChannelSettings   map[uuid.UUID]ChannelPushSettings `json:"channel_settings"`
    TypeSettings      map[NotificationType]bool `json:"type_settings"`
}

type QuietHoursConfig struct {
    Enabled     bool      `json:"enabled"`
    StartTime   TimeOfDay `json:"start_time"`   // 22:00
    EndTime     TimeOfDay `json:"end_time"`     // 08:00
    Timezone    string    `json:"timezone"`
    AllowUrgent bool      `json:"allow_urgent"` // Emergency overrides
}
```

## Dependencies

- **Mobile App Infrastructure**: iOS and Android native app development
- **Push Provider Setup**: APNs (iOS), FCM (Android), Web Push API
- **Certificate Management**: Apple Developer certificates, FCM service keys
- **Analytics Platform**: Delivery tracking, open rates, engagement metrics
- **Template System**: Rich notification content generation

## Success Metrics

- **Push Opt-in Rate**: >70% of mobile users enable push notifications
- **Engagement Lift**: 40% increase in DAU with push notifications enabled
- **Delivery Success**: >95% successful push delivery rate
- **Response Rate**: >25% notification click-through rate
- **Retention Impact**: 50% improvement in 7-day retention for push-enabled users
- **Complaint Rate**: <1% users disable notifications due to spam

## Implementation Phases

### Phase 1 (3 weeks): Core Push Infrastructure
- APNs and FCM client integration
- Basic notification types (DMs, mentions, replies)
- Simple delivery without intelligence
- Basic user preferences (on/off per type)

### Phase 2 (3 weeks): Smart Delivery & Preferences
- Priority scoring algorithm
- Quiet hours and frequency capping
- Per-server/channel notification settings
- Notification batching for non-urgent types

### Phase 3 (2 weeks): Analytics & Optimization
- Delivery analytics dashboard
- A/B testing for notification content
- Engagement tracking and optimization
- Rich notification templates with images/actions

This feature directly addresses the mobile engagement gap and is essential for competing with Discord's mobile-first user experience. Without native push notifications, Hearth cannot achieve sustainable user growth or retention in the mobile era.