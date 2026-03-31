-- Hearth Database Schema
-- Migration 034: Thread Auto-Archive Settings

-- Server-level auto-archive settings
CREATE TABLE thread_auto_archive_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    default_duration INTEGER NOT NULL DEFAULT 1440, -- minutes (default 24 hours)
    allow_override BOOLEAN NOT NULL DEFAULT TRUE, -- allow channel-level override
    archive_duration_options INTEGER[] NOT NULL DEFAULT ARRAY[60, 1440, 4320, 10080], -- available options in minutes
    require_post_author BOOL NOT NULL DEFAULT FALSE, -- require message from post author to bump auto-archive
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(server_id)
);

-- Index for fast lookups
CREATE INDEX idx_thread_auto_archive_settings_server ON thread_auto_archive_settings(server_id);

-- Channel-level auto-archive override settings
CREATE TABLE channel_auto_archive_override (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    auto_archive_duration INTEGER NOT NULL, -- minutes (overrides server default)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(channel_id)
);

-- Index for fast lookups
CREATE INDEX idx_channel_auto_archive_override_channel ON channel_auto_archive_override(channel_id);

-- Thread auto-archive metadata (tracks last activity and archive eligibility)
CREATE TABLE thread_auto_archive_meta (
    thread_id UUID PRIMARY KEY REFERENCES threads(id) ON DELETE CASCADE,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    last_activity_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    next_archive_at TIMESTAMP WITH TIME ZONE,
    archive_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    bumped_by_owner BOOLEAN NOT NULL DEFAULT FALSE, -- true if owner posted since last archive check
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for worker to find threads ready for archiving
CREATE INDEX idx_thread_auto_archive_meta_next_archive ON thread_auto_archive_meta(next_archive_at) WHERE next_archive_at IS NOT NULL AND archive_eligible = TRUE;

-- Index for finding threads by activity
CREATE INDEX idx_thread_auto_archive_meta_activity ON thread_auto_archive_meta(last_activity_at);

-- Function to update thread last activity
CREATE OR REPLACE FUNCTION update_thread_auto_archive_meta()
RETURNS TRIGGER AS $$
BEGIN
    -- Insert or update the auto archive metadata
    INSERT INTO thread_auto_archive_meta (thread_id, last_activity_at, last_activity_message_id, last_activity_user_id, updated_at)
    VALUES (NEW.thread_id, NOW(), NEW.id, NEW.author_id, NOW())
    ON CONFLICT (thread_id) DO UPDATE SET
        last_activity_at = NOW(),
        last_activity_message_id = NEW.id,
        last_activity_user_id = NEW.author_id,
        updated_at = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update thread metadata on new message
DROP TRIGGER IF EXISTS trigger_thread_message_auto_archive ON thread_messages;
CREATE TRIGGER trigger_thread_message_auto_archive
    AFTER INSERT ON thread_messages
    FOR EACH ROW
    EXECUTE FUNCTION update_thread_auto_archive_meta();
