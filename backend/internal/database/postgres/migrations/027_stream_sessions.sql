-- Stream Sessions Migration
-- Migration 027: Add stream sessions for Go Live / screen share functionality

-- Create stream_sessions table
CREATE TABLE IF NOT EXISTS stream_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    type INT NOT NULL DEFAULT 1,
    title VARCHAR(100) NOT NULL DEFAULT '',
    status INT NOT NULL DEFAULT 1,
    viewer_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for stream_sessions
CREATE INDEX IF NOT EXISTS idx_stream_sessions_channel ON stream_sessions(channel_id);
CREATE INDEX IF NOT EXISTS idx_stream_sessions_server ON stream_sessions(server_id);
CREATE INDEX IF NOT EXISTS idx_stream_sessions_user ON stream_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_stream_sessions_status ON stream_sessions(status) WHERE status = 1;
