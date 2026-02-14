# Servers API

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/servers` | Create server |
| GET | `/servers/:id` | Get server |
| PATCH | `/servers/:id` | Update server |
| DELETE | `/servers/:id` | Delete server |
| GET | `/servers/:id/channels` | Get channels |
| POST | `/servers/:id/channels` | Create channel |
| GET | `/servers/:id/members` | Get members |
| GET | `/servers/:id/members/:userId` | Get member |
| PATCH | `/servers/:id/members/:userId` | Update member |
| DELETE | `/servers/:id/members/:userId` | Kick member |
| DELETE | `/servers/:id/members/@me` | Leave server |
| GET | `/servers/:id/bans` | Get bans |
| PUT | `/servers/:id/bans/:userId` | Ban user |
| DELETE | `/servers/:id/bans/:userId` | Unban user |
| GET | `/servers/:id/invites` | Get invites |
| GET | `/servers/:id/roles` | Get roles |
| POST | `/servers/:id/roles` | Create role |
| PATCH | `/servers/:id/roles/:roleId` | Update role |
| DELETE | `/servers/:id/roles/:roleId` | Delete role |

---

## Server Object

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "name": "My Server",
  "icon_url": "https://cdn.hearth.chat/icons/...",
  "banner_url": "https://cdn.hearth.chat/banners/...",
  "description": "A community server",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "features": ["COMMUNITY", "NEWS"],
  "created_at": "2026-02-14T12:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| id | uuid | Server ID |
| name | string | Server name (2-100 chars) |
| icon_url | string? | Server icon |
| banner_url | string? | Server banner |
| description | string? | Server description |
| owner_id | uuid | Owner user ID |
| features | string[] | Enabled features |
| created_at | timestamp | Creation time |

---

## POST /servers

Create a new server.

### Request Body

```json
{
  "name": "My New Server",
  "icon": "https://..."
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| name | string | Yes | 2-100 characters |
| icon | string | No | Valid URL |

### Response (201 Created)

Returns the created server object.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | validation | Name too short/long |
| 403 | max_servers | Reached server limit |

---

## GET /servers/:id

Get server details.

### Response (200 OK)

Returns server object.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | invalid_id | Invalid UUID |
| 404 | not_found | Server not found |

---

## PATCH /servers/:id

Update server settings. Requires `MANAGE_SERVER` permission.

### Request Body

```json
{
  "name": "Updated Name",
  "icon": "https://...",
  "banner": "https://...",
  "description": "New description"
}
```

All fields are optional.

### Response (200 OK)

Returns updated server object.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | forbidden | No permission |
| 404 | not_found | Server not found |

---

## DELETE /servers/:id

Delete a server. **Owner only.**

### Response (204 No Content)

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | not_owner | Only owner can delete |
| 404 | not_found | Server not found |

---

## GET /servers/:id/channels

Get all channels in a server.

### Response (200 OK)

```json
[
  {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "type": "text",
    "name": "general",
    "server_id": "660e8400-e29b-41d4-a716-446655440001",
    "parent_id": null,
    "position": 0,
    "topic": "General chat",
    "nsfw": false,
    "last_message_id": "...",
    "created_at": "2026-02-14T12:00:00Z"
  }
]
```

---

## POST /servers/:id/channels

Create a new channel. Requires `MANAGE_CHANNELS` permission.

### Request Body

```json
{
  "name": "new-channel",
  "type": "text",
  "parent_id": "880e8400-e29b-41d4-a716-446655440003"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Channel name |
| type | string | No | `text` (default), `voice`, `category` |
| parent_id | uuid | No | Parent category ID |

### Response (201 Created)

Returns created channel object.

---

## Members

### Member Object

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "member",
    "discriminator": "0001",
    "avatar_url": null
  },
  "nick": "Server Nickname",
  "roles": ["role-id-1", "role-id-2"],
  "joined_at": "2026-02-14T12:00:00Z"
}
```

---

## GET /servers/:id/members

Get server members with pagination.

### Query Parameters

| Name | Type | Default | Description |
|------|------|---------|-------------|
| limit | int | 100 | Max 1000 |
| offset | int | 0 | Starting offset |

### Response (200 OK)

```json
[
  {
    "user": { ... },
    "nick": null,
    "roles": [],
    "joined_at": "2026-02-14T12:00:00Z"
  }
]
```

---

## PATCH /servers/:id/members/:userId

Update a member's nickname or roles.

### Request Body

```json
{
  "nick": "New Nickname",
  "roles": ["role-id-1"]
}
```

### Response (200 OK)

Returns updated member object.

---

## DELETE /servers/:id/members/:userId

Kick a member. Requires `KICK_MEMBERS` permission.

### Query Parameters

| Name | Type | Description |
|------|------|-------------|
| reason | string | Kick reason (audit log) |

### Response (204 No Content)

---

## DELETE /servers/:id/members/@me

Leave the server.

### Response (204 No Content)

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | owner_cannot_leave | Transfer ownership first |

---

## Bans

### Ban Object

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "banned",
    "discriminator": "0001"
  },
  "reason": "Spam"
}
```

---

## GET /servers/:id/bans

Get banned users. Requires `BAN_MEMBERS` permission.

### Response (200 OK)

```json
[
  {
    "user": { ... },
    "reason": "Rule violation"
  }
]
```

---

## PUT /servers/:id/bans/:userId

Ban a user. Requires `BAN_MEMBERS` permission.

### Request Body

```json
{
  "reason": "Spam",
  "delete_message_days": 7,
  "delete_message_seconds": 604800
}
```

| Field | Type | Description |
|-------|------|-------------|
| reason | string | Ban reason |
| delete_message_days | int | Days of messages to delete (0-7) |
| delete_message_seconds | int | Alternative: seconds of messages |

### Response (204 No Content)

---

## DELETE /servers/:id/bans/:userId

Unban a user. Requires `BAN_MEMBERS` permission.

### Response (204 No Content)

---

## GET /servers/:id/invites

Get all active invites for the server.

### Response (200 OK)

```json
[
  {
    "code": "abc123",
    "channel_id": "770e8400-e29b-41d4-a716-446655440002",
    "creator_id": "550e8400-e29b-41d4-a716-446655440000",
    "max_uses": 100,
    "uses": 5,
    "expires_at": "2026-02-21T12:00:00Z",
    "temporary": false,
    "created_at": "2026-02-14T12:00:00Z"
  }
]
```
