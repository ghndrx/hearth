-- +migrate Up

-- Allow OAuth-only users (no password required)
-- Users who sign up via OAuth won't have a password initially
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- OAuth Providers table for storing linked OAuth accounts
-- Supports GitHub, Google, Discord (extensible for future providers)
CREATE TABLE oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL, -- github, google, discord
    provider_user_id VARCHAR(255) NOT NULL, -- The user's ID from the OAuth provider
    email VARCHAR(255) NOT NULL, -- Email from OAuth provider (may differ from user's primary email)
    username VARCHAR(100), -- Username from OAuth provider
    display_name VARCHAR(100), -- Display name from OAuth provider
    avatar_url TEXT, -- Avatar URL from OAuth provider
    access_token TEXT, -- Encrypted OAuth access token (for future refresh scenarios)
    refresh_token TEXT, -- Encrypted OAuth refresh token
    token_expires_at TIMESTAMPTZ, -- When the access token expires
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure one user can only link one account per provider
    UNIQUE(user_id, provider),
    
    -- Ensure each OAuth account can only be linked to one user
    UNIQUE(provider, provider_user_id)
);

-- Index for looking up user's linked providers
CREATE INDEX idx_oauth_providers_user ON oauth_providers(user_id);

-- Index for finding user by OAuth provider account (login flow)
CREATE INDEX idx_oauth_providers_lookup ON oauth_providers(provider, provider_user_id);

-- Add updated_at trigger
CREATE TRIGGER update_oauth_providers_updated_at 
    BEFORE UPDATE ON oauth_providers
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- +migrate Down
DROP TRIGGER IF EXISTS update_oauth_providers_updated_at ON oauth_providers;
DROP TABLE IF EXISTS oauth_providers;

-- Note: Restoring NOT NULL on password_hash requires all users to have passwords
-- This should only be done if all OAuth-only users have been migrated
-- ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
