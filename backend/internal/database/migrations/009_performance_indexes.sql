-- +migrate Up
-- Performance optimization indexes based on query analysis
-- Created: 2026-02-26

-- ============================================================================
-- HIGH PRIORITY: Frequently accessed query patterns
-- ============================================================================

-- Channel type filtering (GetUserDMs, GetDMChannel queries)
-- Partial index for DM channel lookups - much smaller than full index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_channels_type_dm 
ON channels (type) 
WHERE type IN ('dm', 'group_dm');

-- Channel recipients lookup by user (GetUserDMs N+1 fix support)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_channel_recipients_user_id 
ON channel_recipients (user_id);

-- Relationship type filtering (friend queries, blocks)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_relationships_type 
ON relationships (user_id, type);

-- Pinned messages - partial index (GetPinnedMessages)
-- Only indexes pinned messages, dramatically smaller
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_channel_pinned 
ON messages (channel_id, created_at DESC) 
WHERE pinned = true;

-- ============================================================================
-- FULL-TEXT SEARCH: Message content search
-- ============================================================================

-- GIN index for full-text search (SearchMessages)
-- Use unaccent for better search results across languages
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_content_fts 
ON messages USING GIN (to_tsvector('english', COALESCE(content, '')));

-- ============================================================================
-- ANALYTICS: Reporting and metrics queries
-- ============================================================================

-- Server activity hourly - composite for aggregation queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_server_activity_hourly_lookup 
ON server_activity_hourly (server_id, activity_hour DESC);

-- Daily active users - composite for retention queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_server_dau_lookup 
ON server_daily_active_users (server_id, activity_date DESC);

-- Member snapshots - for growth charts
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_member_snapshots_lookup 
ON server_member_snapshots (server_id, snapshot_date DESC);

-- ============================================================================
-- THREAD MANAGEMENT: Thread presence and membership
-- ============================================================================

-- Thread presence lookup by last_seen_at (for stale presence cleanup queries)
-- Note: Using standard B-tree index, not partial - NOW() in partial indexes
-- is evaluated at creation time only, making the condition static/useless
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_thread_presence_last_seen 
ON thread_presence (last_seen_at);

-- Thread members by user (for "threads you're in" queries)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_thread_members_user 
ON thread_members (user_id);

-- ============================================================================
-- MENTIONS: Unread mention queries
-- ============================================================================

-- Unread mentions by user - partial index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mentions_unread 
ON mentions (user_id, created_at DESC) 
WHERE read_at IS NULL;

-- Mentions by channel (for channel mention cleanup)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mentions_channel 
ON mentions (channel_id);

-- ============================================================================
-- READ STATE: Unread message tracking
-- ============================================================================

-- Read states with mention counts > 0 (for notification badges)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_read_states_mentions 
ON read_states (user_id) 
WHERE mention_count > 0;

-- ============================================================================
-- TIME-SERIES: BRIN index for message timestamps
-- ============================================================================

-- BRIN index for large time-range scans (much smaller than B-tree)
-- Only useful for queries spanning large time ranges
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_created_brin 
ON messages USING BRIN (created_at) 
WITH (pages_per_range = 128);

-- ============================================================================
-- COMPOSITE INDEXES: Multi-column query optimization
-- ============================================================================

-- Messages by author in channel (GetMessagesByAuthor)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_author_channel 
ON messages (author_id, channel_id, created_at DESC);

-- Audit log by target (for "changes to this entity" queries)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_log_target_type_id 
ON audit_log (target_type, target_id, created_at DESC);

-- +migrate Down

DROP INDEX CONCURRENTLY IF EXISTS idx_channels_type_dm;
DROP INDEX CONCURRENTLY IF EXISTS idx_channel_recipients_user_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_relationships_type;
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_channel_pinned;
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_content_fts;
DROP INDEX CONCURRENTLY IF EXISTS idx_server_activity_hourly_lookup;
DROP INDEX CONCURRENTLY IF EXISTS idx_server_dau_lookup;
DROP INDEX CONCURRENTLY IF EXISTS idx_member_snapshots_lookup;
DROP INDEX CONCURRENTLY IF EXISTS idx_thread_presence_last_seen;
DROP INDEX CONCURRENTLY IF EXISTS idx_thread_members_user;
DROP INDEX CONCURRENTLY IF EXISTS idx_mentions_unread;
DROP INDEX CONCURRENTLY IF EXISTS idx_mentions_channel;
DROP INDEX CONCURRENTLY IF EXISTS idx_read_states_mentions;
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_created_brin;
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_author_channel;
DROP INDEX CONCURRENTLY IF EXISTS idx_audit_log_target_type_id;
