-- Migration: 037_forwarded_messages
-- Description: Add forwarded messages table for message forwarding feature
-- Created: 2026-04-01

BEGIN;

-- Create forwarded_messages table
CREATE TABLE IF NOT EXISTS forwarded_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    forwarded_by_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    destination_channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    comment TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX idx_forwarded_messages_original_message ON forwarded_messages(original_message_id);
CREATE INDEX idx_forwarded_messages_forwarded_by ON forwarded_messages(forwarded_by_id);
CREATE INDEX idx_forwarded_messages_destination_channel ON forwarded_messages(destination_channel_id);
CREATE INDEX idx_forwarded_messages_created_at ON forwarded_messages(created_at DESC);

-- Create composite index for checking if a message has been forwarded
CREATE INDEX idx_forwarded_messages_original_message_created ON forwarded_messages(original_message_id, created_at DESC);

COMMIT;
