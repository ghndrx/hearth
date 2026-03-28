-- OAuth Providers table for tracking linked OAuth accounts
-- Migration: 016_oauth_providers.sql

-- Table to store OAuth provider links
CREATE TABLE IF NOT EXISTS oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL, -- 'github', 'google', 'discord'
    provider_user_id VARCHAR(255) NOT NULL, -- The user's ID from the OAuth provider
    email VARCHAR(255) NOT NULL,
    username VARCHAR(255),
    display_name VARCHAR(255),
    avatar_url TEXT,
    access_token TEXT, -- Encrypted access token (for future API calls)
    refresh_token TEXT, -- Encrypted refresh token
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Each provider can only be linked once per user
    CONSTRAINT oauth_providers_user_provider_unique UNIQUE (user_id, provider),
    -- Each provider account can only be linked to one user
    CONSTRAINT oauth_providers_provider_id_unique UNIQUE (provider, provider_user_id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_oauth_providers_user_id ON oauth_providers(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_providers_provider ON oauth_providers(provider);
CREATE INDEX IF NOT EXISTS idx_oauth_providers_provider_user ON oauth_providers(provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_providers_email ON oauth_providers(email);

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_oauth_providers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER oauth_providers_updated_at_trigger
    BEFORE UPDATE ON oauth_providers
    FOR EACH ROW
    EXECUTE FUNCTION update_oauth_providers_updated_at();

-- Add comment for documentation
COMMENT ON TABLE oauth_providers IS 'Stores linked OAuth provider accounts (GitHub, Google, Discord, etc.)';
COMMENT ON COLUMN oauth_providers.provider IS 'OAuth provider name: github, google, discord';
COMMENT ON COLUMN oauth_providers.provider_user_id IS 'User ID from the OAuth provider (unique per provider)';
COMMENT ON COLUMN oauth_providers.access_token IS 'Encrypted OAuth access token for API calls';
COMMENT ON COLUMN oauth_providers.refresh_token IS 'Encrypted OAuth refresh token for token renewal';
