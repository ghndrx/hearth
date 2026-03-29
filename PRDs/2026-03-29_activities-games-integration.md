# Activities & Games Integration PRD

## Feature Overview
Rich presence system with gaming status, Discord Activities integration, and social gaming features within Hearth.

## Discord Equivalent
Discord Rich Presence, Discord Activities, Gaming Status - Deep integration with games and social activities.

## User Value Proposition
- **Social Gaming**: See what games friends are playing, join them directly
- **Community Building**: Organize gaming sessions, tournaments, and events around games
- **Rich Context**: Show detailed game status (what level, which character, etc.)
- **Activity Discovery**: Find new games and activities through friends' activity

## Technical Complexity Estimate
**P1** - Medium-high complexity

## Implementation Sketch
### Rich Presence System
- Game detection through running processes
- Rich presence API for games to provide detailed status
- Custom activity statuses (listening to music, watching streams, etc.)
- Activity history and statistics

### Discord Activities Integration
- Embed web-based games and activities directly in channels
- Party system for multiplayer activities
- Activity invitations and joining
- Screen sharing integration for shared experiences

### Gaming Features
- "Join Game" buttons when friends are playing multiplayer games
- Game library integration (Steam, Epic, etc.)
- Gaming status privacy controls
- Game-based server discovery

### Backend Changes
```go
type Activity struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Type        int       `json:"type"` // 0=game, 1=streaming, 2=listening, 3=watching, 5=competing
    URL         string    `json:"url,omitempty"`
    CreatedAt   int64     `json:"created_at"`
    Timestamps  *struct {
        Start int64 `json:"start,omitempty"`
        End   int64 `json:"end,omitempty"`
    } `json:"timestamps,omitempty"`
    ApplicationID string   `json:"application_id,omitempty"`
    Details       string   `json:"details,omitempty"`
    State         string   `json:"state,omitempty"`
    Party         *struct {
        ID   string `json:"id,omitempty"`
        Size []int  `json:"size,omitempty"` // [current, max]
    } `json:"party,omitempty"`
    Assets *struct {
        LargeImage string `json:"large_image,omitempty"`
        LargeText  string `json:"large_text,omitempty"`
        SmallImage string `json:"small_image,omitempty"`
        SmallText  string `json:"small_text,omitempty"`
    } `json:"assets,omitempty"`
    Secrets *struct {
        Join     string `json:"join,omitempty"`
        Spectate string `json:"spectate,omitempty"`
        Match    string `json:"match,omitempty"`
    } `json:"secrets,omitempty"`
}
```

### Frontend Changes
- User profile showing current activity
- Activity status in member lists and presence indicators
- Game invitation system
- Activity-based server recommendations
- Embedded activity launcher for supported games

### Database Schema
```sql
CREATE TABLE user_activities (
    user_id UUID NOT NULL,
    activity_data JSONB NOT NULL,
    started_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id)
);

CREATE TABLE activity_invites (
    id UUID PRIMARY KEY,
    from_user_id UUID NOT NULL,
    to_user_id UUID NOT NULL,
    activity_data JSONB NOT NULL,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE gaming_sessions (
    id UUID PRIMARY KEY,
    server_id UUID,
    channel_id UUID,
    game_name VARCHAR(255) NOT NULL,
    participants JSONB, -- array of user_ids
    started_at TIMESTAMP DEFAULT NOW(),
    ended_at TIMESTAMP
);
```

## Dependencies
- Rich presence API development
- Game detection system (desktop app integration)
- OAuth integrations with gaming platforms
- WebRTC for activity screen sharing

## Success Metrics
- Percentage of users with active game status
- Game invitation acceptance rate
- Activity-based friend connections
- Time spent in gaming-related channels