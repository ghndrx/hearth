-- Voice Activities & Games
-- Supports Poker, Chess, Watch Together activities in voice channels

CREATE TABLE IF NOT EXISTS voice_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type TEXT NOT NULL CHECK (activity_type IN ('poker', 'chess', 'watch_together')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'finished', 'cancelled')),
    max_participants INT NOT NULL DEFAULT 10,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS voice_activity_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id UUID NOT NULL REFERENCES voice_activities(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    UNIQUE(activity_id, user_id)
);

CREATE TABLE IF NOT EXISTS voice_activity_game_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id UUID NOT NULL REFERENCES voice_activities(id) ON DELETE CASCADE,
    state JSONB NOT NULL DEFAULT '{}',
    version INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(activity_id)
);

CREATE INDEX IF NOT EXISTS idx_voice_activities_channel ON voice_activities(channel_id) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_voice_activities_server ON voice_activities(server_id) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_voice_activity_participants_activity ON voice_activity_participants(activity_id) WHERE left_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_voice_activity_participants_user ON voice_activity_participants(user_id) WHERE left_at IS NULL;
