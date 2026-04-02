-- Migration 041: Advanced Sticker System - Premium Sticker Packs
-- Adds sticker pack system with tiers (Free, Premium) linked to subscription tiers

-- Create sticker pack tier enum
CREATE TYPE sticker_pack_tier AS ENUM ('free', 'basic', 'premium');

-- Create sticker packs table
CREATE TABLE sticker_packs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    icon_url VARCHAR(512),
    tier sticker_pack_tier NOT NULL DEFAULT 'free',
    sticker_count INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_global BOOLEAN NOT NULL DEFAULT true,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create pack_stickers join table (many-to-many relationship)
CREATE TABLE pack_stickers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pack_id UUID NOT NULL REFERENCES sticker_packs(id) ON DELETE CASCADE,
    sticker_id UUID NOT NULL REFERENCES stickers(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(pack_id, sticker_id)
);

-- Add required_tier column to stickers (for premium stickers in global packs)
ALTER TABLE stickers ADD COLUMN IF NOT EXISTS required_tier sticker_pack_tier NOT NULL DEFAULT 'free';

-- Indexes for sticker pack queries
CREATE INDEX idx_sticker_packs_tier ON sticker_packs(tier);
CREATE INDEX idx_sticker_packs_server ON sticker_packs(server_id) WHERE server_id IS NOT NULL;
CREATE INDEX idx_sticker_packs_global ON sticker_packs(is_global) WHERE is_global = true;
CREATE INDEX idx_sticker_packs_active ON sticker_packs(is_active) WHERE is_active = true;

-- Indexes for pack_stickers
CREATE INDEX idx_pack_stickers_pack ON pack_stickers(pack_id);
CREATE INDEX idx_pack_stickers_sticker ON pack_stickers(sticker_id);

-- Index for sticker required_tier
CREATE INDEX idx_stickers_required_tier ON stickers(required_tier);

-- Insert default free sticker pack
INSERT INTO sticker_packs (id, name, description, tier, sticker_count, is_active, is_global, created_by)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Default Pack',
    'The default free sticker pack available to all users',
    'free',
    0,
    true,
    true,
    NULL
) ON CONFLICT DO NOTHING;

-- Function to update sticker_count on pack
CREATE OR REPLACE FUNCTION update_pack_sticker_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE sticker_packs SET sticker_count = sticker_count + 1 WHERE id = NEW.pack_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE sticker_packs SET sticker_count = sticker_count - 1 WHERE id = OLD.pack_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update sticker count on pack
DROP TRIGGER IF EXISTS trg_update_pack_sticker_count ON pack_stickers;
CREATE TRIGGER trg_update_pack_sticker_count
AFTER INSERT OR DELETE ON pack_stickers
FOR EACH ROW EXECUTE FUNCTION update_pack_sticker_count();
