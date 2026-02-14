# Hearth Task Queue

Tasks derived from product research and competitive analysis.
Priority: P0 (Critical) → P1 (High) → P2 (Medium) → P3 (Low)

---

## Accessibility (P0 - Critical)

### A11Y-001: Implement Full Keyboard Navigation
- [ ] Ensure all interactive elements are keyboard accessible
- [ ] Add visible focus indicators (outline/ring) on all focusable elements
- [ ] Implement logical tab order through the interface
- [ ] Test Tab, Shift+Tab, Enter, Escape, Arrow keys throughout app
- **WCAG**: 2.1.1 Keyboard
- **Estimate**: 3-4 days

### A11Y-002: Add ARIA Labels and Screen Reader Support
- [ ] Add ARIA labels to all custom interactive elements
- [ ] Add ARIA live regions for new message notifications
- [ ] Implement proper heading hierarchy in message threads
- [ ] Test with NVDA and VoiceOver
- **WCAG**: 4.1.2 Name, Role, Value
- **Estimate**: 2-3 days

### A11Y-003: Verify Color Contrast Compliance
- [ ] Audit current color palette against WCAG 4.5:1 requirement
- [ ] Fix any contrast issues in design tokens
- [ ] Add contrast checker to PR review process
- **WCAG**: 1.4.3 Contrast (Minimum)
- **Estimate**: 1 day

### A11Y-004: Add Alt Text Support for Images
- [ ] Add "Add description" prompt when uploading images
- [ ] Store and display alt text on hover/screen reader
- [ ] Consider AI-powered alt text suggestions (Phase 2)
- **Estimate**: 1-2 days

---

## Performance (P1 - High)

### PERF-000: [BUG] Fix Token Validation Failure After Registration (P0)
**Added: 2026-02-14 (performance test findings)**
- [ ] Investigate 401 "invalid token" on fresh access tokens
- [ ] Check JWT secret consistency (issue vs validation)
- [ ] Review middleware token parsing
- [ ] Verify Redis session store sync
- **Impact**: Blocks all authenticated endpoint testing
- **Estimate**: 1 day

### PERF-001A: Add Bcrypt Worker Pool for Bounded CPU Usage
**Added: 2026-02-14 (performance test findings)**
- [ ] Implement bounded concurrency for bcrypt operations
- [ ] Registration latency: 220ms → 3.3s under load
- [ ] Consider async registration with job queue
- **Target**: p99 < 500ms under 100 req/s
- **Estimate**: 2 days

### PERF-001B: Add Rate Limit Bypass for Testing
**Added: 2026-02-14 (performance test findings)**
- [ ] Add `RATE_LIMIT_ENABLED=false` env var
- [ ] Or whitelist localhost in dev mode
- **Impact**: Enables local load testing
- **Estimate**: 0.5 day

### PERF-001: Implement Redis Pub/Sub for Message Fan-Out
- [ ] Set up Redis Pub/Sub for channel message distribution
- [ ] Refactor WebSocket handlers to subscribe to channel topics
- [ ] Benchmark message latency before/after
- **Target**: <100ms message delivery
- **Estimate**: 3-4 days

### PERF-002: Add Prometheus Metrics for WebSocket Connections
- [ ] Expose connection count metrics per pod
- [ ] Add message rate metrics (messages/second)
- [ ] Create Grafana dashboard for real-time monitoring
- [ ] Set up HPA based on connection count
- **Estimate**: 2 days

### PERF-003: Implement Graceful Connection Draining
- [ ] Handle SIGTERM in WebSocket server
- [ ] Send reconnect signal to clients before shutdown
- [ ] Add health check endpoint for load balancer
- **Estimate**: 1-2 days

---

## UX Improvements (P1 - High)

### UX-001: Enhanced Search with Filters
- [ ] Implement fuzzy search with type-ahead suggestions
- [ ] Add search filters: `from:`, `in:`, `has:`, `before:`, `after:`
- [ ] Show recent searches
- [ ] Add search results highlighting
- **Estimate**: 4-5 days

### UX-002: Improved Notification Controls
- [ ] Add per-channel notification preferences
- [ ] Implement "Mute for X hours" quick action
- [ ] Add Do Not Disturb scheduling
- [ ] Unify notification settings UI
- **Estimate**: 3 days

### UX-003: User Profile Enhancements
- [ ] Show mutual servers on user profile modal
- [ ] Add shared channels section
- [ ] Display recent activity/status
- **Estimate**: 2 days

---

## New Features (P2 - Medium)

### FEAT-001: Message Threading Enhancements
- [ ] Review `feature-threading-replies.md` research doc
- [ ] Implement thread sidebar view
- [ ] Add thread notifications preferences
- [ ] Thread participant indicators
- **Estimate**: 5-7 days

### FEAT-002: Voice Channel Mini-Player
- [ ] Design floating mini-player for active voice calls
- [ ] Implement persistent docked bar
- [ ] Add quick mute/deafen controls
- **Phase**: 2 (Voice Channels)
- **Estimate**: 3-4 days

### FEAT-003: Split View (Desktop)
- [ ] Design split-view layout for desktop
- [ ] Allow pinning DMs/threads alongside main view
- [ ] Implement resize handles
- **Estimate**: 4-5 days

---

## Security (P2 - Medium)

### SEC-001: Review E2EE Implementation Path
- [ ] Review `security-e2ee-implementation.md` research doc
- [ ] Evaluate Olm/Megolm (Matrix protocol) vs custom
- [ ] Document E2EE architecture for DMs
- **Phase**: 3
- **Estimate**: Research only - 2 days

### SEC-002: JWT Auth Hardening
- [ ] Review `security-jwt-auth.md` research doc
- [ ] Implement token refresh rotation
- [ ] Add device management UI
- **Estimate**: 2-3 days

---

## AI Integration (P3 - Future)

### AI-001: Research AI Features Roadmap
- [ ] Evaluate AI summarization for channels
- [ ] Design `/hearth-ai explain` command
- [ ] Assess AI-powered search
- **Phase**: 2-3
- **Estimate**: Research only - 2 days

---

## Completed Tasks

_Move tasks here when done_

---

*Last updated: 2026-02-14*
*Source: docs/research/insights-2026-02-14.md*
