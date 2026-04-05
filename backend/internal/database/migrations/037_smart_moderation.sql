-- +migrate Up

-- Smart Moderation Settings table
CREATE TABLE moderation_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT true,
    sensitivity_level INT DEFAULT 2, -- 1=low, 2=medium, 3=high
    ml_classification_enabled BOOLEAN DEFAULT false, -- Placeholder for ML integration
    auto_moderation_enabled BOOLEAN DEFAULT true,
    violation_threshold INT DEFAULT 50, -- Score threshold for auto-action (0-100)
    warning_threshold INT DEFAULT 3, -- Violations before auto-warn
    mute_threshold INT DEFAULT 5, -- Violations before auto-mute
    temp_ban_threshold INT DEFAULT 10, -- Violations before auto-temp-ban
    temp_ban_duration INT DEFAULT 86400, -- Duration in seconds (default 1 day)
    mute_duration INT DEFAULT 3600, -- Duration in seconds (default 1 hour)
    log_channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    exempt_roles JSONB DEFAULT '[]',
    exempt_channels JSONB DEFAULT '[]',
    audit_retention_days INT DEFAULT 90,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(server_id)
);

CREATE INDEX idx_moderation_settings_server ON moderation_settings(server_id);

-- Keyword/Regex Rules table for configurable moderation patterns
CREATE TABLE moderation_keyword_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    pattern VARCHAR(500) NOT NULL,
    is_regex BOOLEAN DEFAULT false,
    sensitivity INT DEFAULT 2, -- 1-3 scale matching sensitivity levels
    category INT NOT NULL, -- ToxicityCategory enum
    action INT NOT NULL, -- ModerationActionType enum
    weight DECIMAL(3,2) DEFAULT 0.5, -- Weight for scoring (0.0-1.0)
    enabled BOOLEAN DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_moderation_keyword_rules_server ON moderation_keyword_rules(server_id);
CREATE INDEX idx_moderation_keyword_rules_category ON moderation_keyword_rules(category);
CREATE INDEX idx_moderation_keyword_rules_enabled ON moderation_keyword_rules(enabled) WHERE enabled = true;

-- Moderation Log table for tracking all moderation actions
CREATE TABLE moderation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    moderator_id UUID REFERENCES users(id) ON DELETE SET NULL, -- NULL for auto-mod actions
    action_type INT NOT NULL, -- ModerationActionType enum
    violation_score JSONB, -- ToxicityScore for auto-mod actions
    reason TEXT,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    message_id UUID,
    rule_id UUID REFERENCES moderation_keyword_rules(id) ON DELETE SET NULL,
    rule_name VARCHAR(100),
    duration INT, -- Duration in seconds (for mute/ban)
    resolved BOOLEAN DEFAULT false,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_moderation_logs_server ON moderation_logs(server_id);
CREATE INDEX idx_moderation_logs_member ON moderation_logs(member_id);
CREATE INDEX idx_moderation_logs_moderator ON moderation_logs(moderator_id);
CREATE INDEX idx_moderation_logs_created ON moderation_logs(created_at);
CREATE INDEX idx_moderation_logs_action ON moderation_logs(action_type);
CREATE INDEX idx_moderation_logs_resolved ON moderation_logs(resolved);

-- User Violation Summary table for tracking violation counts per user
CREATE TABLE user_violation_summary (
    user_id UUID NOT NULL,
    server_id UUID NOT NULL,
    violation_count INT DEFAULT 0,
    last_violation_at TIMESTAMPTZ,
    warn_count INT DEFAULT 0,
    mute_count INT DEFAULT 0,
    temp_ban_count INT DEFAULT 0,
    total_score DECIMAL(10,2) DEFAULT 0,
    PRIMARY KEY (user_id, server_id),
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_violation_summary_server ON user_violation_summary(server_id);
CREATE INDEX idx_user_violation_summary_count ON user_violation_summary(violation_count DESC);

-- Moderation Action Rate Limit table (in-memory fallback with persistence)
CREATE TABLE moderation_action_rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    moderator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action_type INT NOT NULL,
    action_count INT DEFAULT 0,
    window_start TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(server_id, moderator_id, action_type, window_start)
);

CREATE INDEX idx_moderation_rate_limits_moderator ON moderation_action_rate_limits(moderator_id);
CREATE INDEX idx_moderation_rate_limits_window ON moderation_action_rate_limits(window_start);

-- +migrate Down

DROP TABLE IF EXISTS moderation_action_rate_limits;
DROP TABLE IF EXISTS user_violation_summary;
DROP TABLE IF EXISTS moderation_logs;
DROP TABLE IF EXISTS moderation_keyword_rules;
DROP TABLE IF EXISTS moderation_settings;
