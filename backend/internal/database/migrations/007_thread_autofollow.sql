-- Migration: 007_thread_autofollow.sql
-- FEAT-004: Thread Autofollow Settings
-- Adds thread autofollow preference columns to user_settings table

-- Add thread autofollow columns with safe defaults
ALTER TABLE user_settings
ADD COLUMN IF NOT EXISTS thread_auto_follow BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN IF NOT EXISTS thread_follow_on_reply BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN IF NOT EXISTS thread_default_notification_level VARCHAR(20) NOT NULL DEFAULT 'all';

-- Add check constraint for valid notification levels
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'user_settings_thread_notif_level_check'
    ) THEN
        ALTER TABLE user_settings
        ADD CONSTRAINT user_settings_thread_notif_level_check
        CHECK (thread_default_notification_level IN ('all', 'mentions', 'none'));
    END IF;
END $$;

-- Index for efficient batch lookup of users with autofollow enabled
-- Used when processing mentions to find users who should be auto-added
CREATE INDEX IF NOT EXISTS idx_user_settings_thread_autofollow 
ON user_settings(thread_auto_follow) 
WHERE thread_auto_follow = true;

-- Comment on columns
COMMENT ON COLUMN user_settings.thread_auto_follow IS 'Automatically follow threads when mentioned or added';
COMMENT ON COLUMN user_settings.thread_follow_on_reply IS 'Automatically follow threads when replying';
COMMENT ON COLUMN user_settings.thread_default_notification_level IS 'Default notification level for auto-followed threads (all, mentions, none)';
