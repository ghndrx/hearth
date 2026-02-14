# WebSocket Gateway

## Overview

The WebSocket gateway provides real-time events for messages, presence, typing indicators, and more.

**Endpoint:** `wss://your-hearth-instance/gateway`

## Connection Flow

```
Client                           Gateway
  |                                 |
  |-------- Connect + Token ------->|
  |                                 |
  |<-------- HELLO (op 10) ---------|
  |                                 |
  |------- IDENTIFY (op 2) -------->|
  |                                 |
  |<-------- READY (op 0) ----------|
  |                                 |
  |<----- Events (op 0) ------------|
  |                                 |
  |--- HEARTBEAT (op 1) every N --->|
  |<--- HEARTBEAT_ACK (op 11) ------|
  |                                 |
```

## Connecting

### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| token | Yes | JWT access token |
| session_id | No | Previous session ID (for resume) |
| client_type | No | `web`, `desktop`, `mobile` |
| resume | No | Resume key from previous session |

### Example

```javascript
const ws = new WebSocket('wss://hearth.chat/gateway?token=eyJ...');
```

---

## Opcodes

| Opcode | Name | Direction | Description |
|--------|------|-----------|-------------|
| 0 | Dispatch | Receive | Event dispatch |
| 1 | Heartbeat | Both | Keep connection alive |
| 2 | Identify | Send | Authentication |
| 3 | Presence Update | Send | Update status |
| 4 | Voice State Update | Send | Join/leave voice |
| 6 | Resume | Send | Resume session |
| 7 | Reconnect | Receive | Server requests reconnect |
| 8 | Request Guild Members | Send | Request member list |
| 9 | Invalid Session | Receive | Session is invalid |
| 10 | Hello | Receive | Connection established |
| 11 | Heartbeat ACK | Receive | Heartbeat acknowledged |

---

## Message Format

All messages are JSON with this structure:

```json
{
  "op": 0,
  "d": { ... },
  "s": 42,
  "t": "MESSAGE_CREATE"
}
```

| Field | Type | Description |
|-------|------|-------------|
| op | int | Opcode |
| d | object | Event data payload |
| s | int? | Sequence number (op 0 only) |
| t | string? | Event type (op 0 only) |

---

## Hello (op 10)

Sent immediately after connecting.

```json
{
  "op": 10,
  "d": {
    "heartbeat_interval": 41250
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| heartbeat_interval | int | Milliseconds between heartbeats |

**Action:** Start heartbeat loop, then send IDENTIFY.

---

## Identify (op 2)

Authenticate the session.

```json
{
  "op": 2,
  "d": {
    "properties": {
      "$os": "linux",
      "$browser": "chrome",
      "$device": "desktop"
    },
    "compress": false
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| properties.$os | string | Operating system |
| properties.$browser | string | Browser/client name |
| properties.$device | string | Device type |
| compress | bool | Request zlib compression |

---

## Ready (READY)

Sent after successful IDENTIFY.

```json
{
  "op": 0,
  "s": 1,
  "t": "READY",
  "d": {
    "v": 10,
    "session_id": "abc123",
    "resume_gateway_url": "wss://...",
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "myuser"
    },
    "guilds": [...],
    "private_channels": [...]
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| v | int | Gateway version |
| session_id | string | Current session ID |
| resume_gateway_url | string | URL for resuming |
| user | object | Current user |
| guilds | array | User's servers |
| private_channels | array | User's DMs |

---

## Heartbeat (op 1)

Send periodically to keep connection alive.

```json
{
  "op": 1,
  "d": null
}
```

Server responds with Heartbeat ACK (op 11):

```json
{
  "op": 11
}
```

**Important:** If no ACK received, connection is dead. Reconnect.

---

## Presence Update (op 3)

Update your online status.

```json
{
  "op": 3,
  "d": {
    "status": "online",
    "activities": [
      {
        "type": 0,
        "name": "Playing a game"
      }
    ],
    "since": null,
    "afk": false
  }
}
```

| Status | Description |
|--------|-------------|
| online | Online |
| idle | Away/idle |
| dnd | Do not disturb |
| invisible | Appear offline |

---

## Resume (op 6)

Resume a disconnected session.

```json
{
  "op": 6,
  "d": {
    "session_id": "abc123",
    "seq": 42
  }
}
```

Server replays missed events, then sends RESUMED.

---

## Request Guild Members (op 8)

Request member list for large servers.

```json
{
  "op": 8,
  "d": {
    "guild_id": "660e8400-e29b-41d4-a716-446655440001",
    "query": "",
    "limit": 100,
    "presences": true,
    "nonce": "request-id"
  }
}
```

Server responds with GUILD_MEMBERS_CHUNK events.

---

## Events

### Message Events

| Event | Description |
|-------|-------------|
| MESSAGE_CREATE | New message |
| MESSAGE_UPDATE | Message edited |
| MESSAGE_DELETE | Message deleted |
| MESSAGE_DELETE_BULK | Multiple messages deleted |
| MESSAGE_REACTION_ADD | Reaction added |
| MESSAGE_REACTION_REMOVE | Reaction removed |
| MESSAGE_REACTION_REMOVE_ALL | All reactions removed |

### MESSAGE_CREATE

```json
{
  "op": 0,
  "s": 15,
  "t": "MESSAGE_CREATE",
  "d": {
    "id": "990e8400-e29b-41d4-a716-446655440004",
    "channel_id": "770e8400-e29b-41d4-a716-446655440002",
    "guild_id": "660e8400-e29b-41d4-a716-446655440001",
    "author": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "sender",
      "discriminator": "0001",
      "avatar": null
    },
    "content": "Hello!",
    "timestamp": "2026-02-14T12:30:00Z",
    "edited_timestamp": null,
    "tts": false,
    "mention_everyone": false,
    "mentions": [],
    "mention_roles": [],
    "attachments": [],
    "embeds": [],
    "pinned": false,
    "type": 0
  }
}
```

### Channel Events

| Event | Description |
|-------|-------------|
| CHANNEL_CREATE | Channel created |
| CHANNEL_UPDATE | Channel updated |
| CHANNEL_DELETE | Channel deleted |
| CHANNEL_PINS_UPDATE | Pins changed |
| TYPING_START | User typing |

### TYPING_START

```json
{
  "op": 0,
  "t": "TYPING_START",
  "d": {
    "channel_id": "770e8400-e29b-41d4-a716-446655440002",
    "guild_id": "660e8400-e29b-41d4-a716-446655440001",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": 1708430400,
    "member": { ... }
  }
}
```

### Guild Events

| Event | Description |
|-------|-------------|
| GUILD_CREATE | Server available (on connect or join) |
| GUILD_UPDATE | Server settings changed |
| GUILD_DELETE | Server unavailable or left |
| GUILD_MEMBER_ADD | Member joined |
| GUILD_MEMBER_UPDATE | Member updated |
| GUILD_MEMBER_REMOVE | Member left/kicked |
| GUILD_MEMBERS_CHUNK | Response to member request |
| GUILD_ROLE_CREATE | Role created |
| GUILD_ROLE_UPDATE | Role updated |
| GUILD_ROLE_DELETE | Role deleted |
| GUILD_BAN_ADD | User banned |
| GUILD_BAN_REMOVE | User unbanned |

### Presence Events

| Event | Description |
|-------|-------------|
| PRESENCE_UPDATE | User status changed |
| USER_UPDATE | Current user updated |

### PRESENCE_UPDATE

```json
{
  "op": 0,
  "t": "PRESENCE_UPDATE",
  "d": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000"
    },
    "guild_id": "660e8400-e29b-41d4-a716-446655440001",
    "status": "online",
    "activities": [],
    "client_status": {
      "desktop": "online"
    }
  }
}
```

### Voice Events

| Event | Description |
|-------|-------------|
| VOICE_STATE_UPDATE | User voice state changed |
| VOICE_SERVER_UPDATE | Voice server info |

---

## Close Codes

| Code | Description | Reconnect? |
|------|-------------|------------|
| 4000 | Unknown error | Yes |
| 4001 | Unknown opcode | Yes |
| 4002 | Decode error | Yes |
| 4003 | Not authenticated | No |
| 4004 | Authentication failed | No |
| 4005 | Already authenticated | Yes |
| 4006 | Invalid session | No (re-identify) |
| 4007 | Invalid seq | No (re-identify) |
| 4008 | Rate limited | Yes (after delay) |
| 4009 | Session timed out | No (re-identify) |
| 4010 | Invalid shard | No |
| 4011 | Sharding required | No |
| 4012 | Invalid API version | No |
| 4013 | Invalid intents | No |
| 4014 | Disallowed intents | No |

---

## Reconnection

1. Close code 4000-4009 (except 4003, 4004): Reconnect with resume
2. Close code 4003, 4004: Re-authenticate (new session)
3. No heartbeat ACK: Reconnect immediately
4. Exponential backoff: 1s, 2s, 4s, 8s... max 60s

```javascript
function reconnect(attempt) {
  const delay = Math.min(1000 * Math.pow(2, attempt), 60000);
  setTimeout(connect, delay);
}
```
