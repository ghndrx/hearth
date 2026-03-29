---
name: Channel Following & Cross-Posting
description: Follow announcement channels from other servers and cross-post content
type: feature
priority: P2
---

# Channel Following & Cross-Posting

## Discord Equivalent
Discord's channel following allows users to receive posts from announcement channels in other servers, creating a content syndication network that promotes community discovery and engagement.

## User Value Proposition
- **Content Discovery**: Access content from multiple communities in one place
- **Community Growth**: Helps smaller servers gain visibility
- **Cross-Pollination**: Encourages users to join source communities
- **Network Effects**: Creates interconnected server ecosystem

## Technical Complexity: P2 (Medium)

### Implementation Sketch
1. **Data Model Enhancement**: Extend existing `follow.go` model with:
   - Cross-server following relationships
   - Announcement channel designation
   - Following permissions and settings
   - Cross-post attribution tracking

2. **Following System**:
   - Channel type: "announcement" designation
   - Follow request/approval workflow
   - Automatic cross-posting pipeline
   - Content filtering and moderation
   - Rate limiting to prevent spam

3. **API Endpoints**:
   - `POST /channels/{id}/follow` - Follow a channel
   - `GET /channels/{id}/followers` - List channel followers
   - `POST /channels/{id}/announce` - Designate as announcement channel
   - `DELETE /follows/{id}` - Unfollow channel
   - `GET /servers/{id}/following` - List followed channels

4. **UI Components**:
   - FollowChannelModal.svelte (follow interface)
   - AnnouncementBadge.svelte (channel designation)
   - FollowedChannelsList.svelte (manage follows)
   - CrossPostIndicator.svelte (show original source)

5. **Permissions**:
   - "Manage Server" to designate announcement channels
   - "Manage Channels" to approve follow requests
   - Channel-specific following permissions
   - Server-level following policies

### Dependencies
- Existing follow model
- Server permission system
- Message routing infrastructure
- Cross-server communication

### Success Metrics
- Channel follow rate
- Cross-post engagement vs original posts
- Server discovery through follows
- Network growth between servers