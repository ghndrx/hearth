# Hearth — Feature Specification

**Version:** 1.0  
**Last Updated:** 2026-02-11

This document provides detailed specifications for every core feature, organized by category. Use this as the implementation reference.

---

## Table of Contents

1. [User Accounts](#1-user-accounts)
2. [Servers](#2-servers)
3. [Channels](#3-channels)
4. [Messaging](#4-messaging)
5. [Direct Messages](#5-direct-messages)
6. [Voice & Video](#6-voice--video)
7. [Roles & Permissions](#7-roles--permissions)
8. [Moderation](#8-moderation)
9. [Social Features](#9-social-features)
10. [Notifications](#10-notifications)
11. [Search](#11-search)
12. [Integrations](#12-integrations)
13. [Accessibility](#13-accessibility)
14. [Mobile Experience](#14-mobile-experience)

---

## 1. User Accounts

### 1.1 Registration

| Feature | Description | Priority |
|---------|-------------|----------|
| Email + Password | Standard registration with email verification | P0 |
| Username selection | Unique username with discriminator (#1234) or unique handle | P0 |
| Password requirements | Min 8 chars, complexity optional by admin | P0 |
| CAPTCHA | Optional hCaptcha/reCAPTCHA for spam prevention | P2 |
| Invite-only mode | Require invite code to register | P1 |
| OAuth registration | Sign up with Google, GitHub, Discord | P2 |

### 1.2 Authentication

| Feature | Description | Priority |
|---------|-------------|----------|
| Email + Password login | Standard login | P0 |
| Remember me | Persistent session option | P0 |
| Two-factor auth (TOTP) | Google Authenticator, Authy, etc. | P1 |
| Backup codes | 10 single-use recovery codes | P1 |
| OAuth login | Google, GitHub, Discord, OIDC | P2 |
| Session management | View active sessions, revoke any | P1 |
| Password reset | Email-based reset flow | P0 |
| Account recovery | Recovery codes, trusted devices | P2 |

### 1.3 User Profile

| Feature | Description | Priority |
|---------|-------------|----------|
| Avatar | Upload image (PNG, JPG, GIF), max 8MB | P0 |
| Avatar cropping | Crop/resize before upload | P1 |
| Animated avatars | GIF support | P2 |
| Banner image | Profile header image | P2 |
| Display name | Customizable name (separate from username) | P0 |
| About me / Bio | Rich text, max 190 chars, Markdown | P1 |
| Pronouns | Optional pronoun field | P2 |
| Connected accounts | Show GitHub, Twitter, etc. | P2 |

### 1.4 Presence & Status

| Feature | Description | Priority |
|---------|-------------|----------|
| Online status | Online, Idle, Do Not Disturb, Invisible | P0 |
| Auto-idle | Switch to Idle after inactivity (configurable) | P1 |
| Custom status | Text + optional emoji, expiration | P1 |
| Activity status | "Playing...", "Listening to..." | P3 |
| Mobile indicator | Show when using mobile app | P2 |

### 1.5 Account Settings

| Feature | Description | Priority |
|---------|-------------|----------|
| Change email | With verification | P0 |
| Change password | Require current password | P0 |
| Change username | Rate limited (2/hour) | P1 |
| Delete account | Permanent, with confirmation | P1 |
| Data export | GDPR export (JSON) | P1 |
| Privacy settings | Who can DM, friend request, etc. | P1 |

---

## 2. Servers

### 2.1 Server Management

| Feature | Description | Priority |
|---------|-------------|----------|
| Create server | Name, icon, template selection | P0 |
| Server icon | 512x512 recommended, GIF for animated | P0 |
| Server banner | Displayed in server header | P2 |
| Server name | 2-100 characters | P0 |
| Server description | For discovery, max 300 chars | P2 |
| Edit server | Update name, icon, settings | P0 |
| Delete server | Owner only, confirmation required | P0 |
| Transfer ownership | To existing member | P1 |

### 2.2 Server Templates

| Feature | Description | Priority |
|---------|-------------|----------|
| Built-in templates | Gaming, Study Group, Community, Friends | P2 |
| Create from template | Pre-configured channels and roles | P2 |
| Save as template | Export current server structure | P3 |

### 2.3 Invites

| Feature | Description | Priority |
|---------|-------------|----------|
| Generate invite | Unique 8-char code | P0 |
| Invite link | https://hearth.example.com/invite/CODE | P0 |
| Expiration | Never, 30min, 1h, 6h, 12h, 1d, 7d, custom | P0 |
| Max uses | Unlimited, 1, 5, 10, 25, 50, 100 | P0 |
| Temporary membership | Kick when disconnect (if no role) | P2 |
| Invite tracking | Who used which invite | P1 |
| Revoke invite | Delete invite code | P0 |
| View all invites | List with stats | P1 |
| Vanity URL | Custom slug (e.g., /join/gamers) | P3 |

### 2.4 Server Discovery

| Feature | Description | Priority |
|---------|-------------|----------|
| Public listing | Opt-in to discovery | P3 |
| Categories | Gaming, Music, Education, etc. | P3 |
| Search | Find public servers | P3 |
| Preview | See description, member count before join | P3 |

### 2.5 Server Organization

| Feature | Description | Priority |
|---------|-------------|----------|
| Server list | Left sidebar with all joined servers | P0 |
| Server order | Drag to reorder | P0 |
| Server folders | Group servers into collapsible folders | P2 |
| Folder naming | Custom folder names | P2 |
| Folder colors | Color-coded folders | P3 |

---

## 3. Channels

### 3.1 Channel Types

| Type | Description | Priority |
|------|-------------|----------|
| Text | Standard messaging channel | P0 |
| Voice | Real-time voice communication | P1 |
| Video | Voice + video/screen share | P2 |
| Announcement | Cross-server following, restricted posting | P2 |
| Forum | Threaded discussions, tags | P2 |
| Stage | Moderated presentations, speakers/audience | P3 |
| Category | Grouping container (not a real channel) | P0 |

### 3.2 Channel Settings

| Feature | Description | Priority |
|---------|-------------|----------|
| Channel name | 1-100 chars, lowercase, hyphens allowed | P0 |
| Channel topic | Description shown in header, max 1024 chars | P0 |
| Position | Drag to reorder within category | P0 |
| Category | Move to/from category | P0 |
| Slowmode | Rate limit: Off, 5s, 10s, 15s, 30s, 1m, 2m, 5m, 10m, 15m, 30m, 1h, 2h, 6h | P1 |
| NSFW | Age-gated content warning | P1 |
| Default thread archive | 1h, 24h, 3d, 1w | P2 |

### 3.3 Channel Permissions

| Feature | Description | Priority |
|---------|-------------|----------|
| Permission overrides | Per-channel role/user permissions | P0 |
| Sync with category | Inherit category permissions | P1 |
| Private channels | Only visible to specific roles | P0 |
| Read-only channels | View but not send | P0 |

### 3.4 Voice Channel Settings

| Feature | Description | Priority |
|---------|-------------|----------|
| Bitrate | 8kbps - 384kbps | P1 |
| User limit | 0 (unlimited) to 99 | P1 |
| Video quality | 720p, 1080p (if available) | P2 |
| Region override | Force specific voice region | P2 |

### 3.5 Forum Channels

| Feature | Description | Priority |
|---------|-------------|----------|
| Create post | Title + content, creates thread | P2 |
| Tags | Categorize posts, filter by tag | P2 |
| Required tags | Must select at least one tag | P2 |
| Default sort | Latest activity, creation date | P2 |
| Post guidelines | Shown when creating post | P2 |

---

## 4. Messaging

### 4.1 Core Messaging

| Feature | Description | Priority |
|---------|-------------|----------|
| Send message | Real-time delivery via WebSocket | P0 |
| Message length | Max 2000 characters | P0 |
| Edit message | Update content, show (edited) indicator | P0 |
| Delete message | Remove own messages | P0 |
| Message history | Infinite scroll, lazy load | P0 |
| Jump to message | Link to specific message | P1 |
| Keyboard shortcuts | Enter to send, Shift+Enter for newline | P0 |

### 4.2 Text Formatting (Markdown)

| Syntax | Result | Priority |
|--------|--------|----------|
| `*italic*` or `_italic_` | *italic* | P0 |
| `**bold**` | **bold** | P0 |
| `***bold italic***` | ***bold italic*** | P0 |
| `~~strikethrough~~` | ~~strikethrough~~ | P0 |
| `__underline__` | underline | P1 |
| `||spoiler||` | Hidden until clicked | P1 |
| `` `inline code` `` | `inline code` | P0 |
| ` ```code block``` ` | Multi-line code | P0 |
| ` ```lang ` | Syntax highlighting | P1 |
| `> quote` | Block quote | P0 |
| `>>> multi-line quote` | Extended quote | P1 |
| `# Heading` | Large text | P2 |
| `- list` | Bulleted list | P1 |
| `1. list` | Numbered list | P1 |
| `[text](url)` | Hyperlink | P1 |

### 4.3 Rich Content

| Feature | Description | Priority |
|---------|-------------|----------|
| Link embeds | Auto-preview with title, description, image | P1 |
| Image embeds | Inline image display, lightbox | P0 |
| Video embeds | YouTube, Vimeo, etc. player | P2 |
| Twitter/X embeds | Tweet preview | P2 |
| GIF picker | Tenor/Giphy integration | P2 |
| Tenor/Giphy search | Search and send GIFs | P2 |

### 4.4 File Attachments

| Feature | Description | Priority |
|---------|-------------|----------|
| Upload files | Drag-drop or file picker | P0 |
| File size limit | Default 8MB, configurable to 100MB | P0 |
| Multiple files | Up to 10 per message | P0 |
| Image preview | Thumbnail in chat | P0 |
| Video preview | Playable in chat | P1 |
| Audio preview | Playable in chat | P2 |
| File download | Click to download | P0 |
| Upload progress | Progress bar during upload | P0 |

**Supported File Types:**

| Category | Extensions |
|----------|------------|
| Images | jpg, jpeg, png, gif, webp |
| Videos | mp4, webm, mov |
| Audio | mp3, wav, ogg, flac |
| Documents | pdf, txt, doc, docx, xls, xlsx, ppt, pptx |
| Archives | zip, rar, 7z, tar, gz |
| Code | js, py, go, rs, java, c, cpp, h, json, yaml, xml, html, css |

### 4.5 Emoji & Reactions

| Feature | Description | Priority |
|---------|-------------|----------|
| Unicode emoji | Full Unicode 15.0 support | P0 |
| Emoji picker | Categorized, searchable | P0 |
| Skin tone variants | Select default skin tone | P1 |
| Custom emoji | Server-specific, up to 50 static + 50 animated | P1 |
| Emoji upload | Admins upload custom emoji | P1 |
| Emoji size | Small in text, large if message is only emoji | P0 |
| Add reaction | Click + emoji to react | P0 |
| Reaction count | Show count per emoji | P0 |
| Reaction tooltip | Show who reacted on hover | P1 |
| Remove reaction | Click own reaction to remove | P0 |
| Reaction permissions | Require permission to react | P2 |

### 4.6 Mentions

| Feature | Description | Priority |
|---------|-------------|----------|
| @user | Mention specific user, autocomplete | P0 |
| @role | Mention all users with role | P0 |
| @everyone | Mention all server members | P0 |
| @here | Mention online members only | P0 |
| #channel | Link to channel | P0 |
| Mention highlight | Visual indicator when mentioned | P0 |
| Mention notification | Alert user of mention | P0 |
| Mention permissions | Require permission for @everyone/@here | P0 |

### 4.7 Replies & References

| Feature | Description | Priority |
|---------|-------------|----------|
| Reply to message | Quote with reference link | P0 |
| Reply preview | Show snippet of original | P0 |
| Jump to original | Click to scroll to original message | P0 |
| Reply ping | Option to notify original author | P0 |
| Forward message | Share to another channel/DM | P2 |

### 4.8 Message Pinning

| Feature | Description | Priority |
|---------|-------------|----------|
| Pin message | Add to pinned messages | P1 |
| Pinned messages panel | View all pins | P1 |
| Pin limit | 50 per channel | P1 |
| Unpin message | Remove from pins | P1 |
| Pin notification | System message when pinned | P2 |

### 4.9 Threads

| Feature | Description | Priority |
|---------|-------------|----------|
| Create thread | Start from any message | P1 |
| Thread name | Auto-generated or custom | P1 |
| Thread sidebar | List of active threads | P1 |
| Thread notifications | Separate from channel | P1 |
| Auto-archive | After 1h, 24h, 3d, 1w inactivity | P1 |
| Thread permissions | Inherit or override | P1 |
| Join/leave thread | Opt-in to notifications | P1 |
| Thread member list | See who's in thread | P2 |

### 4.10 Typing Indicators

| Feature | Description | Priority |
|---------|-------------|----------|
| Show typing | "[User] is typing..." | P0 |
| Multiple typing | "[User1], [User2] are typing..." | P0 |
| Many typing | "Several people are typing..." (4+) | P1 |
| Typing timeout | Clear after 10s of no keystrokes | P0 |

### 4.11 Read State

| Feature | Description | Priority |
|---------|-------------|----------|
| Unread indicator | Dot/badge on unread channels | P0 |
| Unread count | Number of unread messages | P0 |
| Mention badge | Separate count for mentions | P0 |
| Mark as read | Click channel or scroll to bottom | P0 |
| Mark all as read | Clear all unread in server | P1 |
| Jump to unread | Button to scroll to first unread | P1 |
| New message bar | "New messages since [time]" | P0 |

---

## 5. Direct Messages

### 5.1 DM Types

| Type | Description | Priority |
|------|-------------|----------|
| 1:1 DM | Private conversation between two users | P0 |
| Group DM | Up to 10 participants | P1 |

### 5.2 DM Features

| Feature | Description | Priority |
|---------|-------------|----------|
| Start DM | From profile or user list | P0 |
| DM list | Sidebar with recent DMs | P0 |
| DM search | Find DM by username | P1 |
| Close DM | Remove from list (doesn't delete) | P0 |
| DM history | Messages persist until deleted | P0 |
| Voice call | 1:1 voice in DM | P1 |
| Video call | 1:1 video in DM | P2 |
| Screen share | In DM calls | P2 |

### 5.3 Group DM

| Feature | Description | Priority |
|---------|-------------|----------|
| Create group | Select 2-9 other users | P1 |
| Group name | Optional custom name | P1 |
| Group icon | Optional custom icon | P2 |
| Add members | Up to 10 total | P1 |
| Remove members | Creator or self-remove | P1 |
| Leave group | Exit without deleting | P1 |
| Group call | Voice with all members | P2 |

### 5.4 Privacy & Blocking

| Feature | Description | Priority |
|---------|-------------|----------|
| Block user | Prevent DMs, hide messages | P0 |
| Unblock user | Restore communication | P0 |
| Blocked list | View and manage blocked users | P0 |
| DM privacy | Server members only, friends only, everyone | P1 |
| Message requests | Non-friends go to requests | P2 |

---

## 6. Voice & Video

### 6.1 Voice Channels

| Feature | Description | Priority |
|---------|-------------|----------|
| Join voice | Click channel to connect | P1 |
| Leave voice | Click disconnect or X | P1 |
| Voice indicator | Show who's speaking | P1 |
| User list | Show all users in channel | P1 |
| Mute self | Toggle microphone | P1 |
| Deafen self | Mute + disable audio | P1 |
| Push-to-talk | Keybind to transmit | P1 |
| Voice activity | Auto-detect speaking | P1 |
| Input device | Select microphone | P1 |
| Output device | Select speakers/headphones | P1 |
| Input volume | Adjust mic sensitivity | P1 |
| Output volume | Adjust listening volume | P1 |

### 6.2 Voice Quality

| Feature | Description | Priority |
|---------|-------------|----------|
| Bitrate selection | 8-384 kbps | P2 |
| Noise suppression | AI-based background noise removal | P2 |
| Echo cancellation | Prevent feedback | P1 |
| Auto gain control | Normalize volume levels | P2 |
| Connection quality | Show connection indicator | P1 |

### 6.3 Video

| Feature | Description | Priority |
|---------|-------------|----------|
| Enable video | Toggle camera | P2 |
| Camera selection | Choose camera device | P2 |
| Video preview | See self before enabling | P2 |
| Grid layout | Multiple video tiles | P2 |
| Focus mode | Enlarge active speaker | P2 |
| Picture-in-picture | Float video while browsing | P3 |

### 6.4 Screen Sharing

| Feature | Description | Priority |
|---------|-------------|----------|
| Share screen | Full screen capture | P2 |
| Share window | Single window capture | P2 |
| Share tab | Browser tab (Chromium) | P3 |
| Audio sharing | Include system audio | P2 |
| Viewer count | Show who's watching | P2 |
| Stop sharing | End screen share | P2 |

### 6.5 Voice Moderation

| Feature | Description | Priority |
|---------|-------------|----------|
| Server mute | Mute user for all | P1 |
| Server deafen | Deafen user for all | P1 |
| Move user | Move to different channel | P1 |
| Disconnect user | Kick from voice | P1 |
| Priority speaker | Reduce others' volume | P2 |
| Stage permissions | Request to speak, invite to speak | P3 |

---

## 7. Roles & Permissions

### 7.1 Role Management

| Feature | Description | Priority |
|---------|-------------|----------|
| Create role | Name, color, permissions | P0 |
| Edit role | Modify any property | P0 |
| Delete role | Remove role (unassigns from members) | P0 |
| Role order | Drag to set hierarchy | P0 |
| @everyone role | Default role for all members | P0 |
| Role color | Hex color picker | P0 |
| Role icon | Small image next to name | P2 |
| Hoisted role | Show separately in member list | P0 |
| Mentionable | Allow @role mentions | P0 |

### 7.2 Role Assignment

| Feature | Description | Priority |
|---------|-------------|----------|
| Assign role | Add role to member | P0 |
| Remove role | Remove role from member | P0 |
| Bulk assign | Add role to multiple members | P2 |
| Self-assignable | Members can add/remove | P2 |
| Reaction roles | Click reaction to get role | P3 |

### 7.3 Permissions List

**General Server Permissions:**

| Permission | Description |
|------------|-------------|
| View Channels | See channels (text and voice) |
| Manage Channels | Create, edit, delete channels |
| Manage Roles | Create, edit, assign roles below own |
| Manage Emoji | Add, remove, rename custom emoji |
| View Audit Log | See server audit log |
| Manage Webhooks | Create, edit, delete webhooks |
| Manage Server | Edit server name, icon, settings |

**Membership Permissions:**

| Permission | Description |
|------------|-------------|
| Create Invite | Generate invite links |
| Change Nickname | Change own nickname |
| Manage Nicknames | Change others' nicknames |
| Kick Members | Remove members from server |
| Ban Members | Permanently ban members |
| Timeout Members | Temporarily mute members |

**Text Permissions:**

| Permission | Description |
|------------|-------------|
| Send Messages | Send messages in text channels |
| Send Messages in Threads | Send in threads |
| Create Public Threads | Start public threads |
| Create Private Threads | Start private threads |
| Send TTS Messages | Use /tts command |
| Manage Messages | Delete others' messages, pin |
| Manage Threads | Archive, lock threads |
| Embed Links | Auto-embed link previews |
| Attach Files | Upload files |
| Read Message History | View past messages |
| Mention @everyone | Use @everyone, @here |
| Use External Emoji | Use emoji from other servers |
| Use External Stickers | Use stickers from other servers |
| Add Reactions | React to messages |
| Use Slash Commands | Use / commands |

**Voice Permissions:**

| Permission | Description |
|------------|-------------|
| Connect | Join voice channels |
| Speak | Transmit audio |
| Video | Enable camera |
| Use Voice Activity | Speak without push-to-talk |
| Priority Speaker | Louder than others |
| Mute Members | Server mute others |
| Deafen Members | Server deafen others |
| Move Members | Move users between channels |
| Use Soundboard | Play soundboard sounds |

**Advanced:**

| Permission | Description |
|------------|-------------|
| Administrator | All permissions, bypasses overrides |

### 7.4 Channel Overrides

| Feature | Description | Priority |
|---------|-------------|----------|
| Role override | Set allow/deny per role | P0 |
| Member override | Set allow/deny per user | P0 |
| Override UI | Toggle switches for each permission | P0 |
| Inherit | Neutral state (use role permission) | P0 |
| Allow | Explicitly grant | P0 |
| Deny | Explicitly revoke | P0 |

---

## 8. Moderation

### 8.1 Member Actions

| Feature | Description | Priority |
|---------|-------------|----------|
| Kick | Remove from server (can rejoin) | P0 |
| Ban | Remove and prevent rejoin | P0 |
| Unban | Allow banned user to rejoin | P0 |
| Timeout | Temporarily mute (1min - 28days) | P1 |
| Warn | Log warning (no automatic action) | P2 |
| Nickname change | Force change nickname | P1 |
| Role change | Add/remove roles | P0 |

### 8.2 Message Actions

| Feature | Description | Priority |
|---------|-------------|----------|
| Delete message | Remove single message | P0 |
| Bulk delete | Remove up to 100 messages | P1 |
| Purge by user | Delete all messages from user | P2 |
| Purge timeframe | Delete messages in time range | P2 |

### 8.3 Audit Log

| Feature | Description | Priority |
|---------|-------------|----------|
| View audit log | All moderation actions | P0 |
| Filter by action | Type filter | P1 |
| Filter by user | Who performed action | P1 |
| Filter by target | Who was affected | P1 |
| Time range | Date filter | P1 |
| Action details | Before/after values | P1 |
| Export log | Download as JSON/CSV | P2 |

**Logged Actions:**
- Member join/leave/kick/ban/unban/timeout
- Role create/update/delete/assign
- Channel create/update/delete
- Message delete/bulk delete
- Invite create/delete
- Webhook create/update/delete
- Server settings changes
- Emoji changes
- Permission changes

### 8.4 Auto-Moderation

| Feature | Description | Priority |
|---------|-------------|----------|
| Spam detection | Rapid message detection | P2 |
| Duplicate messages | Block repeated content | P2 |
| Mention spam | Limit mentions per message | P2 |
| Link blocking | Block/allow specific domains | P2 |
| Word filter | Block specific words/phrases | P2 |
| Regex filter | Custom pattern matching | P3 |
| Auto-actions | Warn, delete, timeout, kick, ban | P2 |
| Immune roles | Bypass auto-mod | P2 |

### 8.5 Verification

| Feature | Description | Priority |
|---------|-------------|----------|
| Verification level | None, Low, Medium, High, Highest | P1 |
| Email verification | Require verified email | P1 |
| Account age | Require 5/10 min account age | P1 |
| Server membership | Require 10 min membership | P1 |
| Phone verification | Require phone (Highest) | P3 |
| Membership screening | Rules acceptance before access | P2 |

---

## 9. Social Features

### 9.1 Friends

| Feature | Description | Priority |
|---------|-------------|----------|
| Send friend request | From profile or username | P2 |
| Accept/decline request | Pending requests list | P2 |
| Friend list | View all friends | P2 |
| Remove friend | Unfriend user | P2 |
| Friend status | See friends' online status | P2 |
| Friend activity | See what friends are doing | P3 |

### 9.2 User Notes

| Feature | Description | Priority |
|---------|-------------|----------|
| Add note | Private note on any user | P2 |
| Edit note | Update existing note | P2 |
| Note persistence | Synced across devices | P2 |
| Note visibility | Only visible to you | P2 |

### 9.3 Mutual Servers

| Feature | Description | Priority |
|---------|-------------|----------|
| View mutual servers | Servers you share with user | P1 |
| Mutual friends | Friends you share (future) | P3 |

### 9.4 Server Nicknames

| Feature | Description | Priority |
|---------|-------------|----------|
| Set nickname | Per-server display name | P0 |
| View username | Show original on hover | P0 |
| Nickname color | Use highest role color | P0 |
| Change own | If permitted | P0 |
| Reset nickname | Revert to username | P0 |

---

## 10. Notifications

### 10.1 Notification Types

| Type | Description | Priority |
|------|-------------|----------|
| In-app | Badge, sound, visual alert | P0 |
| Desktop | OS-level push notification | P0 |
| Mobile | Push notification (PWA/native) | P2 |
| Email | Digest or instant (configurable) | P2 |

### 10.2 Notification Settings

| Setting | Scope | Priority |
|---------|-------|----------|
| All messages | Every message notifies | P0 |
| Mentions only | @mentions and DMs only | P0 |
| Nothing | No notifications | P0 |
| Suppress @everyone | Ignore @everyone/@here | P0 |
| Suppress @roles | Ignore role mentions | P1 |
| Mobile push | Enable/disable | P2 |

### 10.3 Channel/Server Overrides

| Feature | Description | Priority |
|---------|-------------|----------|
| Per-channel | Override server setting | P0 |
| Mute channel | No notifications, still badge | P0 |
| Mute server | No notifications from server | P0 |
| Mute duration | 15min, 1h, 8h, 24h, forever | P1 |

### 10.4 Notification Sounds

| Feature | Description | Priority |
|---------|-------------|----------|
| Enable/disable | Toggle all sounds | P0 |
| Sound selection | Choose notification sound | P2 |
| Per-type sounds | Different sounds for DM vs mention | P3 |
| Volume | Separate notification volume | P2 |

### 10.5 Do Not Disturb

| Feature | Description | Priority |
|---------|-------------|----------|
| DND status | Suppress all notifications | P0 |
| Scheduled DND | Quiet hours | P2 |
| Allow exceptions | Specific users can notify | P3 |

---

## 11. Search

### 11.1 Message Search

| Feature | Description | Priority |
|---------|-------------|----------|
| Full-text search | Search message content | P1 |
| Search scope | Channel, server, all | P1 |
| Jump to result | Click to scroll to message | P1 |
| Context | Show messages around result | P1 |

### 11.2 Search Filters

| Filter | Syntax | Priority |
|--------|--------|----------|
| From user | `from:username` | P1 |
| In channel | `in:channel-name` | P1 |
| Has attachment | `has:file`, `has:image`, `has:video` | P1 |
| Has link | `has:link` | P1 |
| Has embed | `has:embed` | P2 |
| Before date | `before:2026-02-11` | P1 |
| After date | `after:2026-01-01` | P1 |
| During | `during:2026-02-11` | P2 |
| Mentions | `mentions:username` | P1 |
| Pinned | `is:pinned` | P2 |

### 11.3 Other Search

| Feature | Description | Priority |
|---------|-------------|----------|
| User search | Find members by name | P0 |
| Channel search | Find channels | P0 |
| Emoji search | Find emoji by name | P0 |
| Command search | Search slash commands | P2 |
| GIF search | Search Tenor/Giphy | P2 |

---

## 12. Integrations

### 12.1 Webhooks

| Feature | Description | Priority |
|---------|-------------|----------|
| Create webhook | Generate webhook URL | P1 |
| Webhook avatar | Custom avatar per hook | P1 |
| Webhook name | Custom name per hook | P1 |
| Edit webhook | Update settings | P1 |
| Delete webhook | Remove webhook | P1 |
| Webhook URL | POST endpoint for messages | P1 |
| Webhook payload | JSON with content, embeds | P1 |

### 12.2 Bot Accounts

| Feature | Description | Priority |
|---------|-------------|----------|
| Create bot | Bot user account | P2 |
| Bot token | API authentication | P2 |
| Bot permissions | Scoped access | P2 |
| Add bot to server | OAuth2 authorization flow | P2 |
| Bot badge | Visual indicator for bots | P2 |

### 12.3 Slash Commands

| Feature | Description | Priority |
|---------|-------------|----------|
| Register commands | Bot defines commands | P2 |
| Command autocomplete | Show available commands | P2 |
| Command options | Parameters with types | P2 |
| Command permissions | Restrict to roles/channels | P2 |

### 12.4 Application API

| Feature | Description | Priority |
|---------|-------------|----------|
| REST API | Full CRUD operations | P1 |
| WebSocket API | Real-time events | P1 |
| Rate limiting | Prevent abuse | P1 |
| API documentation | OpenAPI spec | P1 |

---

## 13. Accessibility

### 13.1 Visual

| Feature | Description | Priority |
|---------|-------------|----------|
| Dark/light theme | Color scheme toggle | P0 |
| High contrast | Accessibility theme | P2 |
| Font scaling | Adjust text size | P1 |
| Reduced motion | Disable animations | P1 |
| Color blind modes | Deuteranopia, Protanopia, Tritanopia | P3 |

### 13.2 Navigation

| Feature | Description | Priority |
|---------|-------------|----------|
| Keyboard navigation | Tab through interface | P1 |
| Keyboard shortcuts | Common actions | P1 |
| Focus indicators | Visible focus state | P1 |
| Skip links | Jump to main content | P2 |

### 13.3 Screen Readers

| Feature | Description | Priority |
|---------|-------------|----------|
| ARIA labels | Proper labeling | P1 |
| Live regions | Announce new messages | P2 |
| Alt text | Images, emoji descriptions | P1 |
| Semantic HTML | Proper heading structure | P1 |

---

## 14. Mobile Experience

### 14.1 PWA Features

| Feature | Description | Priority |
|---------|-------------|----------|
| Installable | Add to home screen | P1 |
| Offline support | Cached UI, queue messages | P2 |
| Push notifications | Native-like notifications | P2 |
| App icon | Home screen icon | P1 |
| Splash screen | Loading screen | P2 |

### 14.2 Mobile UI

| Feature | Description | Priority |
|---------|-------------|----------|
| Responsive design | Works on all screen sizes | P0 |
| Touch gestures | Swipe navigation | P1 |
| Pull to refresh | Refresh content | P1 |
| Bottom navigation | Mobile-friendly nav | P1 |
| Compact mode | Denser information display | P2 |

### 14.3 Mobile Voice

| Feature | Description | Priority |
|---------|-------------|----------|
| Background audio | Voice continues in background | P2 |
| Proximity sensor | Disable screen during call | P3 |
| Bluetooth | Headset support | P2 |
| CallKit/ConnectionService | Native call UI | P3 |

---

## Feature Priority Summary

| Priority | Count | Description |
|----------|-------|-------------|
| P0 | ~80 | Must have for MVP |
| P1 | ~70 | Important for v1.0 |
| P2 | ~60 | Nice to have |
| P3 | ~20 | Future consideration |

---

*End of Feature Specification*
