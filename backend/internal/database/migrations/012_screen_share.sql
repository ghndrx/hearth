-- Migration: Screen Share & Application Streaming
-- Description: Add tables for screen share and application streaming functionality

-- Create stream_sessions table
CREATE TABLE IF NOT EXISTS stream_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stream_type INT NOT NULL CHECK (stream_type IN (1, 2)), -- 1=screen, 2=application
    status INT NOT NULL DEFAULT 1 CHECK (status IN (1, 2)), -- 1=active, 2=ended
    resolution VARCHAR(10) DEFAULT '1080p',
    frame_rate INT DEFAULT 30,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    UNIQUE(channel_id) -- Only one active stream per channel at a time
);

-- Create stream_viewers table
CREATE TABLE IF NOT EXISTS stream_viewers (
    session_id UUID NOT NULL REFERENCES stream_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, user_id)
);

-- Create indexes for stream_sessions
CREATE INDEX IF NOT EXISTS idx_stream_sessions_server_id ON stream_sessions(server_id);
CREATE INDEX IF NOT EXISTS idx_stream_sessions_channel_id ON stream_sessions(channel_id);
CREATE INDEX IF NOT EXISTS idx_stream_sessions_user_id ON stream_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_stream_sessions_status ON stream_sessions(status);
CREATE INDEX IF NOT EXISTS idx_stream_sessions_active ON stream_sessions(channel_id, status) WHERE status = 1;

-- Create indexes for stream_viewers
CREATE INDEX IF NOT EXISTS idx_stream_viewers_session_id ON stream_viewers(session_id);
CREATE INDEX IF NOT EXISTS idx_stream_viewers_user_id ON stream_viewers(user_id);

-- Add comment
COMMENT ON TABLE stream_sessions IS 'Active screen share and application stream sessions';
COMMENT ON TABLE stream_viewers IS 'Users viewing a stream session';
