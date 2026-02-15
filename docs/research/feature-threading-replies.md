# FEAT-001: Message Threading Enhancements

## Status: Implemented
**Priority:** P2 (Medium)
**Date Completed:** 2025-02-16

## Overview

This feature enhances message threading in Hearth with:
1. Enhanced thread sidebar view with participant tracking
2. Thread notification preferences in user settings
3. Thread participant indicators

## Implementation Details

### 1. Thread Sidebar View (`ThreadSidebar.svelte`)

An enhanced sidebar component that displays:
- Thread title and reply count
- Thread participant avatars (stacked display)
- Per-thread notification controls (all/mentions/none)
- Parent message context
- Thread replies with full message rendering
- Reply input with keyboard shortcuts

**Key Features:**
- Notification dropdown with three levels: All Messages, Only @mentions, Muted
- Real-time participant tracking from thread messages
- Accessible keyboard navigation (Enter to send, Escape to close)
- Responsive design with proper overflow handling

### 2. Thread Notification Preferences

Added to `NotificationSettings.svelte` and `settings.ts`:

**New Settings:**
- `threadNotifications`: `'all' | 'mentions' | 'none'` - Default notification level for threads
- `threadAutoFollow`: `boolean` - Auto-follow threads when mentioned
- `threadFollowOnReply`: `boolean` - Auto-follow threads when you reply

**UI Section:**
- Radio button group for notification level selection
- Toggle switches for auto-follow behaviors
- Descriptive text for each option

### 3. Thread Participant Indicators

**`ThreadParticipants.svelte`:**
- Stacked avatar display with configurable max count
- Overflow indicator (+N) for many participants
- Optional name display
- Three size variants: xs, sm, md

**`ThreadReplyIndicator.svelte`:**
- Inline indicator shown under messages with thread replies
- Shows participant avatars, reply count, and last reply time
- Click to open thread sidebar
- Hover animation for discoverability

## Files Created/Modified

### New Files:
- `hearth/frontend/src/lib/components/ThreadSidebar.svelte` - Enhanced thread sidebar
- `hearth/frontend/src/lib/components/ThreadParticipants.svelte` - Participant avatar display
- `hearth/frontend/src/lib/components/ThreadReplyIndicator.svelte` - Inline thread indicator
- `hearth/docs/research/feature-threading-replies.md` - This document

### Modified Files:
- `hearth/frontend/src/lib/stores/settings.ts` - Added thread notification types and defaults
- `hearth/frontend/src/lib/stores/thread.ts` - Added ThreadParticipant type and derived store
- `hearth/frontend/src/lib/components/NotificationSettings.svelte` - Added Threads section
- `hearth/frontend/src/lib/components/index.ts` - Exported new components
- `hearth/frontend/src/routes/channels/+layout.svelte` - Updated to use ThreadSidebar

## Data Models

### ThreadNotificationLevel
```typescript
type ThreadNotificationLevel = 'all' | 'mentions' | 'none';
```

### ThreadParticipant
```typescript
interface ThreadParticipant {
  id: string;
  username: string;
  display_name?: string | null;
  avatar?: string | null;
}
```

### NotificationSettings (extended)
```typescript
interface NotificationSettings {
  // ... existing fields ...
  threadNotifications: ThreadNotificationLevel;
  threadAutoFollow: boolean;
  threadFollowOnReply: boolean;
}
```

## Backend Requirements (Future)

The frontend implementation is complete, but the following backend work may be needed:

1. **Per-thread notification settings API**
   - `PUT /threads/{thread_id}/notifications` - Set notification level for a specific thread
   - `GET /threads/{thread_id}/notifications` - Get current notification setting

2. **Thread participant tracking**
   - Backend should track and return `participants` array in thread responses
   - Include participant avatars in thread message responses

3. **Thread follow/unfollow**
   - `PUT /threads/{thread_id}/follow` - Follow a thread
   - `DELETE /threads/{thread_id}/follow` - Unfollow a thread
   - Automatic follow on reply (if enabled in settings)

## Testing Checklist

- [x] Thread sidebar opens when clicking "Reply in thread" on a message
- [x] Participant avatars display correctly (stacked with overflow)
- [x] Notification dropdown shows all three options
- [x] Thread notification settings persist in localStorage
- [x] Settings page shows new Threads section
- [x] Keyboard navigation works (Enter, Escape, Shift+Enter)
- [x] svelte-check passes with 0 errors

## Accessibility

- ARIA labels on all interactive elements
- Keyboard navigation support
- Screen reader announcements for thread state
- Focus management when opening/closing thread
- Role="complementary" on sidebar for landmark navigation
