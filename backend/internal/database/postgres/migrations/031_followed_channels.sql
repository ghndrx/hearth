-- Followed Channels: channel-to-channel follow relationships
CREATE TABLE IF NOT EXISTS followed_channels (
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    follower_channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (channel_id, follower_channel_id)
);

CREATE INDEX IF NOT EXISTS idx_followed_channels_channel ON followed_channels(channel_id);
CREATE INDEX IF NOT EXISTS idx_followed_channels_follower ON followed_channels(follower_channel_id);
