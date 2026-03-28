-- Hearth Database Schema
-- Migration 013: End-to-End Encryption Device Keys
-- Implements key storage for Signal Protocol E2EE

-- Device keys table - stores identity keys for each user device
CREATE TABLE device_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,  -- Unique identifier for this device (generated client-side)
    identity_key BYTEA NOT NULL,  -- Public identity key (encoded)
    registration_id INTEGER NOT NULL,  -- Signal Protocol registration ID
    device_name TEXT,  -- User-friendly device name (e.g., "iPhone 15", "Firefox on MacBook")
    device_type TEXT DEFAULT 'unknown',  -- web, desktop, mobile_ios, mobile_android
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, device_id)
);

-- Signed pre-keys table - signed ephemeral keys rotated periodically
CREATE TABLE signed_prekeys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_keys_id UUID NOT NULL REFERENCES device_keys(id) ON DELETE CASCADE,
    key_id INTEGER NOT NULL,  -- Signal Protocol key ID
    public_key BYTEA NOT NULL,  -- Signed pre-key public key
    signature BYTEA NOT NULL,  -- Signature from identity key
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,  -- Optional expiry
    
    UNIQUE(device_keys_id, key_id)
);

-- One-time pre-keys table - single-use keys for session establishment
CREATE TABLE one_time_prekeys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_keys_id UUID NOT NULL REFERENCES device_keys(id) ON DELETE CASCADE,
    key_id INTEGER NOT NULL,  -- Signal Protocol key ID
    public_key BYTEA NOT NULL,  -- One-time pre-key public key
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    claimed_at TIMESTAMP WITH TIME ZONE,  -- NULL until claimed by another user
    claimed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,  -- Who claimed this key
    
    UNIQUE(device_keys_id, key_id)
);

-- Indexes for efficient lookups
CREATE INDEX idx_device_keys_user_id ON device_keys(user_id);
CREATE INDEX idx_device_keys_user_device ON device_keys(user_id, device_id);
CREATE INDEX idx_device_keys_last_seen ON device_keys(last_seen);

CREATE INDEX idx_signed_prekeys_device ON signed_prekeys(device_keys_id);
CREATE INDEX idx_signed_prekeys_created ON signed_prekeys(created_at);

CREATE INDEX idx_one_time_prekeys_device ON one_time_prekeys(device_keys_id);
CREATE INDEX idx_one_time_prekeys_unclaimed ON one_time_prekeys(device_keys_id) WHERE claimed_at IS NULL;
CREATE INDEX idx_one_time_prekeys_created ON one_time_prekeys(created_at);

-- Function to claim a one-time prekey atomically
CREATE OR REPLACE FUNCTION claim_one_time_prekey(
    p_device_keys_id UUID,
    p_claiming_user_id UUID
) RETURNS TABLE(
    key_id INTEGER,
    public_key BYTEA
) AS $$
DECLARE
    claimed_key RECORD;
BEGIN
    -- Select and claim the oldest unclaimed key atomically
    UPDATE one_time_prekeys
    SET claimed_at = NOW(),
        claimed_by_user_id = p_claiming_user_id
    WHERE id = (
        SELECT id FROM one_time_prekeys
        WHERE device_keys_id = p_device_keys_id
        AND claimed_at IS NULL
        ORDER BY created_at ASC
        LIMIT 1
        FOR UPDATE SKIP LOCKED
    )
    RETURNING one_time_prekeys.key_id, one_time_prekeys.public_key INTO claimed_key;
    
    IF claimed_key IS NOT NULL THEN
        key_id := claimed_key.key_id;
        public_key := claimed_key.public_key;
        RETURN NEXT;
    END IF;
    
    RETURN;
END;
$$ LANGUAGE plpgsql;

-- Function to count remaining one-time prekeys for a device
CREATE OR REPLACE FUNCTION count_unclaimed_prekeys(p_device_keys_id UUID) 
RETURNS INTEGER AS $$
BEGIN
    RETURN (
        SELECT COUNT(*)::INTEGER 
        FROM one_time_prekeys 
        WHERE device_keys_id = p_device_keys_id 
        AND claimed_at IS NULL
    );
END;
$$ LANGUAGE plpgsql;

-- Comment for documentation
COMMENT ON TABLE device_keys IS 'Stores E2EE identity keys for each user device, implementing Signal Protocol key storage';
COMMENT ON TABLE signed_prekeys IS 'Stores signed pre-keys for X3DH key agreement, rotated periodically';
COMMENT ON TABLE one_time_prekeys IS 'Stores single-use pre-keys for enhanced forward secrecy in X3DH';
COMMENT ON FUNCTION claim_one_time_prekey IS 'Atomically claims and returns a one-time prekey for session establishment';
