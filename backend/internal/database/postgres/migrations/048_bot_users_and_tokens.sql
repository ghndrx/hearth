-- Bot API Phase 1: Bot users and token authentication
-- Add is_bot flag to users table and create bot_tokens table

-- Add is_bot column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_bot BOOLEAN NOT NULL DEFAULT false;

-- Create index for bot user lookups
CREATE INDEX IF NOT EXISTS idx_users_is_bot ON users(is_bot) WHERE is_bot = true;

-- Bot tokens table for bot authentication
CREATE TABLE IF NOT EXISTS bot_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL, -- Hashed token (bcrypt)
    token_prefix VARCHAR(10) NOT NULL DEFAULT 'hearth_', -- Token prefix for identification
    name VARCHAR(100), -- Optional name for the token (e.g., "Production", "Development")
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ, -- NULL means never expires
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ, -- NULL means active
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_whitelist TEXT[], -- Optional IP whitelist
    scopes TEXT[] DEFAULT '{}', -- Token scopes/permissions
    metadata JSONB DEFAULT '{}'::jsonb -- Additional metadata
);

-- Indexes for bot token lookups
CREATE INDEX IF NOT EXISTS idx_bot_tokens_user_id ON bot_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_bot_tokens_token_hash ON bot_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_bot_tokens_active ON bot_tokens(user_id) WHERE revoked_at IS NULL;

-- Application bot association
-- Links applications to their bot user accounts
CREATE TABLE IF NOT EXISTS application_bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    bot_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key TEXT, -- For verifying request signatures (Ed25519)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(application_id),
    UNIQUE(bot_user_id)
);

CREATE INDEX IF NOT EXISTS idx_application_bots_app_id ON application_bots(application_id);
CREATE INDEX IF NOT EXISTS idx_application_bots_bot_user_id ON application_bots(bot_user_id);
