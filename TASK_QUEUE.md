## Rate Limiting & Abuse Prevention (from security-ratelimiting-abuse.md)

- [ ] **[HIGH]** Add per-IP and per-user WebSocket connection count limits in gateway layer (Redis-backed)
- [ ] **[HIGH]** Implement message spam pattern detection (duplicate hash, mention flood, DM rate)
- [ ] **[MEDIUM]** Add CAPTCHA (reCAPTCHA v3) on login post-lockout and signup
- [ ] **[MEDIUM]** Add global rate limit counter and `X-RateLimit-Global-*` headers
- [ ] **[MEDIUM]** Add per-user invite creation rate limits (5/hour default)
- [ ] **[MEDIUM]** Emit structured abuse/security events on rate limit hits and login lockout
- [ ] **[LOW]** Change high-risk endpoints (auth, signup) to fail-closed when Redis unavailable
- [ ] **[LOW]** If federation in scope: add per-homeserver federation rate limits
- [ ] **[FUTURE]** Graduated sanctions engine (rate limit → soft mute → temp ban → suspension)

## 2026-04-02 Competitive Pipeline

- [ ] **[P0]** Feature: Group Direct Messages — Multi-user DM conversations for small friend groups (3-50 people), enabling casual group coordination outside server structure
- [ ] **[P0]** Feature: Channel Categories — Hierarchical channel organization with collapsible categories for server structure and navigation at scale
- [ ] **[P1]** Feature: Activity & Rich Presence System — Comprehensive activity detection including gaming, music, and custom activities for social engagement

## 2026-03-29 Vulnerability Findings

### P0 Security Issues

- [x] **[P0] Security (Go stdlib)**: GO-2026-4602 — `os.File.ReadDir` / `os.ReadDir` can escape from Root — FIXED: go1.25.8 deployed ✅ 2026-04-01
- [x] **[P0] Security (Go stdlib)**: GO-2026-4601 — Incorrect parsing of IPv6 host literals in `net/url` — FIXED: go1.25.8 deployed ✅ 2026-04-01

## 2026-03-29 Competitive Pipeline

- [x] **[P0]** Feature: Enhanced Server Discovery — Public server directory with search, categories, and recommendations for user acquisition ✅ 2026-04-01 (commits f119f3e, db25cf9)
- [ ] **[P0]** Feature: Scheduled Events — RSVP system, calendar integration, and event notifications for community building
- [ ] **[P1]** Feature: Comprehensive Sticker System — Custom stickers, animated stickers, and premium sticker packs for engagement and monetization
- [ ] **[P0]** Feature: Server Boosts & Premium Features — Subscription tiers with enhanced capabilities for monetization and sustainability
- [ ] **[P0]** Feature: Stage Channels — Live audio broadcasts for large audiences with speaker management and community building
- [ ] **[P1]** Feature: Advanced Message Search — Full-text search with filters for knowledge management in large communities

## 2026-03-30 Competitive Pipeline

- [ ] **[P0]** Feature: Server Boosts & Premium Subscriptions — Monetization model with tiered server enhancements and sustainable revenue
- [ ] **[P0]** Feature: Scheduled Events with RSVP — Community building through organized activities with calendar integration
- [ ] **[P1]** Feature: Advanced Moderation Tools — Enhanced safety through automated moderation, audit logs, and mod dashboard

## 2026-03-30 Critical Feature Gaps Analysis

- [ ] **[P0]** Feature: Soundboard Integration — Discord's most popular voice feature for audio clips and memes in voice channels
- [ ] **[P0]** Feature: Custom Emoji System — Server emoji upload, management, and cross-server usage for premium users
- [ ] **[P1]** Feature: Voice Activities & Games — Built-in activities like poker, chess, and watch-together for voice engagement

## 2026-03-30 Additional Critical Messaging Features

- [ ] **[P0]** Feature: Message Forwarding — Core messaging feature for sharing content across channels/servers with attribution
- [ ] **[P0]** Feature: Rich Embed Builder — Interactive embed creation for announcements, documentation, and bot responses
- [ ] **[P1]** Feature: Granular Notification Controls — Per-channel notification settings, keywords, and quiet hours for user experience

## 2026-03-31 Competitive Pipeline

- [ ] **[P0]** Feature: Mobile Push Notifications — Essential real-time mobile engagement with granular notification controls for user retention
- [ ] **[P0]** Feature: Advanced Message Search — Full-text search with filters and faceted search for large community knowledge management
- [ ] **[P1]** Feature: Voice Activities & Games — Interactive voice channel activities like poker, chess, and watch-together for engagement

## 2026-04-01 Competitive Pipeline

- [ ] **[P0]** Feature: Interactive Message Components — Buttons, dropdowns, and modals for modern bot interactions and user engagement
- [ ] **[P0]** Feature: Video Calling System — 1-on-1 and group video calls with screen sharing for complete Discord communication parity
- [ ] **[P0]** Feature: Native Mobile Applications — iOS and Android native apps for mobile-first user acquisition and retention
- [ ] **[P0]** Feature: Advanced Developer Platform & App Directory — Comprehensive developer ecosystem with app discovery, OAuth2 scopes, and monetization for third-party growth
- [ ] **[P0]** Feature: Voice Activities Platform — Interactive voice channel activities and games (poker, chess, YouTube together) for unique engagement
- [ ] **[P1]** Feature: Advanced Live Streaming Infrastructure — Go Live broadcasting with audience management and quality controls for content creators

## 2026-04-01 Competitive Pipeline (Critical Gaps Analysis)

- [ ] **[P0]** Feature: Video & Screen Share System — Essential communication parity with 1-on-1/group video calls and screen sharing capabilities
- [ ] **[P0]** Feature: Forum Channels — Thread-based discussion channels with tags, templates, and moderation tools for organized communities
- [ ] **[P0]** Feature: Server Templates — Pre-built server configurations for instant community setup and improved onboarding experience

## 2026-04-01 GitHub Issues Analysis

**Status**: No open issues found in repository
**Action**: HEARTBEAT_OK - No unclaimed issues to process

## 2026-03-31 GitHub Issues Analysis

**Status**: No open issues found in repository
**Action**: HEARTBEAT_OK - No unclaimed issues to process
