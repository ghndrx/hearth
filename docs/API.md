# Hearth API Documentation

For detailed API documentation, see the [api/](./api/) directory:

- **[Overview](./api/README.md)** - Quick reference and common patterns
- **[Authentication](./api/AUTH.md)** - Login, register, OAuth, tokens
- **[Users](./api/USERS.md)** - User profiles and relationships
- **[Servers](./api/SERVERS.md)** - Server management and members
- **[Channels](./api/CHANNELS.md)** - Messages, reactions, pins
- **[Roles](./api/ROLES.md)** - Permission system
- **[Invites](./api/INVITES.md)** - Server invites
- **[WebSocket](./api/WEBSOCKET.md)** - Real-time gateway

## Quick Start

### Base URL
```
https://your-hearth-instance/api/v1
```

### Authentication
```bash
# Login
curl -X POST https://hearth.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret"}'

# Use token
curl https://hearth.example.com/api/v1/users/@me \
  -H "Authorization: Bearer <access_token>"
```

### WebSocket
```javascript
const ws = new WebSocket('wss://hearth.example.com/gateway?token=<access_token>');

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.op === 10) { // HELLO
    // Start heartbeat
    setInterval(() => ws.send(JSON.stringify({op: 1})), msg.d.heartbeat_interval);
    // Identify
    ws.send(JSON.stringify({
      op: 2,
      d: { properties: { $os: 'web', $browser: 'chrome' } }
    }));
  }
};
```

## Error Format

```json
{
  "error": "error_code",
  "message": "Human readable description"
}
```

## Rate Limits

| Endpoint | Limit |
|----------|-------|
| Global | 50 req/s |
| Messages | 5/5s per channel |
| Auth | 5/60s |

## OpenAPI

An OpenAPI 3.0 spec is planned for a future release.
