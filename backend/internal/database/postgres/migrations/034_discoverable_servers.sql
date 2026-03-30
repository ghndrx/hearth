-- Migration: Discoverable Servers (public directory)
-- Creates the discoverable_servers table for the enhanced server discovery feature

CREATE TABLE IF NOT EXISTS discoverable_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL UNIQUE REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(64) NOT NULL DEFAULT 'other',
    icon_url TEXT,
    banner_url TEXT,
    tags TEXT[] NOT NULL DEFAULT '{}',
    member_count INT NOT NULL DEFAULT 0,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_public BOOLEAN NOT NULL DEFAULT true,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    featured_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_public ON discoverable_servers(is_public) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_featured ON discoverable_servers(is_featured, featured_at DESC) WHERE is_featured = true;
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_category ON discoverable_servers(category);
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_member_count ON discoverable_servers(member_count DESC);
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_server_id ON discoverable_servers(server_id);
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_name_trgm ON discoverable_servers USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_tags ON discoverable_servers USING gin (tags);

-- Trigger for auto-updating updated_at
CREATE OR REPLACE FUNCTION update_discoverable_servers_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER discoverable_servers_updated_at
    BEFORE UPDATE ON discoverable_servers
    FOR EACH ROW
    EXECUTE FUNCTION update_discoverable_servers_timestamp();
