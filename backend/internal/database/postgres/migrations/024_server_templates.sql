-- Server Templates Migration
-- Migration 024: Add server templates support for JSON export/import of server structure

-- Create server_templates table
CREATE TABLE server_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(32) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    source_server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    creator_id UUID NOT NULL REFERENCES users(id),
    serialized_data JSONB NOT NULL,
    usage_count INT DEFAULT 0,
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for template queries
CREATE INDEX idx_server_templates_code ON server_templates(code);
CREATE INDEX idx_server_templates_creator ON server_templates(creator_id);
CREATE INDEX idx_server_templates_source_server ON server_templates(source_server_id);
CREATE INDEX idx_server_templates_public ON server_templates(is_public) WHERE is_public = TRUE;

-- Function to generate a unique template code
CREATE OR REPLACE FUNCTION generate_template_code()
RETURNS VARCHAR(32) AS $$
DECLARE
    chars TEXT := 'abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789';
    result VARCHAR(32) := '';
    i INT;
BEGIN
    FOR i IN 1..8 LOOP
        result := result || substr(chars, floor(random() * length(chars) + 1)::int, 1);
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Increment usage_count when a template is used
CREATE OR REPLACE FUNCTION increment_template_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE server_templates SET usage_count = usage_count + 1 WHERE code = NEW.code;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
