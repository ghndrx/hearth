-- +migrate Up

-- Add premium_tier to users table if not exists
ALTER TABLE users ADD COLUMN IF NOT EXISTS premium_tier VARCHAR(20) DEFAULT 'free';

-- Create index for premium_tier lookups
CREATE INDEX IF NOT EXISTS idx_users_premium_tier ON users(premium_tier);

-- Subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier VARCHAR(20) NOT NULL DEFAULT 'free',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    boosts_used INT NOT NULL DEFAULT 0,
    boosts_total INT NOT NULL DEFAULT 0,
    next_billing TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    external_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_external ON subscriptions(external_id);

-- Server boosts table
CREATE TABLE IF NOT EXISTS server_boosts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    UNIQUE(server_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_server_boosts_server ON server_boosts(server_id);
CREATE INDEX IF NOT EXISTS idx_server_boosts_user ON server_boosts(user_id);
CREATE INDEX IF NOT EXISTS idx_server_boosts_active ON server_boosts(server_id) WHERE active = true;

-- Server boost levels table (tracks computed level based on boost count)
CREATE TABLE IF NOT EXISTS server_boost_levels (
    server_id UUID PRIMARY KEY REFERENCES servers(id) ON DELETE CASCADE,
    boost_count INT NOT NULL DEFAULT 0,
    level INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Billing customers table
CREATE TABLE IF NOT EXISTS billing_customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    external_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id),
    UNIQUE(external_id)
);

CREATE INDEX IF NOT EXISTS idx_billing_customers_user ON billing_customers(user_id);
CREATE INDEX IF NOT EXISTS idx_billing_customers_external ON billing_customers(external_id);

-- Billing invoices table
CREATE TABLE IF NOT EXISTS billing_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_id VARCHAR(255),
    amount INT NOT NULL DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    description TEXT,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_billing_invoices_user ON billing_invoices(user_id);

-- Payment methods table
CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_id VARCHAR(255),
    type VARCHAR(20) NOT NULL,
    last4 VARCHAR(4),
    brand VARCHAR(20),
    expires_at TIMESTAMPTZ,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_payment_methods_user ON payment_methods(user_id);

-- Function to update server boost level based on active boost count
CREATE OR REPLACE FUNCTION update_server_boost_level()
RETURNS TRIGGER AS $$
DECLARE
    v_boost_count INT;
    v_new_level INT;
BEGIN
    IF TG_OP = 'INSERT' AND NEW.active = true THEN
        SELECT COUNT(*) INTO v_boost_count
        FROM server_boosts
        WHERE server_id = NEW.server_id AND active = true;
        
        v_new_level := CASE
            WHEN v_boost_count >= 30 THEN 3
            WHEN v_boost_count >= 15 THEN 2
            WHEN v_boost_count >= 2 THEN 1
            ELSE 0
        END;
        
        INSERT INTO server_boost_levels (server_id, boost_count, level, updated_at)
        VALUES (NEW.server_id, v_boost_count, v_new_level, NOW())
        ON CONFLICT (server_id) DO UPDATE SET
            boost_count = v_boost_count,
            level = v_new_level,
            updated_at = NOW();
            
    ELSIF TG_OP = 'DELETE' AND OLD.active = true THEN
        SELECT COUNT(*) INTO v_boost_count
        FROM server_boosts
        WHERE server_id = OLD.server_id AND active = true;
        
        v_new_level := CASE
            WHEN v_boost_count >= 30 THEN 3
            WHEN v_boost_count >= 15 THEN 2
            WHEN v_boost_count >= 2 THEN 1
            ELSE 0
        END;
        
        INSERT INTO server_boost_levels (server_id, boost_count, level, updated_at)
        VALUES (OLD.server_id, v_boost_count, v_new_level, NOW())
        ON CONFLICT (server_id) DO UPDATE SET
            boost_count = v_boost_count,
            level = v_new_level,
            updated_at = NOW();
            
    ELSIF TG_OP = 'UPDATE' AND OLD.active != NEW.active THEN
        SELECT COUNT(*) INTO v_boost_count
        FROM server_boosts
        WHERE server_id = NEW.server_id AND active = true;
        
        v_new_level := CASE
            WHEN v_boost_count >= 30 THEN 3
            WHEN v_boost_count >= 15 THEN 2
            WHEN v_boost_count >= 2 THEN 1
            ELSE 0
        END;
        
        INSERT INTO server_boost_levels (server_id, boost_count, level, updated_at)
        VALUES (NEW.server_id, v_boost_count, v_new_level, NOW())
        ON CONFLICT (server_id) DO UPDATE SET
            boost_count = v_boost_count,
            level = v_new_level,
            updated_at = NOW();
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update server boost level on boost changes
DROP TRIGGER IF EXISTS trigger_update_server_boost_level ON server_boosts;
CREATE TRIGGER trigger_update_server_boost_level
    AFTER INSERT OR UPDATE OR DELETE ON server_boosts
    FOR EACH ROW EXECUTE FUNCTION update_server_boost_level();

-- Function to check if user can boost a server
CREATE OR REPLACE FUNCTION check_user_can_boost(p_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_tier VARCHAR(20);
    v_boosts_used INT;
    v_boosts_total INT;
BEGIN
    SELECT u.premium_tier, COALESCE(s.boosts_used, 0), COALESCE(s.boosts_total, 0)
    INTO v_tier, v_boosts_used, v_boosts_total
    FROM users u
    LEFT JOIN subscriptions s ON s.user_id = u.id AND s.status = 'active'
    WHERE u.id = p_user_id;
    
    IF v_tier IS NULL OR v_tier = 'free' THEN
        RETURN FALSE;
    END IF;
    
    IF v_boosts_used >= v_boosts_total THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- +migrate Down

DROP TRIGGER IF EXISTS trigger_update_server_boost_level ON server_boosts;
DROP FUNCTION IF EXISTS check_user_can_boost(UUID);
DROP FUNCTION IF EXISTS update_server_boost_level();

DROP TABLE IF EXISTS payment_methods CASCADE;
DROP TABLE IF EXISTS billing_invoices CASCADE;
DROP TABLE IF EXISTS billing_customers CASCADE;
DROP TABLE IF EXISTS server_boost_levels CASCADE;
DROP TABLE IF EXISTS server_boosts CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;

ALTER TABLE users DROP COLUMN IF EXISTS premium_tier;
