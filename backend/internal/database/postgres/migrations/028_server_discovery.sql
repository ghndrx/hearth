-- Migration: Server Discovery & Browse
-- Adds support for public server listings, discovery, and search

-- Server Discovery Listings table
CREATE TABLE IF NOT EXISTS server_discovery_listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL UNIQUE REFERENCES servers(id) ON DELETE CASCADE,
    short_description VARCHAR(160) NOT NULL DEFAULT '',
    is_listed BOOLEAN NOT NULL DEFAULT false,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    featured_at TIMESTAMP WITH TIME ZONE,
    approval_status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    approved_at TIMESTAMP WITH TIME ZONE,
    approved_by UUID REFERENCES users(id),
    rejection_reason TEXT,
    member_count_snapshot INT NOT NULL DEFAULT 0,
    online_count_snapshot INT NOT NULL DEFAULT 0,
    weekly_growth_rate DECIMAL(5,2) DEFAULT 0,
    engagement_score DECIMAL(5,2) DEFAULT 0,
    region VARCHAR(64),
    language VARCHAR(10) DEFAULT 'en',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Server Discovery Categories (many-to-many)
CREATE TABLE IF NOT EXISTS server_discovery_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL UNIQUE,
    slug VARCHAR(64) NOT NULL UNIQUE,
    icon VARCHAR(32) NOT NULL DEFAULT '🏠',
    description TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Junction table for server-category relationship
CREATE TABLE IF NOT EXISTS server_discovery_listing_categories (
    listing_id UUID NOT NULL REFERENCES server_discovery_listings(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES server_discovery_categories(id) ON DELETE CASCADE,
    PRIMARY KEY (listing_id, category_id)
);

-- Server Discovery Tags
CREATE TABLE IF NOT EXISTS server_discovery_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL UNIQUE,
    slug VARCHAR(64) NOT NULL UNIQUE,
    usage_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Junction table for server tags
CREATE TABLE IF NOT EXISTS server_discovery_listing_tags (
    listing_id UUID NOT NULL REFERENCES server_discovery_listings(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES server_discovery_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (listing_id, tag_id)
);

-- Server Discovery Reports
CREATE TABLE IF NOT EXISTS server_discovery_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES server_discovery_listings(id) ON DELETE CASCADE,
    reporter_id UUID NOT NULL REFERENCES users(id),
    reason VARCHAR(64) NOT NULL,
    details TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending, reviewed, resolved
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for discovery queries
CREATE INDEX idx_server_discovery_listings_listed ON server_discovery_listings(is_listed) WHERE is_listed = true;
CREATE INDEX idx_server_discovery_listings_featured ON server_discovery_listings(is_featured, featured_at) WHERE is_featured = true;
CREATE INDEX idx_server_discovery_listings_approval ON server_discovery_listings(approval_status) WHERE approval_status = 'pending';
CREATE INDEX idx_server_discovery_listings_server_id ON server_discovery_listings(server_id);
CREATE INDEX idx_server_discovery_listings_member_count ON server_discovery_listings(member_count_snapshot DESC);
CREATE INDEX idx_server_discovery_listings_growth ON server_discovery_listings(weekly_growth_rate DESC);
CREATE INDEX idx_server_discovery_listings_engagement ON server_discovery_listings(engagement_score DESC);

-- Insert default categories
INSERT INTO server_discovery_categories (name, slug, icon, description, sort_order) VALUES
    ('Gaming', 'gaming', '🎮', 'For gamers and gaming communities', 1),
    ('Music', 'music', '🎵', 'Music fans, artists, and producers', 2),
    ('Technology', 'technology', '💻', 'Tech enthusiasts and developers', 3),
    ('Art & Design', 'art', '🎨', 'Artists and creative communities', 4),
    ('Education', 'education', '📚', 'Learning and academic communities', 5),
    ('Science', 'science', '🔬', 'Science and research discussions', 6),
    ('Entertainment', 'entertainment', '🎬', 'Movies, TV, and entertainment', 7),
    ('Social', 'social', '💬', 'General social communities', 8),
    ('Sports', 'sports', '⚽', 'Sports fans and athletes', 9),
    ('Anime & Manga', 'anime', '🍜', 'Anime and manga communities', 10),
    ('Fashion', 'fashion', '👗', 'Fashion and lifestyle', 11),
    ('Food & Cooking', 'food', '🍳', 'Foodies and cooking enthusiasts', 12),
    ('Business', 'business', '💼', 'Professional and business communities', 13),
    ('Language Learning', 'language', '🗣️', 'Language exchange and learning', 14)
ON CONFLICT (slug) DO NOTHING;

-- Add updated_at trigger function
CREATE OR REPLACE FUNCTION update_server_discovery_listing_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for auto-updating timestamp
CREATE TRIGGER server_discovery_listings_updated_at
    BEFORE UPDATE ON server_discovery_listings
    FOR EACH ROW
    EXECUTE FUNCTION update_server_discovery_listing_timestamp();

-- Add is_public column to servers table for discovery eligibility
ALTER TABLE servers ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT false;
