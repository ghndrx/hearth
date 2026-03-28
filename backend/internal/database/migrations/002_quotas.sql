-- +migrate Up

-- Server-level quota overrides
CREATE TABLE server_quotas (
    server_id UUID PRIMARY KEY REFERENCES servers(id) ON DELETE CASCADE,
    storage_mb BIGINT,                  -- NULL = use instance default, 0 = unlimited
    max_file_size_mb INT,
    max_channels INT,
    max_roles INT,
    max_emoji INT,
    max_members INT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Role-level quota overrides
CREATE TABLE role_quotas (
    role_id UUID PRIMARY KEY REFERENCES roles(id) ON DELETE CASCADE,
    storage_mb BIGINT,
    max_file_size_mb INT,
    message_rate_limit INT,             -- messages per rate window
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User-level quota overrides (admin-assigned)
CREATE TABLE user_quotas (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    storage_mb BIGINT,
    max_file_size_mb INT,
    max_servers_owned INT,
    max_servers_joined INT,
    message_rate_limit INT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Storage usage tracking (per user per server)
CREATE TABLE storage_usage (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    used_bytes BIGINT DEFAULT 0,
    file_count INT DEFAULT 0,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, server_id)
);

CREATE INDEX idx_storage_usage_user ON storage_usage(user_id);
CREATE INDEX idx_storage_usage_server ON storage_usage(server_id);

-- Global storage usage per user (across all servers + DMs)
CREATE TABLE user_storage_totals (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_bytes BIGINT DEFAULT 0,
    file_count INT DEFAULT 0,
    last_updated TIMESTAMPTZ DEFAULT NOW()
);

-- Triggers to update storage totals

CREATE OR REPLACE FUNCTION update_storage_totals()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        INSERT INTO user_storage_totals (user_id, total_bytes, file_count, last_updated)
        SELECT 
            NEW.user_id,
            COALESCE(SUM(used_bytes), 0),
            COALESCE(SUM(file_count), 0),
            NOW()
        FROM storage_usage
        WHERE user_id = NEW.user_id
        ON CONFLICT (user_id) DO UPDATE SET
            total_bytes = EXCLUDED.total_bytes,
            file_count = EXCLUDED.file_count,
            last_updated = NOW();
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO user_storage_totals (user_id, total_bytes, file_count, last_updated)
        SELECT 
            OLD.user_id,
            COALESCE(SUM(used_bytes), 0),
            COALESCE(SUM(file_count), 0),
            NOW()
        FROM storage_usage
        WHERE user_id = OLD.user_id
        ON CONFLICT (user_id) DO UPDATE SET
            total_bytes = EXCLUDED.total_bytes,
            file_count = EXCLUDED.file_count,
            last_updated = NOW();
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_storage_totals
    AFTER INSERT OR UPDATE OR DELETE ON storage_usage
    FOR EACH ROW EXECUTE FUNCTION update_storage_totals();

-- Function to check storage quota before upload
CREATE OR REPLACE FUNCTION check_storage_quota(
    p_user_id UUID,
    p_server_id UUID,
    p_file_size BIGINT,
    p_user_limit_bytes BIGINT,    -- 0 or NULL = unlimited
    p_server_limit_bytes BIGINT   -- 0 or NULL = unlimited
) RETURNS BOOLEAN AS $$
DECLARE
    v_user_total BIGINT;
    v_server_total BIGINT;
BEGIN
    -- Check user quota
    IF p_user_limit_bytes IS NOT NULL AND p_user_limit_bytes > 0 THEN
        SELECT COALESCE(total_bytes, 0) INTO v_user_total
        FROM user_storage_totals
        WHERE user_id = p_user_id;
        
        IF v_user_total + p_file_size > p_user_limit_bytes THEN
            RETURN FALSE;
        END IF;
    END IF;
    
    -- Check server quota (if server_id provided)
    IF p_server_id IS NOT NULL AND p_server_limit_bytes IS NOT NULL AND p_server_limit_bytes > 0 THEN
        SELECT COALESCE(SUM(used_bytes), 0) INTO v_server_total
        FROM storage_usage
        WHERE server_id = p_server_id;
        
        IF v_server_total + p_file_size > p_server_limit_bytes THEN
            RETURN FALSE;
        END IF;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- +migrate Down

DROP FUNCTION IF EXISTS check_storage_quota CASCADE;
DROP FUNCTION IF EXISTS update_storage_totals CASCADE;
DROP TRIGGER IF EXISTS trigger_update_storage_totals ON storage_usage;
DROP TABLE IF EXISTS user_storage_totals CASCADE;
DROP TABLE IF EXISTS storage_usage CASCADE;
DROP TABLE IF EXISTS user_quotas CASCADE;
DROP TABLE IF EXISTS role_quotas CASCADE;
DROP TABLE IF EXISTS server_quotas CASCADE;
