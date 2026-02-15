# Invites API

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/invites/:code` | Get invite info |
| POST | `/invites/:code` | Accept invite |
| DELETE | `/invites/:code` | Delete invite |
| POST | `/channels/:id/invites` | Create channel invite |
| GET | `/servers/:id/invites` | Get server invites |

---

## Invite Object

```json
{
  "code": "abc123xy",
  "guild_id": "660e8400-e29b-41d4-a716-446655440001",
  "channel_id": "770e8400-e29b-41d4-a716-446655440002",
  "inviter_id": "550e8400-e29b-41d4-a716-446655440000",
  "max_uses": 100,
  "uses": 5,
  "expires_at": "2026-02-21T12:00:00Z",
  "temporary": false,
  "created_at": "2026-02-14T12:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| code | string | Unique invite code |
| guild_id | uuid | Target server |
| channel_id | uuid | Target channel |
| inviter_id | uuid | Creator user ID |
| max_uses | int | Max uses (0 = unlimited) |
| uses | int | Current use count |
| expires_at | timestamp? | Expiration time (null = never) |
| temporary | bool | Grant temporary membership |
| created_at | timestamp | Creation time |

---

## GET /invites/:code

Get information about an invite.

### Parameters

| Name | Type | Description |
|------|------|-------------|
| code | string | Invite code |

### Response (200 OK)

```json
{
  "code": "abc123xy",
  "guild_id": "660e8400-e29b-41d4-a716-446655440001",
  "channel_id": "770e8400-e29b-41d4-a716-446655440002",
  "max_uses": 100,
  "uses": 5,
  "expires_at": "2026-02-21T12:00:00Z",
  "temporary": false,
  "created_at": "2026-02-14T12:00:00Z",
  "guild": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "Cool Server",
    "icon_url": "https://...",
    "member_count": 150
  },
  "channel": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "name": "general",
    "type": "text"
  }
}
```

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 404 | not_found | Invalid or expired invite |

---

## POST /invites/:code

Accept an invite and join the server.

### Parameters

| Name | Type | Description |
|------|------|-------------|
| code | string | Invite code |

### Response (200 OK)

Returns the server object you joined.

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "name": "Cool Server",
  "icon_url": "https://...",
  "owner_id": "550e8400-e29b-41d4-a716-446655440000",
  "features": [],
  "created_at": "2026-02-14T12:00:00Z"
}
```

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | already_member | Already in this server |
| 400 | invite_expired | Invite has expired |
| 400 | max_uses | Invite has been used too many times |
| 403 | banned | User is banned from server |

---

## DELETE /invites/:code

Delete an invite. Requires `MANAGE_SERVER` or be the invite creator.

### Parameters

| Name | Type | Description |
|------|------|-------------|
| code | string | Invite code |

### Response (204 No Content)

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | no_permission | Not creator and no MANAGE_SERVER |
| 404 | not_found | Invite not found |

---

## POST /channels/:id/invites

Create a new invite for a channel. Requires `CREATE_INVITE`.

### Request Body

```json
{
  "max_age": 86400,
  "max_uses": 10,
  "temporary": false
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| max_age | int | 86400 | Seconds until expiry (0 = never) |
| max_uses | int | 0 | Max uses (0 = unlimited) |
| temporary | bool | false | Grant temporary membership |

### Response (201 Created)

```json
{
  "code": "xyz789ab",
  "guild_id": "660e8400-e29b-41d4-a716-446655440001",
  "channel_id": "770e8400-e29b-41d4-a716-446655440002",
  "inviter_id": "550e8400-e29b-41d4-a716-446655440000",
  "max_uses": 10,
  "uses": 0,
  "expires_at": "2026-02-15T12:00:00Z",
  "temporary": false,
  "created_at": "2026-02-14T12:00:00Z"
}
```

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | no_permission | Missing CREATE_INVITE |
| 403 | max_invites | Server has too many invites |

---

## GET /servers/:id/invites

Get all active invites for a server. Requires `MANAGE_SERVER`.

### Parameters

| Name | Type | Description |
|------|------|-------------|
| id | uuid | Server ID |

### Response (200 OK)

```json
[
  {
    "code": "abc123xy",
    "channel_id": "770e8400-e29b-41d4-a716-446655440002",
    "inviter_id": "550e8400-e29b-41d4-a716-446655440000",
    "max_uses": 100,
    "uses": 5,
    "expires_at": "2026-02-21T12:00:00Z",
    "temporary": false,
    "created_at": "2026-02-14T12:00:00Z"
  }
]
```

---

## Temporary Membership

When `temporary: true`:
1. User joins the server normally
2. If user disconnects from all voice channels without having a role assigned, they are automatically kicked
3. Useful for events or trials

---

## Vanity URLs

Servers with the `VANITY_URL` feature can have a custom invite code:

```
https://hearth.chat/invite/myserver
```

Vanity URLs are managed via server settings, not the invites API.
