-- Stickers System Migration
-- Migration 023: Add stickers support

-- Create sticker format enum
CREATE TYPE sticker_format AS ENUM ('PNG', 'APNG', 'GIF');

-- Create stickers table
CREATE TABLE stickers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(30) NOT NULL,
    tags TEXT[] DEFAULT '{}',
    url VARCHAR(512) NOT NULL,
    format sticker_format NOT NULL DEFAULT 'PNG',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for sticker queries
CREATE INDEX idx_stickers_server ON stickers(server_id) WHERE server_id IS NOT NULL;
CREATE INDEX idx_stickers_created_by ON stickers(created_by);
CREATE INDEX idx_stickers_name ON stickers(name);

-- Add sticker_id column to messages for sticker messages
ALTER TABLE messages ADD COLUMN sticker_id UUID REFERENCES stickers(id);

-- Create index for sticker messages
CREATE INDEX idx_messages_sticker ON messages(sticker_id) WHERE sticker_id IS NOT NULL;
