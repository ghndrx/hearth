-- Migration 050: Channel Notification Overrides
-- Adds per-channel notification override settings for users

CREATE TABLE IF NOT EXISTS channel_notification_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    notification_level VARCHAR(20) NOT NULL DEFAULT 'all_messages' CHECK (notification_level IN ('all_messages', 'mentions_only', 'nothing')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, channel_id)
);

-- Index for fast lookups by user
CREATE INDEX IF NOT EXISTS idx_channel_notification_overrides_user_id ON channel_notification_overrides(user_id);

-- Index for fast lookups by channel
CREATE INDEX IF NOT EXISTS idx_channel_notification_overrides_channel_id ON channel_notification_overrides(channel_id);

-- Index for filtering active overrides (not 'all_messages' which is default behavior)
CREATE INDEX IF NOT EXISTS idx_channel_notification_overrides_level ON channel_notification_overrides(notification_level) WHERE notification_level != 'all_messages';
