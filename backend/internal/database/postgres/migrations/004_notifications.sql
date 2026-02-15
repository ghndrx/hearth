-- 004_notifications.sql
-- Notifications table for user notifications

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    body VARCHAR(2000) NOT NULL,
    read BOOLEAN NOT NULL DEFAULT false,
    data TEXT, -- JSON encoded extra data
    
    -- References
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    message_id UUID, -- Not enforced as messages may be deleted
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for user's notifications (most common query)
CREATE INDEX idx_notifications_user_id ON notifications(user_id);

-- Index for unread notifications
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, read) WHERE read = false;

-- Index for notifications by type
CREATE INDEX idx_notifications_user_type ON notifications(user_id, type);

-- Index for notifications by creation time (for cleanup)
CREATE INDEX idx_notifications_created_at ON notifications(created_at);

-- Composite index for common list query
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);

-- Add check constraint for valid notification types
ALTER TABLE notifications ADD CONSTRAINT chk_notification_type 
    CHECK (type IN ('mention', 'reply', 'direct_message', 'friend_request', 
                    'friend_accept', 'server_invite', 'server_join', 'reaction', 'system'));
