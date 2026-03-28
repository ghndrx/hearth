-- Applications table for bot ecosystem
-- This is the primary application entity for slash commands
CREATE TABLE IF NOT EXISTS applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    icon TEXT,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for looking up applications by owner
CREATE INDEX IF NOT EXISTS idx_applications_owner_id ON applications(owner_id);
-- Index for verified status
CREATE INDEX IF NOT EXISTS idx_applications_verified ON applications(verified) WHERE verified = true;

-- Command permissions table for guild-specific command permissions
CREATE TABLE IF NOT EXISTS command_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id UUID NOT NULL REFERENCES slash_commands(id) ON DELETE CASCADE,
    guild_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    permissions JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_command_guild UNIQUE (command_id, guild_id)
);

-- Index for looking up permissions by command
CREATE INDEX IF NOT EXISTS idx_command_permissions_command_id ON command_permissions(command_id);
-- Index for looking up permissions by guild
CREATE INDEX IF NOT EXISTS idx_command_permissions_guild_id ON command_permissions(guild_id);
