# Hearth — Data Model Reference

**Version:** 1.0  
**Last Updated:** 2026-02-11

---

## Entity Reference

### User

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| email | string | Unique, for auth and notifications |
| username | string | Display name, unique with discriminator |
| discriminator | string(4) | 4-digit suffix (e.g., #1234) |
| password_hash | string | Argon2id hash |
| avatar_url | string? | CDN URL for avatar |
| banner_url | string? | Profile banner image |
| bio | string? | About me text (max 200 chars) |
| status | enum | online, idle, dnd, invisible, offline |
| custom_status | string? | Custom status text |
| mfa_enabled | boolean | 2FA active |
| mfa_secret | string? | TOTP secret (encrypted) |
| verified | boolean | Email verified |
| flags | int | System flags (staff, partner, etc.) |
| created_at | timestamp | Account creation |
| updated_at | timestamp | Last modification |

### Server

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| name | string | Server name (2-100 chars) |
| icon_url | string? | Server icon |
| banner_url | string? | Server banner |
| splash_url | string? | Invite splash image |
| description | string? | Server description |
| owner_id | UUID | FK to users |
| default_channel_id | UUID? | Landing channel for new members |
| afk_channel_id | UUID? | AFK voice channel |
| afk_timeout | int | Seconds before AFK move |
| verification_level | int | 0=none, 1=email, 2=5min, 3=10min, 4=phone |
| explicit_content_filter | int | 0=off, 1=no role, 2=all |
| default_notifications | int | 0=all, 1=mentions |
| features | string[] | Enabled features |
| max_members | int | Member limit (default 500k) |
| vanity_url_code | string? | Custom invite slug |
| created_at | timestamp | Creation time |
| updated_at | timestamp | Last modification |

### Channel

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| server_id | UUID? | FK to servers (null for DMs) |
| category_id | UUID? | FK to parent category channel |
| type | enum | text, voice, category, announcement, forum, stage, dm, group_dm |
| name | string? | Channel name (not for DMs) |
| topic | string? | Channel topic/description |
| position | int | Sort order |
| nsfw | boolean | Age-gated content |
| slowmode_seconds | int | Message rate limit (0-21600) |
| bitrate | int | Voice bitrate (8000-384000) |
| user_limit | int | Voice channel limit (0=unlimited) |
| rtc_region | string? | Voice region override |
| default_auto_archive | int | Thread auto-archive (60, 1440, 4320, 10080 mins) |
| last_message_id | UUID? | For unread calculation |
| created_at | timestamp | Creation time |
| updated_at | timestamp | Last modification |

### Message

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key (snowflake-style) |
| channel_id | UUID | FK to channels |
| author_id | UUID | FK to users |
| content | string | Message text (max 2000 chars) |
| type | enum | default, reply, join, leave, pin, etc. |
| reply_to_id | UUID? | FK to parent message |
| thread_id | UUID? | FK to thread channel |
| pinned | boolean | Is pinned |
| tts | boolean | Text-to-speech |
| mention_everyone | boolean | Contains @everyone/@here |
| mentions | UUID[] | User IDs mentioned |
| mention_roles | UUID[] | Role IDs mentioned |
| mention_channels | UUID[] | Channel IDs mentioned |
| embeds | json[] | Rich embeds |
| attachments | Attachment[] | Files |
| reactions | Reaction[] | Emoji reactions |
| sticker_ids | UUID[] | Stickers used |
| flags | int | Message flags |
| created_at | timestamp | Send time |
| edited_at | timestamp? | Last edit time |

### Role

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| server_id | UUID | FK to servers |
| name | string | Role name |
| color | int | RGB color (0 = no color) |
| permissions | bigint | Permission bitmask |
| position | int | Hierarchy position (higher = more power) |
| hoist | boolean | Show separately in member list |
| managed | boolean | Managed by integration |
| mentionable | boolean | Can be @mentioned |
| icon_url | string? | Role icon |
| unicode_emoji | string? | Role emoji |
| created_at | timestamp | Creation time |

### Member

| Field | Type | Description |
|-------|------|-------------|
| user_id | UUID | FK to users (PK part 1) |
| server_id | UUID | FK to servers (PK part 2) |
| nickname | string? | Server-specific nickname |
| avatar_url | string? | Server-specific avatar |
| roles | UUID[] | Role IDs (ordered) |
| joined_at | timestamp | Join time |
| premium_since | timestamp? | Boost start time |
| deaf | boolean | Server deafened |
| mute | boolean | Server muted |
| pending | boolean | Pending membership screening |
| timeout_until | timestamp? | Timeout expiration |
| flags | int | Member flags |

### Invite

| Field | Type | Description |
|-------|------|-------------|
| code | string | Primary key (8 chars) |
| server_id | UUID | FK to servers |
| channel_id | UUID | FK to target channel |
| inviter_id | UUID? | FK to creating user |
| target_type | int? | 0=stream, 1=embedded app |
| target_user_id | UUID? | For stream invites |
| max_uses | int? | Use limit (null = unlimited) |
| uses | int | Current use count |
| max_age | int? | Seconds until expiry (null = never) |
| temporary | boolean | Kick on disconnect if no role |
| created_at | timestamp | Creation time |
| expires_at | timestamp? | Expiration time |

### Emoji

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| server_id | UUID | FK to servers |
| name | string | Emoji name (alphanumeric + underscore) |
| creator_id | UUID? | FK to uploading user |
| require_colons | boolean | Needs :name: syntax |
| managed | boolean | Managed by integration |
| animated | boolean | Is animated GIF |
| available | boolean | Currently usable |
| roles | UUID[] | Roles that can use (empty = all) |
| url | string | CDN URL |
| created_at | timestamp | Upload time |

### Webhook

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| server_id | UUID | FK to servers |
| channel_id | UUID | FK to target channel |
| creator_id | UUID? | FK to creating user |
| name | string | Webhook name |
| avatar_url | string? | Webhook avatar |
| token | string | Secret token for posting |
| type | int | 1=incoming, 2=channel follower |
| source_server_id | UUID? | For follower webhooks |
| source_channel_id | UUID? | For follower webhooks |
| created_at | timestamp | Creation time |

### AuditLogEntry

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| server_id | UUID | FK to servers |
| user_id | UUID? | Acting user |
| target_id | UUID? | Affected entity |
| action_type | int | Action code |
| changes | json[] | Before/after values |
| options | json? | Additional context |
| reason | string? | Mod reason |
| created_at | timestamp | Action time |

### VoiceState

| Field | Type | Description |
|-------|------|-------------|
| user_id | UUID | FK to users (PK part 1) |
| server_id | UUID? | FK to servers |
| channel_id | UUID | FK to voice channel (PK part 2) |
| session_id | string | Voice session ID |
| deaf | boolean | Server deafened |
| mute | boolean | Server muted |
| self_deaf | boolean | Self deafened |
| self_mute | boolean | Self muted |
| self_stream | boolean | Streaming |
| self_video | boolean | Camera on |
| suppress | boolean | Suppressed (stage) |
| request_to_speak_at | timestamp? | Stage raise hand |

---

## Relationships

```
User 1──────────∞ Member ∞──────────1 Server
                    │
                    ∞
                    │
                  Role

Server 1──────────∞ Channel
                    │
                    ∞
                    │
                 Message ∞──────────1 User (author)
                    │
                    ∞
                    │
                Attachment
                Reaction

Channel 1──────────∞ ChannelOverride
                        │
                        1
                        │
                   Role or User

Server 1──────────∞ Invite
Server 1──────────∞ Emoji  
Server 1──────────∞ Webhook
Server 1──────────∞ AuditLogEntry
Server 1──────────∞ Ban
```

---

## Indexes

### Performance-Critical Indexes

```sql
-- Message retrieval (most common query)
CREATE INDEX idx_messages_channel_time 
ON messages(channel_id, created_at DESC);

-- Message search
CREATE INDEX idx_messages_content_search 
ON messages USING gin(to_tsvector('english', content));

-- Member lookup
CREATE INDEX idx_members_server ON members(server_id);
CREATE INDEX idx_members_user ON members(user_id);

-- Role hierarchy
CREATE INDEX idx_roles_server_position 
ON roles(server_id, position DESC);

-- Invite lookup
CREATE INDEX idx_invites_server ON invites(server_id);
CREATE INDEX idx_invites_expires ON invites(expires_at) 
WHERE expires_at IS NOT NULL;

-- Audit log queries
CREATE INDEX idx_audit_server_time 
ON audit_log(server_id, created_at DESC);
CREATE INDEX idx_audit_user 
ON audit_log(user_id, created_at DESC);

-- Presence queries
CREATE INDEX idx_voice_state_channel 
ON voice_states(channel_id);
```

---

## Data Lifecycle

### Message Retention
- Default: Unlimited retention
- Configurable per-server or instance-wide
- Bulk delete: Up to 14 days old (API limitation)
- Attachments: Deleted with message or after orphan period

### User Deletion (GDPR)
1. Anonymize messages (author becomes "Deleted User")
2. Remove from all servers
3. Delete DM channels (or anonymize)
4. Purge personal data (email, avatar, etc.)
5. Keep audit log entries (anonymized)

### Server Deletion
1. Cascade delete all channels
2. Cascade delete all messages
3. Cascade delete all roles, emojis, webhooks
4. Remove all members
5. Invalidate all invites
6. Keep audit log for 30 days (compliance)

---

## Scaling Considerations

For deployments exceeding single-database capacity, see the detailed research documents:

- **[Database Sharding Strategies](./research/architecture-database-sharding.md)** - Partition/shard strategies for messages at scale (Discord, Slack, Matrix patterns)
- **[WebSocket Scaling](./research/architecture-websocket-scaling.md)** - Horizontal scaling for real-time connections

### Message ID Strategy

Hearth uses Snowflake-style IDs (64-bit integers) for messages, enabling:
- Chronological sorting without secondary indexes
- Distributed ID generation without coordination
- Embedded timestamp extraction
- Efficient cursor-based pagination

### Partition-Ready Schema

All message queries should include `channel_id` to enable future partitioning:
```sql
-- Good: Partition-aware query
SELECT * FROM messages WHERE channel_id = ? ORDER BY id DESC LIMIT 50;

-- Avoid: Full table scan
SELECT * FROM messages WHERE content LIKE '%search%';
```

---

*End of Data Model Reference*
