-- +migrate Up
-- Migration: 034_public_server_directory
-- Adds a simple public server directory with categories for user acquisition

-- Create discoverable_servers table for the public server directory
CREATE TABLE discoverable_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL UNIQUE REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(32) NOT NULL DEFAULT 'other',
    icon_url TEXT,
    banner_url TEXT,
    member_count INT NOT NULL DEFAULT 0,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_public BOOLEAN NOT NULL DEFAULT true,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    featured_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index for discovery queries
CREATE INDEX idx_discoverable_servers_public ON discoverable_servers(is_public) WHERE is_public = true;
CREATE INDEX idx_discoverable_servers_featured ON discoverable_servers(is_featured, featured_at) WHERE is_featured = true;
CREATE INDEX idx_discoverable_servers_category ON discoverable_servers(category) WHERE is_public = true;
CREATE INDEX idx_discoverable_servers_member_count ON discoverable_servers(member_count DESC) WHERE is_public = true;
CREATE INDEX idx_discoverable_servers_name_search ON discoverable_servers(name) WHERE is_public = true;

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_discoverable_server_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for auto-updating timestamp
CREATE TRIGGER discoverable_servers_updated_at
    BEFORE UPDATE ON discoverable_servers
    FOR EACH ROW
    EXECUTE FUNCTION update_discoverable_server_timestamp();

-- +migrate Down
DROP TRIGGER IF EXISTS discoverable_servers_updated_at ON discoverable_servers;
DROP FUNCTION IF EXISTS update_discoverable_server_timestamp();
DROP TABLE IF EXISTS discoverable_servers;
