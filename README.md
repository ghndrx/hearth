# Hearth

A self-hosted, open-source real-time communication platform. Discord-compatible API with full data sovereignty.

## Overview

Hearth is a chat platform designed for teams, communities, and organizations that need to own their communication infrastructure. It provides familiar Discord-like functionality while keeping all data on your servers.

**Key differentiators:**
- Self-hosted with no external dependencies
- End-to-end encryption for DMs (optional for channels)
- Single binary deployment or Docker/Kubernetes
- PostgreSQL or SQLite backend

## Features

### Communication
- Text channels with Markdown, file sharing, and embeds
- Voice channels (WebRTC)
- Direct messages and group DMs
- Threads and forum channels
- Typing indicators and presence

### Server Management
- Servers with categories and channels
- Role-based permissions (30+ granular permissions)
- Invite system with expiration and usage limits
- Audit logging
- Auto-moderation

### User Features
- Profiles with avatars and status
- Friend system
- Notification controls
- Theme support

### Security
- End-to-end encryption for DMs
- JWT authentication with refresh tokens
- Rate limiting
- CORS and CSRF protection

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Frontend  │────▶│   Backend   │────▶│  Database   │
│  SvelteKit  │◀────│     Go      │◀────│ PostgreSQL  │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                    ┌──────▼──────┐
                    │  WebSocket  │
                    │   Gateway   │
                    └─────────────┘
```

| Component | Technology |
|-----------|------------|
| Backend | Go + Chi router |
| Frontend | SvelteKit + TypeScript + Tailwind |
| Database | PostgreSQL (SQLite for development) |
| Real-time | WebSocket |
| Storage | Local filesystem or S3-compatible |
| Cache | Redis (optional) |

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/ghndrx/hearth.git
cd hearth
cp .env.example .env
docker compose up -d
```

Access at `http://localhost:3000`

### Manual Installation

**Prerequisites:**
- Go 1.21+
- Node.js 20+
- PostgreSQL 15+ (or SQLite)

```bash
# Clone
git clone https://github.com/ghndrx/hearth.git
cd hearth

# Backend
cd backend
go build -o hearth ./cmd/hearth
./hearth

# Frontend (separate terminal)
cd frontend
npm install
npm run dev
```

## Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `sqlite://hearth.db` |
| `JWT_SECRET` | Secret for JWT signing | (required) |
| `REDIS_URL` | Redis connection string | (optional) |
| `S3_BUCKET` | S3 bucket for file storage | (optional) |
| `PORT` | Backend port | `8080` |

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for full configuration reference.

## Deployment

### Docker

```bash
docker pull ghcr.io/ghndrx/hearth:latest
docker run -d -p 8080:8080 -e DATABASE_URL=... ghcr.io/ghndrx/hearth
```

### Kubernetes

```bash
helm repo add hearth https://charts.hearth.chat
helm install hearth hearth/hearth -f values.yaml
```

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for detailed deployment guides.

## API

Hearth exposes a REST API and WebSocket gateway.

- REST API: `GET/POST/PUT/DELETE /api/v1/*`
- WebSocket: `ws://host/api/gateway`

API documentation: [docs/API.md](docs/API.md)

## Development

```bash
# Run tests
cd backend && go test ./...
cd frontend && npm test

# Lint
cd backend && golangci-lint run
cd frontend && npm run lint

# Build
docker compose -f docker-compose.dev.yml up
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## Project Status

Hearth is in active development. Current version: **v0.1.0**

### Implemented
- User authentication
- Servers, channels, messages
- Real-time WebSocket gateway
- Roles and permissions
- Invites
- Basic moderation

### Roadmap
- Voice channels (WebRTC)
- Video calls
- File sharing
- Search
- Mobile apps
- Desktop app

See [CHANGELOG.md](CHANGELOG.md) for release history.

## Related Repositories

- [hearth-desktop](https://github.com/ghndrx/hearth-desktop) - Tauri desktop client
- [hearth-mobile](https://github.com/ghndrx/hearth-mobile) - React Native mobile app

## License

MIT License. See [LICENSE](LICENSE).

## Contributing

Contributions welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).
