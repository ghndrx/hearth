-- Stage instances: active stage sessions in stage channels
CREATE TABLE IF NOT EXISTS stage_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    topic VARCHAR(120) NOT NULL DEFAULT '',
    privacy_level INT NOT NULL DEFAULT 1, -- 1=public, 2=guild_only
    started_by UUID NOT NULL REFERENCES users(id),
    speaker_count INT NOT NULL DEFAULT 0,
    audience_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(channel_id) -- only one active stage per channel (enforced by app logic with ended_at IS NULL)
);

-- Stage participants: tracks speakers and audience members
CREATE TABLE IF NOT EXISTS stage_participants (
    stage_id UUID NOT NULL REFERENCES stage_instances(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'audience', -- 'speaker' or 'audience'
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (stage_id, user_id)
);

-- Drop the unique constraint on channel_id so multiple ended stages can exist
ALTER TABLE stage_instances DROP CONSTRAINT IF EXISTS stage_instances_channel_id_key;

-- Index for looking up active stage by channel
CREATE INDEX IF NOT EXISTS idx_stage_instances_channel_active ON stage_instances(channel_id) WHERE ended_at IS NULL;

-- Index for looking up stages by server
CREATE INDEX IF NOT EXISTS idx_stage_instances_server ON stage_instances(server_id);

-- Index for looking up participants by user
CREATE INDEX IF NOT EXISTS idx_stage_participants_user ON stage_participants(user_id);
