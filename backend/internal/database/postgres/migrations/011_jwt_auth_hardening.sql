-- Migration 011: JWT Auth Hardening
-- Implements refresh token rotation and enhanced device tracking

-- Refresh tokens table with rotation support
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    family_id UUID NOT NULL,
    session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
    used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMP WITH TIME ZONE,
    revoked BOOLEAN DEFAULT FALSE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient lookups
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE NOT revoked;

-- Enhance sessions table with device info
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS device_name VARCHAR(255);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS device_type VARCHAR(32) DEFAULT 'unknown';
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS browser VARCHAR(100);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS browser_version VARCHAR(50);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS os VARCHAR(100);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS os_version VARCHAR(50);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS is_current BOOLEAN DEFAULT FALSE;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS location_city VARCHAR(100);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS location_country VARCHAR(100);

-- Rename columns to match Go model (if different)
-- The existing columns are: id, user_id, token_hash, device, ip_address, user_agent, last_used, expires_at, created_at

-- Create function to auto-revoke entire family when a used token is presented
CREATE OR REPLACE FUNCTION check_refresh_token_reuse()
RETURNS TRIGGER AS $$
BEGIN
    -- If token was already used, revoke the entire family (theft detected)
    IF OLD.used = TRUE AND NEW.used = TRUE THEN
        UPDATE refresh_tokens 
        SET revoked = TRUE, revoked_at = NOW()
        WHERE family_id = OLD.family_id AND revoked = FALSE;
        
        -- Also revoke the associated session
        UPDATE sessions
        SET expires_at = NOW()
        WHERE id = OLD.session_id;
        
        RAISE NOTICE 'Token family % revoked due to reuse', OLD.family_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for token reuse detection (optional - can be handled in app layer)
-- DROP TRIGGER IF EXISTS refresh_token_reuse_check ON refresh_tokens;
-- CREATE TRIGGER refresh_token_reuse_check
--     BEFORE UPDATE ON refresh_tokens
--     FOR EACH ROW
--     WHEN (OLD.used = TRUE)
--     EXECUTE FUNCTION check_refresh_token_reuse();

-- Cleanup function for expired tokens (to be called periodically)
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM refresh_tokens 
    WHERE expires_at < NOW() - INTERVAL '7 days'
    OR (revoked = TRUE AND revoked_at < NOW() - INTERVAL '7 days');
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    DELETE FROM sessions
    WHERE expires_at < NOW() - INTERVAL '7 days';
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Comment on tables/columns for documentation
COMMENT ON TABLE refresh_tokens IS 'Stores refresh tokens with rotation support for secure authentication';
COMMENT ON COLUMN refresh_tokens.family_id IS 'Groups tokens from same login session; all revoked if old token reused';
COMMENT ON COLUMN refresh_tokens.used IS 'Set to TRUE after token is exchanged for new token pair';
COMMENT ON COLUMN sessions.device_type IS 'Device category: desktop, mobile, tablet, unknown';
COMMENT ON COLUMN sessions.is_current IS 'TRUE if this session was used for current request';
