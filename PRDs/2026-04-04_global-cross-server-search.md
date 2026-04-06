# PRD: Global Cross-Server Search

**Date:** 2026-04-04
**Status:** In Planning
**Priority:** P0
**Complexity:** High

## Problem Statement

Hearth's search functionality is currently scoped to individual servers/channels only. Users cannot search across all servers they're members of, unlike Discord's global search that allows finding messages, files, and content across their entire server history.

## Discord Equivalent

Discord's global search feature accessible via Ctrl+K or the search bar at the top, allowing users to search across all servers, DMs, and group chats they have access to.

## User Value Proposition

- **Power Users**: Advanced users managing multiple communities need to find conversations/files across their entire Hearth ecosystem
- **Knowledge Workers**: Business/professional users need to locate information shared across different team servers
- **Content Creators**: Need to find past discussions, media, or resources shared across various community servers
- **Mobile Experience**: Critical for mobile users who rely heavily on search due to limited screen space

## Technical Requirements

### Core Functionality
- Global search bar accessible from main navigation
- Search across all servers user has access to
- Support for message content, file names, user mentions, channel names
- Unified results showing context (server name, channel, timestamp)
- Permission-aware results (respect server/channel access controls)

### Advanced Features  
- Filter by server, date range, file type, author
- Quick server switcher integration 
- Search within DMs and group DMs
- Autocomplete suggestions from search history
- Mobile-optimized search interface

## Implementation Sketch

### Backend Changes
1. **New Search Service**: Create `GlobalSearchService` alongside existing server-scoped `SearchService`
2. **Permission Layer**: Build efficient permission checking for cross-server results
3. **Search Index**: Optimize database queries or implement search index for cross-server performance
4. **API Endpoints**: 
   - `GET /search/global/messages`
   - `GET /search/global/files` 
   - `GET /search/global/servers`

### Frontend Changes
1. **Global Search Component**: Top-level search bar component
2. **Results View**: Unified results display with server context
3. **Search Filters**: Advanced filter UI for refining results
4. **Mobile Integration**: Touch-optimized search experience

### Database Optimization
- Add composite indexes for cross-server search performance
- Consider search result caching for common queries
- Optimize permission checking queries

## Technical Complexity: P0 (High)

**Estimated Timeline:** 8-12 weeks

**Complexity Factors:**
- Cross-server permission checking at scale
- Database query optimization for large user bases
- Mobile performance optimization
- Search ranking and relevance algorithms

## Dependencies

1. **Performance Infrastructure**: May require search indexing system (Elasticsearch/similar)
2. **Permission System**: Must leverage existing role/permission infrastructure
3. **Mobile Apps**: Requires mobile development for optimal UX

## Success Metrics

- User engagement: % of users using global search weekly
- Search success rate: % of searches returning clicked results  
- Performance: Search response time <500ms for most queries
- Feature parity: Competitive comparison with Discord's search capabilities

## Future Enhancements

- AI-powered semantic search
- Cross-server conversation threading
- Advanced search operators (from:user, in:channel, etc.)
- Saved search queries
- Search API for third-party integrations