-- Message Components Migration
-- Migration 025: Add message components support (buttons, select menus, action rows, text inputs)

-- Create component types enum
CREATE TYPE component_type AS ENUM ('action_row', 'button', 'select_menu', 'text_input');

-- Create component style enum
CREATE TYPE component_style AS ENUM ('primary', 'secondary', 'success', 'danger', 'link', 'short', 'paragraph');

-- Create message_components table
CREATE TABLE message_components (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    type component_type NOT NULL,
    style component_style,
    label VARCHAR(100),
    custom_id VARCHAR(100),
    url TEXT,
    disabled BOOLEAN DEFAULT FALSE,
    emoji_id UUID,
    emoji_name VARCHAR(64),
    options JSONB DEFAULT '[]',
    min_values INT,
    max_values INT,
    placeholder VARCHAR(150),
    required BOOLEAN DEFAULT FALSE,
    value TEXT,
    min_length INT,
    max_length INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create component_interactions table
CREATE TABLE component_interactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type INT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    channel_id UUID NOT NULL REFERENCES channels(id),
    message_id UUID NOT NULL REFERENCES messages(id),
    component_id UUID NOT NULL REFERENCES message_components(id),
    custom_id VARCHAR(100),
    values TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for message_components
CREATE INDEX idx_message_components_message ON message_components(message_id);
CREATE INDEX idx_message_components_custom_id ON message_components(custom_id) WHERE custom_id IS NOT NULL;

-- Indexes for component_interactions
CREATE INDEX idx_component_interactions_message ON component_interactions(message_id);
CREATE INDEX idx_component_interactions_component ON component_interactions(component_id);
CREATE INDEX idx_component_interactions_user ON component_interactions(user_id);
