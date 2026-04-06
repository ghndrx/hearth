---
name: Temporary Bans
description: Auto-expiring server bans with duration-based moderation actions
type: Feature PRD
priority: P0
---

# Temporary Bans

## Discord Equivalent
Direct 1:1 match with Discord's Temporary Bans:
- Server bans with automatic expiry after specified duration
- Moderation actions with time limits (1 hour to permanent)
- Automatic unban notifications and restoration
- Essential moderation tool for proportional punishment

## User Value Proposition
**Critical moderation gap** - Temporary bans provide:
- **Proportional Punishment**: Escalation from timeout → temp ban → permanent ban
- **Reduced Admin Burden**: No manual tracking of when to unban users
- **Rehabilitation Focus**: Users can return after cooling off period
- **Moderation Consistency**: Standardized ban durations across moderators
- **Community Safety**: Remove disruptive users temporarily without permanent exclusion

## Technical Complexity: P0 (High Impact, Low-Medium Complexity)

### Current Implementation Gap
```go
// CURRENT: Ban model lacks expiry functionality
type Ban struct {
    ID       string    `json:"id" db:"id"`
    ServerID string    `json:"server_id" db:"server_id"`
    UserID   string    `json:"user_id" db:"user_id"`
    Reason   *string   `json:"reason" db:"reason"`
    CreatedBy string   `json:"created_by" db:"created_by"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    // MISSING: ExpiresAt, AutoRemove, Duration fields
}

// NEEDED: Enhanced Ban model with expiry
type Ban struct {
    ID        string     `json:"id" db:"id"`
    ServerID  string     `json:"server_id" db:"server_id"`
    UserID    string     `json:"user_id" db:"user_id"`
    Reason    *string    `json:"reason" db:"reason"`
    CreatedBy string     `json:"created_by" db:"created_by"`
    CreatedAt time.Time  `json:"created_at" db:"created_at"`

    // NEW FIELDS for temporary bans
    ExpiresAt    *time.Time `json:"expires_at" db:"expires_at"`
    Duration     *int       `json:"duration" db:"duration"` // seconds
    IsTemporary  bool       `json:"is_temporary" db:"is_temporary"`
    AutoUnbanned bool       `json:"auto_unbanned" db:"auto_unbanned"`
}
```

### Implementation Sketch
```go
// Enhanced Ban Request
type CreateBanRequest struct {
    UserID         string  `json:"user_id"`
    Reason         *string `json:"reason"`
    DeleteMessages int     `json:"delete_message_days"` // 0-7 days
    Duration       *int    `json:"duration_seconds"`    // NEW: null = permanent
}

// Ban Worker for automatic unbanning
type BanExpiryWorker struct {
    repo     database.BanRepository
    events   pubsub.Publisher
    interval time.Duration
}

func (w *BanExpiryWorker) ProcessExpiredBans() error {
    expiredBans, err := w.repo.GetExpiredBans()
    if err != nil {
        return err
    }

    for _, ban := range expiredBans {
        // Remove ban from database
        if err := w.repo.RemoveBan(ban.ServerID, ban.UserID); err != nil {
            continue
        }

        // Mark as auto-unbanned for audit trail
        ban.AutoUnbanned = true
        w.repo.UpdateBan(ban)

        // Send unban notification
        w.events.Publish("ban.expired", BanExpiredEvent{
            ServerID: ban.ServerID,
            UserID:   ban.UserID,
            BanID:    ban.ID,
        })
    }

    return nil
}
```

### Key Features
1. **Duration-Based Bans**
   - Standard duration presets: 1h, 6h, 12h, 1d, 3d, 7d, 30d, permanent
   - Custom duration input with maximum limits
   - Clear expiry time display in moderation logs
   - Timezone-aware expiry calculations

2. **Automatic Unban System**
   - Background worker checking for expired bans (every 5 minutes)
   - Automatic removal from ban list on expiry
   - System-generated unban audit log entries
   - Notification to user when ban expires

3. **Moderation Interface**
   - Duration selector in ban modal
   - Visual indicators for temporary vs permanent bans
   - Countdown timers showing remaining ban duration
   - Bulk ban management with expiry filtering

4. **Integration with Existing Systems**
   - Compatible with current ban reason and message deletion
   - Audit log integration showing ban duration and expiry
   - Permission system respects existing "Ban Members" permission
   - WebSocket events for real-time ban status updates

### Database Schema Changes
```sql
-- Enhance existing bans table
ALTER TABLE bans ADD COLUMN expires_at TIMESTAMPTZ NULL;
ALTER TABLE bans ADD COLUMN duration_seconds INTEGER NULL;
ALTER TABLE bans ADD COLUMN is_temporary BOOLEAN DEFAULT FALSE;
ALTER TABLE bans ADD COLUMN auto_unbanned BOOLEAN DEFAULT FALSE;

-- Index for efficient expired ban queries
CREATE INDEX idx_bans_expires_at ON bans(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_bans_temporary ON bans(is_temporary, expires_at);

-- Update existing permanent bans
UPDATE bans SET is_temporary = FALSE WHERE expires_at IS NULL;
UPDATE bans SET is_temporary = TRUE WHERE expires_at IS NOT NULL;
```

### API Endpoints
- `POST /servers/{id}/bans/{user_id}` - Enhanced with duration parameter
- `GET /servers/{id}/bans` - Enhanced with expiry information
- `DELETE /servers/{id}/bans/{user_id}` - Manual unban (works for temp bans too)
- `GET /servers/{id}/bans/expired` - List recently expired bans
- `PATCH /servers/{id}/bans/{user_id}` - Modify ban duration

### Background Processing
```go
// Cron job configuration
type BanExpiryScheduler struct {
    worker   *BanExpiryWorker
    ticker   *time.Ticker
    stopChan chan struct{}
}

func (s *BanExpiryScheduler) Start() {
    s.ticker = time.NewTicker(5 * time.Minute) // Check every 5 minutes

    go func() {
        for {
            select {
            case <-s.ticker.C:
                s.worker.ProcessExpiredBans()
            case <-s.stopChan:
                return
            }
        }
    }()
}
```

## Dependencies
1. **Enhanced Ban Repository** - Database queries for expired bans
2. **Background Job System** - Scheduled worker for processing expiry
3. **Audit Log Updates** - Track automatic unbans in moderation logs
4. **UI Component Updates** - Ban duration selector and countdown displays
5. **WebSocket Events** - Real-time ban status notifications

## Success Metrics
- Temporary ban usage rate (% of total bans that are temporary)
- Average temporary ban duration chosen by moderators
- Repeat offense rate after temporary bans expire
- Moderation action escalation patterns (timeout → temp ban → perm ban)
- Community satisfaction with proportional punishment

## Implementation Phases

### Phase 1: Backend Infrastructure (2 weeks)
1. Database schema migration for ban expiry
2. Enhanced ban repository with expiry queries
3. Background worker for processing expired bans
4. API endpoint updates for duration parameter

### Phase 2: UI Integration (1.5 weeks)
1. Ban duration selector in moderation modal
2. Temporary ban indicators in ban list
3. Countdown timers for remaining duration
4. Audit log display of ban expiry events

### Phase 3: Advanced Features (1 week)
1. Bulk ban management with duration filters
2. Ban expiry notification system
3. Performance optimization for large servers
4. Admin dashboard for temporary ban analytics

**Total: 4.5 weeks for complete implementation**

## Risk Assessment
- **Low**: Database changes are additive and backward compatible
- **Low**: Background processing is lightweight and failure-tolerant
- **Low**: Feature builds on existing ban system without major architectural changes

## Business Impact
Temporary bans are essential for:
- **Community Health** - More nuanced moderation reduces permanent user loss
- **Moderator Efficiency** - Automated unban reduces administrative overhead
- **User Experience** - Proportional punishment feels more fair than permanent bans
- **Competitive Parity** - Standard feature expected in modern Discord alternatives

## Current Workarounds
Moderators currently must:
- Manually track ban expiry times in external calendars/spreadsheets
- Remember to manually unban users after cooling off periods
- Risk permanent bans for minor infractions due to lack of automation
- Spend unnecessary time on ban management instead of community building