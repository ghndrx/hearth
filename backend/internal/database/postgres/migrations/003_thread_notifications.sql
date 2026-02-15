-- Hearth Database Schema
-- Migration 003: Thread notification preferences and presence

-- Thread notification preferences table
CREATE TABLE thread_notification_preferences (
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    level VARCHAR(20) NOT NULL DEFAULT 'all', -- 'all', 'mentions', 'none'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (thread_id, user_id)
);

CREATE INDEX idx_thread_notification_prefs_user ON thread_notification_preferences(user_id);

-- Thread presence (who is currently viewing a thread)
-- This is ephemeral and could be stored in Redis, but we use DB for simplicity
CREATE TABLE thread_presence (
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (thread_id, user_id)
);

CREATE INDEX idx_thread_presence_thread ON thread_presence(thread_id);
CREATE INDEX idx_thread_presence_last_seen ON thread_presence(last_seen_at);

-- Function to clean up stale presence records (older than 5 minutes)
CREATE OR REPLACE FUNCTION cleanup_stale_thread_presence() 
RETURNS void AS $$
BEGIN
    DELETE FROM thread_presence 
    WHERE last_seen_at < NOW() - INTERVAL '5 minutes';
END;
$$ LANGUAGE plpgsql;

-- Add notification_type to NotificationType enum for thread notifications
-- Note: This assumes you want to extend the notification system
ALTER TABLE notifications 
ADD COLUMN IF NOT EXISTS thread_id UUID REFERENCES threads(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_notifications_thread ON notifications(thread_id) WHERE thread_id IS NOT NULL;
