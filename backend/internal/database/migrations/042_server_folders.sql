-- +migrate Up

-- Server Folders table - user-created folders to organize servers
CREATE TABLE IF NOT EXISTS server_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES server_folders(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    position INT NOT NULL DEFAULT 0,
    is_collapsed BOOLEAN NOT NULL DEFAULT FALSE,
    depth INT NOT NULL DEFAULT 0,  -- 0 = root level, 1 = nested once, 2 = nested twice (max 3 levels total)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast lookups by user
CREATE INDEX IF NOT EXISTS idx_server_folders_user ON server_folders(user_id);
-- Index for fast lookups by parent
CREATE INDEX IF NOT EXISTS idx_server_folders_parent ON server_folders(parent_id);
-- Index for ordering folders
CREATE INDEX IF NOT EXISTS idx_server_folders_position ON server_folders(user_id, position);
-- Index for efficient folder tree queries
CREATE INDEX IF NOT EXISTS idx_server_folders_user_depth ON server_folders(user_id, depth);

-- User server folder assignments (which servers are in which folders)
CREATE TABLE IF NOT EXISTS user_server_folder (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES server_folders(id) ON DELETE SET NULL,
    position INT NOT NULL DEFAULT 0,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, server_id)
);

-- Index for fast lookups by folder
CREATE INDEX IF NOT EXISTS idx_user_server_folder_folder ON user_server_folder(folder_id);
-- Index for fast lookups by server
CREATE INDEX IF NOT EXISTS idx_user_server_folder_server ON user_server_folder(server_id);
-- Index for ordering servers within folders
CREATE INDEX IF NOT EXISTS idx_user_server_folder_position ON user_server_folder(user_id, folder_id, position);

-- +migrate Down

-- Drop indexes first
DROP INDEX IF EXISTS idx_server_folders_user;
DROP INDEX IF EXISTS idx_server_folders_parent;
DROP INDEX IF EXISTS idx_server_folders_position;
DROP INDEX IF EXISTS idx_server_folders_user_depth;
DROP INDEX IF EXISTS idx_user_server_folder_folder;
DROP INDEX IF EXISTS idx_user_server_folder_server;
DROP INDEX IF EXISTS idx_user_server_folder_position;

-- Drop tables
DROP TABLE IF EXISTS user_server_folder;
DROP TABLE IF EXISTS server_folders;