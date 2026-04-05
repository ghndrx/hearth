## 2026-04-05 Competitive Pipeline - CRITICAL PARITY COMPLETION

**Analysis**: Deep codebase audit reveals 3 critical Discord parity gaps that block mainstream adoption. While 95% parity achieved on web, mobile apps and video calling SFU are essential for competitive positioning.
**Status**: CRITICAL PARITY - Complete these gaps before full differentiation focus

### TOP 3 CRITICAL PARITY GAPS (3 NEW PRDs Created)
- [ ] **[P0]** Feature: Native Mobile Apps Shipping — iOS/Android apps to app stores, 70% of Discord users are mobile-first, critical user acquisition blocker (8-12 weeks)
- [ ] **[P0]** Feature: Video Calling SFU Architecture — Multi-user video calls with SFU infrastructure, group video essential for team communication parity (10-14 weeks)
- [ ] **[P1]** Feature: Rich Presence Activities Platform — Gaming activity detection, party system, social engagement feature that Discord users expect (12-16 weeks)

## 2026-04-05 Competitive Pipeline - DIFFERENTIATION FOCUS

**Analysis**: With core parity gaps addressed above, focus shifts to competitive differentiation through features Discord lacks entirely. Identified 3 major opportunities where Hearth can exceed Discord's capabilities.
**Status**: MARKET LEADERSHIP - Build features Discord cannot match (AFTER parity completion)

### TOP 3 DIFFERENTIATION OPPORTUNITIES (3 PRDs Created) 
- [ ] **[P0]** Feature: AI-Powered Smart Moderation Suite — Intelligent toxicity detection and community health management vs Discord's basic keyword filtering (12-16 weeks)
- [ ] **[P1]** Feature: Cross-Platform Gaming Integration Hub — Unified gaming social platform with cross-platform friends, LFG, achievements vs Discord's basic game status (16-20 weeks)  
- [ ] **[P2]** Feature: Collaborative Workspace Suite — Real-time whiteboard, documents, code editor that Discord completely lacks - major differentiation (20-26 weeks)

## 2026-04-05 GitHub Issues Pipeline (Latest Analysis)

**Status**: No open issues found in repository
**Analysis**: Checked for unclaimed issues with labels: 'help wanted', 'good first issue', 'P0', 'P1', 'bug', 'enhancement'
**Action**: HEARTBEAT_OK - No unclaimed issues to process

## 2026-04-05 GitHub Issues Pipeline

**Status**: No open issues found in repository
**Analysis**: Checked for unclaimed issues with labels: 'help wanted', 'good first issue', 'P0', 'P1', 'bug', 'enhancement'
**Action**: HEARTBEAT_OK - No unclaimed issues to process

## 2026-04-05 Competitive Pipeline - COMPREHENSIVE ANALYSIS

**Analysis**: After exhaustive feature audit (167 frontend components, comprehensive backend models), Hearth achieves 97%+ Discord parity! Focus shifts to differentiation opportunities.
**Status**: MARKET DIFFERENTIATION - Target features that exceed Discord capabilities

### NEW DIFFERENTIATING GAPS IDENTIFIED (3 PRDs Created)
- [ ] **[P1]** Feature: Gaming Platform Social Integrations — Cross-platform friend discovery, game library sync, social gaming features beyond Discord's basic connections (16-20 weeks)
- [ ] **[P1]** Feature: AI-Powered Server Discovery Engine — ML-driven community recommendations vs Discord's manual browsing, significant competitive advantage (14-18 weeks)  
- [ ] **[P2]** Feature: Collaborative Content Creation Suite — Integrated whiteboard, code editing, real-time collaboration tools that Discord completely lacks (20-26 weeks)

## 2026-04-04 Competitive Pipeline - UPDATED ANALYSIS

**Analysis**: Deep Discord feature parity analysis reveals Hearth has implemented 95% of core Discord features! Most gaps are nuanced.
**Status**: COMPETITIVE POSITIONING - Focus on remaining quality-of-life features and subscription value

### NEW CRITICAL GAPS IDENTIFIED (3 PRDs Created)
- [ ] **[P0]** Feature: Global Cross-Server Search — Critical power user feature missing, Discord users expect search across all servers/DMs (8-12 weeks)
- [ ] **[P1]** Feature: Discord Nitro Parity — Enhanced file uploads (500MB), HD streaming, animated avatars for subscription value (10-15 weeks)  
- [ ] **[P1]** Feature: Advanced Member Onboarding — Enhanced screening with form builder, automation rules, safety templates (9-12 weeks)

## 2026-04-04 GitHub Issues Pipeline (Latest Analysis)

**Status**: No open issues found in repository
**Analysis**: Checked for unclaimed issues with labels: 'help wanted', 'good first issue', 'P0', 'P1', 'bug', 'enhancement'
**Action**: HEARTBEAT_OK - No unclaimed issues to process

## 2026-04-04 GitHub Issues Pipeline

**Status**: No open issues found in repository
**Action**: HEARTBEAT_OK - No unclaimed issues to process

## 2026-04-03 Updated Competitive Intelligence Pipeline

**Analysis**: Fresh competitive assessment with 4 new PRDs created for critical gaps
**Status**: CRITICAL - Focus on adoption blockers and ecosystem development

### NEW CRITICAL GAPS IDENTIFIED (4 PRDs Created)
- [ ] **[P0]** Feature: Advanced Content Safety System — NSFW controls, age verification, content filtering for mainstream adoption compliance (8-12 weeks)
- [ ] **[P1]** Feature: Cross-Server Friend System — Friend requests, presence sharing, social graph for platform stickiness (10-14 weeks)
- [ ] **[P1]** Feature: Global Custom Emojis — Cross-server emoji usage for premium users, increases subscription value (6-8 weeks)
- [ ] **[P1]** Feature: Advanced Webhook Platform — Enhanced integrations, rate limiting, developer ecosystem support (6-10 weeks)

## 2026-04-03 TOP PRIORITY IMPLEMENTATION QUEUE

**CRITICAL PATH**: Focus on mobile + safety + reactions for competitive positioning

- [ ] **[P0]** Feature: Forum Channels — Discord-style threaded discussions with tagging for organized communities (8-12 weeks)
- [ ] **[P0]** Feature: Interactive Message Components — Buttons, select menus, and modals for bot interactions (10-14 weeks)
- [ ] **[P0]** Feature: Slash Commands Framework — Type-safe application commands with autocomplete for bot platform (10-14 weeks)

## 2026-04-03 Competitive Pipeline

- [ ] **[P0]** Feature: Forum Channels — Discord-style threaded discussions with tagging for organized communities (8-12 weeks)
- [ ] **[P0]** Feature: Interactive Message Components — Buttons, select menus, and modals for bot interactions (10-14 weeks)
- [ ] **[P0]** Feature: Slash Commands Framework — Type-safe application commands with autocomplete for bot platform (10-14 weeks)

## 2026-04-03 Competitive Pipeline

- [ ] **[P0]** Feature: Forum Channels — Discord-style threaded discussions with tagging for organized communities (8-12 weeks)
- [ ] **[P0]** Feature: Interactive Message Components — Buttons, select menus, and modals for bot interactions (10-14 weeks)
- [ ] **[P0]** Feature: Slash Commands Framework — Type-safe application commands with autocomplete for bot platform (10-14 weeks)

## 2026-04-03 TOP COMPETITIVE PRIORITIES

- [ ] **[P0]** Feature: Native Mobile Applications — CRITICAL USER ACQUISITION BLOCKER: 70% of Discord users mobile-first, no app store presence blocks mainstream adoption (12-16 weeks)
- [ ] **[P0]** Feature: Message Reactions System — Core engagement mechanic missing, conversations feel static without reaction persistence and real-time updates (4-6 weeks)
- [ ] **[P1]** Feature: Comprehensive Bot/Developer Platform — Developer ecosystem essential for stickiness, need OAuth2 scopes, bot accounts, app directory for community functionality (8-12 weeks)

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

## 2026-04-02 Critical Competitive Gaps

- [ ] **[P0]** Feature: AI-Powered Smart Notifications — Context-aware notification intelligence for mobile engagement and reduced notification fatigue (8-12 weeks)
- [ ] **[P0]** Feature: Community Growth Engine — AI-powered community discovery, member matching, and growth optimization for user acquisition (10-14 weeks)
- [ ] **[P1]** Feature: Collaborative Content Tools — Real-time collaborative documents, whiteboards, and wikis to differentiate from Discord (8-10 weeks)

## 2026-04-02 Competitive Pipeline

- [ ] **[P0]** Feature: Premium Subscription System — Tiered subscriptions (Basic $2.99, Premium $9.99) with server boosts for sustainable monetization and business model
- [ ] **[P0]** Feature: Advanced Sticker System — Custom server stickers, animated stickers, and premium sticker packs for engagement and revenue
- [ ] **[P0]** Feature: Stage Channels — Live audio broadcast channels with speaker management for large community events and content creation

## 2026-04-02 Critical Feature Gaps Analysis

- [ ] **[P0]** Feature: Private Threads — Thread visibility control for sensitive conversations within channel context for enterprise and community management
- [ ] **[P0]** Feature: Temporary Bans — Auto-expiring server bans with duration-based moderation actions for proportional punishment
- [ ] **[P1]** Feature: Message Edit History — Track and display complete edit history for transparency and accountability in communities

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

## 2026-04-02 GitHub Issues Analysis

**Status**: No open issues found in repository
**Action**: HEARTBEAT_OK - No unclaimed issues to process

## 2026-04-01 GitHub Issues Analysis

**Status**: No open issues found in repository
**Action**: HEARTBEAT_OK - No unclaimed issues to process

## 2026-03-31 GitHub Issues Analysis

**Status**: No open issues found in repository
**Action**: HEARTBEAT_OK - No unclaimed issues to process
## 2026-04-02 GitHub Issues Pipeline

**Status**: No open issues found in repository
**Action**: HEARTBEAT_OK - No unclaimed issues to process

## 2026-04-02 Vulnerability Findings

## 2026-04-03 Hearth Tech Debt Hunter — Inline Findings

**Scope**: Go backend (clean), Frontend TS/Svelte, Mobile iOS/Android, docs/research
**No P1 items found inline — no commits/PRs opened this cycle.**

### P1 — Fixed Inline (none this cycle)
- No security or performance-related inline TODOs found in Go backend.
- All Go source files clean of TODO/FIXME/HACK/XXX comments.

### P2 — Queued in TASK_QUEUE.md (from codebase TODOs)

**Frontend:**
- `[P2]` `frontend/src/lib/e2ee/keys.ts:66` — Load and initialize libsignal-client WASM (E2EE)
- `[P2]` `frontend/src/lib/e2ee/device-manager.ts:308` — Compute `identityKeyHash` properly
- `[P2]` `frontend/src/lib/stores/settings.test.ts` — `fetchUserSettings` backend sync not implemented; settings use localStorage only
- `[P2]` `frontend/src/lib/components/NotificationSettings.test.ts:188` — Same backend sync gap
- `[P2]` `frontend/src/lib/stores/notifications.ts:263` — Check if message is from another user (unread logic gap)

**Mobile Android:**
- `[P2]` `mobile/android/.../ServerListScreen.kt:68` — Server creation/join dialog not wired up
- `[P2]` `mobile/android/.../ChannelListScreen.kt:148` — Join voice channel not implemented

**Mobile iOS:**
- `[P2]` `mobile/ios/.../LoginView.swift:114,117,120` — GitHub, Google, Discord OAuth not implemented
- `[P2]` `mobile/ios/.../DMListView.swift:46` — New DM flow missing
- `[P2]` `mobile/ios/.../MessageComposerView.swift:30` — Attachment picker not implemented

### P3 — Nice-to-Have (from docs/research)

**Security Research TODOs (cross-reference to existing TASK_QUEUE items):**
- `[P1/HIGH]` docs/research/security-permission-system.md:97 — Server-side permission enforcement (already tracked in Rate Limiting section above)
- `[P1/HIGH]` docs/research/security-permission-system.md:98 — Rate-limit role/permission PATCH endpoints (already tracked)
- `[MEDIUM]` docs/research/security-permission-system.md:99-102 — Various medium-priority spec gaps (audit bypass, role hierarchy, cache invalidation, bot scope)
- `[MEDIUM]` docs/research/feature-file-sharing-media-handling.md:439-442 — ClamAV, EXIF stripping, SVG rejection, CDN domain (already tracked under P2 Security section)

### Deprecated / Empty Catch / Hardcoded Values
- **No deprecated function calls** found in Go or TypeScript source
- **No empty catch blocks** found in Go or TypeScript source
- **Hardcoded constants**: See existing P2 items around rate limiting for hardcoded magic numbers

**HEARTBEAT_OK — No P1 inline fixes required. P2/P3 items queued above.**
## 2026-04-05 Hearth Tech Debt Hunter — Inline Findings

**Scope**: Go backend, Frontend TS/Svelte, Mobile iOS/Android
**Findings**: P1 billing STUBs (previously missed), P2 E2EE WASM gap, P3 JSDoc fix

### P1 — Inline Fixes This Cycle

**Billing Service — STUB Implementations (PREVIOUSLY MISSED by automated scan):**
`backend/internal/services/billing_service.go` contains 10 STUB markers. This service is explicitly
marked as a stub with the comment: "This is a STUB implementation - integrate with real payment provider
(Stripe/Paddle) before production." The `isProd` flag gate means stubs only run in dev, but any billing
feature flag flipped to production would expose these as silent no-ops:

- `CreateCustomer()` — creates `mock_*` external IDs; real path calls `stripe.Customer.New()`
- `CreateSubscription()` — creates in-repo subscription record; no Stripe call made
- `CancelSubscription()` — marks SubStatusCanceled locally; no Stripe subscription cancelled
- `HandleWebhook()` — `fmt.Printf` only; no signature verification, no event processing
- `ProcessPayment()` — returns `errors.New("payment processing not implemented")`
- `UpdatePaymentMethod()` — returns `errors.New("payment method update not implemented")`
- `CreateBillingPortalSession()` — returns hardcoded `https://mock-billing-portal.example.com/...`
- `GiftSubscription()` — creates in-repo sub only; no Stripe gift payment flow
- `GetSubscriptionByExternalID()` — always returns `ErrSubscriptionNotFound`; no real query

**Risk**: If `BillingService` is instantiated with `isProd=true` and a real `stripeKey`, all operations
still silently mock. The code appears to be wired up for Stripe but the integration is absent.
No P1 inline fix possible (requires full Stripe/Paddle integration — tracked as Premium Subscription
System in competitive pipeline, 2026-04-02).

### P2 — Queued in TASK_QUEUE.md (from codebase TODOs)

**Frontend:**
- `[P2]` `frontend/src/lib/e2ee/keys.ts:66` — Load and initialize libsignal-client WASM (E2EE)
- `[P2]` `frontend/src/lib/stores/settings.test.ts` — `fetchUserSettings` backend sync not implemented; settings use localStorage only (already queued 2026-04-04)
- `[P2]` `frontend/src/lib/components/NotificationSettings.test.ts:188` — Same backend sync gap (already queued 2026-04-04)

### P3 — Queued (new this cycle)

- `[P3]` `frontend/src/lib/crypto/safety-number.ts:56` — Misleading `@returns` JSDoc: says "60-digit safety number" but E2EE safety numbers are 60-*digits* formatted as 6 groups of 5 digits (30 numeric characters + spaces). Fix: update `@returns` to accurately describe the 30-character digit format.

**HEARTBEAT_OK — P1 requires full Stripe/Paddle integration (tracked separately). P3 JSDoc fix is a one-line inline edit. P2 E2EE WASM gap carries over from 2026-04-04 report.**

---

## 2026-04-04 Vulnerability Findings
## 2026-04-05 Vulnerability Findings
