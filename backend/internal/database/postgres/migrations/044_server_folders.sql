-- Migration 044: Server Folders
-- Creates tables for user-created server organization folders

-- Server folders table (hierarchical folder structure)
CREATE TABLE server_folders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES server_folders(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    is_collapsed BOOLEAN NOT NULL DEFAULT false,
    depth INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT chk_folder_name_length CHECK (char_length(name) >= 1 AND char_length(name) <= 100),
    CONSTRAINT chk_folder_depth CHECK (depth >= 0 AND depth <= 2),
    CONSTRAINT chk_no_self_parent CHECK (id != parent_id)
);

-- User to server folder assignments (many-to-many via join table)
CREATE TABLE user_server_folder (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES server_folders(id) ON DELETE SET NULL,
    position INTEGER NOT NULL DEFAULT 0,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Composite primary key ensures each server is only in one folder per user
    PRIMARY KEY (user_id, server_id)
);

-- Indexes for server_folders
CREATE INDEX idx_server_folders_user_id ON server_folders(user_id);
CREATE INDEX idx_server_folders_parent_id ON server_folders(parent_id);
CREATE INDEX idx_server_folders_user_position ON server_folders(user_id, position);
CREATE INDEX idx_server_folders_user_parent ON server_folders(user_id, parent_id);

-- Indexes for user_server_folder
CREATE INDEX idx_user_server_folder_user_id ON user_server_folder(user_id);
CREATE INDEX idx_user_server_folder_server_id ON user_server_folder(server_id);
CREATE INDEX idx_user_server_folder_folder_id ON user_server_folder(folder_id);
CREATE INDEX idx_user_server_folder_user_position ON user_server_folder(user_id, folder_id, position);

-- Trigger to update updated_at on server_folders
CREATE OR REPLACE FUNCTION update_server_folders_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_server_folders_updated
    BEFORE UPDATE ON server_folders
    FOR EACH ROW
    EXECUTE FUNCTION update_server_folders_timestamp();
