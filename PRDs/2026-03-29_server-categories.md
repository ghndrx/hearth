# Server Categories PRD

## Feature Overview
Organizational system for grouping channels into collapsible categories with inherited permissions and visual hierarchy.

## Discord Equivalent
Discord Channel Categories - Collapsible channel groups that organize server layout and enable bulk permission management.

## User Value Proposition
- **Server Organization**: Group related channels together (e.g., "General", "Gaming", "Projects")
- **Visual Clarity**: Reduce clutter in large servers with many channels
- **Permission Management**: Set category-level permissions that cascade to child channels
- **User Experience**: Collapse unused categories to focus on relevant content

## Technical Complexity Estimate
**P2** - Medium complexity

## Implementation Sketch
### Category System
- Categories as special channel type with display-only properties
- Drag-and-drop channel organization into categories
- Category collapse/expand state per user
- Inherited permissions from category to channels
- Category-level muting and notification controls

### Backend Changes
```go
type Category struct {
    ID          string             `json:"id"`
    Name        string             `json:"name"`
    ServerID    string             `json:"server_id"`
    Position    int                `json:"position"`
    Permissions []PermissionOverwrite `json:"permission_overwrites"`
    CreatedAt   time.Time          `json:"created_at"`
}

type Channel struct {
    // ... existing fields ...
    CategoryID  *string `json:"category_id,omitempty"`
    Position    int     `json:"position"` // Position within category
}
```

### Permission Inheritance
- Category permissions apply to all child channels by default
- Channel-specific overrides take precedence
- Permission calculation: Server → Category → Channel → User/Role
- Bulk permission updates through category management

### Frontend Changes
- Category headers in channel list with collapse/expand icons
- Drag-and-drop interface for channel organization
- Category creation and management modal
- Visual indentation for channels within categories
- Category-level context menus for bulk operations

### Database Schema
```sql
CREATE TABLE categories (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    server_id UUID NOT NULL REFERENCES servers(id),
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE channels ADD COLUMN category_id UUID REFERENCES categories(id);
ALTER TABLE channels ADD COLUMN position INTEGER NOT NULL DEFAULT 0;

CREATE TABLE category_permissions (
    id UUID PRIMARY KEY,
    category_id UUID NOT NULL REFERENCES categories(id),
    role_id UUID REFERENCES roles(id),
    user_id UUID REFERENCES users(id),
    permissions BIGINT NOT NULL,
    allow BIGINT NOT NULL,
    deny BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT category_permissions_target_check CHECK (
        (role_id IS NOT NULL AND user_id IS NULL) OR
        (role_id IS NULL AND user_id IS NOT NULL)
    )
);

CREATE TABLE user_category_states (
    user_id UUID NOT NULL REFERENCES users(id),
    category_id UUID NOT NULL REFERENCES categories(id),
    collapsed BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (user_id, category_id)
);

-- Indexes for performance
CREATE INDEX idx_channels_category_position ON channels(category_id, position);
CREATE INDEX idx_categories_server_position ON categories(server_id, position);
```

### Channel Ordering
- Categories ordered by position
- Channels within categories ordered by position
- Channels without categories shown at top or bottom (configurable)
- Drag-and-drop reordering updates positions

## Dependencies
- Permissions system must support inheritance
- Channel management UI needs restructuring
- Real-time updates for category state changes

## Success Metrics
- Percentage of servers using categories
- Average number of categories per server
- Reduction in uncategorized channels over time
- User engagement with category collapse/expand features