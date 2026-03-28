-- Hearth Database Schema
-- Migration 009: Enhanced user profiles and status features

-- Connected accounts table (OAuth integrations)
CREATE TABLE connected_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL,  -- 'github', 'twitter', 'spotify', 'steam', etc.
    account_id VARCHAR(255) NOT NULL,
    account_name VARCHAR(255),
    verified BOOLEAN DEFAULT FALSE,
    visibility INTEGER DEFAULT 1,  -- 0=private, 1=friends only, 2=everyone
    show_activity BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    access_token_encrypted TEXT,
    refresh_token_encrypted TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, type)
);

CREATE INDEX idx_connected_accounts_user ON connected_accounts(user_id);

-- User badges/achievements table
CREATE TABLE user_badges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_type VARCHAR(64) NOT NULL,
    badge_name VARCHAR(128) NOT NULL,
    badge_icon VARCHAR(512),
    description TEXT,
    earned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    UNIQUE(user_id, badge_type)
);

CREATE INDEX idx_user_badges_user ON user_badges(user_id);

-- User status persistence table (for custom status with emoji, expiry, etc.)
CREATE TABLE user_statuses (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    custom_text VARCHAR(128),
    emoji VARCHAR(64),
    emoji_id UUID,
    emoji_name VARCHAR(64),
    clear_after TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User activities table (for persistent activity status)
CREATE TABLE user_activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type INTEGER NOT NULL,  -- 0=Playing, 1=Streaming, 2=Listening, 3=Watching, 4=Custom, 5=Competing
    name VARCHAR(128) NOT NULL,
    url VARCHAR(512),
    details VARCHAR(128),
    state VARCHAR(128),
    application_id UUID,
    assets_large_image VARCHAR(512),
    assets_large_text VARCHAR(128),
    assets_small_image VARCHAR(512),
    assets_small_text VARCHAR(128),
    started_at TIMESTAMP WITH TIME ZONE,
    ends_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_user_activities_user ON user_activities(user_id);

-- Add pronouns column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS pronouns VARCHAR(32);

-- Add display_name column (separate from username)
ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name VARCHAR(32);

-- Add accent_color for profile customization
ALTER TABLE users ADD COLUMN IF NOT EXISTS accent_color INTEGER;

-- Add about_me (longer bio, markdown supported)
ALTER TABLE users ADD COLUMN IF NOT EXISTS about_me TEXT;

-- Update user_settings with status preferences
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS status_persist BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS status_auto_idle BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS status_auto_idle_minutes INTEGER DEFAULT 10,
    ADD COLUMN IF NOT EXISTS activity_display BOOLEAN DEFAULT TRUE;

-- Apply updated_at trigger to new tables
CREATE TRIGGER connected_accounts_updated_at BEFORE UPDATE ON connected_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER user_statuses_updated_at BEFORE UPDATE ON user_statuses FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER user_activities_updated_at BEFORE UPDATE ON user_activities FOR EACH ROW EXECUTE FUNCTION update_updated_at();
