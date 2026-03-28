-- Server Analytics Migration
-- Provides optimized queries for server insights dashboard

-- Add indexes for analytics queries on messages table
CREATE INDEX IF NOT EXISTS idx_messages_server_created 
    ON messages(channel_id, created_at DESC);

-- Create analytics helper view for message counts
-- Note: Using a regular view instead of materialized view for simplicity
-- Can be upgraded to materialized view with refresh triggers if performance requires
CREATE OR REPLACE VIEW v_channel_message_stats AS
SELECT 
    c.server_id,
    c.id AS channel_id,
    c.name AS channel_name,
    c.type AS channel_type,
    COUNT(m.id) AS message_count,
    COUNT(DISTINCT m.author_id) AS unique_authors,
    MAX(m.created_at) AS last_activity,
    DATE_TRUNC('day', MIN(m.created_at)) AS first_message_date
FROM channels c
LEFT JOIN messages m ON m.channel_id = c.id
WHERE c.server_id IS NOT NULL
GROUP BY c.server_id, c.id, c.name, c.type;

-- Create server_member_snapshots table for historical tracking
-- This tracks daily member counts for growth charts
CREATE TABLE IF NOT EXISTS server_member_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,
    member_count INT NOT NULL DEFAULT 0,
    online_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(server_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_server_member_snapshots_server_date 
    ON server_member_snapshots(server_id, snapshot_date DESC);

-- Create server_activity_hourly for message heatmaps
-- This stores hourly message counts for activity visualization
CREATE TABLE IF NOT EXISTS server_activity_hourly (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    activity_hour TIMESTAMPTZ NOT NULL, -- Truncated to hour
    message_count INT NOT NULL DEFAULT 0,
    unique_users INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(server_id, activity_hour)
);

CREATE INDEX IF NOT EXISTS idx_server_activity_hourly_server_hour 
    ON server_activity_hourly(server_id, activity_hour DESC);

-- Create server_daily_active_users for retention metrics
CREATE TABLE IF NOT EXISTS server_daily_active_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    activity_date DATE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(server_id, activity_date, user_id)
);

CREATE INDEX IF NOT EXISTS idx_server_daily_active_users_server_date 
    ON server_daily_active_users(server_id, activity_date DESC);

-- Function to update hourly activity stats (called on message insert)
CREATE OR REPLACE FUNCTION update_server_activity_hourly()
RETURNS TRIGGER AS $$
DECLARE
    v_server_id UUID;
    v_hour TIMESTAMPTZ;
BEGIN
    -- Get server_id from channel
    SELECT server_id INTO v_server_id FROM channels WHERE id = NEW.channel_id;
    
    -- Skip if not a server channel (DM, etc.)
    IF v_server_id IS NULL THEN
        RETURN NEW;
    END IF;
    
    v_hour := DATE_TRUNC('hour', NEW.created_at);
    
    -- Upsert hourly activity
    INSERT INTO server_activity_hourly (server_id, activity_hour, message_count, unique_users)
    VALUES (v_server_id, v_hour, 1, 1)
    ON CONFLICT (server_id, activity_hour) DO UPDATE SET
        message_count = server_activity_hourly.message_count + 1,
        updated_at = NOW();
    
    -- Upsert daily active user
    INSERT INTO server_daily_active_users (server_id, activity_date, user_id, message_count)
    VALUES (v_server_id, DATE_TRUNC('day', NEW.created_at)::DATE, NEW.author_id, 1)
    ON CONFLICT (server_id, activity_date, user_id) DO UPDATE SET
        message_count = server_daily_active_users.message_count + 1;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger on messages table
DROP TRIGGER IF EXISTS trg_update_server_activity ON messages;
CREATE TRIGGER trg_update_server_activity
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_server_activity_hourly();

-- Function to take daily member snapshot (should be called via cron/scheduler)
CREATE OR REPLACE FUNCTION take_member_snapshots()
RETURNS void AS $$
BEGIN
    INSERT INTO server_member_snapshots (server_id, snapshot_date, member_count)
    SELECT 
        server_id,
        CURRENT_DATE,
        COUNT(*)
    FROM members
    GROUP BY server_id
    ON CONFLICT (server_id, snapshot_date) DO UPDATE SET
        member_count = EXCLUDED.member_count,
        created_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Backfill member snapshots with current data (one-time)
INSERT INTO server_member_snapshots (server_id, snapshot_date, member_count)
SELECT 
    server_id,
    CURRENT_DATE,
    COUNT(*)
FROM members
GROUP BY server_id
ON CONFLICT (server_id, snapshot_date) DO NOTHING;

COMMENT ON TABLE server_member_snapshots IS 'Daily snapshots of server member counts for growth tracking';
COMMENT ON TABLE server_activity_hourly IS 'Hourly message activity aggregates for heatmap visualization';
COMMENT ON TABLE server_daily_active_users IS 'Daily active user tracking for retention metrics (DAU/MAU)';
