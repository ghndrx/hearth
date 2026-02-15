-- Hearth Database Schema
-- Migration 003: Extended user settings (notifications, privacy)

-- Add notification and privacy settings columns to user_settings
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS notifications_enabled BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS notifications_sound BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS notifications_desktop BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS notifications_mentions_only BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS notifications_dm BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS notifications_server_defaults BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS privacy_dm_from_servers BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS privacy_dm_from_friends_only BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS privacy_show_activity BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS privacy_friend_requests_all BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS privacy_read_receipts BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS locale VARCHAR(10) DEFAULT 'en-US';
