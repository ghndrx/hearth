-- Soundboard System Migration
-- Migration 024: Add soundboard sounds support

-- Create soundboard_sounds table
CREATE TABLE soundboard_sounds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    emoji_name VARCHAR(100),
    volume FLOAT NOT NULL DEFAULT 1.0 CHECK (volume >= 0.0 AND volume <= 1.0),
    audio_url VARCHAR(512) NOT NULL,
    duration_ms INT NOT NULL CHECK (duration_ms > 0 AND duration_ms <= 5000),
    available BOOLEAN NOT NULL DEFAULT TRUE,
    creator_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for soundboard queries
CREATE INDEX idx_soundboard_sounds_server ON soundboard_sounds(server_id) WHERE server_id IS NOT NULL;
CREATE INDEX idx_soundboard_sounds_creator ON soundboard_sounds(creator_id);
CREATE INDEX idx_soundboard_sounds_name ON soundboard_sounds(name);

-- Add soundboard_sound_id column to messages for soundboard messages
ALTER TABLE messages ADD COLUMN soundboard_sound_id UUID REFERENCES soundboard_sounds(id);

-- Create index for soundboard messages
CREATE INDEX idx_messages_soundboard ON messages(soundboard_sound_id) WHERE soundboard_sound_id IS NOT NULL;
