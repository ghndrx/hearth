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
