-- Fix missing owner memberships for existing servers
-- This ensures server owners are members of their own servers

INSERT INTO members (server_id, user_id, joined_at)
SELECT s.id, s.owner_id, s.created_at
FROM servers s
WHERE NOT EXISTS (
    SELECT 1 FROM members m 
    WHERE m.server_id = s.id AND m.user_id = s.owner_id
);
