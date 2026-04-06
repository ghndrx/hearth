# Hearth Mobile Applications — Technical Specification

## Executive Summary

Native mobile applications for iOS (Swift/SwiftUI) and Android (Kotlin/Jetpack Compose) to address the critical user acquisition blocker: 70% of Discord users are mobile-first, and Hearth has no app store presence.

**Timeline:** 12–16 weeks
**Platforms:** iOS 16+ / Android 8.0+ (API 26)

---

## 1. Architecture Overview

### Decision: Native per-platform (not cross-platform)

**Rationale:**
- Push notifications, background WebSocket keep-alive, and voice/video all require deep OS integration
- SwiftUI and Jetpack Compose are both declarative UI frameworks — code structure maps 1:1 across platforms
- Avoids React Native / Flutter bridge overhead for real-time messaging (thousands of events/sec)
- LiveKit has first-class native SDKs for both iOS and Android
- Easier to pass App Store / Play Store review with native implementations

### Shared Design Principles
- **Offline-first**: All viewed data cached locally; messages queue for send when offline
- **Event-driven**: WebSocket gateway drives UI state; REST is for initial loads and mutations
- **Token-based auth**: JWT access/refresh tokens stored in platform keychain/keystore
- **Unidirectional data flow**: Repository → ViewModel → View

### High-Level Architecture (both platforms)

```
┌─────────────────────────────────────────────┐
│                    Views                     │
│         (SwiftUI / Jetpack Compose)          │
├─────────────────────────────────────────────┤
│                 ViewModels                   │
│        (ObservableObject / ViewModel)        │
├─────────────────────────────────────────────┤
│                Repositories                  │
│     (Single source of truth per domain)      │
├──────────┬──────────┬───────────────────────┤
│ API      │ Gateway  │ Local Storage          │
│ Client   │ (WS)    │ (SQLite / CoreData)   │
├──────────┴──────────┴───────────────────────┤
│         Push Notification Manager            │
│     (APNs / FCM — per platform native)       │
└─────────────────────────────────────────────┘
```

---

## 2. Core Screens & Navigation

### Tab Bar (bottom navigation)
1. **Servers** — server list sidebar + channel list
2. **DMs** — direct message conversations
3. **Search** — full-text message search
4. **Notifications** — notification center
5. **Profile** — user settings, status, account

### Screen Inventory

| Screen | Priority | Description |
|--------|----------|-------------|
| Login / Register | P0 | Email/password + OAuth (GitHub, Google, Discord) |
| MFA Prompt | P0 | TOTP code entry during login |
| Server List | P0 | Sidebar with server icons, unread indicators |
| Channel List | P0 | Text/voice channels grouped by category |
| Message View | P0 | Scrollable message list with infinite scroll |
| Message Composer | P0 | Text input, attachments, replies, mentions |
| DM List | P0 | Recent DM conversations |
| DM Chat | P0 | 1:1 and group DM messaging |
| Thread View | P1 | Thread replies within a channel |
| User Profile | P1 | View user info, roles, mutual servers |
| Server Settings | P1 | Name, icon, roles, channels |
| Notification List | P1 | Grouped notifications with read/unread |
| Voice Channel | P2 | Voice participants, mute/deafen controls |
| Server Discovery | P2 | Browse/search public servers |
| Search | P2 | Full-text search with filters |
| Settings | P1 | App preferences, account, notifications |

### Navigation Architecture
- **iOS**: `TabView` → `NavigationStack` per tab with push navigation
- **Android**: Bottom Navigation → `NavHost` with Compose Navigation

---

## 3. Real-Time Infrastructure

### WebSocket Gateway Protocol

The gateway uses the same opcode protocol as the web client:

| Opcode | Name | Direction |
|--------|------|-----------|
| 0 | DISPATCH | Server → Client |
| 1 | HEARTBEAT | Bidirectional |
| 2 | IDENTIFY | Client → Server |
| 3 | PRESENCE_UPDATE | Client → Server |
| 4 | VOICE_STATE_UPDATE | Client → Server |
| 6 | RESUME | Client → Server |
| 7 | RECONNECT | Server → Client |
| 9 | INVALID_SESSION | Server → Client |
| 10 | HELLO | Server → Client |
| 11 | HEARTBEAT_ACK | Server → Client |

### Connection Lifecycle
1. Connect to `wss://{host}/gateway?token={jwt}`
2. Receive `HELLO` with heartbeat interval
3. Send `IDENTIFY` with token and client properties
4. Begin heartbeat loop at server-specified interval
5. Process `DISPATCH` events (MESSAGE_CREATE, TYPING_START, PRESENCE_UPDATE, etc.)
6. On disconnect: attempt `RESUME` with session_id and last sequence number
7. Exponential backoff reconnect (max 10 attempts, jitter)

### Key Dispatch Events
- `READY` — initial state (user, servers, channels, DMs)
- `MESSAGE_CREATE`, `MESSAGE_UPDATE`, `MESSAGE_DELETE`
- `CHANNEL_CREATE`, `CHANNEL_UPDATE`, `CHANNEL_DELETE`
- `TYPING_START`
- `PRESENCE_UPDATE`
- `VOICE_STATE_UPDATE`
- `NOTIFICATION_CREATE`
- `SERVER_MEMBER_ADD`, `SERVER_MEMBER_REMOVE`
- `REACTION_ADD`, `REACTION_REMOVE`

---

## 4. Push Notification Infrastructure

### Overview
Push notifications are the #1 user acquisition blocker. When a user is offline, the server sends push notifications via:
- **iOS**: Apple Push Notification Service (APNs)
- **Android**: Firebase Cloud Messaging (FCM)

### Deduplication Strategy
- **Gateway-connected = user is active**: suppress push if the user's gateway connection is active
- **Badge count**: maintained via `GET /api/v1/users/@me/notifications/unread-count`
- **Notification grouping**: by server/channel to prevent notification spam

### iOS (APNs)

#### Implementation
- `PushNotificationManager.swift` handles the full APNs lifecycle
- Requests `provisional` authorization first (iOS 12+), then upgrades to full
- Device token sent to backend via `POST /api/v1/users/@me/push-tokens`
- VoIP push handled separately via `PushKit` (future voice feature)

#### Push Payload Structure
```json
{
  "aps": {
    "alert": {
      "title": "Server Name",
      "body": "User: Message content..."
    },
    "badge": 3,
    "sound": "default",
    "thread-id": "channel_id"
  },
  "hearth": {
    "type": "MESSAGE_CREATE",
    "channelId": "...",
    "serverId": "...",
    "messageId": "...",
    "senderId": "..."
  }
}
```

#### Background Modes
- `remote-notification` — background wake to sync badge + fetch new messages
- `processing` — allows long-running background tasks

#### Capabilities Required
- Push Notifications
- Background Modes (Remote notifications, Background fetch)

### Android (FCM)

#### Implementation
- `PushNotificationManager.kt` initializes FCM and obtains an FCM registration token
- Token sent to backend via `POST /api/v1/users/@me/push-tokens`
- `PushService.kt` (extending `FirebaseMessagingService`) handles incoming FCM messages

#### Notification Channels
| Channel ID | Importance | Use Case |
|------------|------------|----------|
| `messages` | HIGH | New messages in channels/DMs |
| `mentions` | HIGH | @mentions and @everyone |
| `dms` | DEFAULT | Direct message notifications |
| `activity` | LOW | Server events (member joined, etc.) |

#### Android 13+ (API 33) Runtime Permission
- Request `POST_NOTIFICATIONS` permission at runtime before registering with FCM
- Show in-app explanation before system permission dialog

### Server-Side Push Token API

```
POST /api/v1/users/@me/push-tokens
Body: {
  "token": "device_push_token",
  "platform": "ios" | "android",
  "deviceId": "UUID",
  "appVersion": "1.0.0"
}

DELETE /api/v1/users/@me/push-tokens/:token

GET /api/v1/users/@me/notifications/unread-count
→ { "count": 5 }
```

### Backend Push Flow
```
1. User sends message via WebSocket/REST
2. Gateway checks if recipient has active connection
3. If NO active connection → lookup push token from DB
4. Queue push notification via APNs/FCM
5. Increment unread count in Redis
6. Send push (APNs/FCM handles delivery)
7. Client receives → updates badge count
```

---

## 5. API Integration Points

### Base URL
```
{scheme}://{host}/api/v1
```

### Authentication
```
POST /auth/login          → { token, refresh_token, user }
POST /auth/login/mfa      → { token, refresh_token, user }
POST /auth/register       → { token, refresh_token, user }
POST /auth/refresh        → { token, refresh_token }
POST /auth/logout
GET  /auth/oauth/:provider → redirect
```

### Core Data Loading
```
GET  /users/@me                         → current user
GET  /users/@me/servers                 → server list
GET  /servers/:id                       → server detail
GET  /servers/:id/channels              → channel list
GET  /servers/:id/channels/:id/messages → message history (paginated)
GET  /users/@me/channels                → DM list
GET  /dms/:channelId/messages           → DM messages
```

### Mutations
```
POST   /servers/:id/channels/:id/messages  → send message
PATCH  /messages/:id                       → edit message
DELETE /messages/:id                       → delete message
POST   /messages/:id/reactions/:emoji      → add reaction
POST   /users/@me/channels                 → create DM
PUT    /users/@me/status                   → update status
```

### Push Notifications
```
POST   /users/@me/push-tokens              → register device token
DELETE /users/@me/push-tokens/:token       → unregister device token
GET    /users/@me/notifications/unread-count → badge count
```

### Headers
```
Authorization: Bearer {access_token}
Content-Type: application/json
X-Client-Info: hearth-ios/1.0.0 (iPhone; iOS 17.0)
```

---

## 6. Authentication Flow

### Login Flow
```
┌──────────┐    POST /auth/login     ┌──────────┐
│  Login   │ ───────────────────────→│  Backend  │
│  Screen  │←─── { token, refresh }──│          │
└──────────┘                         └──────────┘
      │
      │  Store in Keychain/KeyStore
      ▼
┌──────────┐    GET /users/@me       ┌──────────┐
│  App     │ ───────────────────────→│  Backend  │
│  Start   │←─── { user }───────────│          │
└──────────┘                         └──────────┘
      │
      │  Connect WebSocket
      ▼
┌──────────┐
│  Ready   │
└──────────┘
```

### Token Management
- **Access token**: 15 min TTL, stored in memory + Keychain/KeyStore
- **Refresh token**: 7 day TTL, stored in Keychain/KeyStore only
- **Auto-refresh**: Intercept 401 responses, call `/auth/refresh`, retry original request
- **Logout**: Clear tokens, disconnect gateway, clear local cache, delete push token

### OAuth Flow
1. Open system browser / ASWebAuthenticationSession / Chrome Custom Tabs
2. Redirect to `hearth://oauth/callback?code=...`
3. Exchange code via backend callback endpoint
4. Receive JWT tokens

---

## 7. Offline Support Strategy

### Tier 1 — Read Cache (Week 1–4)
- Cache all fetched servers, channels, messages in local SQLite
- Show cached data immediately on app launch
- Overlay "Connecting..." banner when offline
- Messages render from cache with "sent" / "pending" / "failed" badges

### Tier 2 — Optimistic Writes (Week 5–8)
- Queue outbound messages locally when offline
- Show pending messages in chat with "sending" indicator
- Flush queue on reconnect in order
- Conflict resolution: server timestamp wins

### Tier 3 — Background Sync (Week 9–12)
- Periodic background fetch of unread counts
- Push notification badge sync
- Incremental message sync (fetch since last_message_id)

### Storage
- **iOS**: SwiftData (backed by SQLite) for structured data; FileManager for attachments
- **Android**: Room (SQLite) for structured data; internal storage for attachments
- **Cache eviction**: LRU by server, keep last 100 messages per channel in cache

---

## 8. State Management

### Architecture Pattern: Repository + ViewModel

```
View ←→ ViewModel ←→ Repository ←→ [API Client | Gateway | LocalDB | Push Manager]
```

### iOS (Swift)
- `@Observable` classes for ViewModels (Swift 5.9+)
- `SwiftData` for persistence
- `@Environment` for dependency injection
- `AsyncSequence` for streaming gateway events

### Android (Kotlin)
- `ViewModel` + `StateFlow` for reactive UI state
- `Room` for persistence
- `Hilt` for dependency injection
- `Flow` for streaming gateway events

### Key State Domains

| Domain | Scope | Source of Truth |
|--------|-------|-----------------|
| Auth | Global | Keychain/KeyStore + memory |
| Current User | Global | API + gateway READY |
| Server List | Global | API + gateway events |
| Channel List | Per-server | API + gateway events |
| Messages | Per-channel | API + gateway events + local cache |
| Presence | Global | Gateway events only |
| Typing | Per-channel | Gateway events (ephemeral, not cached) |
| Notifications | Global | API + gateway events |
| Push Token | Global | APNs/FCM + API |

---

## 9. Testing Strategy

### Unit Tests
- **ViewModels**: Test state transitions, business logic (mock repositories)
- **Repositories**: Test data merging (mock API + mock DB)
- **Gateway Client**: Test opcode handling, reconnect logic (mock WebSocket)
- **Auth Manager**: Test token refresh, expiry detection
- **PushNotificationManager**: Test token storage, API registration, permission handling

### Integration Tests
- **API Client**: Test against local backend (docker-compose)
- **Database**: Test migrations, queries against in-memory SQLite

### UI Tests
- **iOS**: XCTest UI tests for critical flows (login, send message, navigate)
- **Android**: Compose UI tests with `createComposeRule` for critical flows

### Coverage Targets
- Unit: 80%+ on ViewModels and Repositories
- Integration: Key API flows (auth, messages, channels)
- UI: Login flow, message send, channel navigation

### CI Pipeline
- **iOS**: Xcode Cloud or GitHub Actions with macOS runner
- **Android**: GitHub Actions with Gradle build + instrumented tests on emulator

---

## 10. Project Structure

```
mobile/
├── SPEC.md                          # This document
├── PUSH_SETUP.md                    # Push notification setup guide
├── shared/
│   └── api-types.md                # Shared type reference
├── ios/
│   ├── Hearth/
│   │   ├── App/
│   │   │   └── HearthApp.swift      # App entry, notification bootstrap
│   │   ├── Core/
│   │   │   ├── Auth/               # AuthManager, KeychainHelper
│   │   │   ├── Gateway/            # WebSocket client
│   │   │   ├── Network/            # APIClient
│   │   │   ├── Push/               # PushNotificationManager
│   │   │   └── Storage/            # SwiftData models, persistence
│   │   ├── Features/
│   │   │   ├── Auth/               # Login, Register, OAuth screens
│   │   │   ├── Servers/            # Server list, channel list
│   │   │   ├── Messages/           # Message list, composer
│   │   │   ├── DMs/                # Direct messages
│   │   │   └── Profile/            # User profile, settings
│   │   ├── Shared/                # Reusable components
│   │   └── Resources/             # Assets, localization
│   ├── HearthTests/              # Unit tests
│   ├── HearthUITests/            # UI tests
│   └── Package.swift            # Swift Package Manager manifest
└── android/
    ├── app/
    │   ├── src/
    │   │   ├── main/
    │   │   │   ├── java/co/hndrx/hearth/
    │   │   │   │   ├── app/         # Application, DI modules
    │   │   │   │   ├── core/
    │   │   │   │   │   ├── auth/    # Token management
    │   │   │   │   ├── gateway/     # WebSocket client
    │   │   │   │   ├── network/     # API client, interceptors
    │   │   │   │   ├── push/        # PushNotificationManager, PushService
    │   │   │   │   └── storage/     # Room database
    │   │   │   │   ├── features/
    │   │   │   │   │   ├── auth/    # Login, Register screens
    │   │   │   │   ├── servers/     # Server list, channels
    │   │   │   │   ├── messages/    # Message list, composer
    │   │   │   │   ├── dms/         # Direct messages
    │   │   │   │   └── profile/     # Profile, settings
    │   │   │   │   └── shared/      # Reusable composables
    │   │   │   └── res/             # Resources
    │   │   ├── test/                # Unit tests
    │   │   └── androidTest/        # Instrumented tests
    │   └── build.gradle.kts
    ├── build.gradle.kts            # Root build file
    ├── settings.gradle.kts
    └── gradle/
        └── libs.versions.toml      # Version catalog
```

---

## 11. Milestones

| Week | Milestone | Deliverables |
|------|-----------|-------------|
| 1–2 | Scaffolding + Auth | Project setup, CI, login/register, token management |
| 2–3 | Push Notifications | **APNs + FCM registration, token management, basic notifications** |
| 3–4 | Core Messaging | Channel list, message list, send messages, WebSocket |
| 5–6 | DMs + Threads | DM list, DM chat, thread view, group DMs |
| 7–8 | Offline + Polish | Local cache, offline queue, pull-to-refresh, loading states |
| 9–10 | Notifications + Search | Notification list, full-text search |
| 11–12 | Voice + Discovery | Voice channel join, server discovery, invite links |
| 13–14 | Settings + Polish | User settings, server settings, accessibility, dark mode |
| 15–16 | Testing + Release | Beta testing, App Store / Play Store submission |
