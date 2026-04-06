# Global Custom Emojis System

**Discord Feature**: Cross-server emoji usage for premium users, emoji discovery, server emoji browsing
**Priority**: P1 (Premium/Engagement)
**Estimated Complexity**: 6-8 weeks

## User Value Proposition

Premium users can use their favorite custom emojis across all servers, not just where they were uploaded. Increases premium subscription value and creates engagement loops where users discover new servers through emoji usage.

## Discord Equivalent

Discord Nitro features:
- Use custom emojis anywhere (servers where not a member)
- Emoji picker shows emojis from all servers user is in
- Animated emoji support for premium users
- High-resolution emoji uploads
- Cross-server emoji reactions
- Server emoji discovery browser

## Technical Implementation Sketch

### Database Schema Additions
```sql
-- Enhance existing emojis table
ALTER TABLE emojis ADD COLUMN animated BOOLEAN DEFAULT FALSE;
ALTER TABLE emojis ADD COLUMN available BOOLEAN DEFAULT TRUE;
ALTER TABLE emojis ADD COLUMN managed BOOLEAN DEFAULT FALSE; -- bot-uploaded
ALTER TABLE emojis ADD COLUMN require_colons BOOLEAN DEFAULT TRUE;

-- Premium emoji usage permissions
CREATE TABLE global_emoji_permissions (
    user_id UUID NOT NULL REFERENCES users(id),
    emoji_id UUID NOT NULL REFERENCES emojis(id),
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    granted_by VARCHAR(20) DEFAULT 'premium', -- premium, server_boost, etc.
    PRIMARY KEY (user_id, emoji_id)
);

-- Emoji usage analytics
CREATE TABLE emoji_usage_stats (
    emoji_id UUID NOT NULL REFERENCES emojis(id),
    server_id UUID REFERENCES servers(id),
    usage_count BIGINT DEFAULT 0,
    last_used TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (emoji_id, server_id)
);
```

### Implementation Components
1. **Emoji Permission System**: Check user premium status for cross-server usage
2. **Enhanced Emoji Picker**: Show available emojis from all servers + global
3. **Emoji Discovery**: Browse popular emojis across servers user can access
4. **Usage Analytics**: Track emoji popularity and cross-server adoption
5. **Premium Integration**: Tie to existing subscription system

### API Endpoints
- `GET /api/emojis/global` - Global emojis available to user
- `GET /api/emojis/discover` - Discover popular emojis
- `POST /api/emojis/{id}/use` - Use emoji cross-server (premium check)
- `GET /api/servers/{id}/emojis/stats` - Server emoji usage analytics

### WebSocket Events
- `EMOJI_UPDATE` - Emoji permissions changed
- `EMOJI_USAGE` - Real-time emoji usage for analytics

## Dependencies

- Existing premium subscription system ✅ (implemented)
- Enhanced emoji management system
- Cross-server permission validation
- Premium feature flag system
- Analytics dashboard for server owners

## Technical Complexity: P1

**Medium complexity** due to:
- Integration with premium subscription checks
- Cross-server permission validation performance
- Emoji discovery algorithm and popularity tracking
- Real-time emoji availability updates
- Analytics system for usage patterns