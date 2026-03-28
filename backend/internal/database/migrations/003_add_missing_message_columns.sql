-- Add missing columns for messages table
ALTER TABLE messages ADD COLUMN IF NOT EXISTS encrypted_content TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS pinned_at TIMESTAMPTZ;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS mentions_everyone BOOLEAN DEFAULT FALSE;
