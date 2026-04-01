-- Modal Components Migration
-- Migration 038: Add modal support for interactive components

-- Create modal_type enum
CREATE TYPE modal_type AS ENUM ('primary', 'danger');

-- Create modal_components table
CREATE TABLE modal_components (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    custom_id VARCHAR(100) NOT NULL,
    type modal_type NOT NULL DEFAULT 'primary',
    title VARCHAR(100) NOT NULL,
    rows JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create modal_interactions table
CREATE TABLE modal_interactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    channel_id UUID NOT NULL REFERENCES channels(id),
    message_id UUID NOT NULL REFERENCES messages(id),
    modal_id UUID NOT NULL REFERENCES modal_components(id),
    custom_id VARCHAR(100) NOT NULL,
    component_id UUID NOT NULL REFERENCES message_components(id),
    values JSONB DEFAULT '{}',
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for modal_components
CREATE UNIQUE INDEX idx_modal_components_custom_id ON modal_components(custom_id);
CREATE INDEX idx_modal_components_created ON modal_components(created_at);

-- Indexes for modal_interactions
CREATE INDEX idx_modal_interactions_modal ON modal_interactions(modal_id);
CREATE INDEX idx_modal_interactions_user ON modal_interactions(user_id);
CREATE INDEX idx_modal_interactions_message ON modal_interactions(message_id);

-- Add interaction response type to component_interactions
ALTER TABLE component_interactions ADD COLUMN IF NOT EXISTS response_type VARCHAR(50);
ALTER TABLE component_interactions ADD COLUMN IF NOT EXISTS response_data JSONB;
