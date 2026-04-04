# Changelog

All notable changes to Hearth will be documented in this file.

## [2026-04-04](https://github.com/ghndrx/hearth/compare/2026-04-03...HEAD)

### Features
* feat: video calling core system
* feat(channels): add forum channels with threaded discussions and tagging
* feat: add critical competitive feature gap PRDs and update task priorities

### Bug Fixes
* fix/clipboard polyfill test failure (#109)
* fix: remove duplicate declarations causing CI build failure
* fix: add missing GetTotalMessageCount to simpleMockThreadRepo
* fix(slashCommands): handle multi-word string args and fix autocomplete test (#106)
* fix(frontend): import ForumTag type in ForumChannelView (#103)
* fix(security): restore correct G115 guard in CheckMentionAbuse (#100)
* fix(frontend): correct discovery component tests for Svelte 5 compatibility (#96)
* fix(tests): polyfill navigator.clipboard for happy-dom compatibility (#95)
* fix(security): restore G115 integer overflow guard in CheckMentionAbuse (#94)

### Other Changes
* fix/clipboard polyfill test failure (#109)
* chore: fix critical P1 database error handling
* chore: address tech debt in backend/internal/websocket/gateway.go
* docs: auto-generate changelog for 2026-04-03
* fix(slashCommands): handle multi-word string args and fix autocomplete test (#106)
* test(handlers): add AppDirectoryHandler tests (#105)
* chore: address critical tech debt in websocket gateway (#98)
* test(handlers): add soundboard handler tests (#99)
* fix(frontend): correct discovery component tests for Svelte 5 compatibility (#96)
* fix(tests): polyfill navigator.clipboard for happy-dom compatibility (#95)

### Miscellaneous
* Merge remote-tracking branch 'origin/chore/tech-debt-20260403' into fix-ci
* Merge pull request #108 from ghndrx/chore/tech-debt-20260403
* security: fix G115 integer overflow in content filter via proper Unicode handling (#107)
* 🔒 Fix Critical P1 Security Vulnerabilities (#104)
* coverage/improve 20260403 00 (#90)

## [2026-04-03](https://github.com/ghndrx/hearth/compare/2026-04-02...HEAD)

### Features
* feat: comprehensive audit logs and moderation analytics
* feat(handlers): add HTTP handler tests for LiveKitVoiceHandler (#93)
* feat: implement AI-powered smart notifications system
* feat: AI-Powered Smart Notifications system
* feat: add critical competitive advantage PRDs and update task priorities
* feat(stickers): Advanced Sticker System - premium packs and subscription tiers (#82)
* feat(premium): Premium Subscription System - Tiered subscriptions (Basic $2.99, Premium $9.99) with server boosts
* feat: add critical competitive feature gap PRDs and task queue updates

### Bug Fixes
* fix(config_test): add TestMain to set SECRET_KEY before tests run (#92)
* fix(tests): use regular function constructor for WebSocket mock (#89)
* fix(tests): add missing jumpToMessage mock to MessageList.test.ts (#87)
* fix(tests): replace deprecated vi.resetModules and vi.stubGlobal in integration tests (#86)
* fix(frontend): update MessageComponents and Button tests for Svelte 5 compatibility (#84)
* fix(services): replace invalid regexp with manual check in content_filter.go (#81)
* fix(SelectMenu): initialize selectedValues from default option and fix tests (#80)
* fix(frontend): regenerate bun.lock with correct checksums
* fix(frontend): remove stale TODO in MessageArea - file preview already implemented (#79)
* fix(frontend): correct TextInput required attribute test (#77)
* fix(frontend): implement call initiation from user popout (#75)
* fix(frontend): add error logging to 13 empty catch blocks (#20)
* fix(ci): run svelte-kit sync before ESLint to generate .svelte-kit/tsconfig.json
* fix: resolve 4 CI failures in devsecops workflow
* fix: resolve 3 CI failures in devsecops workflow
* fix(ci): correct devsecops workflow syntax
* Fix critical service initialization tech debt (#110)

### Other Changes
* refactor(handlers): add interfaces for LiveKitVoiceHandler dependency injection (#91)
* chore: address tech debt in backend error handling and configuration (#85)
* test(handler): add mock-based tests for AI chat template listing (#83)
* test: improve coverage for ai/providers, cache, ai packages (#78)
* chore: address tech debt in config/auth - CRITICAL SECURITY FIX (#76)
* docs: auto-generate changelog for 2026-04-02

### Other Changes
* feat(handlers): add HTTP handler tests for LiveKitVoiceHandler (#93)
* refactor(handlers): add interfaces for LiveKitVoiceHandler dependency injection (#91)
* fix(config_test): add TestMain to set SECRET_KEY before tests run (#92)
* fix(tests): use regular function constructor for WebSocket mock (#89)
* fix(tests): add missing jumpToMessage mock to MessageList.test.ts (#87)
* fix(tests): replace deprecated vi.resetModules and vi.stubGlobal in integration tests (#86)
* fix(frontend): update MessageComponents and Button tests for Svelte 5 compatibility (#84)
* chore: address tech debt in backend error handling and configuration (#85)
* test(handler): add mock-based tests for AI chat template listing (#83)
* fix(SelectMenu): initialize selectedValues from default option and fix tests (#80)
>>>>>>> 4d1ccba (docs: auto-generate changelog for 2026-04-03)

### Miscellaneous
  (none)

## [2026-04-02](https://github.com/ghndrx/hearth/compare/2026-04-01...HEAD)

### Features
* feat: implement group direct messages for 3-50 person conversations
* feat: add critical competitive feature PRDs - group DMs, channel categories, activity presence
* feat: implement interactive message components (buttons, dropdowns, modals)
* feat(events): add scheduled events with RSVP system and notifications
* feat(messages): add interactive message components (buttons, dropdowns, modals)
* feat(messages): add interactive message components (buttons, dropdowns, modals)
* feat: enhanced server discovery with search and recommendations
* feat: add public server discovery, search, categories, and recommendations
* feature/message file preview (#33)
* feat(message): add message forwarding functionality (#59)

### Bug Fixes
* fix(frontend): replace empty catch blocks with debug logging for media permission denial (#71)
* fix(frontend): use ES module imports in syncStatus test mocks (#68)
* fix(frontend): switch test environment from jsdom to happy-dom (#66)
* fix(frontend): remove invalid eslint-disable for non-existent svelte rule (#67)
* fix(frontend): remove vi.useRealTimers() calls that cause test timeouts (#65)
* fix: resolve CI failures from feat(messages) merge (run #231) (#62)
* fix/permission checks 20260401 (#64)
* fix(frontend): resolve TypeScript type error in EventDetail.svelte permissions check
* fix(backend): add missing mock methods for ComponentServiceInterface in tests
* fix(frontend): use @ts-expect-error instead of @ts-ignore in discovery test
* fix(frontend): resolve TypeScript type errors in role-service-permission-checks
* fix(discovery): link server verification from discoverable_servers table (#29)
* fix: add thread auto-archive handler and tests, update errors
* fix(frontend): replace jsdom with happy-dom for vitest compatibility (#35)
* fix(a11y): add tabindex to soundboard picker dialog (#34)
* fix: add MANAGE_EVENTS permission check for event management (#31)

### Other Changes
* chore: Tech debt cleanup - Remove debug console.log statements (#69)
* fix(frontend): use ES module imports in syncStatus test mocks (#68)
* fix(frontend): switch test environment from jsdom to happy-dom (#66)
* fix(frontend): remove vi.useRealTimers() calls that cause test timeouts (#65)
* chore: mark resolved Go CVE and Enhanced Server Discovery items done in task queue
* chore/tech debt 20260401 (#63)
* fix(backend): add missing mock methods for ComponentServiceInterface in tests
* chore: add GitHub issues analysis for 2026-04-01
* fix(frontend): use @ts-expect-error instead of @ts-ignore in discovery test
* chore(frontend): regenerate package-lock.json for npm ci compatibility
* fix: add thread auto-archive handler and tests, update errors
* test(coverage): add invite tracking handler tests (#36)
* fix(frontend): replace jsdom with happy-dom for vitest compatibility (#35)
* chore/tech debt 20260401 (#60)

### Miscellaneous
* security: fix G304 path traversal in local storage via os.Root (Go 1.24+) (#70)
* continue rebase
* security: fix G304 path traversal in local storage via os.Root (Go 1.24+) (#38)
* security: fix SQL injection vulnerability and tech debt analysis (#37)

## [2026-04-01](https://github.com/ghndrx/hearth/compare/2026-03-31...HEAD)

### Features
* feat(competitive): add critical Discord feature gap PRDs and update task queue
* feat(message): add message forwarding functionality
* feat(websocket): add per-IP and per-user connection limits (Redis-backed) (#57)
* feat(polls): implement PollRepository and wire up PollService (#56)
* feat: implement enhanced server discovery and public directory
* feat(rate limit): add per-user invite creation rate limit (5/hour) (#52)
* feat(discovery): wire up DiscoverableServer service and repository
* feat: add critical Discord feature gap PRDs for mobile parity and community management

### Bug Fixes
* fix(storage): G304 path traversal — use os.Root for atomic scope enforcement (#50)
* fix(api): correct route param names in thread auto-archive tests (#51)
* fix(security): prevent integer overflow in CheckMentionAbuse (G115) (#47)
* fix: resolve go vet failures in thread auto-archive tests (#49)
* fix(frontend): resolve CI type errors in discovery components (#48)
* fix(backend): P0 security — G115 integer overflow guard, G118 context propagation

### Other Changes
* chore: fix critical permission check in EventDetail component
* chore: remove dead legacy HTTP handler code (net/http style) (#58)
* chore: address tech debt in main.go (#54)
* test(api): add ComponentHandler tests and refactor to use interfaces (#55)
* fix(api): correct route param names in thread auto-archive tests (#51)
* fix: resolve go vet failures in thread auto-archive tests (#49)

### Miscellaneous
* coverage/improve 20260331 17 (#53)
* security: fix P0 vulnerabilities G118, G115, G304 (#45)
* security: fix P0 vulnerabilities G118, G115, G304 (#45)

## [2026-03-31](https://github.com/ghndrx/hearth/compare/2026-03-30...HEAD)

### Features
* feature/message file preview (#33)

### Bug Fixes
* fix(frontend): resolve TypeScript type errors in role-service-permission-checks
* fix(frontend): replace jsdom with happy-dom for vitest compatibility (#35)
* fix(a11y): add tabindex to soundboard picker dialog (#34)
* fix(frontend): resolve TypeScript null type errors in discovery components (#30)
* fix(frontend): resolve TypeScript error in EventDetail.svelte (#32)
* fix: add MANAGE_EVENTS permission check for event management (#31)
* fix(discovery): link server verification from discoverable_servers table (#29)

### Other Changes
* chore: update TASK_QUEUE.md with 2026-03-31 GitHub issues analysis
* chore(frontend): regenerate package-lock.json for npm ci compatibility
* fix(frontend): replace jsdom with happy-dom for vitest compatibility (#35)

### Miscellaneous
  (none)

## [2026-03-30](https://github.com/ghndrx/hearth/compare/2026-03-29...HEAD)

### Features
* feat: add enhanced server discovery with public directory
* feat(frontend): implement jump-to-message scroll from pinned/search results (#25)
* feat(MemberList): implement Send Message to DM navigation (#19)
* feature/cache redis miniredis coverage (#5)

### Bug Fixes
* fix(templates): capture server settings when serializing to template (#23)
* fix: resolve conflicts and add public server directory test mocks/routes (#22)
* fix(frontend): add error logging to 13 empty catch blocks (#20)
* fix: correct golangci-lint v2.11.4 config format
* fix: upgrade golangci-lint-action to v7 and use binary install mode for v2.x compatibility
* fix: add version field for golangci-lint v2.x compatibility

### Other Changes
* test(coverage): add discovery handler httptest coverage (#24)
* fix: resolve conflicts and add public server directory test mocks/routes (#22)
* docs: auto-generate changelog for 2026-03-29
* Add comprehensive tests for template handler endpoints (#21)
* test(coverage): add follow and reminders handler tests (#18)
* test(coverage): add notification repository test coverage (#17)
* test(coverage): add pins handler tests with httptest (#15)
* test(coverage): improve analytics handler test coverage with httptest (#14)
* test(coverage): add analytics repository test coverage (#13)
* test(coverage): add storage repository tests (#11)
* chore: address critical event management permission bypass (#12)
* test(coverage): add webhook repository test coverage (#9)
* test(coverage): improve cache test coverage with miniredis (#8)
* Add comprehensive tests for forum_tags helper functions and ListPosts endpoint (#6)

### Miscellaneous
* Add comprehensive tests for template handler endpoints (#21)
* ci: fix CodeQL SARIF upload permissions in DevSecOps workflow (#16)
* security: bump Go version to 1.25.8 in Dockerfiles (GO-2026-4601/4602/4603) (#10)
* security: upgrade Go to 1.25.8 (fixes GO-2026-4601, GO-2026-4602, GO-2026-4603) (#7)
* Add comprehensive tests for forum_tags helper functions and ListPosts endpoint (#6)

## [2026-03-29](https://github.com/ghndrx/hearth/compare/2026-03-28...HEAD)

### Features
* feat(MemberList): implement Send Message to DM navigation (#19)
* feature/cache redis miniredis coverage (#5)
* feat(app-directory): comprehensive app directory and bot store implementation (#123)
* feat(app-directory): comprehensive app directory and bot store implementation (#123)

### Bug Fixes
* fix(frontend): add error logging to 13 empty catch blocks (#20)
* fix: correct golangci-lint v2.11.4 config format
* fix: upgrade golangci-lint-action to v7 and use binary install mode for v2.x compatibility
* fix: add version field for golangci-lint v2.x compatibility
* fix(handlers): properly extract user from request context in getUser stub (#3)
* fix: resolve YAML syntax error in bedrock-agent workflow (#2)
* fix: correct test handler to avoid double-response in TestHandleServiceError (#65)
* fix: resolve WebSocket authentication failures (#31)
* fix(websocket): return early when no auth token provided to prevent ValidateAccessToken("") failures (#20)

### Other Changes
* test(coverage): add follow and reminders handler tests (#18)
* test(coverage): add notification repository test coverage (#17)
* test(coverage): add pins handler tests with httptest (#15)
* test(coverage): improve analytics handler test coverage with httptest (#14)
* test(coverage): add analytics repository test coverage (#13)
* test(coverage): add storage repository tests (#11)
* chore: address critical event management permission bypass (#12)
* test(coverage): add webhook repository test coverage (#9)
* test(coverage): improve cache test coverage with miniredis (#8)
* Add comprehensive tests for forum_tags helper functions and ListPosts endpoint (#6)
* style: apply gofmt to threads_coverage_test.go
* test(handlers): add httptest coverage for VoiceHandler and GatewayHandler (#97)
* fix: correct test handler to avoid double-response in TestHandleServiceError (#65)
* test(coverage): improve metrics package test coverage (#56)
* test(coverage): improve comprehensive service test coverage (#1)
* chore: remove bedrock-agent workflow - not needed
* chore: remove internal agent guidelines file

### Miscellaneous
* ci: fix CodeQL SARIF upload permissions in DevSecOps workflow (#16)
* security: bump Go version to 1.25.8 in Dockerfiles (GO-2026-4601/4602/4603) (#10)
* security: upgrade Go to 1.25.8 (fixes GO-2026-4601, GO-2026-4602, GO-2026-4603) (#7)
* Add comprehensive tests for forum_tags helper functions and ListPosts endpoint (#6)
* style: apply gofmt to threads_coverage_test.go

## [2026-03-28](https://github.com/ghndrx/hearth/compare/2026-03-27...HEAD)

### Features
* feat(app-directory): comprehensive app directory and bot store implementation (#123)
* feat(app-directory): comprehensive app directory and bot store implementation (#123)
* feat: app directory and onboarding/welcome screens (#117)
* feat(premium): finalize server boost & premium system implementation (#118)
* feat(frontend): add scheduled events UI components (#111)

### Bug Fixes
* fix(ci): run svelte-kit sync before ESLint to generate .svelte-kit/tsconfig.json
* fix: resolve 4 CI failures in devsecops workflow
* fix: resolve 3 CI failures in devsecops workflow
* fix(ci): correct devsecops workflow syntax
* Fix critical service initialization tech debt (#110)

### Other Changes
* chore: add CLAUDE.md, PLAN.md, PRD.md, ROADMAP.md, .claude/ to .gitignore
* chore: add comprehensive .gitignore for local-only files, update CLAUDE.md
* chore: remove ci-auto-fix workflow - it creates dirty commits with no actual fixes
* chore: remove auto-fix-conflicts workflow - it created dirty commit history
* test(frontend): fix vitest jsdom compatibility issues
* test(coverage): add channel mute handler coverage tests (#122)
* docs: auto-generate changelog for 2026-03-27
* test: add ThreadServiceInterface and handler coverage tests (#121)
* test: add handler coverage tests for AI endpoints (#114)
* test(coverage): improve middleware ratelimit adapter test coverage (#112)

### Miscellaneous
* security: upgrade Go to 1.24.13 (GO-2026-4337, GO-2025-4012, GO-2025-4011, GO-2025-4008, GO-2026-4601, GO-2025-4010, GO-2026-4341, GO-2025-4175, GO-2025-4155, GO-2025-4013, GO-2025-4007, GO-2025-4009, GO-2025-3956, GO-2026-4602, GO-2025-3849)
* security: patch picomatch ReDoS and method injection vulnerabilities (SEC-P0-001, SEC-P0-002) (#115)
* security: patch npm vulnerabilities (#108)

## [2026-03-27](https://github.com/ghndrx/hearth/compare/2026-03-26...HEAD)

### Features
* feat: add pinned messages, channel follow, reminders, and soundboard handlers
* feat(forum): implement forum channel tags frontend (#72)
* feat(invite): add vanity URLs and invite analytics (#75)
* feat(audio): add per-server audio settings and push-to-talk (#73)
* feat(animations): add smooth animations for components (#71)
* feat(theme): add system theme option with OS preference sync (#68)
* feat: implement slash commands and bot API ecosystem (#67)
* feat(forum): implement forum channel tags (#66)
* feature/sticker database persistence (#63)
* feat: add competitive analysis PRDs for critical Discord feature gaps
* feat: implement stage channels backend (#102)
* feat: merge webhook handler coverage and slash commands (#101)

### Bug Fixes
* fix: remove duplicate severity flag in gosec args
* fix: remove invalid claude auth configure step from auto-fix-conflicts workflow
* fix: use official npm install for Claude Code
* fix(tests): add missing mock methods for ServerRepository interface (#77)
* fix: correct test handler to avoid double-response in TestHandleServiceError (#65)
* fix: add missing interface methods to mock implementations (#64)

### Other Changes
* docs: strengthen guardrails against committing sensitive reports
* docs: add strategic PRDs for platform viability gaps
* test(coverage): improve storage test coverage (#80)
* test(coverage): improve models test coverage (#79)
* test(coverage): improve metrics test coverage from 71.5% to 99.3% (#78)
* test(coverage): improve postgres repository test coverage (#106)
* test(coverage): improve push and audio settings handler test coverage (#105)
* test(coverage): add comprehensive role_repo tests (#104)
* test(postgres): add InviteRepo and BanRepo test coverage (#103)

### Miscellaneous
* ci: enhance DevSecOps workflow - add SAST, quality gate, license audit
* security: validate server membership before sending mention notifications (#99)

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
* docs: add strategic PRDs for platform viability gaps
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
* fix(frontend): resolve SubtleCrypto ArrayBuffer issue in encryption tests
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
