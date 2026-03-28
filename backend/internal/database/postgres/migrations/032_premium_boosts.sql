-- Migration 032: Server Boost & Premium System
-- Adds tables for subscription management, server boosts, and billing

-- Subscriptions table
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier VARCHAR(16) NOT NULL DEFAULT 'free',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    boosts_used INTEGER NOT NULL DEFAULT 0,
    boosts_total INTEGER NOT NULL DEFAULT 0,
    next_billing TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,
    payment_method_id VARCHAR(255),
    payment_method_type VARCHAR(32),
    payment_method_last4 VARCHAR(4),
    payment_method_brand VARCHAR(32),
    payment_method_expires_at TIMESTAMP WITH TIME ZONE,
    external_id VARCHAR(255), -- Stripe/Paddle customer/subscription ID
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_external ON subscriptions(external_id) WHERE external_id IS NOT NULL;
CREATE INDEX idx_subscriptions_status ON subscriptions(status);

-- Server boosts table
CREATE TABLE server_boosts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(server_id, user_id)
);

CREATE INDEX idx_server_boosts_server ON server_boosts(server_id);
CREATE INDEX idx_server_boosts_user ON server_boosts(user_id);
CREATE INDEX idx_server_boosts_active ON server_boosts(server_id, active) WHERE active = TRUE;

-- Billing customers table (for Stripe/Paddle integration)
CREATE TABLE billing_customers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    external_id VARCHAR(255) NOT NULL, -- Stripe customer ID
    provider VARCHAR(16) NOT NULL DEFAULT 'stripe',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id),
    UNIQUE(external_id)
);

CREATE INDEX idx_billing_customers_user ON billing_customers(user_id);
CREATE INDEX idx_billing_customers_external ON billing_customers(external_id);

-- Billing invoices table
CREATE TABLE billing_invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    amount INTEGER NOT NULL, -- cents
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(16) NOT NULL,
    description TEXT,
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

CREATE INDEX idx_billing_invoices_user ON billing_invoices(user_id);
CREATE INDEX idx_billing_invoices_status ON billing_invoices(status);

-- Payment methods table
CREATE TABLE payment_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL,
    last4 VARCHAR(4) NOT NULL,
    brand VARCHAR(32),
    expires_at TIMESTAMP WITH TIME ZONE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, external_id)
);

CREATE INDEX idx_payment_methods_user ON payment_methods(user_id);

-- Server boost features (denormalized for performance)
-- This table stores calculated perks for each server based on boost count
CREATE TABLE server_boost_levels (
    server_id UUID PRIMARY KEY REFERENCES servers(id) ON DELETE CASCADE,
    level INTEGER NOT NULL DEFAULT 0,
    boost_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_server_boost_levels_level ON server_boost_levels(level);

-- Function to update server boost level
CREATE OR REPLACE FUNCTION update_server_boost_level()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate level based on boost count
    IF NEW.boost_count >= 30 THEN
        NEW.level := 3;
    ELSIF NEW.boost_count >= 15 THEN
        NEW.level := 2;
    ELSIF NEW.boost_count >= 2 THEN
        NEW.level := 1;
    ELSE
        NEW.level := 0;
    END IF;
    
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update boost level
CREATE TRIGGER tr_update_server_boost_level
    BEFORE INSERT OR UPDATE ON server_boost_levels
    FOR EACH ROW
    EXECUTE FUNCTION update_server_boost_level();

-- Add premium_since column to members table if not exists
-- (already exists in initial schema based on our review)

-- Add boost_count to servers table for quick lookup
ALTER TABLE servers ADD COLUMN IF NOT EXISTS boost_count INTEGER NOT NULL DEFAULT 0;

-- Add premium flags to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS premium_tier VARCHAR(16) NOT NULL DEFAULT 'free';
ALTER TABLE users ADD COLUMN IF NOT EXISTS premium_since TIMESTAMP WITH TIME ZONE;
