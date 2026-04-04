-- Migration 043: Content Safety System
-- Creates tables for NSFW filtering, age verification, and user content preferences

-- Content filters table
CREATE TABLE content_filters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    type INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    threshold INTEGER NOT NULL DEFAULT 0,
    action INTEGER NOT NULL DEFAULT 0,
    filter_data JSONB NOT NULL DEFAULT '{}',
    exempt_roles JSONB DEFAULT '[]',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_content_filters_server_id ON content_filters(server_id);
CREATE INDEX idx_content_filters_channel_id ON content_filters(channel_id);
CREATE INDEX idx_content_filters_enabled ON content_filters(server_id, enabled);
CREATE INDEX idx_content_filters_type ON content_filters(type);

-- Age verification settings table
CREATE TABLE age_verification_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    required_age INTEGER NOT NULL DEFAULT 18,
    verification_type VARCHAR(50) NOT NULL DEFAULT 'manual',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(server_id, channel_id)
);

CREATE INDEX idx_age_verification_server_id ON age_verification_settings(server_id);
CREATE INDEX idx_age_verification_channel_id ON age_verification_settings(channel_id);

-- User content preferences table
CREATE TABLE user_content_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nsfw_filter_level INTEGER NOT NULL DEFAULT 2,
    hide_nsfw_content BOOLEAN NOT NULL DEFAULT true,
    hide_explicit_content BOOLEAN NOT NULL DEFAULT false,
    auto_collapse_nsfw BOOLEAN NOT NULL DEFAULT false,
    allow_age_verified_channels BOOLEAN NOT NULL DEFAULT true,
    trusted_servers JSONB DEFAULT '[]',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_user_content_preferences_user_id ON user_content_preferences(user_id);
