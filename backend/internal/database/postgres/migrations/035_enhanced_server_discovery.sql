-- Migration: Enhanced Server Discovery
-- Creates the public server directory with search, categories, recommendations,
-- activity tracking, and user acquisition features

-- Core discoverable_servers table (public server directory)
CREATE TABLE IF NOT EXISTS discoverable_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL UNIQUE REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(64) NOT NULL DEFAULT 'other',
    icon_url TEXT,
    banner_url TEXT,
    member_count INT NOT NULL DEFAULT 0,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_public BOOLEAN NOT NULL DEFAULT true,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    featured_at TIMESTAMP WITH TIME ZONE,
    -- Enhanced fields for discovery & user acquisition
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    region VARCHAR(64),
    vanity_url VARCHAR(64) UNIQUE,
    -- Activity & engagement metrics
    online_count INT NOT NULL DEFAULT 0,
    messages_per_day DECIMAL(10,2) NOT NULL DEFAULT 0,
    active_members_7d INT NOT NULL DEFAULT 0,
    weekly_growth_rate DECIMAL(5,2) NOT NULL DEFAULT 0,
    engagement_score DECIMAL(5,2) NOT NULL DEFAULT 0,
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for discovery queries
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_public ON discoverable_servers(is_public) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_featured ON discoverable_servers(is_featured, featured_at DESC) WHERE is_featured = true;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_category ON discoverable_servers(category) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_member_count ON discoverable_servers(member_count DESC) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_engagement ON discoverable_servers(engagement_score DESC) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_growth ON discoverable_servers(weekly_growth_rate DESC) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_language ON discoverable_servers(language) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_region ON discoverable_servers(region) WHERE is_public = true AND region IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_name_trgm ON discoverable_servers USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_created ON discoverable_servers(created_at DESC) WHERE is_public = true;

-- Tags for discoverable servers (many-to-many via junction)
CREATE TABLE IF NOT EXISTS discoverable_server_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL UNIQUE,
    slug VARCHAR(64) NOT NULL UNIQUE,
    usage_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS discoverable_server_tag_map (
    server_id UUID NOT NULL REFERENCES discoverable_servers(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES discoverable_server_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (server_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_discoverable_server_tag_map_tag ON discoverable_server_tag_map(tag_id);

-- Activity tracking: views, impressions, joins from discovery
CREATE TABLE IF NOT EXISTS server_discovery_activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES discoverable_servers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    activity_type VARCHAR(32) NOT NULL, -- 'view', 'impression', 'join', 'search_click'
    source VARCHAR(64), -- 'home', 'search', 'category', 'trending', 'recommended', 'featured'
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discovery_activity_server ON server_discovery_activity(server_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_discovery_activity_type ON server_discovery_activity(activity_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_discovery_activity_created ON server_discovery_activity(created_at DESC);

-- Materialized view for daily discovery stats (refreshed periodically)
CREATE TABLE IF NOT EXISTS server_discovery_daily_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES discoverable_servers(id) ON DELETE CASCADE,
    stat_date DATE NOT NULL DEFAULT CURRENT_DATE,
    views INT NOT NULL DEFAULT 0,
    impressions INT NOT NULL DEFAULT 0,
    joins INT NOT NULL DEFAULT 0,
    search_clicks INT NOT NULL DEFAULT 0,
    UNIQUE(server_id, stat_date)
);

CREATE INDEX IF NOT EXISTS idx_discovery_daily_stats_server ON server_discovery_daily_stats(server_id, stat_date DESC);

-- Auto-update timestamp trigger
CREATE OR REPLACE FUNCTION update_discoverable_servers_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS discoverable_servers_updated_at ON discoverable_servers;
CREATE TRIGGER discoverable_servers_updated_at
    BEFORE UPDATE ON discoverable_servers
    FOR EACH ROW
    EXECUTE FUNCTION update_discoverable_servers_timestamp();

-- Insert default tags for common server topics
INSERT INTO discoverable_server_tags (name, slug, usage_count) VALUES
    ('Competitive', 'competitive', 0),
    ('Casual', 'casual', 0),
    ('Esports', 'esports', 0),
    ('Streaming', 'streaming', 0),
    ('Creative', 'creative', 0),
    ('Programming', 'programming', 0),
    ('Open Source', 'open-source', 0),
    ('Chill', 'chill', 0),
    ('Memes', 'memes', 0),
    ('Study Group', 'study-group', 0),
    ('Roleplay', 'roleplay', 0),
    ('Anime', 'anime', 0),
    ('Music Production', 'music-production', 0),
    ('Game Dev', 'game-dev', 0),
    ('Crypto', 'crypto', 0),
    ('Fitness', 'fitness', 0),
    ('Photography', 'photography', 0),
    ('Book Club', 'book-club', 0),
    ('Language Exchange', 'language-exchange', 0),
    ('NFT', 'nft', 0)
ON CONFLICT (slug) DO NOTHING;

-- Enable trigram extension for fuzzy text search (if not already enabled)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
