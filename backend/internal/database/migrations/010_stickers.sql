-- Migration: Stickers System
-- Description: Add stickers table for custom emoji/sticker support

-- Create stickers table
CREATE TABLE IF NOT EXISTS stickers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(30) NOT NULL,
    tags TEXT[] DEFAULT '{}',
    url VARCHAR(500) NOT NULL,
    format VARCHAR(10) NOT NULL CHECK (format IN ('PNG', 'APNG', 'GIF')),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index for server_id lookups
CREATE INDEX IF NOT EXISTS idx_stickers_server_id ON stickers(server_id);

-- Create index for global stickers (server_id IS NULL)
CREATE INDEX IF NOT EXISTS idx_stickers_global ON stickers(server_id) WHERE server_id IS NULL;

-- Create index for searching by name
CREATE INDEX IF NOT EXISTS idx_stickers_name ON stickers USING gin(to_tsvector('english', name));

-- Create index for tags
CREATE INDEX IF NOT EXISTS idx_stickers_tags ON stickers USING gin(tags);

-- Create composite index for server + name lookups
CREATE INDEX IF NOT EXISTS idx_stickers_server_name ON stickers(server_id, name);

-- Add comment
COMMENT ON TABLE stickers IS 'Custom stickers for servers and global stickers';
