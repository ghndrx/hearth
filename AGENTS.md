# AGENTS.md - Hearth Development Guide

## Project Overview
Hearth is a Discord clone with Go backend + SvelteKit frontend.
**Goal:** MVP that lets users create servers, channels, send messages in real-time.

## Architecture

### Backend (`backend/`)
```
backend/
├── cmd/hearth/main.go       # Entry point, DI wiring
├── internal/
│   ├── api/handlers/        # HTTP handlers
│   ├── services/            # Business logic (24 services exist)
│   ├── models/              # Data models
│   ├── database/postgres/   # Repositories
│   └── gateway/             # WebSocket gateway
```

### Frontend (`frontend/`)
```
frontend/src/lib/
├── components/    # 33 Svelte components exist
├── stores/        # Svelte stores (auth, servers, messages, etc.)
├── api.ts         # API client
└── gateway.ts     # WebSocket client
```

---

## Coding Standards

### Go Services
```go
package services

import (
    "context"
    "sync"
    "hearth/internal/models"
    "github.com/google/uuid"
)

// Interface first
type MyRepository interface {
    Get(ctx context.Context, id uuid.UUID) (*models.Thing, error)
}

// Then implementation with mutex for thread safety
type MyService struct {
    repo MyRepository
    mu   sync.RWMutex
}

func NewMyService(repo MyRepository) *MyService {
    return &MyService{repo: repo}
}

// Always accept context.Context as first param
func (s *MyService) DoThing(ctx context.Context, id uuid.UUID) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.repo.Get(ctx, id)
}
```

### Svelte Components
```svelte
<script lang="ts">
  export let variant: 'primary' | 'secondary' = 'primary';
  export let disabled = false;
  
  $: classes = `btn btn-${variant}`;
</script>

<button class={classes} {disabled} on:click>
  <slot />
</button>
```

---

## MVP Task Queue

Pick ONE task. Complete it fully. Commit and push.

### BACKEND TASKS

#### B1: Wire Services in main.go
**Goal:** All 24 services instantiated and injected into handlers
**File:** `cmd/hearth/main.go`
**Steps:**
1. Import all services from `internal/services`
2. Create NewXxxService() for each service
3. Pass services to handler constructors
4. Verify with `go build ./...`

#### B2: Ban/Mute API Routes
**Goal:** REST endpoints for moderation
**File:** `api/handlers/moderation.go`
**Endpoints:**
- POST /api/servers/{id}/bans - ban user
- DELETE /api/servers/{id}/bans/{userId} - unban
- POST /api/servers/{id}/mutes - mute user
- DELETE /api/servers/{id}/mutes/{userId} - unmute
**Use:** BanService, MuteService from services/

#### B3: Poll API Routes
**Goal:** Create and vote on polls
**File:** `api/handlers/poll.go`
**Endpoints:**
- POST /api/channels/{id}/polls - create poll
- POST /api/polls/{id}/vote - cast vote
- GET /api/polls/{id} - get results
**Use:** PollService from services/

#### B4: File Upload Handler
**Goal:** Upload attachments to messages
**File:** `api/handlers/upload.go`
**Endpoints:**
- POST /api/upload - multipart file upload
- Return: {url, filename, size, contentType}
**Use:** AttachmentService, local storage for MVP

### FRONTEND TASKS

#### F1: ServerSettings Modal
**Goal:** Server owners can edit server settings
**File:** `components/ServerSettings.svelte`
**Features:**
- Server name input
- Server icon upload (preview)
- Delete server button (with confirm)
- Save/Cancel buttons
**Use:** Modal.svelte as wrapper, servers store

#### F2: MemberList Sidebar
**Goal:** Show server members with presence
**File:** `components/MemberList.svelte`
**Features:**
- Group by role (Admin, Moderator, Member)
- Show online/offline/idle status dot
- Click member → show profile popup
- Search/filter members
**Use:** presence store, servers store

#### F3: UserSettings Modal
**Goal:** User preferences and account
**File:** `components/UserSettings.svelte`
**Tabs:**
- Account: avatar, username, email
- Appearance: theme toggle (dark/light)
- Notifications: enable/disable, sounds
- Privacy: block list
**Use:** Modal.svelte, settings store

#### F4: VoiceChannel Component
**Goal:** Basic voice channel UI (no WebRTC yet)
**File:** `components/VoiceChannel.svelte`
**Features:**
- Show users in voice channel
- Join/Leave buttons
- Mute/Deafen toggles
- Speaking indicator (placeholder)

### INTEGRATION TASKS

#### I1: Auth Flow End-to-End
**Goal:** Login works completely
**Test flow:**
1. User enters email/password in LoginForm
2. POST /api/auth/login returns JWT
3. Store token in auth store + localStorage
4. WebSocket connects with token
5. Redirect to server list
**Fix any broken links in this chain.**

#### I2: Message Send Flow
**Goal:** Sending a message works completely
**Test flow:**
1. User types in MessageInput
2. Press Enter or click Send
3. POST /api/channels/{id}/messages
4. WebSocket broadcasts MESSAGE_CREATE
5. MessageList shows new message
6. Author sees their message immediately (optimistic)

#### I3: Typing Indicators
**Goal:** See when others are typing
**Flow:**
1. User types → emit TYPING_START via WebSocket
2. Server broadcasts to channel members
3. Other clients show "User is typing..."
4. Clear after 5 seconds or MESSAGE_CREATE

### TEST TASKS

#### T1: Service Unit Tests
**Goal:** 80%+ coverage on services
**Pattern:**
```go
func TestBanService_BanUser(t *testing.T) {
    repo := &mockBanRepo{}
    svc := NewBanService(repo)
    
    err := svc.BanUser(ctx, serverID, userID, "reason")
    assert.NoError(t, err)
}
```

#### T2: WebSocket Integration Tests
**Goal:** Test real-time events
**Test:** Connect two clients, one sends message, other receives

#### T3: API Integration Tests
**Goal:** Test HTTP endpoints end-to-end
**Use:** httptest.NewServer, test auth + CRUD

---

## Git Workflow
- Branch: use assigned feature branch
- Commit: `feat|fix|test: clear description`
- Build check: `go build ./...` (Go) or `npm run check` (Svelte)
- Push: always push after commit

## Don't
- Don't import `database/postgres` in services (use interfaces)
- Don't skip tests
- Don't use `any` in TypeScript
- Don't commit broken code
- Don't create new files without checking if they exist first
