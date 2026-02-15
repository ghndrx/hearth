-- Hearth Database Schema
-- Migration 002: Threads table

-- Threads table
CREATE TABLE threads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    message_count INTEGER DEFAULT 0,
    member_count INTEGER DEFAULT 1,
    archived BOOLEAN DEFAULT FALSE,
    auto_archive INTEGER DEFAULT 1440, -- minutes (default 24 hours)
    locked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    archive_timestamp TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_threads_parent_channel ON threads(parent_channel_id);
CREATE INDEX idx_threads_owner ON threads(owner_id);
CREATE INDEX idx_threads_archived ON threads(archived);
CREATE INDEX idx_threads_created ON threads(created_at DESC);

-- Thread members (users participating in a thread)
CREATE TABLE thread_members (
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (thread_id, user_id)
);

CREATE INDEX idx_thread_members_user ON thread_members(user_id);

-- Thread messages table (separate from main messages for performance)
CREATE TABLE thread_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    edited_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_thread_messages_thread ON thread_messages(thread_id, created_at DESC);
CREATE INDEX idx_thread_messages_author ON thread_messages(author_id);
