# AGENTS.md - Hearth Development Guide

> **READ THIS FIRST.** Copy code patterns exactly. Don't explore - just build.

---

## B1: Wire Services in main.go

**File:** `backend/cmd/hearth/main.go`
**Goal:** All services instantiated after repos, passed to handlers

**Current state (line ~55):**
```go
// Initialize repositories
repos := postgres.NewRepositories(db)
```

**ADD THIS after repos (copy exactly):**
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

**Verify:** `go build ./...`

---

## B2: Moderation Handler

**Create:** `backend/internal/api/handlers/moderation.go`

**Copy this entire file:**
```go
package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/services"
)

type ModerationHandler struct {
	modService *services.ModerationService
}

func NewModerationHandler(modService *services.ModerationService) *ModerationHandler {
	return &ModerationHandler{modService: modService}
}

// POST /api/servers/:serverId/bans
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

	if err := h.modService.BanUser(c.Context(), serverID, body.UserID, body.Reason); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(204).Send(nil)
}

// DELETE /api/servers/:serverId/bans/:userId
func (h *ModerationHandler) UnbanUser(c *fiber.Ctx) error {
	serverID, _ := uuid.Parse(c.Params("serverId"))
	userID, _ := uuid.Parse(c.Params("userId"))

	if err := h.modService.UnbanUser(c.Context(), serverID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(204).Send(nil)
}

// GET /api/servers/:serverId/bans
func (h *ModerationHandler) GetBans(c *fiber.Ctx) error {
	serverID, _ := uuid.Parse(c.Params("serverId"))

	bans, err := h.modService.GetBans(c.Context(), serverID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(bans)
}
```

**Then add routes in `handlers.go`:**
```go
// Moderation routes
servers.Post("/:serverId/bans", h.moderation.BanUser)
servers.Delete("/:serverId/bans/:userId", h.moderation.UnbanUser)
servers.Get("/:serverId/bans", h.moderation.GetBans)
```

---

## F1: ServerSettings Modal

**Create:** `frontend/src/lib/components/ServerSettings.svelte`

**Copy this:**
```svelte
<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Modal from './Modal.svelte';
	import Button from './Button.svelte';
	import Avatar from './Avatar.svelte';
	import ConfirmDialog from './ConfirmDialog.svelte';
	import { servers, deleteServer, updateServer } from '$lib/stores/servers';

	export let server: { id: string; name: string; icon?: string };
	export let open = false;

	const dispatch = createEventDispatcher<{ close: void }>();

	let name = server.name;
	let iconFile: File | null = null;
	let iconPreview = server.icon || '';
	let showDeleteConfirm = false;
	let saving = false;

	function handleIconChange(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files?.[0]) {
			iconFile = input.files[0];
			iconPreview = URL.createObjectURL(iconFile);
		}
	}

	async function handleSave() {
		saving = true;
		try {
			await updateServer(server.id, { name, icon: iconPreview });
			dispatch('close');
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		await deleteServer(server.id);
		dispatch('close');
	}
</script>

<Modal {open} title="Server Settings" on:close>
	<div class="space-y-4">
		<div class="flex items-center gap-4">
			<Avatar src={iconPreview} size="lg" />
			<input type="file" accept="image/*" on:change={handleIconChange} />
		</div>

		<label class="block">
			<span class="text-sm text-gray-400">Server Name</span>
			<input
				type="text"
				bind:value={name}
				class="w-full mt-1 px-3 py-2 bg-[#1e1f22] rounded border border-gray-700"
			/>
		</label>

		<div class="flex justify-between pt-4">
			<Button variant="danger" on:click={() => (showDeleteConfirm = true)}>
				Delete Server
			</Button>
			<div class="flex gap-2">
				<Button variant="secondary" on:click={() => dispatch('close')}>Cancel</Button>
				<Button on:click={handleSave} disabled={saving}>
					{saving ? 'Saving...' : 'Save'}
				</Button>
			</div>
		</div>
	</div>
</Modal>

<ConfirmDialog
	open={showDeleteConfirm}
	title="Delete Server"
	message="This cannot be undone. All channels and messages will be deleted."
	confirmText="Delete"
	on:confirm={handleDelete}
	on:cancel={() => (showDeleteConfirm = false)}
/>
```

---

## F2: Enhance MemberList

**File:** `frontend/src/lib/components/MemberList.svelte`

**Add to existing (or replace):**
```svelte
<script lang="ts">
	import { presence } from '$lib/stores/presence';
	import Avatar from './Avatar.svelte';
	import PresenceIndicator from './PresenceIndicator.svelte';

	export let members: Array<{
		id: string;
		username: string;
		avatar?: string;
		role: 'admin' | 'moderator' | 'member';
	}> = [];

	// Group by role
	$: admins = members.filter((m) => m.role === 'admin');
	$: mods = members.filter((m) => m.role === 'moderator');
	$: regularMembers = members.filter((m) => m.role === 'member');

	// Sort by online status
	function sortByPresence(list: typeof members) {
		return [...list].sort((a, b) => {
			const aOnline = $presence[a.id]?.status === 'online' ? 0 : 1;
			const bOnline = $presence[b.id]?.status === 'online' ? 0 : 1;
			return aOnline - bOnline;
		});
	}
</script>

<aside class="w-60 bg-[#2b2d31] p-3 overflow-y-auto">
	{#if admins.length}
		<div class="mb-4">
			<h3 class="text-xs uppercase text-gray-400 mb-2">Admin — {admins.length}</h3>
			{#each sortByPresence(admins) as member}
				<button class="flex items-center gap-2 w-full p-2 rounded hover:bg-[#35373c]">
					<div class="relative">
						<Avatar src={member.avatar} size="sm" />
						<PresenceIndicator status={$presence[member.id]?.status || 'offline'} />
					</div>
					<span class="text-sm">{member.username}</span>
				</button>
			{/each}
		</div>
	{/if}

	{#if mods.length}
		<div class="mb-4">
			<h3 class="text-xs uppercase text-gray-400 mb-2">Moderator — {mods.length}</h3>
			{#each sortByPresence(mods) as member}
				<button class="flex items-center gap-2 w-full p-2 rounded hover:bg-[#35373c]">
					<div class="relative">
						<Avatar src={member.avatar} size="sm" />
						<PresenceIndicator status={$presence[member.id]?.status || 'offline'} />
					</div>
					<span class="text-sm">{member.username}</span>
				</button>
			{/each}
		</div>
	{/if}

	<div>
		<h3 class="text-xs uppercase text-gray-400 mb-2">Members — {regularMembers.length}</h3>
		{#each sortByPresence(regularMembers) as member}
			<button class="flex items-center gap-2 w-full p-2 rounded hover:bg-[#35373c]">
				<div class="relative">
					<Avatar src={member.avatar} size="sm" />
					<PresenceIndicator status={$presence[member.id]?.status || 'offline'} />
				</div>
				<span class="text-sm">{member.username}</span>
			</button>
		{/each}
	</div>
</aside>
```

---

## Common Mistakes (DON'T)

1. ❌ `import "hearth/internal/database/postgres"` in services
2. ❌ Creating new stores when one exists
3. ❌ Using `any` type in TypeScript  
4. ❌ Forgetting `go build ./...` before commit
5. ❌ Creating files that already exist (check first!)

## Done Checklist

Before commit:
- [ ] Code compiles (`go build ./...` or `npm run check`)
- [ ] Follows existing patterns exactly
- [ ] No new dependencies added without reason
- [ ] Commit message: `feat|fix|test: description`
- [ ] Push immediately
