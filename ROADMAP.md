# Hearth Roadmap

## Current Status: Alpha (v0.1.0)

---

## 🔴 Priority 1: Critical Fixes (Sprint 1)
*Blocking basic functionality*

- [x] Auth (login/register) working
- [ ] **Server persistence** - Servers disappear after refresh (DB column mismatch - FIX COMMITTED)
- [ ] **Channel creation UI** - No way to create channels in a server
- [ ] **guild-discovery 404** - Missing route
- [ ] **Layout polish** - Match Discord's sidebar structure

---

## 🟠 Priority 2: Core Features (Sprint 2-3)
*Essential for usable chat app*

### Friends System
- [ ] Friends list with online/offline status
- [ ] Friend requests (pending incoming/outgoing)
- [ ] Block users
- [ ] Remove friend
- [ ] "Active Now" sidebar showing what friends are doing

### DM Improvements
- [ ] DM list in left sidebar (under Friends)
- [ ] Group DMs
- [ ] DM search
- [ ] Close DM button

### Server Features
- [ ] Server icon upload
- [ ] Server settings modal (full)
- [ ] Server invite system
- [ ] Channel categories (folders)
- [ ] Channel reordering (drag & drop)
- [ ] Role management UI
- [ ] Member list with role colors

### Messages
- [ ] Message editing
- [ ] Message deletion
- [ ] Reply to message
- [ ] Reactions
- [ ] Embeds (link previews)
- [ ] File uploads
- [ ] Image previews
- [ ] Emoji picker improvements

---

## 🟡 Priority 3: Enhanced Features (Sprint 4-5)
*Better UX, more Discord-like*

### User Settings (Full)
- [ ] **My Account**
  - [ ] Display name edit
  - [ ] Username change
  - [ ] Email change
  - [ ] Phone number
  - [ ] Password change
  - [ ] Two-factor auth (2FA)
  - [ ] Account deletion
- [ ] **User Profile**
  - [ ] Avatar upload (working)
  - [ ] Banner upload
  - [ ] About me bio
  - [ ] Pronouns
  - [ ] Custom status
- [ ] **Privacy & Safety**
  - [ ] Who can DM you
  - [ ] Who can friend request you
  - [ ] Blocked users list
  - [ ] Data download request
- [ ] **Connections**
  - [ ] Link external accounts (GitHub, Spotify, etc.)
- [ ] **Notifications**
  - [ ] Desktop notifications
  - [ ] Sound settings
  - [ ] Per-server notification overrides
- [ ] **Appearance**
  - [ ] Theme (dark/light/OLED)
  - [ ] Font scaling
  - [ ] Compact mode
  - [ ] Message grouping
- [ ] **Voice & Video**
  - [x] Input device selection
  - [x] Output device selection
  - [x] Camera selection
  - [ ] Noise suppression
  - [ ] Echo cancellation
  - [ ] Push-to-talk
  - [ ] Voice activity detection
- [ ] **Keybinds**
  - [x] View keybinds
  - [ ] Custom keybind editing

### Server Settings (Full)
- [ ] Overview (name, icon, banner)
- [ ] Roles (create, edit, delete, reorder)
- [ ] Emoji management
- [ ] Moderation (auto-mod rules)
- [ ] Audit log viewer
- [ ] Integrations/webhooks
- [ ] Widget settings
- [ ] Server template

---

## 🟢 Priority 4: Advanced Features (Sprint 6+)
*Nice-to-have, power user features*

### Voice & Video
- [ ] Voice channels (join/leave)
- [ ] Voice channel UI (connected users)
- [ ] Screen sharing
- [ ] Video calls
- [ ] Go Live (streaming)
- [ ] Stage channels

### Discovery
- [ ] Server discovery/explore
- [ ] Server search
- [ ] Server categories
- [ ] Vanity URLs

### Threads
- [ ] Create thread from message
- [ ] Thread list
- [ ] Thread notifications

### Search
- [ ] Global message search
- [ ] Search filters (from:, in:, has:, before:, after:)
- [ ] Search results UI

### Mobile
- [ ] Responsive design
- [ ] PWA support
- [ ] Push notifications

### Self-Hosting
- [ ] One-click deploy scripts
- [ ] Docker Compose (done)
- [ ] Kubernetes manifests
- [ ] Documentation

---

## Tech Debt
- [ ] E2E encryption (libsignal)
- [ ] WebRTC for voice/video
- [ ] Rate limiting
- [ ] Caching layer (Redis)
- [ ] CDN for media
- [ ] Database migrations system
- [ ] API versioning
- [ ] OpenAPI docs
- [ ] Test coverage > 80%

---

## Timeline Estimate

| Sprint | Duration | Focus |
|--------|----------|-------|
| 1 | 1-2 days | Critical fixes |
| 2-3 | 1 week | Core features |
| 4-5 | 2 weeks | Enhanced features |
| 6+ | Ongoing | Advanced features |

---

*Last updated: 2026-02-15*
