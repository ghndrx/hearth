-- Migration 029: Forum Channel Tags
-- Adds forum tag management and applied tags on threads

-- Forum tags table (server-scoped tag definitions)
CREATE TABLE forum_tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    emoji_name VARCHAR(128),
    moderated BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(channel_id, name)
);

CREATE INDEX idx_forum_tags_server ON forum_tags(server_id);
CREATE INDEX idx_forum_tags_channel ON forum_tags(channel_id);

-- Add applied_tags column to threads table (for forum posts)
ALTER TABLE threads ADD COLUMN applied_tags UUID[] DEFAULT '{}';

-- Add forum_config column to channels table (JSONB for forum-specific settings)
ALTER TABLE channels ADD COLUMN forum_config JSONB DEFAULT NULL;

-- Create index for threads with applied_tags
CREATE INDEX idx_threads_applied_tags ON threads USING GIN(applied_tags);

-- Add columns to threads for forum post metadata
ALTER TABLE threads ADD COLUMN is_pinned BOOLEAN DEFAULT FALSE;
ALTER TABLE threads ADD COLUMN pin_weight INTEGER DEFAULT 0;
