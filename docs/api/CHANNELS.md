# Channels API

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/channels/:id` | Get channel |
| PATCH | `/channels/:id` | Update channel |
| DELETE | `/channels/:id` | Delete channel |
| GET | `/channels/:id/messages` | Get messages |
| POST | `/channels/:id/messages` | Send message |
| GET | `/channels/:id/messages/:messageId` | Get message |
| PATCH | `/channels/:id/messages/:messageId` | Edit message |
| DELETE | `/channels/:id/messages/:messageId` | Delete message |
| PUT | `/channels/:id/messages/:messageId/reactions/:emoji/@me` | Add reaction |
| DELETE | `/channels/:id/messages/:messageId/reactions/:emoji/@me` | Remove reaction |
| GET | `/channels/:id/pins` | Get pinned messages |
| PUT | `/channels/:id/pins/:messageId` | Pin message |
| DELETE | `/channels/:id/pins/:messageId` | Unpin message |
| POST | `/channels/:id/typing` | Send typing indicator |
| POST | `/channels/:id/invites` | Create invite |

---

## Channel Types

| Type | Description |
|------|-------------|
| `text` | Text channel for messages |
| `voice` | Voice channel |
| `dm` | Direct message |
| `group_dm` | Group direct message |
| `category` | Channel category |
| `announcement` | Announcement/news channel |
| `forum` | Forum channel |

---

## Channel Object

```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "type": "text",
  "name": "general",
  "server_id": "660e8400-e29b-41d4-a716-446655440001",
  "parent_id": null,
  "position": 0,
  "topic": "General discussion",
  "nsfw": false,
  "rate_limit_per_user": 0,
  "last_message_id": "990e8400-e29b-41d4-a716-446655440004",
  "permission_overwrites": [],
  "created_at": "2026-02-14T12:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| id | uuid | Channel ID |
| type | string | Channel type |
| name | string | Channel name |
| server_id | uuid? | Parent server (null for DMs) |
| parent_id | uuid? | Parent category |
| position | int | Sort position |
| topic | string? | Channel topic |
| nsfw | bool | Age-restricted content |
| rate_limit_per_user | int | Slowmode in seconds |
| last_message_id | uuid? | Most recent message ID |
| permission_overwrites | array | Permission overrides |
| created_at | timestamp | Creation time |

---

## GET /channels/:id

Get channel details.

### Response (200 OK)

Returns channel object.

---

## PATCH /channels/:id

Update channel settings. Requires `MANAGE_CHANNELS`.

### Request Body

```json
{
  "name": "new-name",
  "topic": "New topic",
  "nsfw": false,
  "rate_limit_per_user": 5,
  "position": 2,
  "parent_id": "category-id"
}
```

All fields optional.

### Response (200 OK)

Returns updated channel object.

---

## DELETE /channels/:id

Delete a channel. Requires `MANAGE_CHANNELS`.

### Response (204 No Content)

---

## Messages

### Message Object

```json
{
  "id": "990e8400-e29b-41d4-a716-446655440004",
  "channel_id": "770e8400-e29b-41d4-a716-446655440002",
  "guild_id": "660e8400-e29b-41d4-a716-446655440001",
  "author_id": "550e8400-e29b-41d4-a716-446655440000",
  "content": "Hello, world!",
  "type": 0,
  "timestamp": "2026-02-14T12:30:00Z",
  "edited_timestamp": null,
  "pinned": false,
  "tts": false,
  "referenced_message_id": null,
  "attachments": [],
  "reactions": [
    {
      "emoji": "👍",
      "count": 3,
      "me": true
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| id | uuid | Message ID |
| channel_id | uuid | Channel ID |
| guild_id | uuid? | Server ID |
| author_id | uuid | Author user ID |
| content | string | Message content |
| type | int | Message type (0 = default) |
| timestamp | timestamp | Send time |
| edited_timestamp | timestamp? | Edit time |
| pinned | bool | Is pinned |
| tts | bool | Text-to-speech |
| referenced_message_id | uuid? | Reply target |
| attachments | array | File attachments |
| reactions | array | Message reactions |

---

## GET /channels/:id/messages

Get messages with pagination.

### Query Parameters

| Name | Type | Default | Description |
|------|------|---------|-------------|
| limit | int | 50 | Max 100 |
| before | uuid | - | Get messages before this ID |
| after | uuid | - | Get messages after this ID |

### Response (200 OK)

```json
[
  {
    "id": "...",
    "content": "Hello!",
    "author_id": "...",
    "timestamp": "2026-02-14T12:30:00Z",
    ...
  }
]
```

---

## POST /channels/:id/messages

Send a message.

### Request Body

```json
{
  "content": "Hello, world!",
  "reply_to": "message-id-to-reply-to",
  "tts": false,
  "nonce": "client-side-id"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| content | string | Yes | Message content (max 2000 chars) |
| reply_to | uuid | No | Message to reply to |
| tts | bool | No | Text-to-speech |
| nonce | string | No | Client de-duplication ID |

### Response (201 Created)

Returns created message object.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | empty_message | Content is required |
| 403 | no_permission | Cannot send in this channel |
| 429 | rate_limited | Sending too fast |

---

## GET /channels/:id/messages/:messageId

Get a specific message.

### Response (200 OK)

Returns message object.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 404 | not_found | Message not found |

---

## PATCH /channels/:id/messages/:messageId

Edit a message. **Author only.**

### Request Body

```json
{
  "content": "Updated message"
}
```

### Response (200 OK)

Returns updated message with `edited_timestamp` set.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | not_author | Can only edit own messages |
| 404 | not_found | Message not found |

---

## DELETE /channels/:id/messages/:messageId

Delete a message. Author or `MANAGE_MESSAGES` required.

### Response (204 No Content)

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | no_permission | Not author and no permission |
| 404 | not_found | Message not found |

---

## Reactions

### PUT /channels/:id/messages/:messageId/reactions/:emoji/@me

Add a reaction to a message.

### Parameters

| Name | Type | Description |
|------|------|-------------|
| emoji | string | URL-encoded emoji (🔥 or custom:id) |

### Response (204 No Content)

---

### DELETE /channels/:id/messages/:messageId/reactions/:emoji/@me

Remove your reaction from a message.

### Response (204 No Content)

---

## Pins

### GET /channels/:id/pins

Get all pinned messages in a channel.

### Response (200 OK)

```json
[
  {
    "id": "...",
    "content": "Important announcement",
    "pinned": true,
    ...
  }
]
```

---

### PUT /channels/:id/pins/:messageId

Pin a message. Requires `MANAGE_MESSAGES`.

### Response (204 No Content)

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | max_pins | Channel has 50 pinned messages |
| 404 | not_found | Message not found |

---

### DELETE /channels/:id/pins/:messageId

Unpin a message. Requires `MANAGE_MESSAGES`.

### Response (204 No Content)

---

## Typing Indicator

### POST /channels/:id/typing

Send a typing indicator. Broadcasts `TYPING_START` event.

### Response (204 No Content)

Typing indicator expires after 10 seconds. Re-send to keep active.
