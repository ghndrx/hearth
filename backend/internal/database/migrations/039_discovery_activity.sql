-- +migrate Up
-- Migration: 039_discovery_activity
-- Adds activity tracking for server discovery analytics

-- Create discovery_activity table for tracking user interactions
CREATE TABLE IF NOT EXISTS discovery_activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    activity_type VARCHAR(32) NOT NULL, -- view, impression, join, search_click
    source VARCHAR(32) NOT NULL DEFAULT '', -- home, search, category, trending, recommended, featured
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for activity queries
CREATE INDEX idx_discovery_activity_server_id ON discovery_activity(server_id);
CREATE INDEX idx_discovery_activity_created_at ON discovery_activity(created_at);
CREATE INDEX idx_discovery_activity_type ON discovery_activity(activity_type, created_at);
CREATE INDEX idx_discovery_activity_server_date ON discovery_activity(server_id, created_at);

-- Create discovery_daily_stats materialized view for aggregated daily stats
CREATE TABLE IF NOT EXISTS discovery_daily_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    stat_date DATE NOT NULL,
    views INT NOT NULL DEFAULT 0,
    impressions INT NOT NULL DEFAULT 0,
    joins INT NOT NULL DEFAULT 0,
    search_clicks INT NOT NULL DEFAULT 0,
    UNIQUE(server_id, stat_date)
);

CREATE INDEX idx_discovery_daily_stats_server ON discovery_daily_stats(server_id, stat_date DESC);

-- +migrate Down
DROP TABLE IF EXISTS discovery_daily_stats;
DROP TABLE IF EXISTS discovery_activity;
