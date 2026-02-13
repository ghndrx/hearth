-- Hearth Database Schema
-- Migration 001: Initial schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(32) NOT NULL,
    discriminator CHAR(4) NOT NULL DEFAULT '0000',
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(512),
    banner_url VARCHAR(512),
    bio TEXT,
    status VARCHAR(16) DEFAULT 'offline',
    custom_status VARCHAR(128),
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret VARCHAR(64),
    verified BOOLEAN DEFAULT FALSE,
    flags BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(username, discriminator)
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- Servers table
CREATE TABLE servers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    icon_url VARCHAR(512),
    banner_url VARCHAR(512),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    region VARCHAR(32) DEFAULT 'auto',
    verification_level INTEGER DEFAULT 0,
    default_notifications INTEGER DEFAULT 0,
    explicit_filter INTEGER DEFAULT 0,
    features TEXT[] DEFAULT '{}',
    system_channel_id UUID,
    rules_channel_id UUID,
    vanity_url VARCHAR(32) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_servers_owner ON servers(owner_id);
CREATE INDEX idx_servers_vanity ON servers(vanity_url) WHERE vanity_url IS NOT NULL;

-- Roles table
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color INTEGER DEFAULT 0,
    position INTEGER DEFAULT 0,
    permissions BIGINT DEFAULT 0,
    hoist BOOLEAN DEFAULT FALSE,
    mentionable BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,
    icon_url VARCHAR(512),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_roles_server ON roles(server_id);
CREATE INDEX idx_roles_position ON roles(server_id, position);

-- Members table (server memberships)
CREATE TABLE members (
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nickname VARCHAR(32),
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    premium_since TIMESTAMP WITH TIME ZONE,
    deaf BOOLEAN DEFAULT FALSE,
    mute BOOLEAN DEFAULT FALSE,
    pending BOOLEAN DEFAULT FALSE,
    temporary BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (server_id, user_id)
);

CREATE INDEX idx_members_user ON members(user_id);
CREATE INDEX idx_members_joined ON members(server_id, joined_at);

-- Member roles junction table
CREATE TABLE member_roles (
    server_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (server_id, user_id, role_id),
    FOREIGN KEY (server_id, user_id) REFERENCES members(server_id, user_id) ON DELETE CASCADE
);

CREATE INDEX idx_member_roles_role ON member_roles(role_id);

-- Channels table
CREATE TABLE channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(100),
    type INTEGER NOT NULL DEFAULT 0,
    topic VARCHAR(1024),
    position INTEGER DEFAULT 0,
    slowmode INTEGER DEFAULT 0,
    nsfw BOOLEAN DEFAULT FALSE,
    e2ee_enabled BOOLEAN DEFAULT FALSE,
    bitrate INTEGER,
    user_limit INTEGER,
    rtc_region VARCHAR(32),
    last_message_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_channels_server ON channels(server_id);
CREATE INDEX idx_channels_parent ON channels(parent_id);
CREATE INDEX idx_channels_position ON channels(server_id, position);

-- DM channel recipients (for DM and Group DM channels)
CREATE TABLE channel_recipients (
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_channel_recipients_user ON channel_recipients(user_id);

-- Messages table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    content TEXT,
    encrypted_content TEXT,
    type INTEGER DEFAULT 0,
    edited_at TIMESTAMP WITH TIME ZONE,
    pinned BOOLEAN DEFAULT FALSE,
    pinned_at TIMESTAMP WITH TIME ZONE,
    tts BOOLEAN DEFAULT FALSE,
    mentions_everyone BOOLEAN DEFAULT FALSE,
    reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    thread_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    flags INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_messages_channel ON messages(channel_id, created_at DESC);
CREATE INDEX idx_messages_author ON messages(author_id);
CREATE INDEX idx_messages_reply ON messages(reply_to_id) WHERE reply_to_id IS NOT NULL;

-- Attachments table
CREATE TABLE attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    url VARCHAR(512) NOT NULL,
    proxy_url VARCHAR(512),
    content_type VARCHAR(100),
    size BIGINT NOT NULL,
    width INTEGER,
    height INTEGER,
    ephemeral BOOLEAN DEFAULT FALSE,
    encrypted BOOLEAN DEFAULT FALSE,
    encrypted_key TEXT,
    iv VARCHAR(32),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_attachments_message ON attachments(message_id);

-- Reactions table
CREATE TABLE reactions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);

CREATE INDEX idx_reactions_message ON reactions(message_id);

-- Invites table
CREATE TABLE invites (
    code VARCHAR(32) PRIMARY KEY,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    max_uses INTEGER DEFAULT 0,
    uses INTEGER DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE,
    temporary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_invites_server ON invites(server_id);
CREATE INDEX idx_invites_expires ON invites(expires_at) WHERE expires_at IS NOT NULL;

-- Bans table
CREATE TABLE bans (
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    banned_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (server_id, user_id)
);

CREATE INDEX idx_bans_user ON bans(user_id);

-- Audit log table
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id UUID,
    action_type VARCHAR(64) NOT NULL,
    changes JSONB,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_log_server ON audit_log(server_id, created_at DESC);
CREATE INDEX idx_audit_log_user ON audit_log(user_id);
CREATE INDEX idx_audit_log_action ON audit_log(server_id, action_type);

-- Webhooks table
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(80) NOT NULL,
    avatar_url VARCHAR(512),
    token VARCHAR(68) NOT NULL,
    url VARCHAR(512),
    source_server_id UUID,
    source_channel_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_webhooks_channel ON webhooks(channel_id);
CREATE INDEX idx_webhooks_server ON webhooks(server_id);

-- Channel permission overwrites
CREATE TABLE permission_overwrites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    target_id UUID NOT NULL, -- Can be role_id or user_id
    target_type INTEGER NOT NULL, -- 0 = role, 1 = user
    allow BIGINT DEFAULT 0,
    deny BIGINT DEFAULT 0
);

CREATE INDEX idx_permission_overwrites_channel ON permission_overwrites(channel_id);
CREATE UNIQUE INDEX idx_permission_overwrites_unique ON permission_overwrites(channel_id, target_id, target_type);

-- User settings (per-user preferences)
CREATE TABLE user_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme VARCHAR(16) DEFAULT 'dark',
    message_display VARCHAR(16) DEFAULT 'cozy',
    inline_embeds BOOLEAN DEFAULT TRUE,
    inline_attachments BOOLEAN DEFAULT TRUE,
    render_reactions BOOLEAN DEFAULT TRUE,
    animate_emoji BOOLEAN DEFAULT TRUE,
    enable_tts BOOLEAN DEFAULT TRUE,
    compact_mode BOOLEAN DEFAULT FALSE,
    developer_mode BOOLEAN DEFAULT FALSE,
    custom_css TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Server-specific user settings
CREATE TABLE member_settings (
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    muted BOOLEAN DEFAULT FALSE,
    mute_until TIMESTAMP WITH TIME ZONE,
    notification_level INTEGER DEFAULT 0,
    suppress_everyone BOOLEAN DEFAULT FALSE,
    suppress_roles BOOLEAN DEFAULT FALSE,
    hide_muted_channels BOOLEAN DEFAULT FALSE,
    mobile_push BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (server_id, user_id)
);

-- Relationships (friends, blocked)
CREATE TABLE relationships (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type INTEGER NOT NULL, -- 1 = friend, 2 = blocked, 3 = pending incoming, 4 = pending outgoing
    nickname VARCHAR(32),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, target_id)
);

CREATE INDEX idx_relationships_target ON relationships(target_id);

-- Read states (tracking what messages have been read)
CREATE TABLE read_states (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    last_message_id UUID,
    mention_count INTEGER DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, channel_id)
);

-- Sessions (for token management)
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    device VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    last_used TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(token_hash);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at trigger to relevant tables
CREATE TRIGGER users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER servers_updated_at BEFORE UPDATE ON servers FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER user_settings_updated_at BEFORE UPDATE ON user_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER read_states_updated_at BEFORE UPDATE ON read_states FOR EACH ROW EXECUTE FUNCTION update_updated_at();
