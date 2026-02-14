# Hearth — Quotas & Limits

**Version:** 1.0  
**Last Updated:** 2026-02-11

---

## Overview

Hearth supports configurable quotas at multiple levels. Operators can set limits to prevent abuse, or disable them entirely for unlimited usage.

**Key principle:** Every limit can be set to `0` or `-1` for unlimited.

---

## Quota Levels

```
┌─────────────────────────────────────────────────────────────┐
│                      Instance Level                          │
│  (Global defaults for the entire Hearth instance)           │
├─────────────────────────────────────────────────────────────┤
│                      Server Level                            │
│  (Override defaults per server)                              │
├─────────────────────────────────────────────────────────────┤
│                      Role Level                              │
│  (Premium roles can have higher limits)                      │
├─────────────────────────────────────────────────────────────┤
│                      User Level                              │
│  (Individual overrides, highest priority)                    │
└─────────────────────────────────────────────────────────────┘
```

---

## Storage Quotas

### Configuration

```yaml
quotas:
  storage:
    # Global defaults (0 = unlimited)
    enabled: true
    
    # Per-user storage limit
    user_storage_mb: 500        # 500 MB per user, 0 = unlimited
    
    # Per-server storage limit (all uploads in server)
    server_storage_mb: 5000     # 5 GB per server, 0 = unlimited
    
    # Per-file limits
    max_file_size_mb: 25        # 25 MB max file, 0 = unlimited
    max_avatar_size_mb: 8       # 8 MB for avatars
    max_emoji_size_mb: 1        # 1 MB for custom emoji
    
    # File count limits
    max_attachments_per_message: 10
    max_files_per_user: 0       # 0 = unlimited
    
    # Allowed file types (empty = allow all)
    allowed_extensions: []
    
    # Blocked file types (always blocked)
    blocked_extensions:
      - exe
      - bat
      - cmd
      - sh
      - ps1
      - msi
```

### Per-Role Overrides

```yaml
# In server settings or via API
roles:
  - name: "Premium Member"
    quotas:
      user_storage_mb: 2000     # 2 GB for premium
      max_file_size_mb: 100     # 100 MB files

  - name: "Moderator"
    quotas:
      user_storage_mb: 0        # Unlimited
      max_file_size_mb: 0       # Unlimited
```

### Per-User Overrides

```yaml
# Via admin API or dashboard
users:
  - id: "user_abc123"
    quotas:
      user_storage_mb: 10000    # 10 GB for this user
      max_file_size_mb: 500     # 500 MB files
```

---

## Message Quotas

### Rate Limiting

```yaml
quotas:
  messages:
    # Messages per time window
    rate_limit_messages: 5      # 5 messages per window
    rate_limit_window_seconds: 5 # 5 second window
    # Result: 5 messages per 5 seconds (1/sec average)
    
    # Set to 0 to disable rate limiting
    rate_limit_messages: 0
    
    # Slowmode (per-channel, overrides global)
    default_slowmode_seconds: 0  # 0 = no slowmode
    max_slowmode_seconds: 21600  # 6 hours max
```

### Content Limits

```yaml
quotas:
  messages:
    max_message_length: 2000    # Characters, 0 = unlimited
    max_embed_count: 10         # Embeds per message
    max_mentions_per_message: 20 # @mentions, 0 = unlimited
    max_reactions_per_message: 20 # Unique emoji reactions
```

---

## Server Quotas

```yaml
quotas:
  servers:
    # Per-user limits
    max_servers_owned: 10       # Servers a user can own
    max_servers_joined: 100     # Servers a user can be in
    
    # Per-server limits
    max_channels: 500           # Channels per server
    max_roles: 250              # Roles per server
    max_emoji: 50               # Custom emoji (static)
    max_emoji_animated: 50      # Animated emoji
    max_members: 500000         # Members per server
    max_invites: 1000           # Active invites
    max_bans: 100000            # Ban list size
    max_webhooks: 15            # Webhooks per channel
    
    # Set any to 0 for unlimited
    max_channels: 0             # Unlimited channels
```

---

## Voice/Video Quotas

```yaml
quotas:
  voice:
    # Quality limits
    max_bitrate_kbps: 384       # Audio bitrate
    max_video_height: 1080      # Video resolution
    max_screen_share_fps: 30    # Screen share framerate
    
    # Concurrent limits
    max_voice_users_per_channel: 99
    max_video_users_per_channel: 25
    
    # Duration limits (0 = unlimited)
    max_call_duration_minutes: 0
    
    # For SaaS tiers
    voice_enabled: true         # Can be disabled entirely
```

---

## API Quotas

```yaml
quotas:
  api:
    # Rate limits for API consumers
    requests_per_minute: 60
    requests_per_hour: 1000
    
    # Burst allowance
    burst_limit: 10
    
    # WebSocket limits
    max_concurrent_connections: 5  # Per user
    max_guilds_per_connection: 100
    
    # Bot-specific (higher limits)
    bot_requests_per_minute: 120
    bot_burst_limit: 20
```

---

## Implementation

### Quota Service

```go
type QuotaService interface {
    // Check if action is allowed
    CheckStorageQuota(ctx context.Context, userID, serverID uuid.UUID, sizeBytes int64) error
    CheckMessageRate(ctx context.Context, userID, channelID uuid.UUID) error
    CheckServerLimit(ctx context.Context, userID uuid.UUID, limitType string) error
    
    // Get current usage
    GetStorageUsage(ctx context.Context, userID uuid.UUID) (*StorageUsage, error)
    GetServerUsage(ctx context.Context, serverID uuid.UUID) (*ServerUsage, error)
    
    // Get effective limits (considering all levels)
    GetEffectiveLimits(ctx context.Context, userID uuid.UUID, serverID *uuid.UUID) (*EffectiveLimits, error)
    
    // Admin overrides
    SetUserQuota(ctx context.Context, userID uuid.UUID, quotas *UserQuotas) error
    SetRoleQuota(ctx context.Context, roleID uuid.UUID, quotas *RoleQuotas) error
    SetServerQuota(ctx context.Context, serverID uuid.UUID, quotas *ServerQuotas) error
}

type StorageUsage struct {
    UserID      uuid.UUID `json:"user_id"`
    UsedBytes   int64     `json:"used_bytes"`
    LimitBytes  int64     `json:"limit_bytes"`  // -1 = unlimited
    FileCount   int       `json:"file_count"`
    Percentage  float64   `json:"percentage"`   // 0-100, -1 if unlimited
}

type EffectiveLimits struct {
    StorageMB         int64 `json:"storage_mb"`          // -1 = unlimited
    MaxFileSizeMB     int64 `json:"max_file_size_mb"`
    MessageRateLimit  int   `json:"message_rate_limit"`
    SlowmodeSeconds   int   `json:"slowmode_seconds"`
    
    // Source of each limit
    Sources map[string]string `json:"sources"` // field -> "instance"|"server"|"role"|"user"
}
```

### Quota Calculation

```go
func (s *quotaService) GetEffectiveLimits(ctx context.Context, userID uuid.UUID, serverID *uuid.UUID) (*EffectiveLimits, error) {
    // Start with instance defaults
    limits := s.instanceDefaults.Clone()
    sources := make(map[string]string)
    
    for field := range limits {
        sources[field] = "instance"
    }
    
    // Apply server overrides
    if serverID != nil {
        serverQuotas, _ := s.serverRepo.GetQuotas(ctx, *serverID)
        if serverQuotas != nil {
            limits.Merge(serverQuotas)
            // Update sources for changed fields
        }
        
        // Apply role overrides (highest role wins)
        member, _ := s.memberRepo.Get(ctx, userID, *serverID)
        if member != nil {
            roles, _ := s.roleRepo.GetByIDs(ctx, member.Roles)
            sort.Slice(roles, func(i, j int) bool {
                return roles[i].Position > roles[j].Position
            })
            for _, role := range roles {
                if role.Quotas != nil {
                    limits.MergeIfHigher(role.Quotas)
                }
            }
        }
    }
    
    // Apply user overrides (highest priority)
    userQuotas, _ := s.userRepo.GetQuotas(ctx, userID)
    if userQuotas != nil {
        limits.Merge(userQuotas)
        // Update sources
    }
    
    return limits, nil
}
```

### Database Schema

```sql
-- Instance-level quotas (in config, not DB)

-- Server-level quota overrides
CREATE TABLE server_quotas (
    server_id UUID PRIMARY KEY REFERENCES servers(id) ON DELETE CASCADE,
    storage_mb BIGINT,           -- NULL = use default
    max_file_size_mb INT,
    max_channels INT,
    max_roles INT,
    max_emoji INT,
    max_members INT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Role-level quota overrides
CREATE TABLE role_quotas (
    role_id UUID PRIMARY KEY REFERENCES roles(id) ON DELETE CASCADE,
    storage_mb BIGINT,
    max_file_size_mb INT,
    message_rate_limit INT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User-level quota overrides
CREATE TABLE user_quotas (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    storage_mb BIGINT,
    max_file_size_mb INT,
    max_servers_owned INT,
    max_servers_joined INT,
    message_rate_limit INT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Storage usage tracking
CREATE TABLE storage_usage (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    used_bytes BIGINT DEFAULT 0,
    file_count INT DEFAULT 0,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, server_id)
);

CREATE INDEX idx_storage_usage_user ON storage_usage(user_id);
CREATE INDEX idx_storage_usage_server ON storage_usage(server_id);
```

---

## Admin Dashboard

### Quota Management UI

```
┌─────────────────────────────────────────────────────────────┐
│  Server Settings > Quotas                                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Storage Limits                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Per-User Storage    [  500  ] MB  ☑ Unlimited      │   │
│  │  Server Total        [ 5000  ] MB  ☐ Unlimited      │   │
│  │  Max File Size       [   25  ] MB  ☐ Unlimited      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Message Limits                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Rate Limit          [    5  ] per [ 5 ] seconds    │   │
│  │  Default Slowmode    [    0  ] seconds (0 = off)    │   │
│  │  Max Message Length  [ 2000  ] chars  ☐ Unlimited   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Role Overrides                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  ⬤ Admin           Unlimited storage, 100 MB files  │   │
│  │  ⬤ Moderator       2 GB storage, 50 MB files        │   │
│  │  ⬤ Premium         1 GB storage, 25 MB files        │   │
│  │  ⬤ @everyone       500 MB storage, 10 MB files      │   │
│  │  [+ Add Role Override]                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [Save Changes]                                             │
└─────────────────────────────────────────────────────────────┘
```

### Usage Dashboard

```
┌─────────────────────────────────────────────────────────────┐
│  Server Storage Usage                                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Total: 2.4 GB / 5.0 GB                                     │
│  ████████████████░░░░░░░░░░░░░░ 48%                        │
│                                                             │
│  Top Users                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  1. @john        450 MB / 500 MB  ██████████ 90%    │   │
│  │  2. @jane        380 MB / 500 MB  ████████░░ 76%    │   │
│  │  3. @bob         220 MB / 2 GB    ██░░░░░░░░ 11%    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [Export Report]  [Clear Old Files]  [Adjust Quotas]       │
└─────────────────────────────────────────────────────────────┘
```

---

## API Endpoints

```
# Get current usage
GET /api/v1/users/@me/storage
GET /api/v1/servers/{id}/storage

# Get effective limits
GET /api/v1/users/@me/limits
GET /api/v1/servers/{id}/limits

# Admin: Set quotas
PATCH /api/v1/servers/{id}/quotas
PATCH /api/v1/servers/{id}/roles/{roleId}/quotas
PATCH /api/v1/admin/users/{userId}/quotas

# Admin: Instance defaults
GET  /api/v1/admin/quotas
PUT  /api/v1/admin/quotas
```

---

## Error Responses

```json
// Storage quota exceeded
{
  "error": "quota_exceeded",
  "message": "You have exceeded your storage quota",
  "details": {
    "used_mb": 500,
    "limit_mb": 500,
    "file_size_mb": 25,
    "would_be_mb": 525
  },
  "upgrade_url": "/settings/premium"
}

// Rate limited
{
  "error": "rate_limited",
  "message": "You are sending messages too quickly",
  "retry_after": 5,
  "details": {
    "limit": 5,
    "window_seconds": 5,
    "slowmode_seconds": 0
  }
}

// File too large
{
  "error": "file_too_large",
  "message": "File exceeds maximum size",
  "details": {
    "file_size_mb": 30,
    "max_size_mb": 25
  }
}
```

---

## Self-Hosted Recommendations

### Unlimited (Homelab/Personal)
```yaml
quotas:
  storage:
    user_storage_mb: 0          # Unlimited
    server_storage_mb: 0        # Unlimited
    max_file_size_mb: 0         # Unlimited
  messages:
    rate_limit_messages: 0      # No rate limit
  servers:
    max_servers_owned: 0        # Unlimited
    max_channels: 0             # Unlimited
```

### Moderate (Small Team)
```yaml
quotas:
  storage:
    user_storage_mb: 5000       # 5 GB per user
    max_file_size_mb: 100       # 100 MB files
  messages:
    rate_limit_messages: 10     # 10 per 5 seconds
```

### Strict (Public Server)
```yaml
quotas:
  storage:
    user_storage_mb: 100        # 100 MB per user
    max_file_size_mb: 10        # 10 MB files
  messages:
    rate_limit_messages: 3      # 3 per 5 seconds
    max_mentions_per_message: 5 # Prevent @spam
```

---

*End of Quotas Documentation*
