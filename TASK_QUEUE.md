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

## 2026-03-29 Vulnerability Findings

### P0 Security Issues

- [ ] **[P0] Security (Go stdlib)**: GO-2026-4602 — `os.File.ReadDir` / `os.ReadDir` can escape from Root — affects `hearth` binary — FIXED IN: go1.25.8 (current: go1.24.13) — https://pkg.go.dev/vuln/GO-2026-4602

- [ ] **[P0] Security (Go stdlib)**: GO-2026-4601 — Incorrect parsing of IPv6 host literals in `net/url` (`url.Parse`, `url.ParseRequestURI`, `url.URL.Parse`, `url.URL.UnmarshalBinary`) — affects `hearth` binary — FIXED IN: go1.25.8 (current: go1.24.13) — https://pkg.go.dev/vuln/GO-2026-4601

## 2026-03-29 Competitive Pipeline

- [ ] **[P0]** Feature: Enhanced Server Discovery — Public server directory with search, categories, and recommendations for user acquisition
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

## 2026-03-31 Competitive Pipeline

- [ ] **[P0]** Feature: Voice Messages — Audio message recording and playback in text channels for mobile-first communication
- [ ] **[P0]** Feature: Enhanced Mobile Experience — Mobile-first UI patterns and optimizations for competitive mobile parity
- [ ] **[P1]** Feature: Server Insights Dashboard — Analytics and metrics dashboard for community management and growth tracking

## 2026-03-31 Critical Feature Gap Analysis

- [ ] **[P0]** Feature: Bot API and Developer Platform — Essential developer ecosystem with bot creation, OAuth2, and API access for third-party integrations
- [ ] **[P0]** Feature: Live Video Streaming — Go Live functionality for streaming games/applications to voice channel audiences
- [ ] **[P0]** Feature: Advanced Permissions System — Fine-grained permission controls with role hierarchies and channel overrides
