-- Slash commands (application commands) table
CREATE TABLE IF NOT EXISTS slash_commands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type SMALLINT NOT NULL DEFAULT 1,
    app_id UUID NOT NULL REFERENCES oauth_apps(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(32) NOT NULL,
    description VARCHAR(100) NOT NULL,
    options JSONB,
    permissions JSONB,
    version VARCHAR(32) NOT NULL DEFAULT gen_random_uuid()::text,
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    default_permission BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Commands must be unique per app (and per server for guild-specific commands)
    CONSTRAINT unique_global_command UNIQUE (app_id, name, server_id)
        -- only enforce when server_id IS NULL for global commands
);

-- Unique constraint: global commands must be unique by name per app
CREATE UNIQUE INDEX IF NOT EXISTS idx_slash_commands_global
    ON slash_commands(app_id, name) WHERE server_id IS NULL;

-- Unique constraint: guild commands must be unique by name per app+server
CREATE UNIQUE INDEX IF NOT EXISTS idx_slash_commands_guild
    ON slash_commands(app_id, server_id, name) WHERE server_id IS NOT NULL;

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_slash_commands_app_id ON slash_commands(app_id);
CREATE INDEX IF NOT EXISTS idx_slash_commands_server_id ON slash_commands(server_id);
CREATE INDEX IF NOT EXISTS idx_slash_commands_name ON slash_commands(name);

-- Command execution logs for analytics
CREATE TABLE IF NOT EXISTS command_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id UUID NOT NULL REFERENCES slash_commands(id) ON DELETE CASCADE,
    app_id UUID NOT NULL,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    channel_id UUID NOT NULL,
    user_id UUID NOT NULL,
    options JSONB,
    response_time_ms BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for analytics queries
CREATE INDEX IF NOT EXISTS idx_command_executions_command_id ON command_executions(command_id);
CREATE INDEX IF NOT EXISTS idx_command_executions_app_id ON command_executions(app_id);
CREATE INDEX IF NOT EXISTS idx_command_executions_server_id ON command_executions(server_id);
CREATE INDEX IF NOT EXISTS idx_command_executions_user_id ON command_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_command_executions_created_at ON command_executions(created_at);
CREATE INDEX IF NOT EXISTS idx_command_executions_status ON command_executions(status);

-- Interaction tokens for ephemeral responses
CREATE TABLE IF NOT EXISTS interaction_tokens (
    token VARCHAR(64) PRIMARY KEY,
    interaction_id UUID NOT NULL,
    app_id UUID NOT NULL,
    user_id UUID NOT NULL,
    server_id UUID,
    channel_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_interaction_tokens_app_id ON interaction_tokens(app_id);
CREATE INDEX IF NOT EXISTS idx_interaction_tokens_expires_at ON interaction_tokens(expires_at);
