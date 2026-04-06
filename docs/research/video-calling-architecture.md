# Video/Audio Calling Architecture - PRD

## Overview

This document defines the architecture for direct video/audio calling in Hearth. The system builds on the existing WebSocket gateway and WebRTC signaling infrastructure (`backend/internal/websocket/video.go`, `frontend/src/lib/stores/videoCall.ts`) to add persistent call tracking, REST API endpoints, and a user-facing CallView UI.

## Current State

### Already Implemented
- **WebSocket signaling service** (`websocket/video.go`): Full WebRTC relay with SDP offer/answer, ICE candidate exchange, ring/accept/decline, screen sharing, and in-memory call tracking via `VideoSignalingService`
- **Frontend store** (`stores/videoCall.ts`): Complete state management for calls with gateway event listeners for VIDEO_RING_START, VIDEO_SERVER_UPDATE, VIDEO_STATE_UPDATE, etc.
- **VideoGrid component** (`components/VideoGrid.svelte`): LiveKit-based participant video grid rendering
- **Voice infrastructure**: LiveKit integration for voice channels with VoiceStateRepository, VoiceService, and LiveKitVoiceHandler

### What This Foundation Adds
- Persistent data models (Call, CallParticipant, CallSession) with DB schema
- REST API for call lifecycle management (create, join, leave, get, signal)
- Call service layer with business logic
- CallView UI component with media controls
- Frontend calls.ts store for REST API integration alongside existing WebSocket store

---

## 1. WebRTC Setup

### STUN/TURN Servers

```
STUN: stun:stun.l.google.com:19302 (default, free)
TURN: Configurable via environment variables:
  - TURN_SERVER_URL
  - TURN_USERNAME
  - TURN_CREDENTIAL
```

### Peer Connection Management

Hearth uses **peer-to-peer mesh** for small calls (up to 4 participants) and delegates to **LiveKit SFU** for larger calls. The existing `VideoSignalingService` handles the mesh signaling.

**Connection Flow:**
1. Caller creates call via `POST /api/v1/calls`
2. Backend persists call record, returns call ID
3. Caller sends `VIDEO_RING` via WebSocket gateway
4. Callee receives `VIDEO_RING_START` event
5. Callee accepts via WebSocket `VIDEO_RING_RESPONSE`
6. Both peers exchange SDP offers/answers via `VIDEO_OFFER` / `VIDEO_ANSWER`
7. ICE candidates exchanged via `VIDEO_ICE_CANDIDATE`
8. Media flows peer-to-peer (or via LiveKit SFU for >4 participants)

### ICE Configuration

```go
type ICEConfig struct {
    STUNServers []string `json:"stun_servers"`
    TURNServers []TURNServer `json:"turn_servers,omitempty"`
}

type TURNServer struct {
    URL        string `json:"url"`
    Username   string `json:"username"`
    Credential string `json:"credential"`
}
```

---

## 2. Backend Signaling Service

### Existing: WebSocket-Based Signaling (`websocket/video.go`)

The `VideoSignalingService` already handles all real-time signaling:

| Client Message | Server Event | Purpose |
|---|---|---|
| `VIDEO_RING` | `VIDEO_RING_START` | Initiate call, ring target |
| `VIDEO_RING_RESPONSE` | `VIDEO_RING_ACCEPT` / `VIDEO_RING_DECLINE` | Accept or decline |
| `VIDEO_LEAVE` | `VIDEO_STATE_UPDATE` (is_left) | Leave call |
| `VIDEO_OFFER` | `VIDEO_OFFER` | SDP offer relay |
| `VIDEO_ANSWER` | `VIDEO_ANSWER` | SDP answer relay |
| `VIDEO_ICE_CANDIDATE` | `VIDEO_ICE_CANDIDATE` | ICE candidate relay |
| `VIDEO_STATE_UPDATE` | `VIDEO_STATE_UPDATE` | Camera/mute/screen state |
| `VIDEO_SCREEN_START` | `VIDEO_STATE_UPDATE` | Start screen share |
| `VIDEO_SCREEN_STOP` | `VIDEO_STATE_UPDATE` | Stop screen share |

### New: REST API for Persistence

The REST API layer provides durable call records for:
- Call history and analytics
- Reconnection after network drops
- Querying active calls in a channel/server

---

## 3. Data Models

### Call

```go
type Call struct {
    ID          uuid.UUID  `json:"id" db:"id"`
    ChannelID   uuid.UUID  `json:"channel_id" db:"channel_id"`
    ServerID    *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
    InitiatorID uuid.UUID  `json:"initiator_id" db:"initiator_id"`
    Type        CallType   `json:"type" db:"type"`            // direct, group, channel
    Status      CallStatus `json:"status" db:"status"`        // ringing, active, ended
    StartedAt   time.Time  `json:"started_at" db:"started_at"`
    EndedAt     *time.Time `json:"ended_at,omitempty" db:"ended_at"`
    EndReason   string     `json:"end_reason,omitempty" db:"end_reason"` // completed, missed, declined, error
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}
```

### CallParticipant

```go
type CallParticipant struct {
    ID        uuid.UUID  `json:"id" db:"id"`
    CallID    uuid.UUID  `json:"call_id" db:"call_id"`
    UserID    uuid.UUID  `json:"user_id" db:"user_id"`
    JoinedAt  time.Time  `json:"joined_at" db:"joined_at"`
    LeftAt    *time.Time `json:"left_at,omitempty" db:"left_at"`
    IsMuted   bool       `json:"is_muted" db:"is_muted"`
    IsVideoOn bool       `json:"is_video_on" db:"is_video_on"`
    // Populated from joins
    Username    string  `json:"username,omitempty" db:"username"`
    DisplayName *string `json:"display_name,omitempty" db:"display_name"`
    Avatar      *string `json:"avatar,omitempty" db:"avatar"`
}
```

### CallSession

```go
type CallSession struct {
    ID            uuid.UUID  `json:"id" db:"id"`
    CallID        uuid.UUID  `json:"call_id" db:"call_id"`
    UserID        uuid.UUID  `json:"user_id" db:"user_id"`
    SessionID     string     `json:"session_id" db:"session_id"`    // Gateway session
    PeerID        string     `json:"peer_id" db:"peer_id"`          // WebRTC peer ID
    ConnectedAt   time.Time  `json:"connected_at" db:"connected_at"`
    DisconnectedAt *time.Time `json:"disconnected_at,omitempty" db:"disconnected_at"`
    ConnectionType string    `json:"connection_type" db:"connection_type"` // peer, sfu
}
```

### DB Schema (Migration 047)

```sql
CREATE TABLE calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    initiator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL DEFAULT 'direct',
    status VARCHAR(20) NOT NULL DEFAULT 'ringing',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    end_reason VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE call_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    is_muted BOOLEAN NOT NULL DEFAULT true,
    is_video_on BOOLEAN NOT NULL DEFAULT false,
    UNIQUE(call_id, user_id)
);

CREATE TABLE call_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255) NOT NULL,
    peer_id VARCHAR(255) NOT NULL DEFAULT '',
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMPTZ,
    connection_type VARCHAR(20) NOT NULL DEFAULT 'peer'
);

CREATE INDEX idx_calls_channel_id ON calls(channel_id);
CREATE INDEX idx_calls_initiator_id ON calls(initiator_id);
CREATE INDEX idx_calls_status ON calls(status) WHERE status != 'ended';
CREATE INDEX idx_call_participants_call_id ON call_participants(call_id);
CREATE INDEX idx_call_participants_user_id ON call_participants(user_id);
CREATE INDEX idx_call_sessions_call_id ON call_sessions(call_id);
```

---

## 4. API Handlers

### Endpoints

| Method | Path | Description |
|---|---|---|
| `POST /api/v1/calls` | Create a new call | Returns call object with ID |
| `GET /api/v1/calls/:id` | Get call details | Returns call with participants |
| `POST /api/v1/calls/:id/join` | Join an existing call | Adds participant record |
| `POST /api/v1/calls/:id/leave` | Leave a call | Updates participant left_at |
| `POST /api/v1/calls/:id/signal` | Send signaling data | Relays via WebSocket (bridge) |

### Request/Response Examples

**POST /api/v1/calls**
```json
// Request
{
    "channel_id": "uuid",
    "server_id": "uuid",        // optional
    "type": "direct",           // direct | group | channel
    "target_user_id": "uuid"    // for direct calls
}

// Response (201)
{
    "id": "uuid",
    "channel_id": "uuid",
    "initiator_id": "uuid",
    "type": "direct",
    "status": "ringing",
    "started_at": "2026-04-06T...",
    "participants": [...]
}
```

**POST /api/v1/calls/:id/join**
```json
// Response (200)
{
    "call_id": "uuid",
    "user_id": "uuid",
    "joined_at": "2026-04-06T...",
    "ice_servers": [
        { "urls": ["stun:stun.l.google.com:19302"] }
    ]
}
```

---

## 5. Gateway WebSocket Events

### Existing Events (already in `websocket/video.go`)

| Event | Direction | Data |
|---|---|---|
| `VIDEO_RING_START` | Server -> Client | call_id, channel_id, from_user_id, call_type |
| `VIDEO_RING_ACCEPT` | Server -> Client | call_id, from_user_id |
| `VIDEO_RING_DECLINE` | Server -> Client | call_id, from_user_id |
| `VIDEO_RING_END` | Server -> Client | call_id, reason |
| `VIDEO_STATE_UPDATE` | Server -> Client | call_id, user_id, is_joined/is_left/state |
| `VIDEO_SERVER_UPDATE` | Server -> Client | Full call state snapshot |
| `VIDEO_OFFER` | Server -> Client | call_id, from_user_id, sdp |
| `VIDEO_ANSWER` | Server -> Client | call_id, from_user_id, sdp |
| `VIDEO_ICE_CANDIDATE` | Server -> Client | call_id, from_user_id, candidate |

### New Events (added by this foundation)

| Event | Direction | Data |
|---|---|---|
| `CALL_CREATED` | Server -> Channel | call object with participants |
| `CALL_ENDED` | Server -> Channel | call_id, end_reason, duration |

---

## 6. Frontend Store: calls.ts

### Purpose

`calls.ts` provides REST API integration for persistent call operations, complementing the existing `videoCall.ts` WebSocket store.

```typescript
// Active calls indexed by channel
export const activeCalls = writable<Map<string, Call>>(new Map());

// Functions
export async function createCall(channelId: string, type: CallType, targetUserId?: string): Promise<Call>
export async function getCall(callId: string): Promise<Call>
export async function joinCall(callId: string): Promise<JoinCallResponse>
export async function leaveCall(callId: string): Promise<void>
export async function getChannelCalls(channelId: string): Promise<Call[]>
```

### Integration with videoCall.ts

`calls.ts` handles the REST API lifecycle. `videoCall.ts` handles real-time WebSocket state. They coordinate:

1. `createCall()` -> POST /calls -> gets call ID -> `videoCallStore.startCall()` -> sends VIDEO_RING
2. `joinCall()` -> POST /calls/:id/join -> gets ICE servers -> `videoCallStore.acceptCall()`
3. `leaveCall()` -> POST /calls/:id/leave -> `videoCallStore.endCall()`

---

## 7. Frontend UI: CallView Component

### CallView.svelte

Full-screen overlay component activated when a call is active.

**Layout:**
```
+------------------------------------------+
|  Call with @username          [duration]  |
|                                          |
|  +----------------+  +----------------+ |
|  |                |  |                | |
|  |  Remote Video  |  |  Local Video   | |
|  |  (or Avatar)   |  |  (PiP corner)  | |
|  |                |  |                | |
|  +----------------+  +----------------+ |
|                                          |
|  [Mute] [Camera] [ScreenShare] [EndCall] |
+------------------------------------------+
```

**Controls:**
- Mute/Unmute microphone
- Camera on/off
- Screen share toggle
- End call button (red)
- Participant list (for group calls)

**States:**
- Ringing out: "Calling..." with cancel button
- Ringing in: Accept/Decline buttons with caller info
- Connecting: "Connecting..." spinner
- Connected: Full video view with controls
- Ended: "Call ended" with duration summary

---

## 8. Mobile Stubs

### iOS (Future)
- `CallViewController.swift` - Main call screen
- `CallKit` integration for system-level call UI
- Uses same WebSocket signaling protocol

### Android (Future)
- `CallActivity.kt` - Main call screen
- `ConnectionService` for system call management
- Uses same WebSocket signaling protocol

Mobile implementations will use the same REST API and WebSocket protocol documented here.

---

## 9. Security

### Authentication
- All `/api/v1/calls/*` endpoints require `RequireAuth` middleware (JWT)
- WebSocket signaling requires authenticated gateway session
- User ID extracted from JWT claims, never from request body

### Call Participant Validation
- Only call participants can send signaling messages for that call
- `VideoSignalingService` validates `client.UserID` membership in `call.Participants`
- Join requires channel membership (verified via ChannelService)
- Direct calls require existing DM channel or friendship

### Rate Limiting
- Call creation: 10 calls/minute per user
- Signal relay: 100 messages/second per user (ICE candidates can be bursty)
- Join/leave: 30 requests/minute per user

### Data Privacy
- SDP offers/answers are relayed, never stored
- ICE candidates are relayed, never stored
- Call metadata (duration, participants) stored for history
- Media streams are peer-to-peer, never touch the server (mesh mode)

---

## 10. Future Work

- [ ] TURN server deployment and credential rotation
- [ ] SFU mode via LiveKit for calls with >4 participants
- [ ] Call recording (opt-in, with consent UI)
- [ ] Call quality metrics (bitrate, packet loss, jitter)
- [ ] Push notifications for incoming calls (mobile)
- [ ] CallKit/ConnectionService integration (mobile)
- [ ] Group call management (add/remove participants mid-call)
- [ ] Call transfer and hold functionality
