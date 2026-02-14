# B2: Moderation API Routes

## Goal
REST endpoints for banning/unbanning and muting/unmuting users.

## Input Files (READ THESE FIRST)
- `backend/internal/api/handlers/users.go` - Handler pattern to copy
- `backend/internal/services/moderation_service.go` - Service to use

## Output Files
- `backend/internal/api/handlers/moderation.go` - NEW FILE

## Required Endpoints

### POST /api/servers/:serverId/bans
Ban a user from a server.
```json
Request: {"user_id": "uuid", "reason": "string"}
Response: 204 No Content
```

### DELETE /api/servers/:serverId/bans/:userId
Unban a user.
```json
Response: 204 No Content
```

### GET /api/servers/:serverId/bans
List all bans.
```json
Response: [{"user_id": "...", "reason": "...", "banned_at": "..."}]
```

## Complete Code
```go
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"hearth/internal/services"
)

type ModerationHandler struct {
	svc *services.ModerationService
}

func NewModerationHandler(svc *services.ModerationService) *ModerationHandler {
	return &ModerationHandler{svc: svc}
}

func (h *ModerationHandler) BanUser(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid server id"})
	}

	var body struct {
		UserID uuid.UUID `json:"user_id"`
		Reason string    `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	if err := h.svc.BanUser(c.Context(), serverID, body.UserID, body.Reason); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(204)
}

func (h *ModerationHandler) UnbanUser(c *fiber.Ctx) error {
	serverID, _ := uuid.Parse(c.Params("serverId"))
	userID, _ := uuid.Parse(c.Params("userId"))

	if err := h.svc.UnbanUser(c.Context(), serverID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(204)
}

func (h *ModerationHandler) GetBans(c *fiber.Ctx) error {
	serverID, _ := uuid.Parse(c.Params("serverId"))

	bans, err := h.svc.GetBans(c.Context(), serverID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(bans)
}
```

## Wire Routes
Add to `handlers.go` router:
```go
servers.Post("/:serverId/bans", h.moderation.BanUser)
servers.Delete("/:serverId/bans/:userId", h.moderation.UnbanUser)
servers.Get("/:serverId/bans", h.moderation.GetBans)
```

## Acceptance Criteria
- [ ] `go build ./...` passes
- [ ] Handler follows existing patterns
- [ ] Routes wired in handlers.go

## Verification Command
```bash
cd backend && go build ./... && echo "✅ Build passes"
```

## Commit Message
```
feat: add moderation API routes for ban/unban
```
