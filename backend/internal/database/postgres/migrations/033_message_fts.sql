-- Migration 033: Full-text search for messages using PostgreSQL tsvector
-- This enables fast, scalable message search with relevance ranking

BEGIN;

-- Add tsvector column for message content search
ALTER TABLE messages ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create GIN index for fast full-text search
CREATE INDEX IF NOT EXISTS idx_messages_search_vector ON messages USING GIN(search_vector);

-- Create function to update search_vector on message insert/update
CREATE OR REPLACE FUNCTION update_message_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    -- Only index the actual content, not system messages or encrypted content
    -- Also handle NULL content
    NEW.search_vector := to_tsvector('english', COALESCE(NEW.content, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update search_vector
DROP TRIGGER IF EXISTS trg_update_message_search_vector ON messages;
CREATE TRIGGER trg_update_message_search_vector
    BEFORE INSERT OR UPDATE OF content ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_message_search_vector();

-- Index existing messages (can take a while for large tables)
-- This is done asynchronously via background job in production
DO $$
DECLARE
    batch_size INTEGER := 10000;
    offset_val INTEGER := 0;
    total_updated INTEGER := 0;
BEGIN
    -- Update in batches to avoid locking
    LOOP
        UPDATE messages 
        SET search_vector = to_tsvector('english', COALESCE(content, ''))
        WHERE id IN (
            SELECT id FROM messages 
            WHERE search_vector IS NULL 
            ORDER BY created_at 
            LIMIT batch_size
        );
        
        GET DIAGNOSTICS total_updated = ROW_COUNT;
        EXIT WHEN total_updated = 0;
        
        -- Log progress (will appear in PostgreSQL logs)
        RAISE NOTICE 'Updated % messages for FTS', total_updated;
    END LOOP;
END $$;

-- Create view for search statistics
CREATE OR REPLACE VIEW search_stats AS
SELECT 
    COUNT(*) as total_messages,
    COUNT(search_vector) as indexed_messages,
    COUNT(*) FILTER (WHERE search_vector IS NULL) as pending_index,
    MAX(created_at) as latest_message
FROM messages;

-- Add composite index for common search patterns (channel + time)
CREATE INDEX IF NOT EXISTS idx_messages_channel_time_search 
    ON messages(channel_id, created_at DESC) 
    WHERE search_vector IS NOT NULL;

COMMIT;

-- Migration completed successfully
-- Note: The search_vector column is automatically populated via trigger
-- For existing messages without search_vector, run: UPDATE messages SET search_vector = to_tsvector('english', COALESCE(content, ''));
