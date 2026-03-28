-- Voice states for tracking users connected to voice channels
CREATE TABLE IF NOT EXISTS voice_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    
    -- Self states (controlled by user)
    self_muted BOOLEAN NOT NULL DEFAULT FALSE,
    self_deafened BOOLEAN NOT NULL DEFAULT FALSE,
    self_video BOOLEAN NOT NULL DEFAULT FALSE,
    self_stream BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Server states (controlled by moderators)
    muted BOOLEAN NOT NULL DEFAULT FALSE,
    deafened BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Session tracking
    session_id VARCHAR(64) NOT NULL,
    connected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Only one voice state per user per server
    UNIQUE(user_id, server_id)
);

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_voice_states_channel ON voice_states(channel_id);
CREATE INDEX IF NOT EXISTS idx_voice_states_server ON voice_states(server_id);
CREATE INDEX IF NOT EXISTS idx_voice_states_user ON voice_states(user_id);
CREATE INDEX IF NOT EXISTS idx_voice_states_session ON voice_states(session_id);
