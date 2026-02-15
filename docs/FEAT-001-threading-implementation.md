# FEAT-001: Message Threading Enhancements

## Implementation Summary

This document describes the implementation of enhanced threading features for Hearth.

## Features Implemented

### 1. Thread Sidebar View ✅
- **Location**: `frontend/src/lib/components/ThreadSidebar.svelte`
- Opens when clicking on a thread in the message list
- Shows parent message with context
- Displays all thread replies with date dividers
- Auto-scrolls to latest messages
- Reply input with Enter to send, Shift+Enter for newlines

### 2. Thread Notification Preferences ✅
- **Backend**: 
  - Database migration: `backend/internal/database/postgres/migrations/003_thread_notifications.sql`
  - Model: `backend/internal/models/thread.go` - `ThreadNotificationPreference`
  - Service: `backend/internal/services/thread_service.go` - `GetNotificationPreference`, `SetNotificationPreference`
  - Handler: `backend/internal/api/handlers/threads.go` - `GetNotificationPreference`, `SetNotificationPreference`
  - Routes: `GET/PUT /threads/:id/notifications`

- **Frontend**:
  - Store: `frontend/src/lib/stores/thread.ts` - `threadNotificationLevel`, `setNotificationPreference`
  - UI: Dropdown in ThreadSidebar with three options:
    - **All Messages**: Notify for every reply
    - **Only @mentions**: Notify only when mentioned
    - **Muted**: No notifications from this thread

### 3. Thread Participant Indicators ✅

#### Historical Participants (who has participated)
- **Component**: `frontend/src/lib/components/ThreadParticipants.svelte`
- Shows stacked avatars of users who have posted in the thread
- Extracted from thread messages
- Shows overflow count when many participants

#### Active Viewers (real-time presence) ✅
- **Backend**:
  - Database table: `thread_presence` in migration 003
  - Service methods: `EnterThread`, `ExitThread`, `GetActiveViewers`, `HeartbeatPresence`
  - Handler methods: `EnterThread`, `ExitThreadPresence`, `GetActiveViewers`, `HeartbeatPresence`
  - Routes: 
    - `POST /threads/:id/presence` - Enter thread (start viewing)
    - `DELETE /threads/:id/presence` - Exit thread (stop viewing)
    - `GET /threads/:id/presence` - Get active viewers
    - `PATCH /threads/:id/presence` - Heartbeat (keep alive)

- **Frontend**:
  - Store: `threadActiveViewers` derived store
  - Presence heartbeat every 30 seconds while thread is open
  - WebSocket subscription to `THREAD_PRESENCE_UPDATE` events
  - UI: Green pulsing indicator with "X is viewing" or "X and Y others are viewing"

## Database Changes

New tables in migration `003_thread_notifications.sql`:

```sql
-- Thread notification preferences
CREATE TABLE thread_notification_preferences (
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    level VARCHAR(20) NOT NULL DEFAULT 'all', -- 'all', 'mentions', 'none'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (thread_id, user_id)
);

-- Thread presence (active viewers)
CREATE TABLE thread_presence (
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (thread_id, user_id)
);
```

## API Endpoints Added

| Method | Path | Description |
|--------|------|-------------|
| GET | `/threads/:id/notifications` | Get user's notification preference for thread |
| PUT | `/threads/:id/notifications` | Set notification preference (`{level: "all"|"mentions"|"none"}`) |
| GET | `/threads/:id/presence` | Get active viewers in thread |
| POST | `/threads/:id/presence` | Enter thread (mark as viewing) |
| PATCH | `/threads/:id/presence` | Heartbeat (keep presence alive) |
| DELETE | `/threads/:id/presence` | Exit thread (stop viewing) |

## WebSocket Events Added

| Event | Direction | Description |
|-------|-----------|-------------|
| `THREAD_PRESENCE_UPDATE` | Server → Client | Broadcast when thread viewers change |
| `THREAD_MESSAGE_CREATE` | Server → Client | Broadcast when new thread message arrives |

## Files Modified

### Backend
- `internal/models/thread.go` - New notification/presence types
- `internal/models/websocket.go` - New WS event constants
- `internal/database/postgres/thread_repo.go` - Notification/presence repo methods
- `internal/services/thread_service.go` - Notification/presence service methods
- `internal/api/handlers/threads.go` - New API handlers
- `internal/api/routes.go` - New routes
- `internal/services/thread_service_test.go` - Updated mocks

### Frontend
- `src/lib/stores/thread.ts` - Enhanced with notifications and presence
- `src/lib/components/ThreadSidebar.svelte` - Enhanced UI with all features
- `src/lib/components/ThreadParticipants.svelte` - Existing, shows historical participants

### New Files
- `backend/internal/database/postgres/migrations/003_thread_notifications.sql`
- `docs/FEAT-001-threading-implementation.md` (this file)

## Testing

Run backend tests:
```bash
cd backend && go test ./internal/services/... -run Thread -v
cd backend && go test ./internal/api/handlers/... -run Thread -v
```

Build and verify:
```bash
cd backend && go build ./...
cd frontend && npm run build
```

## Usage Flow

1. User clicks "Thread" on a message → Thread sidebar opens
2. Frontend calls `POST /threads/:id/presence` → User marked as viewing
3. Frontend starts 30-second heartbeat interval → Calls `PATCH /threads/:id/presence`
4. Frontend subscribes to `THREAD_PRESENCE_UPDATE` WebSocket events
5. Active viewers shown with pulsing green indicator
6. User changes notification setting → `PUT /threads/:id/notifications`
7. User closes thread → `DELETE /threads/:id/presence` called, heartbeat stops
