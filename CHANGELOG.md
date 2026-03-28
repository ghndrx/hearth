# Changelog

All notable changes to Hearth will be documented in this file.

## [2026-03-26](https://github.com/ghndrx/hearth/compare/2026-03-25...HEAD)

### Features
* feat: implement DMs and Group DMs
* feat(threading): add message threads with auto-creation and archive (#62)
* feat(webhooks): add webhook system for external integrations (#61)
* feat(user-status): add custom status with emoji, message, and expiration (#60)
* feat(frontend): add scheduled events UI to server settings (#54)
* feat(messages): add sticker_id support to message sending (#51)
* feat(discovery): implement guild discovery and browse system (#53)
* feat(auth): add FusionAuth SSO provider and deployment configs (#50)
* feat(custom-status): wire up UserStatusPicker in UserPanel (#49)
* feat(metrics): add Prometheus metrics endpoint and Grafana dashboard (#47)
* feat(soundboard): implement soundboard system (#46)
* feat: implement live streaming to channels (P0)
* feat(events): wire up EventService and EventHandler to complete backend integration (#45)
* feat(templates): implement server templates feature (#44)
* feat(automod): wire up AutoMod handler, routes, and repository (#37)
* feat(ratelimit): wire up api/middleware.RateLimiter with Redis-backed implementation (#36)
* feat(competitive): major breakthrough - 82% Discord parity achieved
* feat: implement message components with interactive buttons and select menus (#23)

### Bug Fixes
* fix: update mocks to implement interface methods after feature merges
* fix: add missing ValidateRefreshToken to mockSessionService
* fix: add missing stickerID argument to SendMessage call in test
* fix(ratelimit): tune aggressive rate limits, add per-endpoint limits (#48)
* fix: add missing fetchApi export and mock Publish calls
* fix(frontend): add missing fetchApi export to $lib/api
* fix(security): patch npm vulnerabilities (undici, cookie, flatted, devalue) (#43)
* fix: correct MockServerRepository signature for AddMember
* fix(security): patch P0 vulnerabilities in undici and flatted (#35)
* fix(websocket): add debugging logs and improve token extraction (#22)
* fix: add missing StorageRepository methods
* fix: add missing StorageRepository methods and remove broken automod files

### Other Changes
* fix: add missing stickerID argument to SendMessage call in test
* test(coverage): improve api handlers test coverage (#59)
* test(coverage): improve websocket package test coverage (#58)
* test(coverage): improve models package test coverage (#57)
* test(coverage): improve metrics package test coverage (#56)
* test(coverage): improve metrics package test coverage (#55)
* chore/tech debt 20260325 (#52)
* docs: auto-generate changelog for 2026-03-25
* chore: address tech debt in component handler
* docs: add multi-agent workflow with worktree pool management
* docs: update CLAUDE.md to exclude audit reports
* chore: remove audit reports from repo, add to .gitignore
* chore: remove 'AI' references from workflows
* docs: add PR template without AI attribution
* docs: add CLAUDE.md with no-AI-attribution guidelines

### Miscellaneous
* Merge feature/scheduled-events-frontend into develop
* Merge branch 'develop' of https://github.com/ghndrx/hearth into chore/tech-debt-20260325
* Merge: resolve conflicts keeping ScreenShare, AutoMod, Templates handlers

## [2026-03-25](https://github.com/ghndrx/hearth/compare/2026-03-24...HEAD)

### Features
* feat(competitive): major breakthrough - 82% Discord parity achieved

### Bug Fixes
* fix: add missing StorageRepository methods
* fix: add missing StorageRepository methods and remove broken automod files

### Other Changes
* docs: add multi-agent workflow with worktree pool management
* docs: update CLAUDE.md to exclude audit reports
* chore: remove audit reports from repo, add to .gitignore
* chore: remove 'AI' references from workflows
* docs: add PR template without AI attribution
* docs: add CLAUDE.md with no-AI-attribution guidelines

### Miscellaneous
* 9 commits total


## [2026-03-24](https://github.com/ghndrx/hearth/compare/2026-03-24
2026-01-01...HEAD)

### Features
* feat(stickers): integrate StickerManager into ServerSettings sidebar
* feat(stickers): add sticker system (backend models, services, handlers)
* feat(analysis): add competitive intelligence pipeline with Discord feature gap analysis
* feat(e2ee): add E2EESessionManager with Double Ratchet support
* feat(e2ee): integrate E2EE frontend with backend and gateway
* feat(e2ee): add E2EE frontend module with X3DH implementation
* feat(e2ee): wire up E2EE service, handler, and routes to complete Phase 1 integration
* feat: add guild-discovery route and page for server exploration

### Bug Fixes
* fix(storage): add empty bucket validation to S3 backend and update tests
* fix(stickers): add proper type annotations to StickerPicker API calls
* fix(messages): handle null from decryptMessage in E2EE flow
* fix(storage): add browser import for environment check
* fix(frontend): resolve SubtleCrypto argument type errors in encryption tests
* fix(frontend): resolve SubtleCrypto argument type errors in encryption tests
* fix(frontend): resolve SubtleCrypto SharedArrayBuffer error in encryption tests
* fix: resolve TypeScript errors in safety-number.ts (BufferSource type, byteLength)
* fix: upgrade grpc to v1.79.3 to resolve CVE-2026-33186
* fix: resolve CI failures from mock type mismatch and missing exports
* fix(frontend): resolve SubtleCrypto ArrayBuffer issue in encryption tests
* fix: resolve duplicate declarations in message_service.go
* fix(livekit): update LiveKitManager for livekit-client 2.17.2 API compatibility

### Other Changes
* fix(storage): add empty bucket validation to S3 backend and update tests
* docs: auto-generate changelog for 2026-03-24
* docs: add 2026-03-24 competitive analysis PRDs and feature gap report
* fix(frontend): resolve SubtleCrypto argument type errors in encryption tests
* fix(frontend): resolve SubtleCrypto argument type errors in encryption tests
* fix(frontend): resolve SubtleCrypto SharedArrayBuffer error in encryption tests
* fix(frontend): resolve SubtleCrypto ArrayBuffer issue in encryption tests

### Miscellaneous
* Merge branch 'feature/20260324-stickers-system' into develop

## [2026-03-24](https://github.com/ghndrx/hearth/compare/2026-01-01...HEAD)

### Features
* feat(stickers): integrate StickerManager into ServerSettings sidebar
* feat(stickers): add sticker system (backend models, services, handlers)
* feat(analysis): add competitive intelligence pipeline with Discord feature gap analysis
* feat(e2ee): add E2EESessionManager with Double Ratchet support
* feat(e2ee): integrate E2EE frontend with backend and gateway
* feat(e2ee): add E2EE frontend module with X3DH implementation
* feat(e2ee): wire up E2EE service, handler, and routes to complete Phase 1 integration
* feat: add guild-discovery route and page for server exploration

### Bug Fixes
* fix(stickers): add proper type annotations to StickerPicker API calls
* fix(messages): handle null from decryptMessage in E2EE flow
* fix(storage): add browser import for environment check
* fix(frontend): resolve SubtleCrypto argument type errors in encryption tests
* fix(frontend): resolve SubtleCrypto argument type errors in encryption tests
* fix(frontend): resolve SubtleCrypto SharedArrayBuffer error in encryption tests
* fix: resolve TypeScript errors in safety-number.ts (BufferSource type, byteLength)
* fix: upgrade grpc to v1.79.3 to resolve CVE-2026-33186
* fix: resolve CI failures from mock type mismatch and missing exports
* fix(frontend): resolve SubtleCrypto ArrayBuffer issue in encryption tests
* fix: resolve duplicate declarations in message_service.go
* fix(livekit): update LiveKitManager for livekit-client 2.17.2 API compatibility

### Other Changes
* docs: add 2026-03-24 competitive analysis PRDs and feature gap report
* fix(frontend): resolve SubtleCrypto argument type errors in encryption tests
* fix(frontend): resolve SubtleCrypto argument type errors in encryption tests
* fix(frontend): resolve SubtleCrypto SharedArrayBuffer error in encryption tests
* fix(frontend): resolve SubtleCrypto ArrayBuffer issue in encryption tests

### Miscellaneous
* Merge branch 'feature/20260324-stickers-system' into develop

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
