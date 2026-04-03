-- +migrate Up

-- Comprehensive audit logs table for server moderation and compliance
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL, -- Who performed the action
    action_type VARCHAR(50) NOT NULL, -- Action category code (e.g., MEMBER_BAN, CHANNEL_CREATE)
    action_category INT NOT NULL, -- Numeric category (10-89) for efficient querying
    target_id UUID, -- What was affected (user_id, channel_id, role_id, etc.)
    target_type VARCHAR(30), -- Type of target (member, message, channel, role, webhook, emoji, etc.)
    reason TEXT, -- Moderation reason or description
    metadata JSONB DEFAULT '{}', -- Additional context (old/new values, IP, etc.)
    ip_address INET, -- Actor's IP address for security events
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Efficient indexes for common query patterns
CREATE INDEX idx_audit_logs_server_time ON audit_logs (server_id, created_at DESC);
CREATE INDEX idx_audit_logs_action_type ON audit_logs (action_type);
CREATE INDEX idx_audit_logs_action_category ON audit_logs (action_category);
CREATE INDEX idx_audit_logs_actor ON audit_logs (actor_id);
CREATE INDEX idx_audit_logs_target ON audit_logs (target_id) WHERE target_id IS NOT NULL;
CREATE INDEX idx_audit_logs_server_actor_time ON audit_logs (server_id, actor_id, created_at DESC);
CREATE INDEX idx_audit_logs_server_category_time ON audit_logs (server_id, action_category, created_at DESC);

-- Moderation analytics summary table for fast dashboard queries
CREATE TABLE IF NOT EXISTS moderation_analytics_daily (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    total_actions INT DEFAULT 0,
    member_bans INT DEFAULT 0,
    member_unbans INT DEFAULT 0,
    member_kicks INT DEFAULT 0,
    member_mutes INT DEFAULT 0,
    member_unmutes INT DEFAULT 0,
    member_warns INT DEFAULT 0,
    message_deletes INT DEFAULT 0,
    message_bulk_deletes INT DEFAULT 0,
    channel_creates INT DEFAULT 0,
    channel_updates INT DEFAULT 0,
    channel_deletes INT DEFAULT 0,
    role_creates INT DEFAULT 0,
    role_updates INT DEFAULT 0,
    role_deletes INT DEFAULT 0,
    webhook_creates INT DEFAULT 0,
    webhook_updates INT DEFAULT 0,
    webhook_deletes INT DEFAULT 0,
    emoji_creates INT DEFAULT 0,
    emoji_updates INT DEFAULT 0,
    emoji_deletes INT DEFAULT 0,
    invite_creates INT DEFAULT 0,
    invite_deletes INT DEFAULT 0,
    automod_triggers INT DEFAULT 0,
    automod_actions INT DEFAULT 0,
    UNIQUE(server_id, date)
);

CREATE INDEX idx_mod_analytics_server_date ON moderation_analytics_daily (server_id, date DESC);
CREATE INDEX idx_mod_analytics_date ON moderation_analytics_daily (date DESC);

-- +migrate Down

DROP INDEX IF EXISTS idx_mod_analytics_date;
DROP INDEX IF EXISTS idx_mod_analytics_server_date;
DROP TABLE IF EXISTS moderation_analytics_daily;
DROP INDEX IF EXISTS idx_audit_logs_server_category_time;
DROP INDEX IF EXISTS idx_audit_logs_server_actor_time;
DROP INDEX IF EXISTS idx_audit_logs_target;
DROP INDEX IF EXISTS idx_audit_logs_actor;
DROP INDEX IF EXISTS idx_audit_logs_action_category;
DROP INDEX IF EXISTS idx_audit_logs_action_type;
DROP INDEX IF EXISTS idx_audit_logs_server_time;
DROP TABLE IF EXISTS audit_logs;
