---
name: Channel Following & Cross-Server Content
description: Follow announcement channels across servers for cross-community content syndication
type: feature
priority: P1
complexity: High
dependencies: Webhook system, permission system, content moderation
---

# Channel Following & Cross-Server Content

## Discord Equivalent
Discord's "Follow" feature that allows announcement channels to be followed by other servers, syndicating content across communities.

## User Value Proposition
- **Content Distribution**: Share announcements across multiple related servers
- **Community Networks**: Connect related communities without duplicating content
- **Creator Reach**: Content creators can broadcast to multiple audiences simultaneously
- **Event Coordination**: Coordinate events across partner servers and communities

## Technical Complexity: P1 (High)
**Backend Changes:**
- Cross-server content syndication system
- Follow relationship management (follower/following tracking)
- Content filtering and moderation for followed content
- Webhook-based delivery system for followed messages
- Permission system for follow/unfollow actions
- Rate limiting for cross-server content to prevent spam

**Frontend Changes:**
- Follow/unfollow button on announcement channels
- Cross-server content indicators in message UI
- Followed channels management interface
- Content source attribution in syndicated messages
- Follow permissions configuration for server admins

**Database Schema:**
```sql
-- Channel following relationships
CREATE TABLE channel_follows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_channel_id UUID NOT NULL REFERENCES channels(id),
    target_channel_id UUID NOT NULL REFERENCES channels(id),
    follower_server_id UUID NOT NULL REFERENCES servers(id),
    followed_server_id UUID NOT NULL REFERENCES servers(id),
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(id),
    UNIQUE(source_channel_id, target_channel_id)
);

-- Syndicated message tracking
CREATE TABLE syndicated_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_message_id UUID NOT NULL REFERENCES messages(id),
    syndicated_message_id UUID NOT NULL REFERENCES messages(id),
    follow_id UUID NOT NULL REFERENCES channel_follows(id),
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Implementation Sketch

### Follow Management
```go
type ChannelFollow struct {
    ID                uuid.UUID
    SourceChannelID   uuid.UUID // Channel being followed
    TargetChannelID   uuid.UUID // Channel receiving content
    FollowerServerID  uuid.UUID
    FollowedServerID  uuid.UUID
    Enabled           bool
    CreatedBy         uuid.UUID
}

func (s *FollowService) CreateFollow(
    sourceChannelID, targetChannelID, userID uuid.UUID,
) (*ChannelFollow, error) {
    // Check permissions on both servers
    // Validate channel types (announcement -> any text)
    // Create follow relationship
    // Set up webhook delivery
}
```

### Content Syndication
```go
func (s *SyndicationService) ProcessMessage(message *models.Message) {
    follows := s.getActiveFollows(message.ChannelID)

    for _, follow := range follows {
        // Filter content (no replies, embeds only, etc.)
        if !s.isEligibleForSyndication(message) {
            continue
        }

        // Create syndicated message
        syndicatedMsg := s.createSyndicatedMessage(message, follow)

        // Send to target channel with attribution
        s.deliverSyndicatedMessage(syndicatedMsg, follow.TargetChannelID)
    }
}
```

### Content Attribution
- Source server name and icon
- Original author attribution (opt-in)
- "Followed from #channel-name" indicators
- Original message link for context

## Dependencies
1. **Webhook System**: Reliable cross-server message delivery ✅
2. **Permission System**: Complex cross-server permission checks ✅
3. **Rate Limiting**: Prevent follow-spam across servers
4. **Content Moderation**: Filter inappropriate syndicated content

## Success Metrics
- Follow relationship creation rate
- Cross-server engagement on syndicated content
- Content creator reach amplification
- Server partnership growth through follows

## Implementation Priority
**P1** - Nice-to-have feature that enhances community interconnection. Not blocking for core functionality but valuable for content creators and community networks. Helps drive organic server discovery and growth.

## Security Considerations
- **Spam Prevention**: Rate limits on follows per server
- **Content Filtering**: Block NSFW syndication to SFW servers
- **Permission Validation**: Cross-server permission checks
- **Privacy Controls**: Opt-out of being followed, anonymous following options

## Feature Variations
### Basic Following
- Announcement channels only
- Public servers only
- Simple content syndication

### Advanced Following
- Multiple channel types
- Private server following with approval
- Content filtering rules
- Selective message syndication (tags/keywords)

## Integration Points
- **Server Directory**: Promote followable servers
- **Webhooks**: Leverage existing webhook infrastructure
- **Discovery**: Surface popular followed channels