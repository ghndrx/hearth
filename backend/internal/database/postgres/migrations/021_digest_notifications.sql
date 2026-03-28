-- 021_digest_notifications.sql
-- Digest notifications: batch multiple messages into periodic summaries

-- Digest preferences per user (global defaults)
CREATE TABLE IF NOT EXISTS digest_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Global toggle for digest feature
    enabled BOOLEAN NOT NULL DEFAULT false,
    
    -- Frequency: 'hourly', 'daily', 'weekly'
    frequency VARCHAR(20) NOT NULL DEFAULT 'daily',
    
    -- Preferred delivery time (hour in UTC, 0-23)
    preferred_hour INTEGER NOT NULL DEFAULT 9 CHECK (preferred_hour >= 0 AND preferred_hour <= 23),
    
    -- For weekly digests: day of week (0=Sunday, 6=Saturday)
    preferred_day INTEGER NOT NULL DEFAULT 1 CHECK (preferred_day >= 0 AND preferred_day <= 6),
    
    -- Aggregation mode: 'channel' (per-channel summaries) or 'server' (per-server summaries)
    aggregation_mode VARCHAR(20) NOT NULL DEFAULT 'server',
    
    -- Maximum messages to include per channel/server
    max_messages_per_source INTEGER NOT NULL DEFAULT 50 CHECK (max_messages_per_source > 0 AND max_messages_per_source <= 200),
    
    -- Include muted channels only (true) or all channels (false when muted)
    muted_channels_only BOOLEAN NOT NULL DEFAULT true,
    
    -- Timezone for delivery time calculation (IANA timezone)
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id)
);

-- Channel-specific digest overrides
CREATE TABLE IF NOT EXISTS digest_channel_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    
    -- Override: 'inherit' (use global), 'include' (always digest), 'exclude' (never digest), 'immediate' (never batch)
    digest_mode VARCHAR(20) NOT NULL DEFAULT 'inherit',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id, channel_id)
);

-- Server-specific digest overrides
CREATE TABLE IF NOT EXISTS digest_server_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    
    -- Override: 'inherit' (use global), 'include' (always digest), 'exclude' (never digest)
    digest_mode VARCHAR(20) NOT NULL DEFAULT 'inherit',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id, server_id)
);

-- Queue of messages pending for digest (buffered notifications)
CREATE TABLE IF NOT EXISTS digest_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Source references
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    message_id UUID, -- Not foreign key - messages may be deleted
    
    -- Message metadata for digest generation
    message_content TEXT NOT NULL,
    message_author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    message_author_name VARCHAR(100) NOT NULL,
    message_created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Whether this is a mention of the user
    is_mention BOOLEAN NOT NULL DEFAULT false,
    
    -- Notification type (mention, reply, direct_message, etc.)
    notification_type VARCHAR(50) NOT NULL,
    
    -- When this was queued
    queued_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- For which digest period this belongs to
    -- Allows grouping by period when generating digests
    digest_period TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Digest history - tracking sent digests
CREATE TABLE IF NOT EXISTS digest_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- When the digest was sent
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Period this digest covers
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Frequency at the time of generation
    frequency VARCHAR(20) NOT NULL,
    
    -- Summary stats
    total_messages INTEGER NOT NULL DEFAULT 0,
    total_mentions INTEGER NOT NULL DEFAULT 0,
    servers_included INTEGER NOT NULL DEFAULT 0,
    channels_included INTEGER NOT NULL DEFAULT 0,
    
    -- Digest content (JSON)
    content_json TEXT NOT NULL,
    
    -- Delivery status: 'pending', 'sent', 'failed', 'skipped'
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    -- Error message if failed
    error_message TEXT,
    
    -- Retry count for failed digests
    retry_count INTEGER NOT NULL DEFAULT 0
);

-- Indexes for digest_preferences
CREATE INDEX idx_digest_preferences_user_id ON digest_preferences(user_id);
CREATE INDEX idx_digest_preferences_enabled ON digest_preferences(enabled) WHERE enabled = true;
CREATE INDEX idx_digest_preferences_frequency ON digest_preferences(frequency);

-- Indexes for digest_channel_preferences
CREATE INDEX idx_digest_channel_prefs_user ON digest_channel_preferences(user_id);
CREATE INDEX idx_digest_channel_prefs_channel ON digest_channel_preferences(channel_id);

-- Indexes for digest_server_preferences
CREATE INDEX idx_digest_server_prefs_user ON digest_server_preferences(user_id);
CREATE INDEX idx_digest_server_prefs_server ON digest_server_preferences(server_id);

-- Indexes for digest_queue
CREATE INDEX idx_digest_queue_user_id ON digest_queue(user_id);
CREATE INDEX idx_digest_queue_period ON digest_queue(digest_period);
CREATE INDEX idx_digest_queue_user_period ON digest_queue(user_id, digest_period);
CREATE INDEX idx_digest_queue_queued_at ON digest_queue(queued_at);
CREATE INDEX idx_digest_queue_channel ON digest_queue(channel_id);
CREATE INDEX idx_digest_queue_server ON digest_queue(server_id);

-- Indexes for digest_history
CREATE INDEX idx_digest_history_user_id ON digest_history(user_id);
CREATE INDEX idx_digest_history_sent_at ON digest_history(sent_at DESC);
CREATE INDEX idx_digest_history_status ON digest_history(status);
CREATE INDEX idx_digest_history_user_sent ON digest_history(user_id, sent_at DESC);

-- Constraints
ALTER TABLE digest_preferences ADD CONSTRAINT chk_digest_frequency 
    CHECK (frequency IN ('hourly', 'daily', 'weekly'));

ALTER TABLE digest_preferences ADD CONSTRAINT chk_digest_aggregation_mode 
    CHECK (aggregation_mode IN ('channel', 'server'));

ALTER TABLE digest_channel_preferences ADD CONSTRAINT chk_channel_digest_mode 
    CHECK (digest_mode IN ('inherit', 'include', 'exclude', 'immediate'));

ALTER TABLE digest_server_preferences ADD CONSTRAINT chk_server_digest_mode 
    CHECK (digest_mode IN ('inherit', 'include', 'exclude'));

ALTER TABLE digest_history ADD CONSTRAINT chk_digest_status 
    CHECK (status IN ('pending', 'sent', 'failed', 'skipped'));

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_digest_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER trigger_digest_preferences_updated_at
    BEFORE UPDATE ON digest_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_digest_updated_at();

CREATE TRIGGER trigger_digest_channel_preferences_updated_at
    BEFORE UPDATE ON digest_channel_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_digest_updated_at();

CREATE TRIGGER trigger_digest_server_preferences_updated_at
    BEFORE UPDATE ON digest_server_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_digest_updated_at();

-- Add comments
COMMENT ON TABLE digest_preferences IS 'User global preferences for digest notifications';
COMMENT ON TABLE digest_channel_preferences IS 'Per-channel digest preference overrides';
COMMENT ON TABLE digest_server_preferences IS 'Per-server digest preference overrides';
COMMENT ON TABLE digest_queue IS 'Queue of messages pending inclusion in next digest';
COMMENT ON TABLE digest_history IS 'History of sent digests for tracking and retry';
