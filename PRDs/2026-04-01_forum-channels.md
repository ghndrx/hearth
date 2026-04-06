---
name: Forum Channels
description: Thread-based discussion channels with tags, sorting, and moderation tools
type: feature
priority: P0
discord_equivalent: Forum channels with tags, post templates, and sorting
estimated_complexity: High
---

# Forum Channels

## Discord Equivalent
Direct match to Discord's Forum channels - a thread-based discussion format replacing traditional forums.

## User Value Proposition
- **Organized discussions**: Replace scattered threads with structured forum-style posts
- **Knowledge management**: Searchable, categorized discussions with tags
- **Community building**: Encourage longer-form discussions and help-seeking
- **Moderation efficiency**: Template enforcement and tag-based organization

## Technical Complexity: P0 (High)
- New channel type with different message flow
- Tag system with colors and moderation
- Post templates and validation
- Advanced sorting and filtering
- Thread lifecycle management

## Implementation Sketch

### Backend Components
1. **Database Schema**
   ```sql
   CREATE TABLE forum_channels (
     id UUID PRIMARY KEY REFERENCES channels(id),
     default_sort_order VARCHAR(20) DEFAULT 'recent_activity',
     post_guidelines TEXT,
     require_tags BOOLEAN DEFAULT false,
     max_tags_per_post INTEGER DEFAULT 5
   );

   CREATE TABLE forum_tags (
     id UUID PRIMARY KEY,
     channel_id UUID NOT NULL REFERENCES forum_channels(id),
     name VARCHAR(50) NOT NULL,
     emoji VARCHAR(10),
     color VARCHAR(7),
     moderated BOOLEAN DEFAULT false
   );

   CREATE TABLE forum_posts (
     id UUID PRIMARY KEY,
     channel_id UUID NOT NULL REFERENCES forum_channels(id),
     author_id UUID NOT NULL REFERENCES users(id),
     title VARCHAR(200) NOT NULL,
     tags UUID[] DEFAULT '{}',
     pinned BOOLEAN DEFAULT false,
     locked BOOLEAN DEFAULT false,
     archived BOOLEAN DEFAULT false,
     last_activity_at TIMESTAMP DEFAULT NOW()
   );

   CREATE TABLE post_templates (
     id UUID PRIMARY KEY,
     channel_id UUID NOT NULL REFERENCES forum_channels(id),
     name VARCHAR(100) NOT NULL,
     template_content TEXT NOT NULL,
     required_fields JSONB DEFAULT '{}'
   );
   ```

2. **API Endpoints**
   - `POST /channels/{id}/posts` - Create forum post
   - `GET /channels/{id}/posts?sort=recent&tags=help` - List posts
   - `PUT /posts/{id}/tags` - Update post tags
   - `POST /channels/{id}/tags` - Manage channel tags

### Frontend Components
1. **ForumChannelView.svelte** - Main forum interface
2. **ForumPostCard.svelte** - Individual post preview
3. **PostCreator.svelte** - New post creation with templates
4. **TagManager.svelte** - Tag selection and management
5. **ForumSettings.svelte** - Channel-specific forum configuration

### Key Features
1. **Post Management**
   - Create posts with title, tags, and initial message
   - Pin/lock/archive posts (moderator controls)
   - Post templates with required fields

2. **Tag System**
   - Color-coded tags with optional emoji
   - Moderated tags (require permission to use)
   - Filter posts by single or multiple tags

3. **Sorting Options**
   - Recent activity (default)
   - Creation date (newest/oldest)
   - Most replies
   - Most reactions

4. **Search & Discovery**
   - Full-text search within forum posts
   - Tag-based filtering
   - "Solved" marking for help channels

## Dependencies
- [ ] Thread system working (✅ implemented)
- [ ] Message reactions (✅ implemented)
- [ ] Advanced permissions system
- [ ] Full-text search (✅ implemented)

## Success Metrics
- Forum post creation rate vs regular thread creation
- Tag usage adoption rate > 60%
- Search success rate in forum channels > 80%
- Moderator efficiency improvements (time to resolve)

## Implementation Timeline
- Phase 1: Basic forum channel type and posts (3 weeks)
- Phase 2: Tag system and filtering (2 weeks)
- Phase 3: Post templates and guidelines (2 weeks)
- Phase 4: Advanced moderation tools (2 weeks)