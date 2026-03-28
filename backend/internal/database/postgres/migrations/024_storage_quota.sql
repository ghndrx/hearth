-- Migration 024: Storage quota tracking tables
-- Tracks user and server storage usage for quota enforcement

-- User storage totals (aggregated across all servers)
CREATE TABLE IF NOT EXISTS user_storage_totals (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_bytes BIGINT NOT NULL DEFAULT 0,
    file_count INTEGER NOT NULL DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_storage_totals_user ON user_storage_totals(user_id);

-- Per-server storage usage (for server-specific quota tracking)
CREATE TABLE IF NOT EXISTS storage_usage (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    used_bytes BIGINT NOT NULL DEFAULT 0,
    file_count INTEGER NOT NULL DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, server_id)
);

CREATE INDEX IF NOT EXISTS idx_storage_usage_user ON storage_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_storage_usage_server ON storage_usage(server_id);

-- Add path column to attachments for storage tracking
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS path VARCHAR(512);

COMMENT ON TABLE user_storage_totals IS 'Tracks total storage usage per user across all servers';
COMMENT ON TABLE storage_usage IS 'Tracks storage usage per user per server for quota enforcement';
COMMENT ON COLUMN attachments.path IS 'Storage path for the file (used for deletion from storage backend)';
