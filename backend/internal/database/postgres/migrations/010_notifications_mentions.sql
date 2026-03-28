-- Migration: Push subscriptions and notification preferences
-- Up migration

-- Push subscriptions table for Web Push API
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    UNIQUE(user_id, endpoint)
);

CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user_id ON push_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_expires_at ON push_subscriptions(expires_at) WHERE expires_at IS NOT NULL;

-- Notification preferences table
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    push_enabled BOOLEAN NOT NULL DEFAULT true,
    push_mentions BOOLEAN NOT NULL DEFAULT true,
    push_direct_messages BOOLEAN NOT NULL DEFAULT true,
    push_replies BOOLEAN NOT NULL DEFAULT true,
    push_friend_requests BOOLEAN NOT NULL DEFAULT true,
    push_server_invites BOOLEAN NOT NULL DEFAULT true,
    sound_enabled BOOLEAN NOT NULL DEFAULT true,
    sound_message TEXT NOT NULL DEFAULT 'default',
    sound_mention TEXT NOT NULL DEFAULT 'mention',
    desktop_enabled BOOLEAN NOT NULL DEFAULT true,
    desktop_previews BOOLEAN NOT NULL DEFAULT true,
    do_not_disturb BOOLEAN NOT NULL DEFAULT false,
    do_not_disturb_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add mention_everyone column to messages if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'messages' AND column_name = 'mention_everyone'
    ) THEN
        ALTER TABLE messages ADD COLUMN mention_everyone BOOLEAN NOT NULL DEFAULT false;
    END IF;
END $$;

-- Message mentions table (for storing user mentions in messages)
CREATE TABLE IF NOT EXISTS message_mentions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (message_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_message_mentions_user_id ON message_mentions(user_id);

-- Role mentions table (for storing role mentions in messages)
CREATE TABLE IF NOT EXISTS message_role_mentions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (message_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_message_role_mentions_role_id ON message_role_mentions(role_id);

-- Notifications table enhancements (add unique constraint to prevent duplicates)
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, read) WHERE read = false;
CREATE INDEX IF NOT EXISTS idx_notifications_message_id ON notifications(message_id) WHERE message_id IS NOT NULL;

-- Down migration (commented out, run manually if needed)
-- DROP TABLE IF EXISTS message_role_mentions;
-- DROP TABLE IF EXISTS message_mentions;
-- DROP TABLE IF EXISTS notification_preferences;
-- DROP TABLE IF EXISTS push_subscriptions;
-- ALTER TABLE messages DROP COLUMN IF EXISTS mention_everyone;
