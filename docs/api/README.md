# Hearth API Documentation

## Overview

Hearth exposes a RESTful HTTP API and WebSocket gateway for real-time events. The API follows Discord-compatible patterns for easy migration.

**Base URL:** `https://your-hearth-instance/api/v1`  
**WebSocket Gateway:** `wss://your-hearth-instance/gateway`

## Quick Reference

| Resource | Endpoints |
|----------|-----------|
| [Authentication](./AUTH.md) | Login, register, OAuth, tokens |
| [Users](./USERS.md) | Profiles, relationships, settings |
| [Servers](./SERVERS.md) | Create, join, manage servers |
| [Channels](./CHANNELS.md) | Messages, reactions, pins |
| [Roles](./ROLES.md) | Permissions, role management |
| [Invites](./INVITES.md) | Create and accept invites |
| [WebSocket](./WEBSOCKET.md) | Real-time events |

## Authentication

All API requests (except auth endpoints) require a valid JWT access token.

```http
Authorization: Bearer <access_token>
```

Tokens are obtained via `/api/v1/auth/login` or `/api/v1/auth/register`.

## Response Format

### Success

```json
{
  "id": "uuid",
  "field": "value"
}
```

### Error

```json
{
  "error": "error_code",
  "message": "Human readable description"
}
```

## HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 204 | No Content (success, no body) |
| 400 | Bad Request (validation error) |
| 401 | Unauthorized (invalid/missing token) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Not Found |
| 409 | Conflict (duplicate resource) |
| 429 | Rate Limited |
| 500 | Internal Server Error |

## Rate Limits

Rate limits vary by endpoint:

| Scope | Limit | Window |
|-------|-------|--------|
| Global | 50 req | 1 second |
| Message Send | 5 msg | 5 seconds |
| Auth | 5 req | 60 seconds |

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Max requests
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Unix timestamp when limit resets

## Pagination

List endpoints support cursor-based pagination:

| Parameter | Description |
|-----------|-------------|
| `limit` | Max items to return (1-100, default 50) |
| `before` | Get items before this ID |
| `after` | Get items after this ID |

## Endpoints Summary

### Health Check
```
GET /health
```
Returns `{"status": "ok"}` if the server is running.

### Authentication
```
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/auth/oauth/:provider
GET  /api/v1/auth/oauth/:provider/callback
```

### Users
```
GET   /api/v1/users/@me
PATCH /api/v1/users/@me
GET   /api/v1/users/@me/servers
GET   /api/v1/users/@me/channels
GET   /api/v1/users/@me/relationships
POST  /api/v1/users/@me/relationships
DELETE /api/v1/users/@me/relationships/:id
GET   /api/v1/users/:id
```

### Servers
```
POST   /api/v1/servers
GET    /api/v1/servers/:id
PATCH  /api/v1/servers/:id
DELETE /api/v1/servers/:id
GET    /api/v1/servers/:id/channels
POST   /api/v1/servers/:id/channels
GET    /api/v1/servers/:id/members
GET    /api/v1/servers/:id/members/:userId
PATCH  /api/v1/servers/:id/members/:userId
DELETE /api/v1/servers/:id/members/:userId
DELETE /api/v1/servers/:id/members/@me
GET    /api/v1/servers/:id/bans
PUT    /api/v1/servers/:id/bans/:userId
DELETE /api/v1/servers/:id/bans/:userId
GET    /api/v1/servers/:id/invites
GET    /api/v1/servers/:id/roles
POST   /api/v1/servers/:id/roles
PATCH  /api/v1/servers/:id/roles/:roleId
DELETE /api/v1/servers/:id/roles/:roleId
```

### Channels
```
GET    /api/v1/channels/:id
PATCH  /api/v1/channels/:id
DELETE /api/v1/channels/:id
GET    /api/v1/channels/:id/messages
POST   /api/v1/channels/:id/messages
GET    /api/v1/channels/:id/messages/:messageId
PATCH  /api/v1/channels/:id/messages/:messageId
DELETE /api/v1/channels/:id/messages/:messageId
PUT    /api/v1/channels/:id/messages/:messageId/reactions/:emoji/@me
DELETE /api/v1/channels/:id/messages/:messageId/reactions/:emoji/@me
GET    /api/v1/channels/:id/pins
PUT    /api/v1/channels/:id/pins/:messageId
DELETE /api/v1/channels/:id/pins/:messageId
POST   /api/v1/channels/:id/typing
POST   /api/v1/channels/:id/invites
```

### Invites
```
GET    /api/v1/invites/:code
POST   /api/v1/invites/:code
DELETE /api/v1/invites/:code
```

### Voice
```
GET /api/v1/voice/regions
```

### Gateway
```
GET /api/v1/gateway/stats
GET /gateway (WebSocket)
```
