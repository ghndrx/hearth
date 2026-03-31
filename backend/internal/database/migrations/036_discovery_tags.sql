-- +migrate Up
-- Migration: 036_discovery_tags
-- Adds tags column to discoverable_servers for enhanced server discovery

-- Add tags array column to discoverable_servers
ALTER TABLE discoverable_servers ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';

-- Create GIN index for efficient tag queries
CREATE INDEX IF NOT EXISTS idx_discoverable_servers_tags ON discoverable_servers USING GIN (tags) WHERE is_public = true;

-- +migrate Down
DROP INDEX IF EXISTS idx_discoverable_servers_tags;
ALTER TABLE discoverable_servers DROP COLUMN IF EXISTS tags;
