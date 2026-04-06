-- +migrate Up
-- Migration: 044_embed_templates
-- Adds embed templates for reusable rich embed configurations

-- Create embeds table for message embeds
CREATE TABLE IF NOT EXISTS embeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(256),
    description TEXT,
    url VARCHAR(2048),
    color INTEGER,
    author_name VARCHAR(256),
    author_url VARCHAR(2048),
    author_icon VARCHAR(2048),
    footer_text VARCHAR(2048),
    footer_icon VARCHAR(2048),
    image_url VARCHAR(2048),
    thumbnail_url VARCHAR(2048),
    timestamp TIMESTAMP WITH TIME ZONE,
    type VARCHAR(50) NOT NULL DEFAULT 'rich',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create embed_fields table for embed fields
CREATE TABLE IF NOT EXISTS embed_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    embed_id UUID NOT NULL REFERENCES embeds(id) ON DELETE CASCADE,
    name VARCHAR(256) NOT NULL,
    value TEXT NOT NULL,
    inline BOOLEAN NOT NULL DEFAULT FALSE
);

-- Create embed_templates table for saved embed templates
CREATE TABLE IF NOT EXISTS embed_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    title VARCHAR(256),
    description TEXT,
    url VARCHAR(2048),
    color INTEGER,
    author_name VARCHAR(256),
    author_url VARCHAR(2048),
    author_icon VARCHAR(2048),
    footer_text VARCHAR(2048),
    footer_icon VARCHAR(2048),
    image_url VARCHAR(2048),
    thumbnail_url VARCHAR(2048),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_embeds_id ON embeds(id);
CREATE INDEX IF NOT EXISTS idx_embed_fields_embed_id ON embed_fields(embed_id);
CREATE INDEX IF NOT EXISTS idx_embed_templates_user_id ON embed_templates(user_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_embed_templates_user_id;
DROP INDEX IF EXISTS idx_embed_fields_embed_id;
DROP INDEX IF EXISTS idx_embeds_id;
DROP TABLE IF EXISTS embed_templates;
DROP TABLE IF EXISTS embed_fields;
DROP TABLE IF EXISTS embeds;
