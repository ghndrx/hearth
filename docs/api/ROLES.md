# Roles API

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/servers/:id/roles` | Get all roles |
| POST | `/servers/:id/roles` | Create role |
| PATCH | `/servers/:id/roles/:roleId` | Update role |
| DELETE | `/servers/:id/roles/:roleId` | Delete role |

---

## Role Object

```json
{
  "id": "role-uuid",
  "server_id": "660e8400-e29b-41d4-a716-446655440001",
  "name": "Moderator",
  "color": 3447003,
  "hoist": true,
  "position": 5,
  "permissions": 1099511627775,
  "mentionable": true,
  "created_at": "2026-02-14T12:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| id | uuid | Role ID |
| server_id | uuid | Parent server |
| name | string | Role name |
| color | int | RGB color (decimal) |
| hoist | bool | Show separately in member list |
| position | int | Hierarchy position (higher = more power) |
| permissions | int64 | Permission bitfield |
| mentionable | bool | Can be @mentioned |
| created_at | timestamp | Creation time |

---

## Permissions

Permissions are stored as a 64-bit bitfield.

| Permission | Bit | Value | Description |
|------------|-----|-------|-------------|
| CREATE_INVITE | 0 | 1 | Create invites |
| KICK_MEMBERS | 1 | 2 | Kick members |
| BAN_MEMBERS | 2 | 4 | Ban members |
| ADMINISTRATOR | 3 | 8 | Full permissions |
| MANAGE_CHANNELS | 4 | 16 | Manage channels |
| MANAGE_SERVER | 5 | 32 | Manage server settings |
| ADD_REACTIONS | 6 | 64 | Add reactions |
| VIEW_AUDIT_LOG | 7 | 128 | View audit log |
| PRIORITY_SPEAKER | 8 | 256 | Priority speaker in voice |
| STREAM | 9 | 512 | Go live/screen share |
| VIEW_CHANNEL | 10 | 1024 | View channels |
| SEND_MESSAGES | 11 | 2048 | Send messages |
| SEND_TTS | 12 | 4096 | Send TTS messages |
| MANAGE_MESSAGES | 13 | 8192 | Delete/pin messages |
| EMBED_LINKS | 14 | 16384 | Embed links |
| ATTACH_FILES | 15 | 32768 | Upload files |
| READ_HISTORY | 16 | 65536 | Read message history |
| MENTION_EVERYONE | 17 | 131072 | Mention @everyone |
| USE_EXTERNAL_EMOJI | 18 | 262144 | Use external emojis |
| VIEW_INSIGHTS | 19 | 524288 | View server insights |
| CONNECT | 20 | 1048576 | Connect to voice |
| SPEAK | 21 | 2097152 | Speak in voice |
| MUTE_MEMBERS | 22 | 4194304 | Mute others |
| DEAFEN_MEMBERS | 23 | 8388608 | Deafen others |
| MOVE_MEMBERS | 24 | 16777216 | Move members in voice |
| USE_VAD | 25 | 33554432 | Use voice activity |
| CHANGE_NICKNAME | 26 | 67108864 | Change own nickname |
| MANAGE_NICKNAMES | 27 | 134217728 | Change others' nicknames |
| MANAGE_ROLES | 28 | 268435456 | Manage roles |
| MANAGE_WEBHOOKS | 29 | 536870912 | Manage webhooks |
| MANAGE_EMOJIS | 30 | 1073741824 | Manage emojis |

### Common Permission Values

| Role Type | Permissions |
|-----------|-------------|
| @everyone default | `104324673` (view, send, read history, reactions) |
| Moderator | `104324701` (+ kick, ban) |
| Admin | `2147483647` (all except administrator) |
| Full Admin | `-1` (all permissions) |

---

## GET /servers/:id/roles

Get all roles for a server.

### Response (200 OK)

```json
[
  {
    "id": "@everyone",
    "name": "@everyone",
    "position": 0,
    "permissions": 104324673,
    ...
  },
  {
    "id": "role-uuid",
    "name": "Admin",
    "position": 10,
    "permissions": -1,
    ...
  }
]
```

Roles are sorted by position (ascending).

---

## POST /servers/:id/roles

Create a new role. Requires `MANAGE_ROLES`.

### Request Body

```json
{
  "name": "New Role",
  "color": 15844367,
  "permissions": 104324673
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | No | Default: "new role" |
| color | int | No | RGB as decimal |
| permissions | int64 | No | Permission bitfield |

### Response (201 Created)

Returns created role object.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | no_permission | Missing MANAGE_ROLES |
| 403 | max_roles | Server has 250 roles |

---

## PATCH /servers/:id/roles/:roleId

Update a role. Requires `MANAGE_ROLES`.

### Request Body

```json
{
  "name": "Updated Name",
  "color": 3066993,
  "hoist": true,
  "permissions": 268435455,
  "mentionable": true,
  "position": 3
}
```

All fields optional.

### Response (200 OK)

Returns updated role object.

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | hierarchy | Cannot modify role higher than yours |
| 403 | cannot_modify_everyone | Cannot delete @everyone |

---

## DELETE /servers/:id/roles/:roleId

Delete a role. Requires `MANAGE_ROLES`.

### Response (204 No Content)

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 403 | hierarchy | Cannot delete role higher than yours |
| 403 | cannot_delete_everyone | Cannot delete @everyone |

---

## Role Hierarchy

Roles follow a strict hierarchy:
1. Server owner can manage all roles
2. Users can only manage roles **below** their highest role
3. @everyone is always position 0 and cannot be deleted

### Checking Permissions

```
user_permissions = 0

for role in user.roles (sorted by position descending):
    user_permissions |= role.permissions

if user_permissions & ADMINISTRATOR:
    # Has all permissions
```
