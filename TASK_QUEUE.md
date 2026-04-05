# TASK_QUEUE.md - Hearth Development Queue

> Last updated: 2026-04-05

## 🚨 P0 - Matrix Federation (MAIN PRIORITY)

> **Matrix Federation turns Hearth into a true decentralized platform.**
> DMs and channels across self-hosted Hearth instances, compatible with Matrix.org ecosystem.

- [ ] **[P0]** Feature: Matrix Federation Core — Server-Server federation API, MXID identity, room state, event signing, key distribution (16-24 weeks)
  - PRD: `PRDs/2026-04-05_matrix-federation-core.md`
  - **Phase 1**: Identity layer (MXID, homeserver config, profile API)
  - **Phase 2**: Client-Server core (sync, join/leave, send/receive)
  - **Phase 3**: Server-Server federation (send_join, backfill, state resolution)
  - **Phase 4**: Room directory & discovery
  - **Phase 5**: Cross-server DMs

## 📋 P1 - Essential Parity & Platform

- [ ] **[P1]** Feature: Native Mobile Applications — iOS/Android apps, 70% of users are mobile-first (12-16 weeks)
- [ ] **[P1]** Feature: Forum Channels — Threaded discussions with tagging (8-12 weeks)
- [ ] **[P1]** Feature: Interactive Message Components — Buttons, select menus, modals for bot interactions (10-14 weeks)
- [ ] **[P1]** Feature: Slash Commands Framework — Application commands with autocomplete (10-14 weeks)
- [ ] **[P1]** Feature: Mobile Push Notifications — Granular push with notification intelligence (8-12 weeks)

## 🛡️ P1 - Security & Safety

- [ ] **[P1]** Feature: Advanced Content Safety System — NSFW controls, age verification, content filtering (8-12 weeks)
- [ ] **[P1]** Feature: Advanced Moderation Tools — Automated mod, audit logs, mod dashboard (8-12 weeks)

## 💰 P1 - Monetization

- [ ] **[P1]** Feature: Premium Subscription System — Tiered subscriptions, server boosts (10-15 weeks)
- [ ] **[P1]** Feature: Stage Channels — Live audio broadcasts with speaker management (8-10 weeks)

## ✅ Recently Completed

- Video Calling (LiveKit) ✅
- Screen Sharing ✅
- Reactions System ✅
- Global Cross-Server Search ✅
- Server Discovery Directory ✅
- Soundboard ✅
- Message Edit History ✅
- Temporary Bans ✅
- WebRTC Voice Infrastructure ✅
- Desktop Themes ✅
- Mobile Themes + Widgets ✅

## ⚠️ Open PRs

| # | Title | Status |
|---|-------|--------|
| #124 | `fix(frontend): resolve svelte-check and type errors for CI` | Open |

---

*Matrix Federation is the core architectural priority. All other work is secondary until federation is shipped.*
