# Feature Research: Message Threading and Replies

**Research Date:** 2026-02-13  
**Author:** Automated Research (Clawd)  
**Focus Area:** Threading models, reply UX, and data architecture

---

## Executive Summary

Message threading is one of the most impactful UX decisions in chat applications. The approach chosen affects everything from information organization to user cognitive load. This document analyzes how major platforms implement threading, identifies best practices, and provides recommendations for Hearth.

**Key Finding:** There's no universal best approach. Discord's inline replies, Slack's side-panel threads, and Zulip's topic-based model each excel in different contexts. Hearth should implement **hybrid threading** that supports multiple interaction patterns.

---

## Platform Comparison

### 1. Discord: Inline Replies + Optional Threads

**Model:** Flat timeline with optional inline replies and dedicated thread channels

**How it works:**
- Primary: Messages flow chronologically in channels
- Replies: Reference a parent message with visual indicator (shows snippet of original)
- Threads: Explicit "Create Thread" spawns a sub-channel attached to a message
- Threads appear in sidebar and can have their own activity

**Data Model:**
```
Message {
  id
  channel_id
  content
  message_reference: {     // For replies
    message_id
    channel_id
    guild_id
  }
}

Thread (Channel) {
  id
  parent_id              // Original channel
  owner_id               // Thread creator
  message_count
  auto_archive_duration  // 60min, 24h, 3d, 7d
}
```

**UX Strengths:**
- Low friction for quick replies (just hit reply button)
- Threads keep detailed discussions out of main flow
- Threading is opt-in, not forced
- Visual continuity — replies show parent context inline

**UX Weaknesses:**
- Multiple rapid-fire replies clutter the timeline
- Hard to follow multiple simultaneous discussions
- Thread discovery can be poor (users miss thread updates)
- No topic labeling within threads

**Best For:** Casual communities, gaming groups, real-time coordination

---

### 2. Slack: Side-Panel Threads

**Model:** Channels with mandatory reply-to-thread for long discussions

**How it works:**
- Messages post to channel timeline
- Clicking reply opens side panel for threaded conversation
- "Also send to #channel" option broadcasts thread reply to main timeline
- Thread indicator shows reply count on parent message

**Data Model:**
```
Message {
  ts               // Timestamp as ID
  channel
  text
  thread_ts        // If part of thread, references parent ts
  reply_count      // Denormalized for performance
  reply_users_count
}
```

**UX Strengths:**
- Clear separation of main timeline vs. detailed discussion
- Side-by-side view allows following both
- Thread replies don't pollute main channel
- Good for asynchronous work (catch up on specific threads)

**UX Weaknesses:**
- Extra click to read/reply to threads
- Can't see full context without opening panel
- Multiple open threads = cognitive overhead
- Users forget to check threads (notification tuning required)

**Best For:** Workplace communication, project coordination, async teams

---

### 3. Zulip: Topic-Based Threading (Unique Approach)

**Model:** Every message belongs to a Stream (channel) AND a Topic (thread)

**How it works:**
- Streams = broad categories (like channels)
- Topics = required subject line for every message
- Messages with same topic are threaded together
- Topics auto-suggest based on recent activity
- Easy to rename/move topics retroactively

**Data Model:**
```
Message {
  id
  stream_id        // Channel equivalent
  topic            // Required string, the "subject"
  content
  sender_id
}
```

**UX Strengths:**
- Forces organization from the start
- Easy to catch up on specific topics while skipping others
- Asynchronous-first design (topics float to top when active)
- Topics are searchable and form natural knowledge base
- No messages "fall through the cracks"

**UX Weaknesses:**
- Requires discipline to use topics correctly
- Extra friction (must specify topic for every message)
- Unfamiliar UX for users coming from Slack/Discord
- Mobile UX more complex due to topic selection

**Best For:** Open source projects, academic teams, knowledge workers, distributed async teams

---

### 4. Microsoft Teams: Hybrid Posts + Quote Replies

**Model:** Channels use threaded posts; 1:1 chats use quote replies

**How it works:**
- **Channels:** "Start a post" creates threaded discussion (like email thread)
- Posts collapse to show only first message + reply count
- **Chats:** Linear timeline with quote-reply (references original message)
- No side-panel threads in 1:1 chat

**UX Strengths:**
- Channel threading prevents chaos in busy channels
- Good for formal discussions and decisions
- Rich text formatting in posts
- Integration with Office documents

**UX Weaknesses:**
- Confusing dual model (channels vs chats work differently)
- Chat threading is half-baked compared to Slack
- Heavy, slow interface
- Notification management is complex

**Best For:** Enterprise environments already using Microsoft 365

---

### 5. Matrix (Element): Relations-Based Threading

**Model:** Protocol-level message relations with multiple threading implementations

**How it works:**
- MSC2836/MSC3440 define threading via `m.relates_to` field
- Messages can reference parents with different relation types:
  - `m.in_reply_to` — inline reply
  - `m.thread` — thread participation
- Clients can render these relations differently

**Data Model (Protocol):**
```
Event {
  event_id
  room_id
  content: {
    body: "message text"
    "m.relates_to": {
      "rel_type": "m.thread",
      "event_id": "$parent_event_id"
    }
  }
}
```

**UX Strengths:**
- Protocol flexibility allows different client UX experiments
- Federation-aware threading
- Can retrofit threading to existing rooms
- Server aggregates thread counts efficiently

**UX Weaknesses:**
- Implementation varies across clients
- Some clients handle threads poorly
- E2EE + threading has edge cases
- Complexity in syncing thread state across federation

**Best For:** Privacy-focused communities, federated deployments

---

## Open Source Implementations

### Mattermost
- Slack-like side-panel threads
- `Posts` table with `root_id` for threading
- Good mobile thread support
- Source: github.com/mattermost/mattermost-server

### Rocket.Chat
- Configurable threading (can enable/disable)
- Messages have `tmid` (thread message ID) field
- Thread list panel similar to Slack
- Source: github.com/RocketChat/Rocket.Chat

### Zulip
- Fully open source topic-based implementation
- Python/Django backend, React frontend
- Excellent reference for topic-based architecture
- Source: github.com/zulip/zulip

---

## UX Best Practices

### 1. Keep Replies Minimized but Accessible
- Show collapsed reply indicator on parent message
- One-click expansion to see replies
- Don't let replies dominate main timeline

### 2. Preserve Context
- Always show parent message snippet when viewing reply
- Allow jumping to original context with one click
- Show thread participant avatars on collapsed view

### 3. Smart Notifications
- Thread replies: notify only thread participants by default
- @mentions in threads: notify mentioned users
- Allow "follow thread" without participating
- Digest mode for high-traffic threads

### 4. Easy Thread Creation
- Reply button should be prominent and immediate
- Optional "create thread" for longer discussions
- Auto-suggest threading when reply chains get long (>3 messages)

### 5. Thread Discovery
- Show "N new messages in N threads" on channel return
- Thread activity panel/sidebar
- Mark-all-as-read per thread

### 6. Host/Moderator Control
- Allow mods to move messages into threads
- Lock threads after resolution
- Archive old threads automatically
- Rename/merge threads

---

## Recommended Approach for Hearth

### Hybrid Threading Model

Given Hearth's Discord-alternative positioning, recommend implementing **Discord-style with enhancements**:

#### Core Features (P0-P1)

1. **Inline Replies** (P0)
   - Click reply → shows parent snippet inline
   - `reply_to_id` on message (already in data model ✓)
   - Visual line connecting reply to parent
   - Click parent snippet to jump to original

2. **Thread Channels** (P1)
   - "Create Thread" on any message
   - Thread becomes sub-channel with own permission inheritance
   - Auto-archive after configurable inactivity
   - `thread_id` field on messages (already in data model ✓)

3. **Thread Discovery Panel** (P1)
   - Sidebar section showing active threads
   - Badge with unread count
   - "Following" vs "All Threads" filter

#### Enhanced Features (P2)

4. **Optional Topic Labels**
   - Allow tagging threads with topic strings
   - Searchable topic index
   - Borrows from Zulip without forcing the paradigm

5. **Thread Summary**
   - AI-generated summary for catching up on long threads
   - "Last 24h: 47 messages, 3 decisions made"

6. **Smart Thread Suggestions**
   - After 4+ rapid replies to same message, suggest "Create thread?"
   - Reduces main channel clutter organically

### Data Model Additions

```sql
-- Thread metadata (extends channel)
ALTER TABLE channels ADD COLUMN IF NOT EXISTS
  thread_metadata JSONB DEFAULT NULL;

-- Example thread_metadata:
-- {
--   "parent_message_id": "uuid",
--   "auto_archive_minutes": 4320,
--   "archived_at": null,
--   "member_count": 5,
--   "message_count": 23,
--   "topic_tags": ["design", "urgent"]
-- }

-- Efficient thread listing
CREATE INDEX idx_channels_thread_parent 
  ON channels ((thread_metadata->>'parent_message_id')) 
  WHERE type = 'thread';
```

### API Endpoints

```
POST /channels/{channel_id}/messages/{message_id}/threads
  → Creates thread from message

GET /channels/{channel_id}/threads
  → Lists active threads in channel

GET /users/@me/threads
  → Lists threads user is participating in

POST /threads/{thread_id}/archive
POST /threads/{thread_id}/unarchive
```

---

## Metrics to Track

1. **Thread Adoption Rate** — % of channels using threads
2. **Reply vs Thread Usage** — Ratio of inline replies to threads
3. **Thread Resolution Time** — How long threads stay active
4. **Thread Discovery** — Do users find and read threads?
5. **Message Organization** — Reduction in main channel message volume after thread adoption

---

## References

- [Stream.io: Exploring Livestream Chat UX - Threads and Replies](https://getstream.io/blog/exploring-livestream-chat-ux-threads-and-replies/)
- [Stream.io: Chat UX Best Practices](https://getstream.io/blog/chat-ux/)
- [Zulip: Why Zulip?](https://zulip.com/why-zulip/)
- [Matrix MSC3440: Threading via Relations](https://github.com/matrix-org/matrix-spec-proposals/blob/gsouquet/threading-via-relations/proposals/3440-threading-via-relations.md)
- [UX Stack Exchange: Threaded Messaging Simplicity](https://ux.stackexchange.com/questions/29208/threaded-messaging-simplicity)
- Discord Developer Documentation: Message Reference
- Slack API Documentation: Conversations and Threads

---

## Appendix: Threading Comparison Matrix

| Feature | Discord | Slack | Zulip | Teams | Matrix |
|---------|---------|-------|-------|-------|--------|
| Inline replies | ✓ | ✓ | ✓ | ✓ (chat) | ✓ |
| Side-panel threads | ✗ | ✓ | ✗ | ✗ | varies |
| Dedicated thread channels | ✓ | ✗ | ✗ | ✗ | ✓ |
| Required topic/subject | ✗ | ✗ | ✓ | ✗ | ✗ |
| Thread auto-archive | ✓ | ✗ | ✗ | ✗ | ✗ |
| Thread notifications | limited | ✓ | ✓ | limited | varies |
| Move messages to thread | ✗ | ✗ | ✓ | ✗ | ✗ |
| Thread search | ✓ | ✓ | ✓ | ✓ | ✓ |
| Mobile thread UX | good | good | fair | poor | varies |

---

*Next research topic suggestion: Rich Embeds and Link Previews (Area #2)*
