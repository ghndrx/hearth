-- Add roles array column to members table
-- This stores role IDs directly on the member for faster lookup

ALTER TABLE members ADD COLUMN IF NOT EXISTS roles UUID[] DEFAULT '{}';

-- Create index for role-based queries
CREATE INDEX IF NOT EXISTS idx_members_roles ON members USING GIN (roles);

-- Migrate existing data from member_roles junction table
UPDATE members m
SET roles = COALESCE(
    (SELECT array_agg(mr.role_id) FROM member_roles mr WHERE mr.server_id = m.server_id AND mr.user_id = m.user_id),
    '{}'
);
