# Users API

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/users/@me` | Get current user |
| PATCH | `/users/@me` | Update current user |
| GET | `/users/@me/servers` | Get user's servers |
| GET | `/users/@me/channels` | Get user's DMs |
| GET | `/users/@me/relationships` | Get friends/blocked |
| POST | `/users/@me/relationships` | Add friend/block user |
| DELETE | `/users/@me/relationships/:id` | Remove relationship |
| GET | `/users/:id` | Get user by ID |

---

## User Object

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "myuser",
  "discriminator": "0001",
  "email": "user@example.com",
  "avatar_url": "https://cdn.hearth.chat/avatars/...",
  "banner_url": "https://cdn.hearth.chat/banners/...",
  "bio": "Hello, world!",
  "custom_status": "Working on Hearth",
  "flags": 0,
  "created_at": "2026-02-14T12:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| id | uuid | User's unique ID |
| username | string | Username (2-32 chars) |
| discriminator | string | 4-digit tag |
| email | string | Email (only for @me) |
| avatar_url | string? | Avatar URL |
| banner_url | string? | Profile banner URL |
| bio | string? | Profile bio (max 190 chars) |
| custom_status | string? | Custom status message |
| flags | int64 | User flags bitmask |
| created_at | timestamp | Account creation time |

---

## GET /users/@me

Get the authenticated user's profile.

### Response (200 OK)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "myuser",
  "discriminator": "0001",
  "email": "user@example.com",
  "avatar_url": null,
  "bio": "Hello!",
  "flags": 0,
  "created_at": "2026-02-14T12:00:00Z"
}
```

---

## PATCH /users/@me

Update the authenticated user's profile.

### Request Body

```json
{
  "username": "newusername",
  "avatar_url": "https://...",
  "banner_url": "https://...",
  "bio": "New bio",
  "custom_status": "Away"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| username | string | No | 2-32 characters |
| avatar_url | string | No | Valid URL |
| banner_url | string | No | Valid URL |
| bio | string | No | Max 190 characters |
| custom_status | string | No | Status message |

### Response (200 OK)

Returns updated user object.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | validation | Invalid field values |
| 409 | username_taken | Username already in use |

---

## GET /users/@me/servers

Get servers the user is a member of.

### Response (200 OK)

```json
[
  {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "My Server",
    "icon_url": "https://...",
    "banner_url": null,
    "description": "A cool server",
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "features": ["COMMUNITY"],
    "created_at": "2026-02-14T12:00:00Z"
  }
]
```

---

## GET /users/@me/channels

Get the user's direct message channels.

### Response (200 OK)

```json
[
  {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "type": "dm",
    "recipients": [
      {
        "id": "880e8400-e29b-41d4-a716-446655440003",
        "username": "friend",
        "discriminator": "0002",
        "avatar_url": null
      }
    ],
    "last_message_id": "990e8400-e29b-41d4-a716-446655440004",
    "created_at": "2026-02-14T12:00:00Z"
  }
]
```

---

## GET /users/:id

Get a user's public profile.

### Parameters

| Name | Type | Description |
|------|------|-------------|
| id | uuid | User ID |

### Response (200 OK)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "someuser",
  "discriminator": "0001",
  "avatar_url": null,
  "banner_url": null,
  "bio": "Hello!",
  "flags": 0,
  "created_at": "2026-02-14T12:00:00Z"
}
```

**Note:** Email is not included for other users.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | invalid_id | Invalid UUID format |
| 404 | not_found | User not found |

---

## Relationships

### Relationship Types

| Type | Value | Description |
|------|-------|-------------|
| Friend | 1 | Mutual friends |
| Blocked | 2 | User is blocked |
| Pending Incoming | 3 | Received friend request |
| Pending Outgoing | 4 | Sent friend request |

---

## GET /users/@me/relationships

Get all relationships (friends, blocked, pending).

### Response (200 OK)

```json
[
  {
    "id": "880e8400-e29b-41d4-a716-446655440003",
    "type": 1,
    "user": {
      "id": "880e8400-e29b-41d4-a716-446655440003",
      "username": "friend",
      "discriminator": "0002",
      "avatar_url": null,
      "flags": 0
    }
  }
]
```

---

## POST /users/@me/relationships

Create a friend request or block a user.

### Request Body

```json
{
  "user_id": "880e8400-e29b-41d4-a716-446655440003",
  "type": 1
}
```

Or by username:

```json
{
  "username": "friend#0002",
  "type": 1
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| user_id | uuid | Conditional | Target user ID |
| username | string | Conditional | Target username (alternative) |
| type | int | Yes | 1 = friend, 2 = block |

### Response (204 No Content)

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | validation | Missing user_id/username |
| 400 | self_relationship | Cannot add yourself |
| 404 | not_found | User not found |

---

## DELETE /users/@me/relationships/:id

Remove a friend or unblock a user.

### Parameters

| Name | Type | Description |
|------|------|-------------|
| id | uuid | Target user ID |

### Response (204 No Content)

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | invalid_id | Invalid UUID format |
| 500 | failed | Failed to remove relationship |
