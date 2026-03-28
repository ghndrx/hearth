-- 022_fix_channels_type.sql
-- Fix channels.type column to use VARCHAR instead of INTEGER
-- This aligns the schema with the Go code which uses string ChannelType

-- Drop the view that depends on the column
DROP VIEW IF EXISTS v_channel_message_stats;

-- Check if the column is already VARCHAR (migration is idempotent)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'channels' 
        AND column_name = 'type' 
        AND data_type = 'integer'
    ) THEN
        -- Convert integer type to string
        ALTER TABLE channels 
        ALTER COLUMN type TYPE VARCHAR(32) 
        USING CASE 
            WHEN type = 0 THEN 'text'
            WHEN type = 1 THEN 'voice'
            WHEN type = 2 THEN 'category'
            WHEN type = 3 THEN 'announcement'
            WHEN type = 4 THEN 'forum'
            WHEN type = 5 THEN 'stage'
            WHEN type = 6 THEN 'dm'
            WHEN type = 7 THEN 'group_dm'
            ELSE 'text'
        END;
        
        ALTER TABLE channels ALTER COLUMN type SET DEFAULT 'text';
    END IF;
END $$;

-- Recreate the view
CREATE OR REPLACE VIEW v_channel_message_stats AS
SELECT c.server_id,
    c.id AS channel_id,
    c.name AS channel_name,
    c.type AS channel_type,
    count(m.id) AS message_count,
    count(DISTINCT m.author_id) AS unique_authors,
    max(m.created_at) AS last_activity,
    date_trunc('day'::text, min(m.created_at)) AS first_message_date
FROM channels c
LEFT JOIN messages m ON m.channel_id = c.id
WHERE c.server_id IS NOT NULL
GROUP BY c.server_id, c.id, c.name, c.type;
