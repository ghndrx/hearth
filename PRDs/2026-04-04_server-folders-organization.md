---
name: Server Folders & Organization System
description: Discord-style server folders for organizing servers into collapsible groups
type: competitive
---

# Server Folders & Organization System

## Discord Equivalent
Server Folders - Organize servers into named, collapsible folders for better server management

## User Value Proposition
Users with many servers need organization tools. Without folders, server lists become unwieldy and hard to navigate. Folders are a core Discord organizational feature that users rely on daily.

**Key Benefits:**
- Organize servers by category (work, gaming, communities, friends)
- Collapse/expand folders to reduce visual clutter
- Drag-and-drop server management
- Custom folder colors and names
- Improved server discovery and navigation

## Technical Complexity Estimate
**P0 - High Priority** (6-8 weeks)

**Complexity Factors:**
- Database schema for folder-server relationships
- Real-time folder sync across devices
- Drag-and-drop reordering UI
- Folder collapse/expand state persistence
- Migration for existing server lists

## Implementation Sketch

### Backend Models
```go
type ServerFolder struct {
    ID        uuid.UUID `json:"id" db:"id"`
    UserID    uuid.UUID `json:"user_id" db:"user_id"`
    Name      string    `json:"name" db:"name"`
    Color     *string   `json:"color,omitempty" db:"color"`
    Position  int       `json:"position" db:"position"`
    Collapsed bool      `json:"collapsed" db:"collapsed"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type ServerFolderMembership struct {
    FolderID  uuid.UUID `json:"folder_id" db:"folder_id"`
    ServerID  uuid.UUID `json:"server_id" db:"server_id"`
    Position  int       `json:"position" db:"position"`
}
```

### API Endpoints
- `POST /api/v1/folders` - Create folder
- `PUT /api/v1/folders/{id}` - Update folder (name, color, collapsed)
- `DELETE /api/v1/folders/{id}` - Delete folder (moves servers to root)
- `PUT /api/v1/folders/{id}/servers` - Bulk reorder servers in folder
- `POST /api/v1/servers/{id}/folder` - Move server to folder

### Frontend Components
- `ServerFolderView.svelte` - Collapsible folder with servers
- `FolderCreationModal.svelte` - Create/edit folder dialog
- `ServerDragDrop.svelte` - Drag-and-drop server management
- Enhanced `ServerList.svelte` with folder support

## Dependencies
- Enhanced server sidebar with reorder capabilities
- User settings sync for folder states
- WebSocket events for real-time folder updates
- Mobile app folder support (when native apps ship)

## Success Metrics
- Folder adoption rate (% of users with 3+ servers who create folders)
- Server navigation efficiency (reduced time to find servers)
- User retention improvement for multi-server users