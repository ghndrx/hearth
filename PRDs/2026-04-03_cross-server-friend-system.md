# Cross-Server Friend System

**Discord Feature**: Friend requests, friend lists, cross-server DMs, friend presence
**Priority**: P1 (Social Graph Foundation)
**Estimated Complexity**: 10-14 weeks

## User Value Proposition

Enable users to maintain relationships across servers through friend requests, presence sharing, and direct messaging. Creates social graph that increases platform stickiness and enables cross-server discovery.

## Discord Equivalent

Discord's friend system allows users to:
- Send friend requests via username or mutual servers
- See friend online status across all servers
- Send DMs to friends regardless of shared servers
- Join friends' voice channels (with permissions)
- Rich presence showing game/activity status

## Technical Implementation Sketch

### Database Schema
```sql
-- Friend relationships
CREATE TABLE friends (
    requester_id UUID NOT NULL REFERENCES users(id),
    addressee_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, accepted, blocked
    created_at TIMESTAMPTZ DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,
    PRIMARY KEY (requester_id, addressee_id)
);

-- Friend presence/activity sharing
CREATE TABLE user_presence (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    status VARCHAR(20) DEFAULT 'offline', -- online, idle, dnd, offline
    activity JSONB, -- current activity/game
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### API Endpoints
- `POST /api/friends/requests` - Send friend request
- `GET /api/friends` - List accepted friends with presence
- `PATCH /api/friends/{userId}` - Accept/reject/block friend
- `DELETE /api/friends/{userId}` - Remove friend
- `GET /api/friends/requests` - List pending requests

### WebSocket Events
- `FRIEND_REQUEST` - Incoming friend request
- `FRIEND_ACCEPTED` - Friend request accepted
- `PRESENCE_UPDATE` - Friend status/activity changed

## Dependencies

- Enhanced DM system for cross-server messaging
- User presence tracking system
- Privacy controls for friend discoverability
- Server member mutual friend indicators

## Technical Complexity: P1

**High complexity** due to:
- Complex permission system (can friends DM? see presence?)
- Real-time presence distribution at scale
- Privacy implications and consent management
- Integration with existing server-based permissions
- Friend discovery mechanisms across servers