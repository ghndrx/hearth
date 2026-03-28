-- E2EE Key Infrastructure
-- Implements Signal Protocol X3DH key storage

-- Device keys table (identity keys per device)
CREATE TABLE IF NOT EXISTS device_keys (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(64) NOT NULL,
    identity_key BYTEA NOT NULL,  -- Public identity key (33 bytes for P-256, 32 for Curve25519)
    registration_id INTEGER NOT NULL CHECK (registration_id >= 0 AND registration_id < 16384),
    device_name VARCHAR(64),
    device_type VARCHAR(20) DEFAULT 'unknown' CHECK (device_type IN ('web', 'desktop', 'mobile_ios', 'mobile_android', 'unknown')),
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, device_id)
);

-- Index for looking up devices by user
CREATE INDEX IF NOT EXISTS idx_device_keys_user_id ON device_keys(user_id);

-- Signed prekeys table (rotated periodically, typically weekly)
CREATE TABLE IF NOT EXISTS signed_prekeys (
    id UUID PRIMARY KEY,
    device_keys_id UUID NOT NULL REFERENCES device_keys(id) ON DELETE CASCADE,
    key_id INTEGER NOT NULL CHECK (key_id >= 0),
    public_key BYTEA NOT NULL CHECK (LENGTH(public_key) >= 32 AND LENGTH(public_key) <= 65),
    signature BYTEA NOT NULL CHECK (LENGTH(signature) >= 64 AND LENGTH(signature) <= 128),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(device_keys_id, key_id)
);

-- Index for looking up signed prekeys by device
CREATE INDEX IF NOT EXISTS idx_signed_prekeys_device ON signed_prekeys(device_keys_id);

-- One-time prekeys table (single use, claimed during session establishment)
CREATE TABLE IF NOT EXISTS one_time_prekeys (
    id UUID PRIMARY KEY,
    device_keys_id UUID NOT NULL REFERENCES device_keys(id) ON DELETE CASCADE,
    key_id INTEGER NOT NULL CHECK (key_id >= 0),
    public_key BYTEA NOT NULL CHECK (LENGTH(public_key) >= 32 AND LENGTH(public_key) <= 65),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    claimed_at TIMESTAMP WITH TIME ZONE,
    claimed_by_user_id UUID REFERENCES users(id),
    UNIQUE(device_keys_id, key_id)
);

-- Index for finding unclaimed prekeys
CREATE INDEX IF NOT EXISTS idx_one_time_prekeys_device_unclaimed 
    ON one_time_prekeys(device_keys_id) 
    WHERE claimed_at IS NULL;

-- Index for cleanup of old claimed prekeys
CREATE INDEX IF NOT EXISTS idx_one_time_prekeys_claimed_at 
    ON one_time_prekeys(claimed_at) 
    WHERE claimed_at IS NOT NULL;

-- Function to atomically claim a one-time prekey
-- Returns the claimed key_id and public_key, or NULL if none available
CREATE OR REPLACE FUNCTION claim_one_time_prekey(
    p_device_keys_id UUID,
    p_claiming_user_id UUID
)
RETURNS TABLE (
    key_id INTEGER,
    public_key BYTEA
) AS $$
DECLARE
    v_prekey_id UUID;
    v_key_id INTEGER;
    v_public_key BYTEA;
BEGIN
    -- Select and lock an unclaimed prekey
    SELECT 
        otp.id, otp.key_id, otp.public_key 
    INTO 
        v_prekey_id, v_key_id, v_public_key
    FROM one_time_prekeys otp
    WHERE otp.device_keys_id = p_device_keys_id
      AND otp.claimed_at IS NULL
    ORDER BY otp.key_id
    LIMIT 1
    FOR UPDATE SKIP LOCKED;
    
    -- If no prekey available, return empty result
    IF v_prekey_id IS NULL THEN
        RETURN;
    END IF;
    
    -- Mark as claimed
    UPDATE one_time_prekeys
    SET 
        claimed_at = NOW(),
        claimed_by_user_id = p_claiming_user_id
    WHERE id = v_prekey_id;
    
    -- Return the claimed key
    key_id := v_key_id;
    public_key := v_public_key;
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

-- Add comment explaining the tables
COMMENT ON TABLE device_keys IS 'E2EE device identity keys - one per device per user';
COMMENT ON TABLE signed_prekeys IS 'Signed prekeys for X3DH key agreement - rotated periodically';
COMMENT ON TABLE one_time_prekeys IS 'One-time prekeys for forward secrecy - consumed on first use';
COMMENT ON FUNCTION claim_one_time_prekey IS 'Atomically claim a one-time prekey for session establishment';
