# AGENTS.md - Hearth Development Guide

## Quick Context
- **Backend:** 15+ services exist, most not wired to handlers yet
- **Frontend:** 20+ components, 11 stores exist
- **Goal:** Wire everything together for working MVP

---

## BACKEND TASKS

### B1: Wire Services in main.go
**Goal:** Instantiate all services and inject into handlers

**File to modify:** `backend/cmd/hearth/main.go`

**Services that exist (need wiring):**
```
attachment_service.go   bookmark_service.go    channel_service.go
emoji_service.go        invite_service.go      message_service.go
moderation_service.go   notification_service.go presence_service.go
reaction_service.go     readstate_service.go   report_service.go
thread_service.go       typing_service.go      voicestate_service.go
auditlog_service.go     auth_service.go
```

**Pattern to follow:**
```go
// In main.go
attachmentSvc := services.NewAttachmentService()
bookmarkSvc := services.NewBookmarkService()
// ... etc

handlers := api.NewHandlers(
    authSvc,
    channelSvc,
    messageSvc,
    attachmentSvc,
    // ... pass all services
)
```

**Verify:** `go build ./...` must pass

---

### B2: Ban/Mute API Routes
**Goal:** Moderation endpoints

**Create:** `backend/internal/api/handlers/moderation.go`

**Existing services to use:**
- `services.BanService` (if exists) or `services.ModerationService`

**Existing handler pattern (copy from):** `handlers/users.go`

**Endpoints needed:**
```go
POST   /api/servers/{serverId}/bans          → BanUser(serverID, userID, reason)
DELETE /api/servers/{serverId}/bans/{userId} → UnbanUser(serverID, userID)
GET    /api/servers/{serverId}/bans          → GetBans(serverID)
```

**Wire in:** `handlers/handlers.go` router setup

---

### B3: Poll API Routes
**Goal:** Polls in channels

**Create:** `backend/internal/api/handlers/polls.go`

**Existing service:** Check if `poll_service.go` exists, else create

**Endpoints:**
```go
POST /api/channels/{channelId}/polls     → CreatePoll(channelID, question, options)
POST /api/polls/{pollId}/vote            → Vote(pollID, optionIndex)
GET  /api/polls/{pollId}                 → GetPoll(pollID)
```

---

### B4: File Upload Handler
**Goal:** Upload attachments

**Create:** `backend/internal/api/handlers/upload.go`

**Existing service:** `attachment_service.go`

**Endpoint:**
```go
POST /api/upload  (multipart/form-data)
Response: {"url": "...", "filename": "...", "size": 12345, "contentType": "image/png"}
```

**Storage:** Use local `./uploads/` dir for MVP, return `/uploads/{filename}` URL

---

## FRONTEND TASKS

### F1: ServerSettings Modal
**Goal:** Server owners edit settings

**Create:** `frontend/src/lib/components/ServerSettings.svelte`

**Existing components to use:**
- `Modal.svelte` - wrapper
- `Button.svelte` - actions
- `Avatar.svelte` - server icon preview

**Existing stores:**
- `servers.ts` - has `updateServer()`, `deleteServer()`

**Props:** `export let server: Server; export let onClose: () => void;`

**Features:**
- Input for server name (bind to local state)
- Icon upload with preview
- Delete button → ConfirmDialog → deleteServer()
- Save button → updateServer()

---

### F2: MemberList Sidebar  
**Goal:** Show server members

**File exists:** `frontend/src/lib/components/MemberList.svelte` (enhance it)

**Existing stores:**
- `presence.ts` - has user online status
- `servers.ts` - has members

**Enhance to include:**
- Group by role (Admin → Mod → Member)
- Show colored dot: green=online, yellow=idle, gray=offline
- Click member → show mini profile

---

### F3: UserSettings Modal
**Goal:** User preferences

**Create:** `frontend/src/lib/components/UserSettings.svelte`

**Existing stores:**
- `settings.ts` - theme, notifications
- `auth.ts` - user info

**Tabs (use simple div toggle):**
1. **Account:** avatar, username, email (readonly for MVP)
2. **Appearance:** theme select (dark/light)
3. **Notifications:** toggles for sounds, desktop notifs

---

### F4: VoiceChannel Component
**Goal:** Voice UI (no WebRTC yet - just UI)

**Create:** `frontend/src/lib/components/VoiceChannel.svelte`

**Show:**
- List of users in voice channel
- Join / Leave button
- Mute mic toggle
- Deafen audio toggle
- Speaking indicator (just visual, no real audio)

---

## INTEGRATION TASKS

### I1: Auth Flow
**Files to check/wire:**
1. `frontend/src/routes/login/+page.svelte` → form submit
2. `frontend/src/lib/api.ts` → POST /api/auth/login
3. `frontend/src/lib/stores/auth.ts` → store token
4. `frontend/src/lib/gateway.ts` → connect with token
5. Redirect to `/` after login

**Test:** Login with test user, verify WebSocket connects

---

### I2: Message Send Flow
**Files:**
1. `components/MessageInput.svelte` → on submit
2. `stores/messages.ts` → sendMessage()
3. `api.ts` → POST /api/channels/{id}/messages
4. `gateway.ts` → receive MESSAGE_CREATE
5. `components/MessageList.svelte` → shows new message

---

### I3: Typing Indicators
**Files:**
1. `components/MessageInput.svelte` → on keypress emit typing
2. `stores/typing.ts` → track who's typing
3. `gateway.ts` → send/receive TYPING_START
4. Show "{user} is typing..." below MessageList

---

## TEST TASKS

### T1: Service Unit Tests
**Add tests for:** Any service in `services/` missing `_test.go`

**Pattern:**
```go
func TestXxxService_Method(t *testing.T) {
    svc := NewXxxService(mockRepo)
    result, err := svc.Method(ctx, input)
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

---

### T2: WebSocket Integration Tests
**File:** `backend/internal/gateway/gateway_test.go`

**Test:** Two clients connect, client A sends message, client B receives

---

## Coding Standards

### Go
- Package `services`, import `hearth/internal/models`
- Always use `context.Context` first param
- Use `sync.RWMutex` for in-memory state
- Don't import `database/postgres` in services

### Svelte
- TypeScript for all components
- Export props at top
- Use existing stores, don't create new ones
- Forward events with `on:click`

## Git
- Commit: `feat|fix|test: description`
- Always `go build ./...` or `npm run check` before commit
- Push immediately after commit
