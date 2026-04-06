-- Bot API Phase 2: Application Commands (Slash Command Registry)
-- Stores registered slash commands for applications

CREATE TABLE IF NOT EXISTS application_commands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name VARCHAR(32) NOT NULL,
    description VARCHAR(100) NOT NULL,
    options JSONB DEFAULT '[]'::jsonb, -- Command options/parameters
    default_member_permissions BIGINT, -- Default permissions required to use command
    dm_permission BOOLEAN NOT NULL DEFAULT true, -- Whether command works in DMs
    type SMALLINT NOT NULL DEFAULT 1, -- 1=Slash, 2=User, 3=Message
    version VARCHAR(32) NOT NULL DEFAULT gen_random_uuid()::text,
    nsfw BOOLEAN NOT NULL DEFAULT false, -- Whether command is age-restricted
    integration_types INTEGER[] DEFAULT '{0,1}', -- 0=Guild install, 1=User install
    contexts INTEGER[] DEFAULT '{0,1,2}', -- Where command works: 0=Guild, 1=Bot DM, 2=Private channel
    
    -- Localized names and descriptions (stored as JSONB)
    name_localizations JSONB,
    description_localizations JSONB,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Unique constraint: command names must be unique per application
    CONSTRAINT unique_app_command_name UNIQUE (application_id, name)
);

-- Indexes for application commands
CREATE INDEX IF NOT EXISTS idx_application_commands_app_id ON application_commands(application_id);
CREATE INDEX IF NOT EXISTS idx_application_commands_name ON application_commands(name);
CREATE INDEX IF NOT EXISTS idx_application_commands_type ON application_commands(type);

-- Command versions table for tracking command updates
CREATE TABLE IF NOT EXISTS command_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id UUID NOT NULL REFERENCES application_commands(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,
    data JSONB NOT NULL, -- Full command data at this version
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_command_versions_command_id ON command_versions(command_id);

-- Guild-specific command overrides
-- Allows customizing commands per guild (server)
CREATE TABLE IF NOT EXISTS guild_command_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id UUID NOT NULL REFERENCES application_commands(id) ON DELETE CASCADE,
    guild_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    permissions BIGINT, -- Override default permissions
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_guild_command_override UNIQUE (command_id, guild_id)
);

CREATE INDEX IF NOT EXISTS idx_guild_command_overrides_command_id ON guild_command_overrides(command_id);
CREATE INDEX IF NOT EXISTS idx_guild_command_overrides_guild_id ON guild_command_overrides(guild_id);

-- Command usage analytics
CREATE TABLE IF NOT EXISTS command_usage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id UUID NOT NULL REFERENCES application_commands(id) ON DELETE CASCADE,
    guild_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE SET NULL,
    options_used JSONB DEFAULT '{}'::jsonb, -- Which options were used
    response_time_ms INTEGER, -- How long the bot took to respond
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_command_usage_stats_command_id ON command_usage_stats(command_id);
CREATE INDEX IF NOT EXISTS idx_command_usage_stats_guild_id ON command_usage_stats(guild_id);
CREATE INDEX IF NOT EXISTS idx_command_usage_stats_created_at ON command_usage_stats(created_at);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to auto-update updated_at on application_commands
DROP TRIGGER IF EXISTS update_application_commands_updated_at ON application_commands;
CREATE TRIGGER update_application_commands_updated_at
    BEFORE UPDATE ON application_commands
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to auto-update updated_at on guild_command_overrides
DROP TRIGGER IF EXISTS update_guild_command_overrides_updated_at ON guild_command_overrides;
CREATE TRIGGER update_guild_command_overrides_updated_at
    BEFORE UPDATE ON guild_command_overrides
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
