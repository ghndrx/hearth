-- Hearth Database Schema
-- Migration 014: Thread autofollow settings
-- FEAT-004: Add per-user thread autofollow preferences

-- Add thread autofollow settings to user_settings
ALTER TABLE user_settings
    ADD COLUMN IF NOT EXISTS thread_auto_follow BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS thread_follow_on_reply BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS thread_default_notification_level VARCHAR(20) DEFAULT 'all';

-- Add index for efficient lookup when processing mentions
CREATE INDEX IF NOT EXISTS idx_user_settings_thread_auto_follow 
ON user_settings(user_id) WHERE thread_auto_follow = TRUE;

-- Add comment for documentation
COMMENT ON COLUMN user_settings.thread_auto_follow IS 'Automatically follow threads when mentioned or added';
COMMENT ON COLUMN user_settings.thread_follow_on_reply IS 'Automatically follow threads when replying to them';
COMMENT ON COLUMN user_settings.thread_default_notification_level IS 'Default notification level for auto-followed threads (all, mentions, none)';
