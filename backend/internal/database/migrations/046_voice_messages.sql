-- Voice messages table for storing voice message recordings
CREATE TABLE IF NOT EXISTS voice_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_url TEXT NOT NULL,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    waveform_data JSONB DEFAULT '[]'::jsonb,
    transcription TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for fetching voice messages by channel
CREATE INDEX IF NOT EXISTS idx_voice_messages_channel_id ON voice_messages(channel_id);
CREATE INDEX IF NOT EXISTS idx_voice_messages_user_id ON voice_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_voice_messages_created_at ON voice_messages(created_at);

-- Comment for documentation
COMMENT ON TABLE voice_messages IS 'Stores voice message recordings with waveform data for visualization';
