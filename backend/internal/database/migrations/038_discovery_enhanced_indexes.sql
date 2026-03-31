-- +migrate Up
-- Migration: 038_discovery_enhanced_indexes
-- Adds enhanced indexes for improved discovery search performance and recommendations

-- Add full-text search index for server name and description
CREATE INDEX idx_discoverable_servers_name_trgm ON discoverable_servers USING gin (name gin_trgm_ops) WHERE is_public = true;
CREATE INDEX idx_discoverable_servers_description_trgm ON discoverable_servers USING gin (description gin_trgm_ops) WHERE is_public = true AND description IS NOT NULL;

-- Add index for recommendations query optimization (user's joined servers)
CREATE INDEX idx_members_user_id ON members(user_id) WHERE is_banned = false;

-- Add index for discoverable servers by server_id and is_public for quick lookups
CREATE INDEX idx_discoverable_servers_server_public ON discoverable_servers(server_id, is_public);

-- Add partial index for servers with high member counts (for trending calculations)
CREATE INDEX idx_discoverable_servers_high_members ON discoverable_servers(member_count DESC, is_public) WHERE is_public = true AND member_count > 100;

-- Add index for category-based recommendations
CREATE INDEX idx_discoverable_servers_category_public ON discoverable_servers(category, is_public) WHERE is_public = true;

-- +migrate Down
DROP INDEX IF EXISTS idx_discoverable_servers_name_trgm;
DROP INDEX IF EXISTS idx_discoverable_servers_description_trgm;
DROP INDEX IF EXISTS idx_members_user_id;
DROP INDEX IF EXISTS idx_discoverable_servers_server_public;
DROP INDEX IF EXISTS idx_discoverable_servers_high_members;
DROP INDEX IF EXISTS idx_discoverable_servers_category_public;
