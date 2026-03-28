-- Migration: 017_oauth_apps.sql
-- Description: OAuth 2.0 Provider - Third-party app authorization

-- OAuth Applications (third-party apps that use Hearth as OAuth provider)
CREATE TABLE IF NOT EXISTS oauth_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    client_id VARCHAR(64) UNIQUE NOT NULL,
    client_secret_hash VARCHAR(255) NOT NULL, -- bcrypt hash
    redirect_uris TEXT[] NOT NULL DEFAULT '{}',
    scopes TEXT[] NOT NULL DEFAULT '{}',
    icon_url TEXT,
    homepage_url TEXT,
    privacy_url TEXT,
    terms_url TEXT,
    is_public BOOLEAN NOT NULL DEFAULT false, -- Public clients use PKCE only
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_apps_owner_id ON oauth_apps(owner_id);
CREATE INDEX IF NOT EXISTS idx_oauth_apps_client_id ON oauth_apps(client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_apps_is_active ON oauth_apps(is_active) WHERE is_active = true;

-- OAuth Authorization Codes (short-lived, exchanged for tokens)
CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(128) UNIQUE NOT NULL, -- SHA256 hash of actual code
    client_id VARCHAR(64) NOT NULL REFERENCES oauth_apps(client_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    redirect_uri TEXT NOT NULL,
    code_challenge VARCHAR(128), -- PKCE
    code_challenge_method VARCHAR(10), -- 'plain' or 'S256'
    nonce VARCHAR(128), -- OpenID Connect
    state VARCHAR(255),
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_auth_codes_code ON oauth_authorization_codes(code);
CREATE INDEX IF NOT EXISTS idx_oauth_auth_codes_client ON oauth_authorization_codes(client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_auth_codes_user ON oauth_authorization_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_auth_codes_expires ON oauth_authorization_codes(expires_at);

-- OAuth Access Tokens
CREATE TABLE IF NOT EXISTS oauth_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash VARCHAR(64) UNIQUE NOT NULL, -- SHA256 hash
    client_id VARCHAR(64) NOT NULL REFERENCES oauth_apps(client_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_access_tokens_hash ON oauth_access_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_access_tokens_client ON oauth_access_tokens(client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_access_tokens_user ON oauth_access_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_access_tokens_expires ON oauth_access_tokens(expires_at);

-- OAuth Refresh Tokens (with rotation tracking for reuse detection)
CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash VARCHAR(64) UNIQUE NOT NULL, -- SHA256 hash
    access_token_id UUID NOT NULL REFERENCES oauth_access_tokens(id) ON DELETE CASCADE,
    client_id VARCHAR(64) NOT NULL REFERENCES oauth_apps(client_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ NOT NULL,
    rotated_at TIMESTAMPTZ, -- When this token was rotated
    rotated_to_id UUID REFERENCES oauth_refresh_tokens(id), -- The new refresh token
    revoked_at TIMESTAMPTZ,
    revoked_reason VARCHAR(100), -- 'user_revoked', 'reuse_detected', 'expired', 'logout'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_hash ON oauth_refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_client ON oauth_refresh_tokens(client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_user ON oauth_refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_expires ON oauth_refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_access ON oauth_refresh_tokens(access_token_id);

-- OAuth User Authorizations (tracks which apps user has authorized)
CREATE TABLE IF NOT EXISTS oauth_user_authorizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id VARCHAR(64) NOT NULL REFERENCES oauth_apps(client_id) ON DELETE CASCADE,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    authorized_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    CONSTRAINT oauth_user_auth_unique UNIQUE (user_id, client_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth_user_auths_user ON oauth_user_authorizations(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_user_auths_client ON oauth_user_authorizations(client_id);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_oauth_apps_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER oauth_apps_updated_at_trigger
    BEFORE UPDATE ON oauth_apps
    FOR EACH ROW
    EXECUTE FUNCTION update_oauth_apps_updated_at();

-- Comments for documentation
COMMENT ON TABLE oauth_apps IS 'Third-party OAuth applications registered with Hearth as the OAuth provider';
COMMENT ON COLUMN oauth_apps.client_id IS 'Public identifier for the application';
COMMENT ON COLUMN oauth_apps.client_secret_hash IS 'bcrypt hashed client secret (confidential clients only)';
COMMENT ON COLUMN oauth_apps.is_public IS 'Public clients (mobile/SPA) do not have client secrets and must use PKCE';
COMMENT ON COLUMN oauth_apps.is_verified IS 'Verified apps have been reviewed by platform administrators';

COMMENT ON TABLE oauth_authorization_codes IS 'Short-lived authorization codes (10 min expiry) for OAuth code flow';
COMMENT ON COLUMN oauth_authorization_codes.code_challenge IS 'PKCE code challenge for public clients';
COMMENT ON COLUMN oauth_authorization_codes.code_challenge_method IS 'PKCE method: plain or S256 (SHA256)';

COMMENT ON TABLE oauth_refresh_tokens IS 'Refresh tokens with rotation tracking for reuse detection (SEC-002)';
COMMENT ON COLUMN oauth_refresh_tokens.rotated_to_id IS 'Points to the new refresh token after rotation';
COMMENT ON COLUMN oauth_refresh_tokens.revoked_reason IS 'Why the token was revoked: user_revoked, reuse_detected, expired, logout';
