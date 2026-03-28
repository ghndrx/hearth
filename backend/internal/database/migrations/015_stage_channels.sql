-- Migration: Stage Channels
-- Description: Add tables for stage channels - structured events with speakers and audience

-- Create stages table
CREATE TABLE IF NOT EXISTS stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL UNIQUE REFERENCES channels(id) ON DELETE CASCADE,
    topic VARCHAR(128) NOT NULL DEFAULT '',
    description TEXT,
    status INT NOT NULL DEFAULT 1 CHECK (status IN (1, 2, 3, 4)), -- 1=scheduled, 2=live, 3=paused, 4=ended
    host_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    discovery_disabled BOOLEAN NOT NULL DEFAULT FALSE,
    request_to_speak BOOLEAN NOT NULL DEFAULT TRUE,
    moderator_only BOOLEAN NOT NULL DEFAULT FALSE,
    max_speakers INT NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create stage_participants table
CREATE TABLE IF NOT EXISTS stage_participants (
    stage_id UUID NOT NULL REFERENCES stages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role INT NOT NULL DEFAULT 1 CHECK (role IN (1, 2, 3, 4)), -- 1=audience, 2=speaker, 3=moderator, 4=host
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_muted BOOLEAN NOT NULL DEFAULT TRUE,
    is_deafened BOOLEAN NOT NULL DEFAULT FALSE,
    requested_at TIMESTAMPTZ, -- For pending speaker requests
    approved_at TIMESTAMPTZ,
    PRIMARY KEY (stage_id, user_id)
);

-- Create indexes for stages
CREATE INDEX IF NOT EXISTS idx_stages_channel_id ON stages(channel_id);
CREATE INDEX IF NOT EXISTS idx_stages_host_user_id ON stages(host_user_id);
CREATE INDEX IF NOT EXISTS idx_stages_status ON stages(status);
CREATE INDEX IF NOT EXISTS idx_stages_active ON stages(channel_id, status) WHERE status IN (1, 2, 3);

-- Create indexes for stage_participants
CREATE INDEX IF NOT EXISTS idx_stage_participants_stage_id ON stage_participants(stage_id);
CREATE INDEX IF NOT EXISTS idx_stage_participants_user_id ON stage_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_stage_participants_role ON stage_participants(role);
CREATE INDEX IF NOT EXISTS idx_stage_participants_pending_requests ON stage_participants(stage_id, requested_at) WHERE requested_at IS NOT NULL AND approved_at IS NULL;

-- Add comments
COMMENT ON TABLE stages IS 'Stage channel sessions with speakers and audience';
COMMENT ON TABLE stage_participants IS 'Users participating in a stage channel';
COMMENT ON COLUMN stages.status IS '1=scheduled, 2=live, 3=paused, 4=ended';
COMMENT ON COLUMN stages.request_to_speak IS 'Whether audience can request to become speakers';
COMMENT ON COLUMN stage_participants.role IS '1=audience, 2=speaker, 3=moderator, 4=host';
COMMENT ON COLUMN stage_participants.requested_at IS 'Timestamp when user requested to speak (null if not pending)';
COMMENT ON COLUMN stage_participants.approved_at IS 'Timestamp when speaker request was approved (null if pending or not requested)';

-- Auto-update trigger for updated_at on stages
CREATE OR REPLACE TRIGGER update_stages_updated_at
    BEFORE UPDATE ON stages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
