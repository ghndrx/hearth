-- Migration 045: Thread Notification Settings
-- Adds thread auto-follow and notification level settings to user_settings

ALTER TABLE user_settings
ADD COLUMN IF NOT EXISTS thread_auto_follow BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN IF NOT EXISTS thread_follow_on_reply BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN IF NOT EXISTS thread_default_notification_level VARCHAR(20) NOT NULL DEFAULT 'all';

-- Add index for potential queries on thread notification level
CREATE INDEX IF NOT EXISTS idx_user_settings_thread_notif ON user_settings(thread_default_notification_level);
