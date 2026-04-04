-- Migration 044: Server Folders
-- Creates tables for user-organized server folders

-- Server folders table
CREATE TABLE server_folders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES server_folders(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    is_collapsed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_server_folders_user ON server_folders(user_id);
CREATE INDEX idx_server_folders_parent ON server_folders(parent_id);
CREATE INDEX idx_server_folders_user_position ON server_folders(user_id, position);

-- Server-to-folder assignments table
CREATE TABLE server_folder_servers (
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES server_folders(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (server_id, user_id)
);

CREATE INDEX idx_server_folder_servers_user ON server_folder_servers(user_id);
CREATE INDEX idx_server_folder_servers_folder ON server_folder_servers(folder_id);
CREATE INDEX idx_server_folder_servers_user_position ON server_folder_servers(user_id, position);
