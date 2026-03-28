-- User custom statuses table for rich status with emoji and expiration
CREATE TABLE IF NOT EXISTS user_custom_statuses (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    custom_text VARCHAR(128),
    emoji VARCHAR(64),
    emoji_id UUID,
    emoji_name VARCHAR(64),
    clear_after TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for expiration cleanup
CREATE INDEX IF NOT EXISTS idx_user_custom_statuses_clear_after
    ON user_custom_statuses(clear_after)
    WHERE clear_after IS NOT NULL;

-- Auto-update trigger for updated_at
CREATE OR REPLACE TRIGGER update_user_custom_statuses_updated_at
    BEFORE UPDATE ON user_custom_statuses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
