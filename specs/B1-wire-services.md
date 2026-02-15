# B1: Wire Services in main.go

## Goal
All 24 services instantiated in main.go and injected into handlers.

## Input Files (READ THESE FIRST)
- `backend/cmd/hearth/main.go` - Current state
- `backend/internal/services/*.go` - All service constructors

## Output Files
- `backend/cmd/hearth/main.go` - Modified with service wiring

## Current State
```go
// Line ~55 - repos initialized
repos := postgres.NewRepositories(db)
// MISSING: service initialization
// MISSING: handler injection
```

## Required Changes

### 1. Add service imports (if not present)
```go
import (
    "hearth/internal/services"
)
```

### 2. After repos initialization, add:
```go
// Initialize services
authSvc := services.NewAuthService(repos.Users, cfg.JWTSecret)
userSvc := services.NewUserService(repos.Users)
serverSvc := services.NewServerService(repos.Servers, repos.Members)
channelSvc := services.NewChannelService(repos.Channels)
messageSvc := services.NewMessageService(repos.Messages)
inviteSvc := services.NewInviteService(repos.Invites)
attachmentSvc := services.NewAttachmentService()
bookmarkSvc := services.NewBookmarkService()
emojiSvc := services.NewEmojiService()
moderationSvc := services.NewModerationService()
notificationSvc := services.NewNotificationService()
presenceSvc := services.NewPresenceService()
reactionSvc := services.NewReactionService()
readstateSvc := services.NewReadStateService()
threadSvc := services.NewThreadService()
typingSvc := services.NewTypingService()
voicestateSvc := services.NewVoiceStateService()
auditlogSvc := services.NewAuditLogService()
reportSvc := services.NewReportService()
```

### 3. Pass services to handlers constructor

## Acceptance Criteria
- [ ] `go build ./...` passes
- [ ] All services from `internal/services/` are instantiated
- [ ] Services are passed to handler constructors
- [ ] No unused variable errors

## Verification Command
```bash
cd backend && go build ./... && echo "✅ Build passes"
```

## Commit Message
```
feat: wire all services in main.go dependency injection
```
