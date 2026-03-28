-- User channel settings for per-user, per-channel preferences (e.g. muting)
CREATE TABLE IF NOT EXISTS user_channel_settings (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    muted      BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, channel_id)
);

CREATE INDEX idx_user_channel_settings_user ON user_channel_settings(user_id);
CREATE INDEX idx_user_channel_settings_channel ON user_channel_settings(channel_id);
CREATE INDEX idx_user_channel_settings_muted ON user_channel_settings(user_id, muted) WHERE muted = true;
