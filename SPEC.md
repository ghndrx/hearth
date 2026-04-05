# Hearth — Architecture Overview

> **Core Principle**: Self-hosted Discord alternative with **Matrix Federation** as the primary architectural priority.

## Architecture

### Backend
- **Stack**: Go, PostgreSQL, Redis, WebSocket gateway
- **Location**: `backend/internal/`
- **Handlers**: `api/handlers/`, **WebSocket**: `websocket/`
- **Services**: `services/`, **Database**: `database/postgres/`

### Frontend
- **Stack**: SvelteKit, TypeScript, Bun
- **Location**: `frontend/src/`

### Mobile
- **iOS**: Swift, `mobile/ios/`
- **Android**: Kotlin, `mobile/android/`

## Core Systems

### 1. Matrix Federation (P0 — IN PROGRESS)
- **PRD**: `PRDs/2026-04-05_matrix-federation-core.md`
- MXID identity (`@user:homeserver.example.com`)
- Server-Server (Federation) API — compatible with Matrix.org spec
- Client-Server API — compatible with Element, Hydrogen, etc.
- Room state, event signatures, key distribution
- Cross-server DMs and room participation
- Room version 6, Matrix r0.6.1

### 2. Messaging (Shipped)
- Real-time WebSocket messaging
- End-to-end encryption (Signal Protocol + MLS — future)
- Message editing, reactions, threads
- Rich embeds, link previews

### 3. Voice & Video (Shipped)
- WebRTC voice connections via LiveKit
- Video calling (1-on-1, screen share)
- Voice activity detection

### 4. Server Management
- Roles and permissions (RBAC)
- Channels, categories, threads
- Invites, bans, moderation actions

## Development

- `make test` — run all tests
- `make lint` — run linters
- `docker compose -f deploy/docker/docker-compose.dev.yml up` — local dev
- `make ci` — full CI pipeline

## Documentation

- PRDs: `PRDs/` — feature specifications
- Security: `.github/SECURITY.md`
- Contributing: `CONTRIBUTING.md`
