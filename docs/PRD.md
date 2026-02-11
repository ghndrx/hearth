# Hearth — Product Requirements Document (PRD)

**Version:** 1.0  
**Last Updated:** 2026-02-11  
**Author:** Greg Hendrickson  
**Status:** Draft

---

## 1. Executive Summary

### 1.1 Vision
Hearth is a self-hosted, open-source real-time communication platform that provides Discord-equivalent functionality while giving users complete control over their data and infrastructure.

### 1.2 Problem Statement
- Discord owns all user data and can access, mine, or monetize it
- Centralized platforms can ban communities arbitrarily
- No self-hosted alternative offers Discord's full feature set
- Existing alternatives (Matrix, Revolt) have UX or complexity issues

### 1.3 Solution
A polished, easy-to-deploy communication platform that:
- Runs entirely on user-owned infrastructure
- Provides familiar Discord-like UX
- Requires minimal technical expertise to deploy
- Scales from single-user to thousands

### 1.4 Success Metrics
- Single-command deployment (Docker)
- <5 minute time to first message
- Feature parity with Discord core (text, voice, video, roles)
- <100MB RAM for small instances
- Active community of self-hosters

---

## 2. User Personas

### 2.1 Homelab Enthusiast (Primary)
**Name:** Alex  
**Background:** Runs a home server, self-hosts various services  
**Goals:** Replace Discord for friend groups, full data control  
**Pain Points:** Complex setup, missing features in alternatives  
**Needs:** Docker deployment, low resource usage, good docs

### 2.2 Small Community Owner
**Name:** Jordan  
**Background:** Runs a gaming clan, hobby group, or study community  
**Goals:** Private space for 20-200 members  
**Pain Points:** Discord's policies, lack of control  
**Needs:** Roles, moderation tools, reliability

### 2.3 Organization IT Admin
**Name:** Sam  
**Background:** Manages internal communications for company  
**Goals:** Secure, compliant team communication  
**Pain Points:** Slack costs, data residency requirements  
**Needs:** OIDC/SSO, audit logs, enterprise features

### 2.4 Privacy Advocate
**Name:** Riley  
**Background:** Values digital privacy and sovereignty  
**Goals:** Communicate without corporate surveillance  
**Pain Points:** All mainstream platforms track users  
**Needs:** No telemetry, E2EE option, transparency

---

## 3. Feature Requirements

### 3.1 Authentication & Users

#### 3.1.1 Registration & Login
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| AUTH-001 | Email + password registration | P0 | With email verification option |
| AUTH-002 | OAuth2 login (Google, GitHub, Discord) | P1 | Optional, configurable |
| AUTH-003 | OIDC/SAML SSO | P2 | Enterprise feature |
| AUTH-004 | Two-factor authentication (TOTP) | P1 | Google Authenticator compatible |
| AUTH-005 | Password reset flow | P0 | Email-based |
| AUTH-006 | Session management | P0 | View/revoke active sessions |
| AUTH-007 | Account deletion | P1 | GDPR compliance |

#### 3.1.2 User Profiles
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| USER-001 | Username (unique, changeable) | P0 | With discriminator or unique handle |
| USER-002 | Display name | P0 | Per-server nicknames |
| USER-003 | Avatar upload | P0 | With size limits |
| USER-004 | Banner image | P2 | Profile customization |
| USER-005 | Bio/about me | P1 | Markdown supported |
| USER-006 | Status (online/idle/dnd/invisible) | P0 | Auto-idle after timeout |
| USER-007 | Custom status message | P1 | With optional emoji |
| USER-008 | Connected accounts display | P2 | GitHub, Twitter, etc. |

### 3.2 Servers

#### 3.2.1 Server Management
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| SRV-001 | Create server | P0 | With name and icon |
| SRV-002 | Server icon/banner | P0 | Upload with cropping |
| SRV-003 | Server description | P1 | Shown in discovery |
| SRV-004 | Delete server | P0 | Owner only, confirmation required |
| SRV-005 | Transfer ownership | P1 | To existing member |
| SRV-006 | Server templates | P2 | Gaming, study, community presets |
| SRV-007 | Server discovery | P2 | Public server listing |
| SRV-008 | Vanity URL | P2 | Custom invite slug |

#### 3.2.2 Invites
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| INV-001 | Generate invite link | P0 | Unique code |
| INV-002 | Invite expiration | P0 | Never, 24h, 7d, 30d, custom |
| INV-003 | Max uses limit | P0 | Unlimited or specific count |
| INV-004 | Invite tracking | P1 | Who invited whom |
| INV-005 | Revoke invites | P0 | By code or bulk |
| INV-006 | Temporary membership | P2 | Auto-kick after disconnect |

### 3.3 Channels

#### 3.3.1 Channel Types
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| CHN-001 | Text channels | P0 | Core messaging |
| CHN-002 | Voice channels | P1 | Real-time voice |
| CHN-003 | Video channels | P2 | Voice + video |
| CHN-004 | Announcement channels | P1 | Follow from other servers |
| CHN-005 | Forum channels | P2 | Threaded discussions |
| CHN-006 | Stage channels | P3 | Moderated presentations |
| CHN-007 | Category grouping | P0 | Organize channels |

#### 3.3.2 Channel Settings
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| CHN-010 | Channel name/topic | P0 | With character limits |
| CHN-011 | Slowmode | P1 | Rate limit per user |
| CHN-012 | NSFW flag | P1 | Age gate |
| CHN-013 | Permission overrides | P0 | Per-role, per-user |
| CHN-014 | Channel position | P0 | Drag-and-drop reorder |
| CHN-015 | Auto-archive threads | P1 | After inactivity |

### 3.4 Messaging

#### 3.4.1 Core Messaging
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| MSG-001 | Send text messages | P0 | Real-time delivery |
| MSG-002 | Edit messages | P0 | With edit history |
| MSG-003 | Delete messages | P0 | Own or with permission |
| MSG-004 | Message history | P0 | Infinite scroll, lazy load |
| MSG-005 | Unread indicators | P0 | Channel and mention counts |
| MSG-006 | Jump to message | P1 | By link or search |

#### 3.4.2 Rich Content
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| MSG-010 | Markdown formatting | P0 | Bold, italic, code, etc. |
| MSG-011 | Code blocks | P0 | With syntax highlighting |
| MSG-012 | Link embeds | P1 | Auto-preview for URLs |
| MSG-013 | Image embeds | P0 | Inline display |
| MSG-014 | Video embeds | P1 | YouTube, etc. |
| MSG-015 | File attachments | P0 | With size limits |
| MSG-016 | Emoji (Unicode) | P0 | Full Unicode support |
| MSG-017 | Custom emoji | P1 | Server-specific |
| MSG-018 | Reactions | P0 | Add/remove, count display |
| MSG-019 | Stickers | P3 | Animated images |

#### 3.4.3 Mentions & References
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| MSG-020 | @user mentions | P0 | Autocomplete, notification |
| MSG-021 | @role mentions | P0 | Notify all role members |
| MSG-022 | @everyone / @here | P0 | Require permission |
| MSG-023 | #channel links | P0 | Cross-reference |
| MSG-024 | Reply to message | P0 | Quote with reference |
| MSG-025 | Message pinning | P1 | With pin limit |

#### 3.4.4 Threads
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| MSG-030 | Create thread from message | P1 | Branch conversation |
| MSG-031 | Thread naming | P1 | Auto or custom |
| MSG-032 | Thread membership | P1 | Join/leave |
| MSG-033 | Thread archiving | P1 | Auto after inactivity |
| MSG-034 | Thread permissions | P1 | Inherit or override |

### 3.5 Direct Messages

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| DM-001 | 1:1 direct messages | P0 | Private conversation |
| DM-002 | Group DMs | P1 | Up to 10 participants |
| DM-003 | DM voice calls | P1 | 1:1 voice |
| DM-004 | DM video calls | P2 | 1:1 video |
| DM-005 | Block users | P0 | Prevent DMs |
| DM-006 | DM spam settings | P1 | Server members only, etc. |

### 3.6 Voice & Video

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| VV-001 | Join voice channel | P1 | WebRTC connection |
| VV-002 | Voice activity detection | P1 | Auto-detect speaking |
| VV-003 | Push-to-talk | P1 | Keybind option |
| VV-004 | Mute/deafen self | P1 | Local controls |
| VV-005 | Server mute/deafen | P1 | Moderation |
| VV-006 | Video toggle | P2 | Camera on/off |
| VV-007 | Screen sharing | P2 | Window or full screen |
| VV-008 | Voice quality settings | P2 | Bitrate selection |
| VV-009 | Noise suppression | P2 | AI-based |
| VV-010 | Channel user limit | P1 | Max occupancy |

### 3.7 Roles & Permissions

#### 3.7.1 Role Management
| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| ROLE-001 | Create/edit/delete roles | P0 | With name and color |
| ROLE-002 | Role hierarchy | P0 | Higher = more power |
| ROLE-003 | Role icon | P2 | Visual indicator |
| ROLE-004 | Display role separately | P0 | Sidebar grouping |
| ROLE-005 | Mentionable toggle | P0 | Allow @role |
| ROLE-006 | Self-assignable roles | P2 | Via reaction or command |
| ROLE-007 | Role member list | P0 | View who has role |

#### 3.7.2 Permissions (Comprehensive)
| Category | Permissions |
|----------|-------------|
| **General** | View Channels, Manage Channels, Manage Roles, Manage Emoji, View Audit Log, Manage Webhooks, Manage Server |
| **Membership** | Create Invite, Change Nickname, Manage Nicknames, Kick Members, Ban Members, Timeout Members |
| **Text** | Send Messages, Send TTS, Manage Messages, Embed Links, Attach Files, Read Message History, Mention Everyone, Use External Emoji, Add Reactions, Use Slash Commands |
| **Voice** | Connect, Speak, Video, Use Voice Activity, Priority Speaker, Mute Members, Deafen Members, Move Members |
| **Advanced** | Administrator (bypasses all) |

### 3.8 Moderation

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| MOD-001 | Kick member | P0 | Remove from server |
| MOD-002 | Ban member | P0 | Permanent removal + IP |
| MOD-003 | Unban member | P0 | Restore access |
| MOD-004 | Timeout member | P1 | Temporary mute (1min-28d) |
| MOD-005 | Bulk message delete | P1 | Purge up to 100 |
| MOD-006 | Audit log | P0 | All moderation actions |
| MOD-007 | Auto-moderation | P2 | Spam, link, word filters |
| MOD-008 | Verification levels | P1 | Email, age, 2FA requirements |
| MOD-009 | Raid protection | P2 | Auto-detect and mitigate |
| MOD-010 | Report system | P2 | User reports to mods |

### 3.9 Notifications

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| NOTIF-001 | In-app notifications | P0 | Real-time |
| NOTIF-002 | Desktop notifications | P0 | Browser/electron push |
| NOTIF-003 | Mobile push | P2 | Via FCM/APNs |
| NOTIF-004 | Email notifications | P1 | Digest or instant |
| NOTIF-005 | Per-channel settings | P0 | All, mentions, none |
| NOTIF-006 | Per-server settings | P0 | Override defaults |
| NOTIF-007 | Do not disturb schedule | P1 | Quiet hours |
| NOTIF-008 | Notification sounds | P1 | Customizable |

### 3.10 Search

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| SRCH-001 | Full-text message search | P1 | Across accessible channels |
| SRCH-002 | Search filters | P1 | From, in, has, before, after |
| SRCH-003 | Search in DMs | P1 | Private message search |
| SRCH-004 | User search | P0 | Find members |
| SRCH-005 | Jump to result | P1 | Context around match |

### 3.11 Integrations

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| INT-001 | Incoming webhooks | P1 | Post messages from external |
| INT-002 | Outgoing webhooks | P2 | Events to external URL |
| INT-003 | Bot accounts | P1 | API access with token |
| INT-004 | Bot permissions | P1 | Scoped access |
| INT-005 | Slash commands | P2 | Registered bot commands |
| INT-006 | OAuth2 for bots | P2 | Add bot flow |

---

## 4. Non-Functional Requirements

### 4.1 Performance
| Metric | Target |
|--------|--------|
| Message delivery latency | <100ms (same region) |
| Voice latency | <50ms |
| Time to interactive (web) | <3s on 3G |
| API response time | <200ms (p95) |
| Concurrent users per instance | 10,000+ |
| Messages per second | 1,000+ |

### 4.2 Scalability
- Horizontal scaling via multiple instances behind load balancer
- Database sharding for large deployments
- CDN support for static assets and media
- Stateless backend (Redis for session/pubsub)

### 4.3 Reliability
- 99.9% uptime target (self-hosted dependent on operator)
- Graceful degradation (voice fails → text works)
- Automatic reconnection on network issues
- Message persistence (no message loss)

### 4.4 Security
- All traffic over TLS 1.3
- Passwords hashed with Argon2id
- Rate limiting on all endpoints
- CSRF protection
- XSS prevention (CSP, sanitization)
- SQL injection prevention (parameterized queries)
- Regular security audits
- Optional E2EE for DMs (future)

### 4.5 Compliance
- GDPR: Data export, deletion, consent
- CCPA: Similar rights for California users
- Audit logging for enterprise compliance
- Data residency (user controls server location)

### 4.6 Accessibility
- WCAG 2.1 AA compliance
- Screen reader support
- Keyboard navigation
- Reduced motion option
- High contrast themes

---

## 5. Technical Architecture

*See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed technical design.*

### 5.1 High-Level Overview
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Clients   │────▶│   Hearth    │────▶│  Database   │
│  (Web/App)  │◀────│   Server    │◀────│  (Postgres) │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │
       │              ┌────┴────┐
       │              ▼         ▼
       │         ┌───────┐ ┌───────┐
       └────────▶│ Redis │ │  S3   │
                 │(pubsub)│ │(media)│
                 └───────┘ └───────┘
```

### 5.2 Key Technologies
- **Backend:** Go with Fiber framework
- **Frontend:** SvelteKit + TypeScript
- **Database:** PostgreSQL (prod) / SQLite (dev)
- **Real-time:** WebSocket + Redis Pub/Sub
- **Voice/Video:** WebRTC via Pion
- **Search:** Bleve (embedded) or Meilisearch

---

## 6. User Flows

### 6.1 New User Registration
1. User visits Hearth instance URL
2. Clicks "Create Account"
3. Enters email, username, password
4. (Optional) Verifies email
5. Redirected to home (empty server list)
6. Prompted to create or join a server

### 6.2 Creating a Server
1. User clicks "+" in server list
2. Selects "Create Server"
3. Enters server name, uploads icon
4. (Optional) Selects template
5. Server created with default channels (#general, #voice)
6. User is owner with full permissions
7. Invite link generated automatically

### 6.3 Sending a Message
1. User selects text channel
2. Types in message input
3. (Optional) Adds emoji, attachment, mention
4. Presses Enter or clicks Send
5. Message appears instantly (optimistic UI)
6. Server confirms delivery
7. Other users see message in real-time

### 6.4 Joining Voice
1. User clicks voice channel
2. Browser requests microphone permission
3. WebRTC connection established
4. User appears in voice channel member list
5. Voice activity indicator shows when speaking
6. User can toggle mute/deafen
7. Clicking another channel or X disconnects

---

## 7. Milestones & Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| **Phase 1: Foundation** | 4 weeks | Auth, user profiles, basic UI shell |
| **Phase 2: Core Messaging** | 4 weeks | Servers, channels, text messaging, roles |
| **Phase 3: Real-time** | 3 weeks | WebSocket, presence, typing indicators |
| **Phase 4: Voice** | 4 weeks | WebRTC voice channels |
| **Phase 5: Polish** | 3 weeks | Search, notifications, DMs |
| **Phase 6: Video** | 3 weeks | Video calls, screen share |
| **Phase 7: Integrations** | 2 weeks | Webhooks, bot API |
| **MVP Total** | ~23 weeks | Full Discord-equivalent MVP |

---

## 8. Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| WebRTC complexity | High | Medium | Use Pion library, progressive rollout |
| Scaling issues | Medium | Low | Design for horizontal scale from start |
| Security vulnerabilities | High | Medium | Security-first design, audits, bounty |
| Feature creep | Medium | High | Strict MVP scope, defer to v2 |
| Browser compatibility | Medium | Medium | Test matrix, graceful degradation |

---

## 9. Open Questions

1. **Federation:** Should v1 support server-to-server communication?
   - *Decision:* Defer to v2, focus on single-instance excellence

2. **Mobile apps:** Native or PWA first?
   - *Decision:* PWA for MVP, native for v2

3. **E2EE:** Implement in v1 DMs?
   - *Decision:* Optional feature, not blocking MVP

4. **Monetization:** Open source sustainability?
   - *Decision:* MIT license, optional paid hosting/support tiers

---

## 10. Appendix

### 10.1 Competitive Analysis

| Feature | Discord | Slack | Matrix | Revolt | Hearth |
|---------|---------|-------|--------|--------|--------|
| Self-hosted | ❌ | ❌ | ✅ | ✅ | ✅ |
| Voice/Video | ✅ | ✅ | ⚠️ | ⚠️ | ✅ |
| Roles/Perms | ✅ | ⚠️ | ✅ | ✅ | ✅ |
| Easy deploy | N/A | N/A | ❌ | ⚠️ | ✅ |
| UX polish | ✅ | ✅ | ❌ | ⚠️ | ✅ |
| Open source | ❌ | ❌ | ✅ | ✅ | ✅ |

### 10.2 Glossary
- **Server:** A community instance (called "guild" in Discord API)
- **Channel:** A communication space within a server
- **Role:** A set of permissions assignable to users
- **DM:** Direct message between users outside servers
- **Webhook:** HTTP callback for external integrations

---

*End of PRD*
