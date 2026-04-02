---
name: Message Edit History
description: Track and display complete edit history for transparency and accountability
type: Feature PRD
priority: P1
---

# Message Edit History

## Discord Equivalent
Direct 1:1 match with Discord's Message Edit History:
- Complete edit history accessible via "(edited)" hover tooltip
- Chronological list of all message revisions with timestamps
- Original content preservation for transparency
- Essential for accountability in communities and workplaces

## User Value Proposition
**Critical transparency feature** - Message edit history provides:
- **Accountability**: Prevent malicious edits that change context after replies
- **Transparency**: Community members can see what was originally said
- **Moderation**: Moderators can investigate edited content for policy violations
- **Trust**: Users feel confident that conversations maintain integrity
- **Context Preservation**: Replies and reactions make sense even after edits

## Technical Complexity: P1 (Medium Impact, Medium Complexity)

### Current Implementation Gap
```go
// CURRENT: Only tracks if/when message was edited
type Message struct {
    ID        string     `json:"id" db:"id"`
    Content   string     `json:"content" db:"content"`
    CreatedAt time.Time  `json:"created_at" db:"created_at"`
    EditedAt  *time.Time `json:"edited_at,omitempty" db:"edited_at"`
    // MISSING: Edit history tracking
}

// NEEDED: Message edit revision system
type MessageRevision struct {
    ID          string    `json:"id" db:"id"`
    MessageID   string    `json:"message_id" db:"message_id"`
    Content     string    `json:"content" db:"content"`
    EditedAt    time.Time `json:"edited_at" db:"edited_at"`
    EditedBy    string    `json:"edited_by" db:"edited_by"`
    RevisionNum int       `json:"revision_num" db:"revision_num"`
}
```

### Implementation Sketch
```go
// Message Edit History Service
type MessageEditHistory struct {
    MessageID   string              `json:"message_id"`
    CurrentContent string           `json:"current_content"`
    EditCount   int                 `json:"edit_count"`
    Revisions   []MessageRevision   `json:"revisions"`
    LastEditedAt time.Time          `json:"last_edited_at"`
}

// Enhanced Message Service
func (s *MessageService) UpdateMessage(ctx context.Context, messageID, userID, newContent string) error {
    // Get current message
    currentMsg, err := s.repo.GetMessage(messageID)
    if err != nil {
        return err
    }

    // Store current content as revision before updating
    revision := MessageRevision{
        ID:          uuid.New(),
        MessageID:   messageID,
        Content:     currentMsg.Content,
        EditedAt:    time.Now(),
        EditedBy:    userID,
        RevisionNum: s.getNextRevisionNumber(messageID),
    }

    // Begin transaction
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Save revision
    if err := s.revisionRepo.CreateRevision(ctx, revision); err != nil {
        return err
    }

    // Update message
    currentMsg.Content = newContent
    currentMsg.EditedAt = &revision.EditedAt
    if err := s.repo.UpdateMessage(ctx, currentMsg); err != nil {
        return err
    }

    return tx.Commit()
}

// Get Edit History
func (s *MessageService) GetEditHistory(messageID string) (*MessageEditHistory, error) {
    revisions, err := s.revisionRepo.GetMessageRevisions(messageID)
    if err != nil {
        return nil, err
    }

    currentMsg, err := s.repo.GetMessage(messageID)
    if err != nil {
        return nil, err
    }

    return &MessageEditHistory{
        MessageID:      messageID,
        CurrentContent: currentMsg.Content,
        EditCount:      len(revisions),
        Revisions:      revisions,
        LastEditedAt:   *currentMsg.EditedAt,
    }, nil
}
```

### Key Features
1. **Complete Revision Tracking**
   - Store every version of message content with timestamp
   - Track which user made each edit
   - Maintain chronological revision order
   - Preserve original creation content as "revision 0"

2. **UI Integration**
   - "(edited)" indicator on all edited messages
   - Hover tooltip showing latest edit timestamp
   - Click to view full edit history modal
   - Diff highlighting showing changes between revisions

3. **Edit History Modal**
   - Chronological list of all revisions
   - "Show changes" toggle for content diff view
   - User avatars and timestamps for each edit
   - Copy/quote original content functionality

4. **Performance Optimization**
   - Edit history loaded lazily (only when requested)
   - Automatic cleanup of very old revisions (configurable retention)
   - Efficient database queries with proper indexing
   - Content compression for storage optimization

### Database Schema
```sql
-- New message revisions table
CREATE TABLE message_revisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    edited_at TIMESTAMPTZ NOT NULL,
    edited_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    revision_num INTEGER NOT NULL,

    CONSTRAINT unique_message_revision UNIQUE (message_id, revision_num),
    INDEX idx_message_revisions_message (message_id),
    INDEX idx_message_revisions_edited_at (edited_at)
);

-- Function to get next revision number
CREATE OR REPLACE FUNCTION get_next_revision_number(msg_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN COALESCE(
        (SELECT MAX(revision_num) + 1 FROM message_revisions WHERE message_id = msg_id),
        1
    );
END;
$$ LANGUAGE plpgsql;
```

### API Endpoints
- `GET /messages/{id}/history` - Get complete edit history
- `GET /messages/{id}/revisions/{num}` - Get specific revision content
- `PUT /messages/{id}` - Enhanced to save revision before update
- `GET /channels/{id}/messages` - Enhanced to include edit indicators

### Frontend Components
```typescript
// Edit History Modal Component
interface EditHistoryModalProps {
    messageId: string;
    isOpen: boolean;
    onClose: () => void;
}

interface MessageRevision {
    id: string;
    content: string;
    editedAt: string;
    editedBy: User;
    revisionNum: number;
}

// Edit Indicator Component
interface EditIndicatorProps {
    lastEditedAt: string;
    editCount: number;
    onClick: () => void;
}
```

### Content Diff Display
```typescript
// Text diff algorithm for showing changes
function generateDiff(oldContent: string, newContent: string): DiffResult {
    // Use library like 'diff' to generate character-level differences
    // Return additions (green), deletions (red), and unchanged text
}

// Revision comparison view
interface RevisionCompareProps {
    fromRevision: MessageRevision;
    toRevision: MessageRevision;
    showDiff: boolean;
}
```

## Dependencies
1. **Database Migration** - New message_revisions table
2. **Message Service Enhancement** - Revision tracking on edit operations
3. **UI Components** - Edit history modal and diff display
4. **Performance Monitoring** - Track storage and query impact
5. **Cleanup Jobs** - Automated revision pruning for storage management

## Privacy & Moderation Considerations
- **Edit History Visibility**: All channel members can view edit history (matches Discord)
- **Moderation Access**: Moderators can view edit history for policy enforcement
- **Data Retention**: Configurable revision cleanup (default: keep 30 days or 10 revisions)
- **Content Policy**: Edits don't escape content filtering or moderation review

## Success Metrics
- Edit history access rate (% of edited messages where history is viewed)
- Average revisions per edited message
- Storage impact and query performance
- User trust surveys regarding message integrity
- Moderation effectiveness with edit history access

## Implementation Phases

### Phase 1: Backend Infrastructure (2 weeks)
1. Database schema for message revisions
2. Enhanced message service with revision tracking
3. API endpoints for edit history access
4. Efficient queries and indexing

### Phase 2: Basic UI Integration (2 weeks)
1. "(edited)" indicator display
2. Edit timestamp tooltip
3. Basic edit history modal
4. Revision list with timestamps

### Phase 3: Advanced Features (1.5 weeks)
1. Content diff highlighting
2. Revision comparison view
3. Performance optimization
4. Automated cleanup jobs

**Total: 5.5 weeks for complete implementation**

## Storage Impact Analysis
```sql
-- Conservative storage estimates
-- Average message: 100 characters
-- Average edits per message: 2
-- Additional storage per edit: ~200 bytes

-- For 1M messages with 20% edit rate:
-- 200,000 edited messages × 2 revisions × 200 bytes = 80MB additional storage
-- Manageable with proper cleanup policies
```

## Risk Assessment
- **Medium**: Storage growth requires monitoring and cleanup strategies
- **Low**: Database performance impact mitigated by efficient indexing
- **Low**: Feature adoption risk mitigated by Discord's proven user demand

## Business Impact
Message edit history is essential for:
- **Enterprise Trust** - Companies need audit trails for compliance
- **Community Integrity** - Prevents malicious context manipulation
- **Moderation Effectiveness** - Helps moderators investigate edited content
- **User Confidence** - Transparency builds trust in platform integrity
- **Competitive Parity** - Expected feature for professional Discord alternative