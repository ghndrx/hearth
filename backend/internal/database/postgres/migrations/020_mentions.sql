-- Migration: Mentions table for tracking individual mentions with read/unread status
-- Up migration

-- Create mentions table
CREATE TABLE IF NOT EXISTS mentions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- who is mentioned
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    mentioned_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- who did the mentioning
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    guild_id UUID REFERENCES servers(id) ON DELETE CASCADE, -- null for DMs
    mention_type VARCHAR(20) NOT NULL CHECK (mention_type IN ('user', 'role', 'channel', 'everyone', 'here')),
    mentioned_role_id UUID REFERENCES roles(id) ON DELETE CASCADE, -- for role mentions (nullable)
    mentioned_channel_id UUID REFERENCES channels(id) ON DELETE CASCADE, -- for channel mentions (nullable)
    read_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_mentions_user_id ON mentions(user_id);
CREATE INDEX IF NOT EXISTS idx_mentions_user_unread ON mentions(user_id, read_at) WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mentions_message_id ON mentions(message_id);
CREATE INDEX IF NOT EXISTS idx_mentions_channel_id ON mentions(channel_id);
CREATE INDEX IF NOT EXISTS idx_mentions_guild_id ON mentions(guild_id) WHERE guild_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_mentions_created_at ON mentions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mentions_mentioned_by ON mentions(mentioned_by);

-- Unique constraint to prevent duplicate mentions per message/user
CREATE UNIQUE INDEX IF NOT EXISTS idx_mentions_unique_user_message 
ON mentions(user_id, message_id, mention_type, COALESCE(mentioned_role_id, '00000000-0000-0000-0000-000000000000'));

-- Down migration (commented out, run manually if needed)
-- DROP TABLE IF EXISTS mentions;
