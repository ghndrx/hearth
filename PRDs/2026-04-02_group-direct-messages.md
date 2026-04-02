---
name: Group Direct Messages
description: Multi-user direct message conversations for small friend groups (3-50 people)
type: feature
priority: P0
discord_equivalent: Group DMs
---

# Group Direct Messages

## Overview

**Discord equivalent**: Group DMs
**User value proposition**: Enable small group conversations outside of server structure for casual friend coordination and intimate group chats
**Technical complexity**: P0 (Critical blocking feature)

## Problem Statement

Currently Hearth only supports 1-on-1 direct messages. Users who want to coordinate with friend groups (3-8 people typically) are forced to either:
- Create a full server (overkill for casual conversation)
- Use multiple 1-on-1 DMs (fragmented, inefficient)
- Use external platforms (reduces Hearth adoption)

This is a **fundamental Discord feature** that 60-80% of users expect for small group coordination.

## Solution

Implement Discord-compatible group direct messages with the following capabilities:

### Core Features
- **Create group DMs**: Start group conversation by selecting 2+ friends
- **Named groups**: Custom group names with optional icons/photos
- **Member management**: Add/remove members (with permissions)
- **Group ownership**: Transfer ownership, manage settings
- **Message history**: Full persistence and search
- **Notifications**: Per-group notification settings

### User Experience
- **Creation flow**: Select friends → optional name/icon → create
- **Group list**: Dedicated section in DM sidebar showing active groups
- **Member limit**: 50 people maximum (Discord standard)
- **Permissions**: Owner can add/remove, members can leave

## Technical Implementation

### Backend Changes
```go
// New models
type GroupDM struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name,omitempty"`
    IconURL     string    `json:"icon_url,omitempty"`
    OwnerID     uuid.UUID `json:"owner_id"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type GroupDMMember struct {
    GroupDMID   uuid.UUID `json:"group_dm_id"`
    UserID      uuid.UUID `json:"user_id"`
    JoinedAt    time.Time `json:"joined_at"`
}
```

### Database Schema
- `group_dms` table (id, name, icon_url, owner_id, created_at, updated_at)
- `group_dm_members` table (group_dm_id, user_id, joined_at)
- Extend `channels` table with `type: "GROUP_DM"`

### API Endpoints
- `POST /api/v1/users/@me/channels` - Create group DM
- `GET /api/v1/channels/{id}/recipients` - Get group members
- `PUT /api/v1/channels/{id}/recipients/{user_id}` - Add member
- `DELETE /api/v1/channels/{id}/recipients/{user_id}` - Remove member
- `PATCH /api/v1/channels/{id}` - Update name/icon

## Dependencies

### Prerequisites
1. **DM system** ✅ (already implemented)
2. **Friend system** ✅ (already implemented)
3. **Channel message system** ✅ (already implemented)

### New Dependencies
- **File upload for group icons** (extend existing attachment system)
- **Notification system enhancement** (group-specific settings)

## Success Metrics

- **Adoption rate**: 40% of users create at least one group DM within 30 days
- **Engagement**: Average 15+ messages per active group DM per week
- **Retention**: Groups with 3+ active members have 80% 7-day retention
- **User feedback**: <2% support tickets about "missing group chat"

## Implementation Phases

### Phase 1: Core Functionality (2 weeks)
- Basic group creation and messaging
- Member management (add/remove)
- Group persistence and message history

### Phase 2: Enhanced UX (1 week)
- Custom names and icons
- Improved group management UI
- Notification settings

### Phase 3: Polish (1 week)
- Member permissions (kick vs leave)
- Group ownership transfer
- Advanced group settings

## Risk Assessment

**Low Risk**: Builds on existing DM and messaging infrastructure

**Potential Issues**:
- **Performance**: Groups with 50 members and high message volume
- **Privacy**: Clear member visibility and permissions
- **Spam**: Prevent mass group creation abuse

**Mitigation**:
- Rate limiting on group creation (5 per hour per user)
- Message rate limiting within groups
- Clear privacy controls and member consent