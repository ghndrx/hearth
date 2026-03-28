-- Server Audio Settings: per-server audio device and volume preferences per user
CREATE TABLE IF NOT EXISTS server_audio_settings (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    input_device_id TEXT NOT NULL DEFAULT '',
    output_device_id TEXT NOT NULL DEFAULT '',
    input_volume INTEGER NOT NULL DEFAULT 100 CHECK (input_volume >= 0 AND input_volume <= 100),
    output_volume INTEGER NOT NULL DEFAULT 100 CHECK (output_volume >= 0 AND output_volume <= 100),
    push_to_talk_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    push_to_talk_key TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, server_id)
);

CREATE INDEX IF NOT EXISTS idx_server_audio_settings_user ON server_audio_settings(user_id);
CREATE INDEX IF NOT EXISTS idx_server_audio_settings_server ON server_audio_settings(server_id);
