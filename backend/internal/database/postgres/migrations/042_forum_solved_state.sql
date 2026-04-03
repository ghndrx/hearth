-- Migration 042: Forum Post Solved State and User Sort Preferences
-- Adds solved/answered state to forum posts and user sort preferences

-- Add solved state columns to threads table
ALTER TABLE threads ADD COLUMN is_solved BOOLEAN DEFAULT FALSE;
ALTER TABLE threads ADD COLUMN solved_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE threads ADD COLUMN solved_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE threads ADD COLUMN solved_message_id UUID REFERENCES messages(id) ON DELETE SET NULL;

-- Create index for finding solved/unsolved posts quickly
CREATE INDEX idx_threads_is_solved ON threads(parent_channel_id, is_solved) WHERE is_solved = TRUE;
CREATE INDEX idx_threads_solved_by ON threads(solved_by) WHERE solved_by IS NOT NULL;

-- Forum sort preferences (per-user, per-channel)
CREATE TABLE forum_sort_preferences (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0, -- 0=latest_activity, 1=creation_date, 2=pin_weight, 3=most_reactions
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, channel_id)
);

-- Index for fast lookups
CREATE INDEX idx_forum_sort_preferences_channel ON forum_sort_preferences(channel_id);

-- Forum post view tracking (for "recently viewed" sorting option)
CREATE TABLE forum_post_views (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    last_viewed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, thread_id)
);

-- Index for finding recently viewed posts
CREATE INDEX idx_forum_post_views_user ON forum_post_views(user_id, last_viewed_at DESC);

-- Function to update thread solved state
CREATE OR REPLACE FUNCTION update_thread_solved_state()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_solved = TRUE AND OLD.is_solved = FALSE THEN
        -- Marking as solved
        NEW.solved_at = NOW();
    ELSIF NEW.is_solved = FALSE AND OLD.is_solved = TRUE THEN
        -- Unmarking as solved
        NEW.solved_at = NULL;
        NEW.solved_by = NULL;
        NEW.solved_message_id = NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically set solved_at timestamp when is_solved changes
DROP TRIGGER IF EXISTS trigger_thread_solved_state ON threads;
CREATE TRIGGER trigger_thread_solved_state
    BEFORE UPDATE OF is_solved ON threads
    FOR EACH ROW
    EXECUTE FUNCTION update_thread_solved_state();
