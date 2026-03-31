-- Hearth Database Schema
-- Migration 036: Poll System

-- Polls table
CREATE TABLE polls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    is_multiple BOOLEAN NOT NULL DEFAULT FALSE,
    end_time TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for fetching polls by channel
CREATE INDEX idx_polls_channel_id ON polls(channel_id);

-- Index for fetching polls by server (via channel)
CREATE INDEX idx_polls_server ON polls(channel_id);

-- Poll options table
CREATE TABLE poll_options (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    votes INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for fetching options by poll
CREATE INDEX idx_poll_options_poll_id ON poll_options(poll_id);

-- Poll votes table (tracks who voted for what)
CREATE TABLE poll_votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    option_id UUID NOT NULL REFERENCES poll_options(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(poll_id, user_id) -- one vote per user per poll
);

-- Index for checking if user has voted
CREATE INDEX idx_poll_votes_poll_user ON poll_votes(poll_id, user_id);

-- Index for counting votes per option
CREATE INDEX idx_poll_votes_option_id ON poll_votes(option_id);

-- Function to update poll updated_at timestamp
CREATE OR REPLACE FUNCTION update_poll_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update poll timestamp
DROP TRIGGER IF EXISTS trigger_poll_updated_at ON polls;
CREATE TRIGGER trigger_poll_updated_at
    BEFORE UPDATE ON polls
    FOR EACH ROW
    EXECUTE FUNCTION update_poll_timestamp();

-- Trigger to update option vote count on insert
CREATE OR REPLACE FUNCTION increment_poll_option_votes()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE poll_options SET votes = votes + 1 WHERE id = NEW.option_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_poll_vote_increment ON poll_votes;
CREATE TRIGGER trigger_poll_vote_increment
    AFTER INSERT ON poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION increment_poll_option_votes();

-- Trigger to decrement vote count on vote deletion (for vote changing)
CREATE OR REPLACE FUNCTION decrement_poll_option_votes()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE poll_options SET votes = GREATEST(0, votes - 1) WHERE id = OLD.option_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_poll_vote_decrement ON poll_votes;
CREATE TRIGGER trigger_poll_vote_decrement
    AFTER DELETE ON poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION decrement_poll_option_votes();
