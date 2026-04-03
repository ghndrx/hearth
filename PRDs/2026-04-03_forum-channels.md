---
name: Forum Channels
description: Discord-style forum channels with threaded discussions and tagging system
type: feature
priority: P0
complexity: High
dependencies: Thread system, forum tags, permission system
---

# Forum Channels

## Discord Equivalent
Discord's forum channels allow organized threaded discussions similar to Reddit or traditional forums, with tags for categorization and moderation tools.

## User Value Proposition
- **Organized Discussions**: Threaded conversations around specific topics vs chaotic general chat
- **Community Knowledge Base**: Persistent, searchable discussions that build community knowledge
- **Better Moderation**: Tag-based organization and thread-level moderation controls
- **User Engagement**: Encourages longer-form discussion and community participation

## Technical Complexity: P0 (High)
**Backend Changes:**
- New channel type: `forum` with thread creation permissions
- Forum post creation system (automatic thread creation)
- Forum tags system (predefined tags per forum)
- Thread sorting algorithms (hot, new, trending)
- Forum-specific permissions (create posts, manage tags)

**Frontend Changes:**
- Forum channel view with thread list
- Thread preview cards with tag display
- Forum post creation flow (title + initial message)
- Tag selection and filtering UI
- Sort/filter controls for thread browsing

## Implementation Sketch
1. **Database Schema**:
   - Extend channels table with `forum_tags_enabled`, `default_sort_order`
   - Add forum_tags table (id, channel_id, name, color, emoji_id, position)
   - Add forum_posts table (thread_id, title, tags[], created_at)

2. **Permission Model**:
   - `CREATE_FORUM_POSTS` - Can create new threads in forum
   - `MANAGE_FORUM_TAGS` - Can add/edit forum tags
   - `MODERATE_FORUM` - Can pin, lock, archive forum threads

3. **API Endpoints**:
   - `POST /channels/:id/threads` - Create forum post (thread)
   - `GET /channels/:id/threads` - List forum threads with pagination
   - `POST /channels/:id/tags` - Create forum tag
   - `GET /channels/:id/tags` - List forum tags

4. **Frontend Components**:
   - ForumView component with thread listing
   - CreatePostModal with title + tag selection
   - ForumThreadCard with preview and stats
   - TagFilter component for browsing

## Dependencies
- Thread system (✅ implemented)
- Tag system (❌ needs implementation)
- Permission system (✅ implemented)
- Search functionality (❌ advanced search needed)

## Success Metrics
- Forum adoption rate >40% of servers with >100 members
- Average threads per active forum >25
- Thread engagement (replies per thread) >8
- User retention from forum participation +15%