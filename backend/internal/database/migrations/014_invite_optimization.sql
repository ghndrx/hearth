-- Invite optimization: vanity URLs, analytics tracking, one-time invites

-- Add vanity and one-time flags to invites table
ALTER TABLE invites ADD COLUMN IF NOT EXISTS is_vanity BOOLEAN DEFAULT FALSE;
ALTER TABLE invites ADD COLUMN IF NOT EXISTS vanity_code VARCHAR(32);

-- Unique constraint: only one vanity code per server, and vanity codes must be globally unique
CREATE UNIQUE INDEX IF NOT EXISTS idx_invites_vanity_code ON invites(vanity_code) WHERE vanity_code IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_invites_server_vanity ON invites(server_id) WHERE is_vanity = TRUE;

-- Invite use logs for analytics
CREATE TABLE IF NOT EXISTS invite_use_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invite_code VARCHAR(16) NOT NULL,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    account_created_at TIMESTAMPTZ,
    account_age_days INT GENERATED ALWAYS AS (
        EXTRACT(DAY FROM (joined_at - account_created_at))
    ) STORED
);

CREATE INDEX IF NOT EXISTS idx_invite_use_logs_code ON invite_use_logs(invite_code);
CREATE INDEX IF NOT EXISTS idx_invite_use_logs_server ON invite_use_logs(server_id);
CREATE INDEX IF NOT EXISTS idx_invite_use_logs_joined ON invite_use_logs(joined_at);
