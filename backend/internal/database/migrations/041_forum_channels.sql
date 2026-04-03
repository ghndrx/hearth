-- +migrate Up

-- Forum tags table
CREATE TABLE IF NOT EXISTS forum_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7),
    emoji_name VARCHAR(128),
    moderated BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast lookups by channel
CREATE INDEX IF NOT EXISTS idx_forum_tags_channel ON forum_tags(channel_id);
-- Index for fast lookups by server
CREATE INDEX IF NOT EXISTS idx_forum_tags_server ON forum_tags(server_id);

-- Add applied_tags column to threads for forum post tagging
ALTER TABLE threads ADD COLUMN IF NOT EXISTS applied_tags UUID[] DEFAULT '{}';
ALTER TABLE threads ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE threads ADD COLUMN IF NOT EXISTS pin_weight INT NOT NULL DEFAULT 0;

-- Index for filtering threads by applied tags
CREATE INDEX IF NOT EXISTS idx_threads_applied_tags ON threads USING GIN(applied_tags);
-- Index for pinned threads
CREATE INDEX IF NOT EXISTS idx_threads_pinned ON threads(parent_channel_id, is_pinned, pin_weight DESC, archive_timestamp DESC NULLS LAST);

-- Forum channel-specific settings stored in channel metadata (JSONB)
-- Settings: default_reaction_emoji, default_sort_order, default_auto_archive, require_tag, default_layout, post_guidelines

-- Add forum metadata to channels table
ALTER TABLE channels ADD COLUMN IF NOT EXISTS forum_metadata JSONB DEFAULT NULL;

-- +migrate Down

-- Drop indexes first
DROP INDEX IF EXISTS idx_forum_tags_channel;
DROP INDEX IF EXISTS idx_forum_tags_server;
DROP INDEX IF EXISTS idx_threads_applied_tags;
DROP INDEX IF EXISTS idx_threads_pinned;

-- Drop forum_tags table
DROP TABLE IF EXISTS forum_tags;

-- Remove columns from threads
ALTER TABLE threads DROP COLUMN IF EXISTS applied_tags;
ALTER TABLE threads DROP COLUMN IF EXISTS is_pinned;
ALTER TABLE threads DROP COLUMN IF EXISTS pin_weight;

-- Remove forum metadata
ALTER TABLE channels DROP COLUMN IF EXISTS forum_metadata;
