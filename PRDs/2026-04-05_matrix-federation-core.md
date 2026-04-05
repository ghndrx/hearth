# Matrix Federation - Core P0

> **Status**: NOT STARTED — P0 Priority
> **Estimated**: 16-24 weeks
> **Updated**: 2026-04-05

## Overview

Enable Hearth servers to federate with each other over the Matrix protocol, allowing users to:
- Send DMs to users on other Hearth servers
- Join rooms (channels) on remote Hearth servers
- Participate in cross-server communities without a single centralized server

This makes Hearth a true decentralized alternative to Discord, not just self-hosted.

## Why Matrix Protocol

Matrix (via matrix.org) is the open standard for decentralized chat. Key benefits:
- **No vendor lock-in**: Any Matrix-compatible server can talk to any other
- **Existing ecosystem**: Clients (Element), servers (Synapse, Dendrite, Conduit), bridges
- **Proven at scale**: Used by governments, enterprises, communities
- **Spec is stable**: Client-Server API r0.6.1, Server-Server API r6.0

## User Value Proposition

- **True decentralization**: No single Hearth server required — anyone can run their own
- **Censorship resistance**: No single point of control or failure
- **Interoperability**: Can DM users on any Matrix server (Element, Synapse, etc.)
- **Data sovereignty**: Users own their messages on their server
- **Bridge potential**: Eventually bridge to IRC, Slack, Discord, Telegram

## Technical Architecture

### Identity
- MXIDs: `@username:homeserver.example.com`
- Users get a localpart + homeserver domain
- 3PID (email/phone) association for discovery (future)

### Server-Server (Federation) API
Implemented endpoints per [Matrix Federation API spec](https://matrix.org/docs/spec/server_server/unstable):

```
GET  /_matrix/federation/v1/version        — Server capabilities
GET  /_matrix/federation/v1/make_join/{roomId}/{userId}  — Begin join
POST /_matrix/federation/v1/make_join/{roomId}/{userId}  — Submit join
GET  /_matrix/federation/v1/send_join/{roomId}/{eventId} — Send join event
POST /_matrix/federation/v1/send_join/{roomId}/{eventId} — Join via proxy
GET  /_matrix/federation/v1/send/{roomId}/{txnId}        — Transaction send
GET  /_matrix/federation/v1/state/{roomId}                — Room state
GET  /_matrix/federation/v1/state_ids/{roomId}            — Room state IDs
GET  /_matrix/federation/v1/backfill/{roomId}             — Backfill history
GET  /_matrix/federation/v1/event_auth/{roomId}/{eventId} — Event auth chain
POST /_matrix/federation/v1/get_missing_events/{roomId}   — Fetch missing events
PUT  /_matrix/federation/v1/invite/{roomId}/{eventId}    — Forward invite
GET  /_matrix/federation/v1/query/{queryType}             — Directory queries
GET  /_matrix/federation/v1/device/{userId}/{deviceId}    — Device updates
PUT  /_matrix/federation/v1/send_device_message/{userId}  — Device messages
GET  /_matrix/key/v2/query/{domain}                        — Server key
GET  /_matrix/key/v2/auth/{userId}                        — User key (3PID)
POST /_matrix/identity/v2/store-invite                    — 3PID invite store
```

### Client-Server API (Core subset for MVP)
```
POST /_matrix/client/r0/login                      — Login (password, token)
GET  /_matrix/client/r0/sync                      — Initial sync + incremental
GET  /_matrix/client/r0/rooms/{roomId}/messages  — Message history (backfill)
PUT  /_matrix/client/r0/rooms/{roomId}/send/{txnId} — Send message event
POST /_matrix/client/r0/join/{roomId}              — Join room
POST /_matrix/client/r0/leave/{roomId}             — Leave room
POST /_matrix/client/r0/invite/{roomId}            — Invite user
GET  /_matrix/client/r0/profile/{userId}           — User profile
PUT  /_matrix/client/r0/account/whoami              — Current user MXID
```

### Data Model Changes
```
Users:
  - localpart: string         — e.g., "alice"
  - homeserver: string        — e.g., "hearth.example.com"
  - mxid: string (computed)   — e.g., "@alice:hearth.example.com"
  - display_name: string
  - avatar_url: string?

Rooms (Channels/Servers):
  - room_id: string           — "!AbcDef:homeserver.example.com"
  - room_version: string      — "6" (Matrix room version)
  - predecessor: room_id?     — For room upgrades
  - is_public: bool          — Listed in room directory

Membership:
  - room_id + user_id + membership (join/invite/leave/ban/knock)
  - event_id for each join/leave event

Events:
  - event_id: string          — Unique per server
  - room_id: string
  - sender: mxid string
  - type: string              — m.room.message, m.room.member, etc.
  - content: json
  - origin_server_ts: int64
  - depth: int64
  - prev_events: [event_id, hash]
  - auth_events: [event_id]
  - hashes: string             — Event content hash
  - signatures: map           — Server signatures for integrity
```

### Key Storage
- Ed25519 server signing key (queryable via `/_matrix/key/v2`)
- Ed25519 user signing keys (stored in account data)
- NACL (Curve25519) device keys for E2EE (future: Megolm)

### Server Key Distribution
- Server publishes keys at `/_matrix/key/v2`
- Includes utime timestamp for key lifetime
- Keys signed by server's Ed25519 master key

## Implementation Phases

### Phase 1: Identity Layer (2-3 weeks)
- Update user model with MXID computation
- Add homeserver domain to config
- Implement `/_matrix/client/r0/account/whoami`
- Implement `/_matrix/client/r0/profile/{userId}`
- Update authentication tokens to include homeserver

### Phase 2: Client-Server Core (4-6 weeks)
- Implement `/_matrix/client/r0/sync` (full + incremental)
- Implement room join/leave
- Implement message send/receive
- Implement `/_matrix/client/r0/login` (password flow)
- E2EE: Olm/Megolm for DM rooms (later: can be out of scope for MVP)

### Phase 3: Server-Server Federation (6-10 weeks)
- Implement `/_matrix/federation/v1/send/{roomId}/{txnId}`
- Implement join flow across servers
- Implement invite forwarding
- Implement backfill (history sharing)
- Implement state resolution (v2 algorithm)
- Implement key distribution `/_matrix/key/v2`

### Phase 4: Room Directory & Discovery (2-3 weeks)
- `/_matrix/federation/v1/query/directory` — room alias resolution
- Public room list federation
- Server-side room directory

### Phase 5: DM Federation (2-4 weeks)
- One-to-one DM rooms across servers
- 3PID invite flow (email-based invite to Matrix users)

## Out of Scope (v1)
- Room upgrades
- Space rooms
- Threading (beyond Matrix's threading support)
- End-to-end encryption (Olm/Megolm) — keep DMs unencrypted for v1
- 3PID identity server (beyond basic invite flow)
- Federation with non-Hearth Matrix servers initially

## Dependencies
- PostgreSQL (already in use) — room state, events table
- Redis (already in use) — device lists, sync filters
- No new external dependencies for MVP

## Compatibility
- Target Matrix room version 6
- Support Matrix protocol r0.6.1 (stable)
- Must be able to communicate with Synapse, Dendrite, Conduit servers

## Testing Strategy
- Federation with local second Hearth instance
- Federation with public Matrix server (matrix.org if possible)
- Client tests against Element web client
