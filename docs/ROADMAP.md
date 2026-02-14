# Hearth — Development Roadmap

**Version:** 1.0  
**Last Updated:** 2026-02-11

---

## Overview

Hearth development is organized into phases, each delivering a usable milestone. The goal is always to have a deployable, functional application at the end of each phase.

---

## Phase 0: Foundation (Weeks 1-2)

**Goal:** Project scaffolding and core infrastructure.

### Deliverables
- [ ] Go project structure with Fiber
- [ ] SvelteKit frontend scaffold
- [ ] Database migrations (PostgreSQL + SQLite)
- [ ] Docker development environment
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Basic logging and error handling
- [ ] Configuration management
- [ ] Health check endpoints

### Exit Criteria
- `docker-compose up` starts all services
- `/health` returns 200
- Frontend loads at localhost:3000

---

## Phase 1: Authentication (Weeks 3-4)

**Goal:** User registration, login, and session management.

### Deliverables
- [ ] User registration (email + password)
- [ ] Password hashing (Argon2id)
- [ ] Login flow with JWT
- [ ] Refresh token rotation
- [ ] Email verification (optional)
- [ ] Password reset flow
- [ ] Session management (view/revoke)
- [ ] Basic user profile page
- [ ] Avatar upload

### Exit Criteria
- Users can register and log in
- Sessions persist across browser refresh
- Password reset emails send (if SMTP configured)

---

## Phase 2: Servers & Channels (Weeks 5-7)

**Goal:** Core server and channel management.

### Deliverables
- [ ] Create/edit/delete servers
- [ ] Server icon upload
- [ ] Create/edit/delete channels
- [ ] Channel types: text, category
- [ ] Channel ordering (drag-and-drop)
- [ ] Basic permission system
- [ ] Role CRUD
- [ ] Role hierarchy enforcement
- [ ] Member management (join, leave, kick)
- [ ] Invite generation and redemption

### Exit Criteria
- Users can create servers with channels
- Invites work to add new members
- Roles control basic access

---

## Phase 3: Text Messaging (Weeks 8-10)

**Goal:** Real-time text messaging.

### Deliverables
- [ ] WebSocket gateway
- [ ] Message send/receive
- [ ] Message edit/delete
- [ ] Markdown rendering
- [ ] Code block syntax highlighting
- [ ] Link embeds (preview cards)
- [ ] File attachments
- [ ] Image inline display
- [ ] Emoji picker (Unicode)
- [ ] @mention autocomplete
- [ ] Typing indicators
- [ ] Read state tracking
- [ ] Unread indicators

### Exit Criteria
- Real-time messaging works
- Messages persist and load on refresh
- Basic rich content displays

---

## Phase 4: Direct Messages (Weeks 11-12)

**Goal:** Private messaging between users.

### Deliverables
- [ ] 1:1 DM channels
- [ ] Group DMs (up to 10)
- [ ] DM channel creation
- [ ] User blocking
- [ ] DM privacy settings
- [ ] DM notification preferences

### Exit Criteria
- Users can DM each other
- Block prevents DM receipt
- DMs appear in sidebar

---

## Phase 5: Voice Channels (Weeks 13-16)

**Goal:** Real-time voice communication.

### Deliverables
- [ ] WebRTC integration (Pion SFU)
- [ ] Voice channel join/leave
- [ ] Voice activity detection
- [ ] Push-to-talk option
- [ ] Mute/deafen controls
- [ ] Server mute/deafen (moderation)
- [ ] Voice channel user limit
- [ ] Voice quality settings
- [ ] Connection quality indicator
- [ ] TURN server integration

### Exit Criteria
- 5+ users can voice chat simultaneously
- Audio quality is acceptable
- Moderation controls work

---

## Phase 6: Polish & Search (Weeks 17-18)

**Goal:** Quality of life improvements.

### Deliverables
- [ ] Full-text message search
- [ ] Search filters (from, in, has, date)
- [ ] Notification system (in-app)
- [ ] Desktop notifications (browser)
- [ ] Notification preferences
- [ ] Server discovery (public listing)
- [ ] User presence (online/idle/dnd)
- [ ] Custom status messages
- [ ] Reaction emoji on messages
- [ ] Message pinning
- [ ] Reply threading

### Exit Criteria
- Search returns relevant results
- Notifications work reliably
- Reactions and replies functional

---

## Phase 7: Video & Screen Share (Weeks 19-21)

**Goal:** Video calling and screen sharing.

### Deliverables
- [ ] Video toggle in voice channels
- [ ] Camera device selection
- [ ] Screen share (full screen)
- [ ] Screen share (window)
- [ ] Video grid layout
- [ ] Picture-in-picture mode
- [ ] Video quality settings
- [ ] Bandwidth adaptation

### Exit Criteria
- Video calls work with 4+ participants
- Screen share is usable for presentations
- Quality adapts to network conditions

---

## Phase 8: Moderation (Weeks 22-23)

**Goal:** Community management tools.

### Deliverables
- [ ] Ban/unban members
- [ ] Kick members
- [ ] Timeout (temporary mute)
- [ ] Audit log (all actions)
- [ ] Bulk message delete
- [ ] Slowmode per channel
- [ ] Verification levels
- [ ] Auto-moderation rules
- [ ] Word filters
- [ ] Spam detection

### Exit Criteria
- Moderators can manage communities
- Audit log tracks all changes
- Auto-mod catches basic spam

---

## Phase 9: Integrations (Weeks 24-25)

**Goal:** Extensibility for external services.

### Deliverables
- [ ] Incoming webhooks
- [ ] Bot accounts
- [ ] Bot token authentication
- [ ] Bot permission scopes
- [ ] Slash command registration
- [ ] OAuth2 bot flow
- [ ] Webhook management UI
- [ ] Rate limiting for bots

### Exit Criteria
- External services can post via webhooks
- Bots can be created and added to servers
- Slash commands execute

---

## Phase 10: Mobile & PWA (Weeks 26-27)

**Goal:** Mobile-friendly experience.

### Deliverables
- [ ] Responsive design polish
- [ ] PWA manifest
- [ ] Service worker (offline support)
- [ ] Push notifications (web push)
- [ ] Mobile-optimized voice UI
- [ ] Touch gestures
- [ ] Install prompts

### Exit Criteria
- PWA installable on mobile
- Push notifications work on mobile
- Voice works on mobile browsers

---

## v1.0 Release (Week 28)

**Goal:** Production-ready release.

### Deliverables
- [ ] Security audit
- [ ] Performance optimization
- [ ] Documentation complete
- [ ] Docker Hub / GHCR images
- [ ] Helm chart for Kubernetes
- [ ] One-click deploy templates
- [ ] Landing page
- [ ] Demo instance

---

## Future (v2.0+)

### Federation
- Server-to-server communication
- Shared channels across instances
- Federated identity

### End-to-End Encryption
- Optional E2EE for DMs
- Key management
- Device verification

### Native Mobile Apps
- React Native or Flutter
- Native voice/video
- Background notifications

### Advanced Features
- Scheduled messages
- Polls
- Events/calendar
- Threads (full implementation)
- Forum channels
- Stage channels
- Soundboard
- Activities (embedded apps)

### Enterprise
- SAML SSO
- SCIM provisioning
- Compliance exports
- Data loss prevention
- Advanced audit logs

---

## Versioning

| Version | Focus | Target |
|---------|-------|--------|
| 0.1.0 | Auth + Servers + Text | Week 10 |
| 0.2.0 | DMs + Voice | Week 16 |
| 0.3.0 | Video + Polish | Week 21 |
| 0.4.0 | Moderation + Bots | Week 25 |
| 1.0.0 | Production Ready | Week 28 |
| 2.0.0 | Federation + E2EE | TBD |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to contribute to Hearth development.

Priority areas for contributors:
- Frontend UI components
- Voice/video testing
- Documentation
- Translations
- Security review

---

*End of Roadmap*
