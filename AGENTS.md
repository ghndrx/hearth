# AGENTS.md - Hearth Development Guide

## Project Overview
Hearth is a Discord clone with Go backend + SvelteKit frontend.

## Architecture

### Backend (`backend/`)
```
backend/
├── cmd/hearth/main.go       # Entry point
├── internal/
│   ├── api/handlers/        # HTTP handlers
│   ├── services/            # Business logic (interfaces + impl)
│   ├── models/              # Data models
│   ├── database/postgres/   # Repositories
│   └── gateway/             # WebSocket gateway
```

### Frontend (`frontend/`)
```
frontend/src/lib/
├── components/    # Reusable Svelte components
├── stores/        # Svelte stores (auth, servers, messages, etc.)
├── api.ts         # API client
└── gateway.ts     # WebSocket client
```

## Coding Standards

### Go Services
```go
package services

import (
    "context"
    "hearth/internal/models"
    "github.com/google/uuid"
)

// Interface first
type MyRepository interface {
    Get(ctx context.Context, id uuid.UUID) (*models.Thing, error)
}

// Then implementation
type MyService struct {
    repo MyRepository
    mu   sync.RWMutex  // For in-memory state
}

// Always accept context.Context as first param
func (s *MyService) DoThing(ctx context.Context, id uuid.UUID) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    // ...
}
```

### Svelte Components
```svelte
<script lang="ts">
  // Props with TypeScript
  export let variant: 'primary' | 'secondary' = 'primary';
  export let size: 'sm' | 'md' | 'lg' = 'md';
  
  // Reactive
  $: classes = `btn btn-${variant} btn-${size}`;
</script>

<!-- Forward events -->
<button class={classes} on:click>
  <slot />
</button>
```

## MVP Tasks (Priority Order)

### Backend - Remaining
1. Wire all services in `cmd/hearth/main.go` dependency injection
2. Add missing API routes in `api/handlers/` for new services
3. WebSocket gateway: handle all event types (typing, presence, reactions)
4. Rate limiting middleware
5. File upload handler (S3/local storage)

### Frontend - Remaining
1. `ServerSettings.svelte` - server config modal
2. `UserSettings.svelte` - user preferences
3. `MemberList.svelte` - server member sidebar
4. `VoiceChannel.svelte` - voice channel UI (WebRTC)
5. Wire stores to API endpoints
6. Polish message rendering (embeds, reactions, threads)

### Integration
1. Auth flow: login → token → WebSocket connect
2. Server creation → channel creation → send message
3. Real-time: typing indicators, presence updates
4. File uploads with progress

## Git Workflow
- Work on feature branches: `feat/backend-handlers`, `feat/frontend-components`
- Commit format: `feat|fix|docs|test: description`
- Always run `go build ./...` before committing Go code
- Run `npm run check` for Svelte

## Don't
- Don't import `database/postgres` directly in services (use interfaces)
- Don't skip tests for new services
- Don't use `any` type in TypeScript
- Don't commit broken code (build must pass)
