-- +migrate Up

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(32) NOT NULL,
    discriminator CHAR(4) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    banner_url TEXT,
    bio TEXT,
    status VARCHAR(20) DEFAULT 'offline',
    custom_status TEXT,
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret VARCHAR(64),
    verified BOOLEAN DEFAULT FALSE,
    flags BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(username, discriminator)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);

-- Sessions table
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    user_agent TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(refresh_token_hash);

-- Servers table
CREATE TABLE servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    icon_url TEXT,
    banner_url TEXT,
    description TEXT,
    owner_id UUID NOT NULL REFERENCES users(id),
    default_channel_id UUID,
    afk_channel_id UUID,
    afk_timeout INT DEFAULT 300,
    verification_level INT DEFAULT 0,
    explicit_content_filter INT DEFAULT 0,
    default_notifications INT DEFAULT 0,
    features TEXT[] DEFAULT '{}',
    max_members INT DEFAULT 500000,
    vanity_url_code VARCHAR(32) UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_servers_owner ON servers(owner_id);
CREATE INDEX idx_servers_vanity ON servers(vanity_url_code) WHERE vanity_url_code IS NOT NULL;

-- Channels table
CREATE TABLE channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    category_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    type VARCHAR(20) NOT NULL,
    name VARCHAR(100),
    topic TEXT,
    position INT DEFAULT 0,
    nsfw BOOLEAN DEFAULT FALSE,
    slowmode_seconds INT DEFAULT 0,
    bitrate INT DEFAULT 64000,
    user_limit INT DEFAULT 0,
    rtc_region VARCHAR(32),
    default_auto_archive INT DEFAULT 1440,
    last_message_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_channels_server ON channels(server_id);
CREATE INDEX idx_channels_category ON channels(category_id);

-- Add foreign key for default channel after channels table exists
ALTER TABLE servers ADD CONSTRAINT fk_servers_default_channel 
    FOREIGN KEY (default_channel_id) REFERENCES channels(id) ON DELETE SET NULL;
ALTER TABLE servers ADD CONSTRAINT fk_servers_afk_channel 
    FOREIGN KEY (afk_channel_id) REFERENCES channels(id) ON DELETE SET NULL;

-- Roles table
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color INT DEFAULT 0,
    permissions BIGINT DEFAULT 0,
    position INT DEFAULT 0,
    hoist BOOLEAN DEFAULT FALSE,
    managed BOOLEAN DEFAULT FALSE,
    mentionable BOOLEAN DEFAULT FALSE,
    icon_url TEXT,
    unicode_emoji VARCHAR(64),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_roles_server ON roles(server_id);
CREATE INDEX idx_roles_position ON roles(server_id, position DESC);

-- Members table
CREATE TABLE members (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    nickname VARCHAR(32),
    avatar_url TEXT,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    premium_since TIMESTAMPTZ,
    deaf BOOLEAN DEFAULT FALSE,
    mute BOOLEAN DEFAULT FALSE,
    pending BOOLEAN DEFAULT FALSE,
    timeout_until TIMESTAMPTZ,
    flags INT DEFAULT 0,
    PRIMARY KEY (user_id, server_id)
);

CREATE INDEX idx_members_server ON members(server_id);
CREATE INDEX idx_members_user ON members(user_id);

-- Member roles junction table
CREATE TABLE member_roles (
    member_user_id UUID,
    member_server_id UUID,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (member_user_id, member_server_id, role_id),
    FOREIGN KEY (member_user_id, member_server_id) 
        REFERENCES members(user_id, server_id) ON DELETE CASCADE
);

-- Channel permission overrides
CREATE TABLE channel_overrides (
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    target_type VARCHAR(10) NOT NULL, -- 'role' or 'user'
    target_id UUID NOT NULL,
    allow BIGINT DEFAULT 0,
    deny BIGINT DEFAULT 0,
    PRIMARY KEY (channel_id, target_type, target_id)
);

-- Messages table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    content TEXT,
    type VARCHAR(30) DEFAULT 'default',
    reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    thread_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    pinned BOOLEAN DEFAULT FALSE,
    tts BOOLEAN DEFAULT FALSE,
    mention_everyone BOOLEAN DEFAULT FALSE,
    flags INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    edited_at TIMESTAMPTZ
);

CREATE INDEX idx_messages_channel_time ON messages(channel_id, created_at DESC);
CREATE INDEX idx_messages_author ON messages(author_id);
CREATE INDEX idx_messages_thread ON messages(thread_id) WHERE thread_id IS NOT NULL;

-- Attachments table
CREATE TABLE attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    proxy_url TEXT,
    size BIGINT NOT NULL,
    content_type VARCHAR(100),
    width INT,
    height INT,
    ephemeral BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_attachments_message ON attachments(message_id);

-- Reactions table
CREATE TABLE reactions (
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    emoji VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);

CREATE INDEX idx_reactions_message ON reactions(message_id);

-- Message mentions (for tracking)
CREATE TABLE message_mentions (
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (message_id, user_id)
);

CREATE TABLE message_role_mentions (
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (message_id, role_id)
);

-- Pins table
CREATE TABLE pins (
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    pinned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    pinned_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (channel_id, message_id)
);

-- Read state tracking
CREATE TABLE read_states (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    last_message_id UUID,
    mention_count INT DEFAULT 0,
    last_read_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, channel_id)
);

-- Invites table
CREATE TABLE invites (
    code VARCHAR(16) PRIMARY KEY,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    inviter_id UUID REFERENCES users(id) ON DELETE SET NULL,
    max_uses INT,
    uses INT DEFAULT 0,
    max_age INT,
    temporary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_invites_server ON invites(server_id);
CREATE INDEX idx_invites_expires ON invites(expires_at) WHERE expires_at IS NOT NULL;

-- Bans table
CREATE TABLE bans (
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    banned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (server_id, user_id)
);

CREATE INDEX idx_bans_server ON bans(server_id);

-- DM channel recipients (for DM and group DM channels)
CREATE TABLE dm_recipients (
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_dm_recipients_user ON dm_recipients(user_id);

-- Audit log
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action_type VARCHAR(50) NOT NULL,
    target_type VARCHAR(50),
    target_id UUID,
    changes JSONB,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_log_server_time ON audit_log(server_id, created_at DESC);
CREATE INDEX idx_audit_log_user ON audit_log(user_id);
CREATE INDEX idx_audit_log_target ON audit_log(target_type, target_id);

-- Custom emoji
CREATE TABLE emoji (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(32) NOT NULL,
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    require_colons BOOLEAN DEFAULT TRUE,
    managed BOOLEAN DEFAULT FALSE,
    animated BOOLEAN DEFAULT FALSE,
    available BOOLEAN DEFAULT TRUE,
    url TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_emoji_server ON emoji(server_id);

-- Emoji role restrictions
CREATE TABLE emoji_roles (
    emoji_id UUID REFERENCES emoji(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (emoji_id, role_id)
);

-- Webhooks
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(80) NOT NULL,
    avatar_url TEXT,
    token VARCHAR(68) NOT NULL,
    type INT DEFAULT 1, -- 1=incoming, 2=channel follower
    source_server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    source_channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_webhooks_server ON webhooks(server_id);
CREATE INDEX idx_webhooks_channel ON webhooks(channel_id);

-- Friends and relationships
CREATE TABLE relationships (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    target_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type INT NOT NULL, -- 1=friend, 2=blocked, 3=pending_incoming, 4=pending_outgoing
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, target_id)
);

CREATE INDEX idx_relationships_user ON relationships(user_id);
CREATE INDEX idx_relationships_target ON relationships(target_id);

-- User notes
CREATE TABLE user_notes (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    target_id UUID REFERENCES users(id) ON DELETE CASCADE,
    note TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, target_id)
);

-- Updated at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add triggers
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_servers_updated_at BEFORE UPDATE ON servers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_channels_updated_at BEFORE UPDATE ON channels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +migrate Down

DROP TABLE IF EXISTS user_notes CASCADE;
DROP TABLE IF EXISTS relationships CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;
DROP TABLE IF EXISTS emoji_roles CASCADE;
DROP TABLE IF EXISTS emoji CASCADE;
DROP TABLE IF EXISTS audit_log CASCADE;
DROP TABLE IF EXISTS dm_recipients CASCADE;
DROP TABLE IF EXISTS bans CASCADE;
DROP TABLE IF EXISTS invites CASCADE;
DROP TABLE IF EXISTS read_states CASCADE;
DROP TABLE IF EXISTS pins CASCADE;
DROP TABLE IF EXISTS message_role_mentions CASCADE;
DROP TABLE IF EXISTS message_mentions CASCADE;
DROP TABLE IF EXISTS reactions CASCADE;
DROP TABLE IF EXISTS attachments CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS channel_overrides CASCADE;
DROP TABLE IF EXISTS member_roles CASCADE;
DROP TABLE IF EXISTS members CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS channels CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS servers CASCADE;
DROP TABLE IF EXISTS users CASCADE;

DROP FUNCTION IF EXISTS update_updated_at_column CASCADE;
