# Hearth - Product Requirements Document
## Discord Clone Specification

---

## 1. Overview

Hearth is a Discord-clone chat platform with web, desktop (Tauri), and mobile (React Native) clients sharing a common Go backend.

**Goal:** Pixel-perfect Discord UX so users feel instantly familiar.

---

## 2. Design System

### 2.1 Color Palette

```
/* Background Colors */
--bg-primary: #313338;      /* Main content area */
--bg-secondary: #2b2d31;    /* Sidebars, cards */
--bg-tertiary: #1e1f22;     /* Darkest - server bar, inputs */
--bg-modifier-hover: #35373c;
--bg-modifier-active: #404249;
--bg-modifier-selected: #404249;

/* Text Colors */
--text-normal: #f2f3f5;
--text-muted: #b5bac1;
--text-faint: #6d6f78;
--text-link: #00aff4;

/* Brand Colors */
--blurple: #5865f2;
--blurple-hover: #4752c4;
--green: #23a559;
--yellow: #f0b232;
--red: #da373c;
--fuchsia: #eb459e;

/* Status Colors */
--status-online: #23a559;
--status-idle: #f0b232;
--status-dnd: #f23f43;
--status-offline: #80848e;
```

### 2.2 Typography

```
/* Font Stack */
font-family: 'gg sans', 'Noto Sans', 'Helvetica Neue', Helvetica, Arial, sans-serif;

/* Sizes */
--font-size-xs: 12px;
--font-size-sm: 14px;
--font-size-md: 16px;
--font-size-lg: 20px;
--font-size-xl: 24px;

/* Line Heights */
--line-height-tight: 1.25;
--line-height-normal: 1.375;
--line-height-relaxed: 1.5;
```

### 2.3 Spacing

```
--spacing-xs: 4px;
--spacing-sm: 8px;
--spacing-md: 16px;
--spacing-lg: 24px;
--spacing-xl: 32px;
```

### 2.4 Border Radius

```
--radius-sm: 3px;
--radius-md: 4px;
--radius-lg: 8px;
--radius-xl: 16px;
--radius-full: 50%;
```

---

## 3. Layout Structure

### 3.1 Main Layout (Desktop/Web)

```
┌────────────────────────────────────────────────────────────────┐
│ [72px]           [240px]        [flex]           [240px]       │
│ Server    │    Channel    │               │     Member         │
│ Sidebar   │    Sidebar    │   Chat Area   │     Sidebar        │
│           │               │               │                    │
│ ┌──────┐  │ ┌───────────┐ │ ┌───────────┐ │ ┌────────────────┐ │
│ │ Home │  │ │Server Name│ │ │ #channel  │ │ │ ONLINE — 5     │ │
│ │ ──── │  │ ├───────────┤ │ ├───────────┤ │ ├────────────────┤ │
│ │ 🟢   │  │ │▸ CATEGORY │ │ │ Messages  │ │ │ 👤 User        │ │
│ │ 🔴   │  │ │  # general│ │ │           │ │ │ 👤 User        │ │
│ │ 🟢   │  │ │  # help   │ │ │           │ │ ├────────────────┤ │
│ │ ──── │  │ │▸ VOICE    │ │ │           │ │ │ OFFLINE — 12   │ │
│ │  +   │  │ │  🔊 chat  │ │ │           │ │ │ 👤 User        │ │
│ └──────┘  │ ├───────────┤ │ ├───────────┤ │ └────────────────┘ │
│           │ │👤 User    │ │ │ [Input]   │ │                    │
│           │ │🎤 🔇 ⚙️   │ │ └───────────┘ │                    │
│           │ └───────────┘ │               │                    │
└────────────────────────────────────────────────────────────────┘
```

### 3.2 Server Sidebar (72px wide)

**Components:**
- `ServerList.svelte` - Container for all server icons
- `ServerIcon.svelte` - Individual server icon
- `ServerFolder.svelte` - Expandable folder containing servers

**Server Icon States:**
| State | Visual |
|-------|--------|
| Default | Circle (48px), rounded-full |
| Hover | Rounded-2xl, slight scale up |
| Selected | Rounded-2xl, white pill left |
| Unread | White dot left of icon |
| Mentions | Red badge with count |

**Layout:**
1. Home button (Discord logo)
2. Separator line
3. Server icons (scrollable)
4. Add Server button (+)
5. Explore button (compass)

### 3.3 Channel Sidebar (240px wide)

**Components:**
- `ServerHeader.svelte` - Server name + dropdown
- `ChannelCategory.svelte` - Collapsible category
- `ChannelItem.svelte` - Text/voice channel
- `UserPanel.svelte` - Current user controls

**Channel Icons:**
| Type | Icon |
|------|------|
| Text | # |
| Voice | 🔊 |
| Announcement | 📢 |
| Stage | 🎭 |
| Forum | 💬 |
| Private | 🔒 (prefix) |

**User Panel:**
- Avatar (32px)
- Username + discriminator (or display name)
- Mute button (mic icon)
- Deafen button (headphones icon)
- Settings button (gear icon)

### 3.4 Chat Area (flexible width)

**Components:**
- `ChannelHeader.svelte` - Channel name, topic, actions
- `MessageList.svelte` - Scrollable message container
- `Message.svelte` - Individual message
- `MessageGroup.svelte` - Grouped messages from same author
- `MessageInput.svelte` - Compose area

**Message Structure:**
```
┌─────────────────────────────────────────────────────┐
│ [Avatar]  Username              Today at 3:45 PM   │
│           Message content here                      │
│           Second line of message                    │
│                                                     │
│           Another message (no avatar repeat)        │
│           [😂 3] [❤️ 1]  <- Reactions               │
└─────────────────────────────────────────────────────┘
```

**Message Grouping Rules:**
- Same author within 7 minutes = grouped (no repeat avatar)
- System messages break groups
- Reply to message breaks groups

**Hover Actions:**
- React (emoji icon)
- Edit (pencil - own messages only)
- Reply (arrow)
- More (three dots)

### 3.5 Member Sidebar (240px wide)

**Components:**
- `MemberList.svelte` - Container
- `MemberSection.svelte` - Role group (ONLINE, OFFLINE, role names)
- `MemberItem.svelte` - User row

**Member Row:**
- Avatar (32px) with status dot
- Display name (colored by highest role)
- Activity/game name if present

---

## 4. Component Specifications

### 4.1 Buttons

**Primary Button:**
```css
background: var(--blurple);
color: white;
padding: 8px 16px;
border-radius: 3px;
font-size: 14px;
font-weight: 500;
```

**Secondary Button:**
```css
background: transparent;
color: var(--text-normal);
border: none;
```

**Danger Button:**
```css
background: var(--red);
color: white;
```

### 4.2 Inputs

**Text Input:**
```css
background: var(--bg-tertiary);
border: none;
border-radius: 3px;
padding: 10px;
color: var(--text-normal);
```

**Focus State:**
```css
outline: none;
box-shadow: 0 0 0 2px var(--blurple);
```

### 4.3 Modals

- Centered, max-width 440px (small) or 600px (large)
- Background overlay: rgba(0, 0, 0, 0.85)
- Modal background: var(--bg-primary)
- Border radius: 4px
- Header: 60px height
- Footer: 60px height, bg var(--bg-secondary)

### 4.4 Tooltips

- Background: #111214
- Color: var(--text-normal)
- Padding: 8px 12px
- Border radius: 4px
- Font size: 14px
- Shadow: 0 8px 16px rgba(0, 0, 0, 0.24)

### 4.5 Context Menus

- Background: #111214
- Border radius: 4px
- Padding: 6px 8px
- Min-width: 188px
- Item hover: var(--blurple)

---

## 5. Animations

### 5.1 Transitions

```css
--transition-fast: 0.1s ease;
--transition-normal: 0.2s ease;
--transition-slow: 0.3s ease;
```

### 5.2 Server Icon Hover

```css
.server-icon {
  transition: border-radius 0.15s ease, background-color 0.15s ease;
}
.server-icon:hover {
  border-radius: 16px; /* from 50% to 16px */
}
```

### 5.3 Channel Pill Indicator

- Appears on left of server icon
- Height: 8px (unread), 20px (hover), 40px (selected)
- Width: 4px
- Border radius: 0 4px 4px 0
- Background: white
- Transition: height 0.15s ease

---

## 6. Mobile Layout

### 6.1 Navigation

- Bottom tab bar: Servers, Messages, Notifications, You
- Swipe right: Server list
- Swipe left: Member list

### 6.2 Server List (Mobile)

- Horizontal scrollable row at top
- Or drawer from left

### 6.3 Chat (Mobile)

- Full screen messages
- Floating action button for compose
- Swipe actions on messages

---

## 7. Feature Parity Checklist

### 7.1 Core Features (MVP)
- [ ] User registration/login
- [ ] Server creation/joining
- [ ] Channel management
- [ ] Real-time messaging
- [ ] Message editing/deletion
- [ ] File uploads (images)
- [ ] Emoji reactions
- [ ] Reply to messages
- [ ] User profiles
- [ ] Role management
- [ ] Permissions

### 7.2 Phase 2
- [ ] Voice channels (WebRTC)
- [ ] Video calls
- [ ] Screen sharing
- [ ] Threads
- [ ] Pins
- [ ] Search
- [ ] DMs
- [ ] Friend system

### 7.3 Phase 3
- [ ] Bots API
- [ ] Webhooks
- [ ] Integrations
- [ ] Custom emojis
- [ ] Server boosts
- [ ] Nitro-equivalent

---

## 8. API Endpoints Reference

See `backend/docs/api.md` for full API documentation.

Key endpoints:
- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/servers`
- `POST /api/servers`
- `GET /api/servers/:id/channels`
- `GET /api/channels/:id/messages`
- `POST /api/channels/:id/messages`
- `WebSocket /api/ws`

---

## 9. File Structure

### 9.1 Frontend (Svelte)
```
frontend/src/
├── routes/
│   ├── +layout.svelte        # Main app layout
│   ├── +page.svelte           # Home/login
│   ├── channels/
│   │   └── [id]/+page.svelte  # Channel view
│   └── servers/
│       └── [id]/+page.svelte  # Server view
├── lib/
│   ├── components/
│   │   ├── server/            # Server-related
│   │   ├── channel/           # Channel-related
│   │   ├── message/           # Message-related
│   │   ├── user/              # User-related
│   │   └── common/            # Shared components
│   ├── stores/                # Svelte stores
│   └── utils/                 # Helpers
└── app.css                    # Global styles + CSS vars
```

### 9.2 Backend (Go)
```
backend/
├── cmd/server/main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes.go
│   ├── models/
│   ├── services/
│   ├── database/
│   └── websocket/
└── pkg/
```

---

## 10. Agent Instructions

When working on Hearth UI components:

1. **Always reference this PRD** for colors, spacing, and component specs
2. **Match Discord exactly** - when in doubt, inspect Discord's actual UI
3. **Use CSS variables** defined in the Design System section
4. **Follow component structure** outlined in Section 4
5. **Test dark mode only** - light mode is not supported initially
6. **Commit often** with descriptive messages
7. **Write tests** for logic-heavy components

---

*Last updated: 2026-02-14*
*Version: 1.0*
