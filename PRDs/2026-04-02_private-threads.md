---
name: Private Threads
description: Thread visibility control for sensitive conversations within channel context
type: Feature PRD
priority: P0
---

# Private Threads

## Discord Equivalent
Direct 1:1 match with Discord's Private Threads:
- Threads visible only to invited participants and moderators
- Thread creator can invite specific users
- Maintains channel context without broad visibility
- Essential for sensitive discussions in public channels

## User Value Proposition
**Critical for enterprise and community management** - Private threads enable:
- **Sensitive Discussions**: HR, disciplinary, or confidential conversations in context
- **Small Group Collaboration**: Project planning without cluttering main channel
- **Moderation**: Private discussion about users or incidents
- **Customer Support**: Private assistance without exposing personal details
- **Leadership Planning**: Strategy discussions accessible only to leadership team

## Technical Complexity: P0 (High Impact, Medium Complexity)

### Implementation Sketch
```go
// Enhanced Thread Model
type Thread struct {
    ID               string    `json:"id" db:"id"`
    ChannelID        string    `json:"channel_id" db:"channel_id"`
    Name             string    `json:"name" db:"name"`
    IsPrivate        bool      `json:"is_private" db:"is_private"` // NEW FIELD
    CreatedBy        string    `json:"created_by" db:"created_by"`
    ParticipantLimit int       `json:"participant_limit" db:"participant_limit"`

    // Existing fields...
    Archived         bool      `json:"archived" db:"archived"`
    Locked          bool      `json:"locked" db:"locked"`
    AutoArchive     int       `json:"auto_archive" db:"auto_archive"`
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// New Thread Participants Table
type ThreadParticipant struct {
    ThreadID     string    `json:"thread_id" db:"thread_id"`
    UserID       string    `json:"user_id" db:"user_id"`
    InvitedBy    string    `json:"invited_by" db:"invited_by"`
    JoinedAt     time.Time `json:"joined_at" db:"joined_at"`
    CanInvite    bool      `json:"can_invite" db:"can_invite"`
    IsActive     bool      `json:"is_active" db:"is_active"`
}

// Thread Creation Request Enhancement
type CreateThreadRequest struct {
    Name        string   `json:"name"`
    IsPrivate   bool     `json:"is_private"`   // NEW FIELD
    InviteUsers []string `json:"invite_users"` // Initial participants

    // Existing fields...
    MessageID   *string  `json:"message_id"`
    AutoArchive int      `json:"auto_archive"`
}
```

### Key Features
1. **Thread Visibility Control**
   - Private threads only visible to participants and channel moderators
   - Thread list filtering based on user permissions
   - Private thread icon and UI indicators
   - Search exclusion for non-participants

2. **Participant Management**
   - Thread creator can invite/remove participants
   - Moderators can always access private threads
   - Invitation notifications and consent flow
   - Participant limit enforcement (Discord limit: 10 participants)

3. **Permission Integration**
   - Respect channel's "Create Private Threads" permission
   - Thread-level permissions override system
   - Moderator bypass for all private threads
   - Read/write access control per participant

4. **UI/UX Enhancements**
   - Private thread creation toggle in UI
   - Distinct visual styling (lock icon, dimmed appearance)
   - "Add People" button for thread participants
   - Privacy warnings when creating private threads

### Database Schema Changes
```sql
-- Add private thread support to existing threads table
ALTER TABLE threads ADD COLUMN is_private BOOLEAN DEFAULT FALSE;
ALTER TABLE threads ADD COLUMN participant_limit INT DEFAULT 10;

-- New thread participants table
CREATE TABLE thread_participants (
    thread_id UUID REFERENCES threads(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    can_invite BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,

    PRIMARY KEY (thread_id, user_id),
    INDEX idx_thread_participants_thread (thread_id),
    INDEX idx_thread_participants_user (user_id)
);
```

### API Endpoints
- `POST /channels/{id}/threads` - Enhanced with `is_private` field
- `PUT /threads/{id}/participants/{user_id}` - Add/remove participants
- `GET /threads/{id}/participants` - List thread participants
- `POST /threads/{id}/invite` - Invite users to private thread
- `DELETE /threads/{id}/participants/@me` - Leave private thread

## Dependencies
1. **Enhanced Permission System** - Thread-level permission checking
2. **Notification Updates** - Private thread invitation notifications
3. **UI Component Updates** - Private thread creation and management
4. **Database Migration** - Add private thread support to existing schema
5. **WebSocket Events** - Thread participant updates and notifications

## Privacy & Security Considerations
- **Data Protection**: Private thread content never appears in public search
- **Audit Logging**: Track private thread creation and participant changes
- **Moderation Access**: Server moderators can always access for safety
- **Deletion Behavior**: Private threads delete all participant data on removal

## Success Metrics
- Private thread creation rate (% of total threads)
- Average participants per private thread
- Private thread message engagement vs public threads
- Moderation incident resolution time in private contexts
- User satisfaction with sensitive conversation handling

## Implementation Phases

### Phase 1: Core Infrastructure (3 weeks)
1. Database schema migration for private threads
2. Thread model and participant management
3. Basic permission checking for thread visibility
4. API endpoints for participant management

### Phase 2: UI Integration (2 weeks)
1. Private thread creation toggle
2. Thread list filtering and visual indicators
3. Participant management interface
4. Invitation flow and notifications

### Phase 3: Advanced Features (2 weeks)
1. Search exclusion for private threads
2. Advanced permission overwrites
3. Moderation tools for private threads
4. Performance optimization for large servers

**Total: 7 weeks for complete Discord parity**

## Risk Assessment
- **Low**: Database schema changes are additive and non-breaking
- **Medium**: Permission system complexity requires careful testing
- **Low**: Feature adoption risk mitigated by Discord's proven success

## Business Impact
Private threads are essential for:
- **Enterprise adoption** - Companies need confidential discussion capabilities
- **Community management** - Moderators need private incident discussion
- **User retention** - Sensitive conversations drive deeper engagement
- **Competitive parity** - Table stakes feature for Discord alternative