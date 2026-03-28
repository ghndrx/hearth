-- Migration 026: Auto-Moderation System
-- Creates tables for auto-mod rules and alerts

-- AutoMod rules table
CREATE TABLE automod_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    event_type INTEGER NOT NULL,
    trigger_type INTEGER NOT NULL,
    trigger JSONB NOT NULL DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_automod_rules_server_id ON automod_rules(server_id);
CREATE INDEX idx_automod_rules_enabled ON automod_rules(server_id, enabled);

-- AutoMod alerts table
CREATE TABLE automod_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id UUID NOT NULL REFERENCES automod_rules(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES users(id),
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    message_id UUID,
    content TEXT NOT NULL,
    action_taken INTEGER NOT NULL,
    resolved BOOLEAN NOT NULL DEFAULT false,
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMP WITH TIME ZONE,
    matched_keyword VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_automod_alerts_server_id ON automod_alerts(server_id);
CREATE INDEX idx_automod_alerts_rule_id ON automod_alerts(rule_id);
CREATE INDEX idx_automod_alerts_member_id ON automod_alerts(member_id);
CREATE INDEX idx_automod_alerts_resolved ON automod_alerts(server_id, resolved);
CREATE INDEX idx_automod_alerts_created_at ON automod_alerts(created_at DESC);

-- AutoMod rule statistics table
CREATE TABLE automod_rule_stats (
    rule_id UUID PRIMARY KEY REFERENCES automod_rules(id) ON DELETE CASCADE,
    trigger_count INTEGER NOT NULL DEFAULT 0,
    block_count INTEGER NOT NULL DEFAULT 0,
    flag_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
