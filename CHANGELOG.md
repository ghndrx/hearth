# Changelog

All notable changes to Hearth will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Initial project structure (Go backend, SvelteKit frontend)
- User authentication (register, login, JWT)
- Server management (create, join, update, delete)
- Channel management (text channels, categories)
- Real-time messaging via WebSocket
- Message features (edit, delete, reactions, replies)
- Role management with permissions
- Invite system with expirable links
- Member management (kick, ban)
- Typing indicators
- User presence/status
- Search functionality (messages, users, channels)
- Discord-style UI components
- Desktop client (Tauri + Svelte) - scaffold
- Mobile client (React Native + Expo) - scaffold

### Backend
- RESTful API with Chi router
- PostgreSQL database with migrations
- WebSocket hub for real-time events
- Comprehensive test coverage (130+ tests)
- Service layer architecture
- Middleware (auth, rate limiting, permissions)
- Bcrypt worker pool for bounded CPU usage (prevents p99 latency spikes under load)

### Frontend
- SvelteKit with TypeScript
- Tailwind CSS with Discord color palette
- Component library (ServerIcon, ChannelList, MessageList, etc.)
- Zustand-style stores
- WebSocket integration

---

## Release Process

To create a new release:

```bash
# Single repo
~/clawd/scripts/release.sh ~/clawd/hearth v0.1.0

# All repos (hearth, hearth-desktop, hearth-mobile)
~/clawd/scripts/release-all.sh v0.1.0
```

This will:
1. Squash merge `develop` → `master`
2. Create annotated tag
3. Generate changelog
4. Create GitHub release
5. Push everything

---

## Version History

| Version | Date | Milestone |
|---------|------|-----------|
| v0.1.0 | TBD | MVP - Core messaging |
| v0.2.0 | TBD | Voice channels |
| v0.3.0 | TBD | File sharing |
| v1.0.0 | TBD | Production ready |
