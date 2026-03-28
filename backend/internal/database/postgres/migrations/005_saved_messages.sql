-- Hearth Database Schema
-- Migration 004: Saved/Bookmarked Messages

-- Saved messages table (bookmarks)
CREATE TABLE saved_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, message_id)
);

CREATE INDEX idx_saved_messages_user ON saved_messages(user_id, created_at DESC);
CREATE INDEX idx_saved_messages_message ON saved_messages(message_id);
