---
name: Channel Categories & Organization
description: Hierarchical channel organization with collapsible categories for server structure
type: feature
priority: P0
discord_equivalent: Channel Categories
---

# Channel Categories & Organization

## Overview

**Discord equivalent**: Channel Categories
**User value proposition**: Essential server organization for communities with >10 channels, providing logical grouping and navigation
**Technical complexity**: P0 (Critical infrastructure feature)

## Problem Statement

Currently Hearth displays all channels in a flat list without organization. For servers with 15+ channels, this creates:
- **Navigation chaos**: Users can't find channels quickly
- **Poor UX scaling**: Becomes unusable for communities >30 channels
- **No logical grouping**: Related channels scattered in the list
- **Admin burden**: Server admins can't organize their community structure

Discord considers categories so fundamental they're part of core channel management, not a "feature". Without categories, Hearth servers feel primitive compared to Discord.

## Solution

Implement Discord-compatible channel categories with hierarchical organization:

### Core Features
- **Create categories**: Named sections to group related channels
- **Channel nesting**: Drag channels into categories
- **Visual hierarchy**: Clear parent-child relationship display
- **Collapsible UI**: Expand/collapse categories to reduce visual clutter
- **Category permissions**: Inherit permissions to child channels
- **Reordering**: Drag-and-drop category and channel organization

### User Experience
- **Category creation**: Right-click sidebar → "Create Category"
- **Channel organization**: Drag channels between categories
- **Collapsible sections**: Click arrow to expand/collapse
- **Permission inheritance**: Category settings apply to all child channels
- **Visual distinction**: Clear visual hierarchy with indentation

## Technical Implementation

### Backend Changes
```go
// Extend existing Channel model
type Channel struct {
    ID          uuid.UUID  `json:"id"`
    Name        string     `json:"name"`
    Type        string     `json:"type"` // "GUILD_TEXT", "GUILD_VOICE", "GUILD_CATEGORY"
    ServerID    uuid.UUID  `json:"server_id"`
    ParentID    *uuid.UUID `json:"parent_id,omitempty"` // Category ID for channels
    Position    int        `json:"position"`
    // ... existing fields
}
```

### Database Schema Changes
- **Add to channels table**: `parent_id` (nullable, references categories)
- **Add to channels table**: `position` (int, for ordering)
- **New channel type**: `"GUILD_CATEGORY"`
- **Index on**: `(server_id, parent_id, position)` for efficient queries

### API Endpoints
- `POST /api/v1/servers/{id}/channels` - Create category (type: GUILD_CATEGORY)
- `PATCH /api/v1/channels/{id}` - Update parent_id (move to category)
- `PATCH /api/v1/servers/{id}/channels` - Bulk reorder channels
- `GET /api/v1/servers/{id}/channels` - Returns hierarchical structure

### Frontend Changes
```typescript
// Channel tree structure
interface ChannelTreeNode {
  id: string;
  name: string;
  type: 'GUILD_CATEGORY' | 'GUILD_TEXT' | 'GUILD_VOICE';
  position: number;
  children?: ChannelTreeNode[]; // For categories
  parentId?: string;
  collapsed?: boolean; // UI state
}
```

## Dependencies

### Prerequisites
1. **Channel system** ✅ (already implemented)
2. **Server management** ✅ (already implemented)
3. **Permission system** ✅ (already implemented)

### New Dependencies
- **Drag-and-drop UI library** (frontend component library)
- **Hierarchical permission logic** (category → child inheritance)

## Success Metrics

- **Adoption rate**: 85% of servers with 10+ channels create at least one category
- **Organization improvement**: Average channels per category = 4-8
- **User satisfaction**: <5% support tickets about "can't find channels"
- **Scaling validation**: Servers with 50+ channels maintain good UX

## Implementation Phases

### Phase 1: Core Infrastructure (2 weeks)
- Database schema changes
- Category creation/deletion
- Parent-child channel relationships
- Basic API endpoints

### Phase 2: UI & Reordering (2 weeks)
- Hierarchical channel display
- Drag-and-drop channel organization
- Category collapse/expand
- Visual hierarchy styling

### Phase 3: Permission Inheritance (1 week)
- Category permission propagation
- Override logic for individual channels
- Admin UX for category-level permissions

## Risk Assessment

**Medium Risk**: Requires database migration and significant UI changes

**Potential Issues**:
- **Data migration**: Existing channels need position assignment
- **Performance**: Hierarchical queries at scale
- **UI complexity**: Drag-and-drop state management

**Mitigation**:
- **Migration script**: Assign default positions to existing channels
- **Database indexing**: Optimize queries with proper indexes
- **Progressive enhancement**: Fall back to flat list if drag-and-drop fails

## Example Server Structure
```
📁 General
  #general
  #announcements
  #rules

📁 Gaming
  #gaming-general
  🔊 Gaming Voice 1
  🔊 Gaming Voice 2

📁 Development
  #dev-discussion
  #bug-reports
  #feature-requests
```

This transforms a chaotic 9-channel list into an organized, navigable structure that scales to 50+ channels.