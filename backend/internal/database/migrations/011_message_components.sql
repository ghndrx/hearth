-- Migration: Message Components
-- Description: Add message components (buttons, select menus, action rows) and component interactions

-- Create message_components table
CREATE TABLE IF NOT EXISTS message_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('action_row', 'button', 'select_menu', 'text_input')),
    style VARCHAR(20) CHECK (style IN ('primary', 'secondary', 'success', 'danger', 'link', 'short', 'paragraph')),
    label VARCHAR(80),
    custom_id VARCHAR(100),
    url VARCHAR(500),
    disabled BOOLEAN DEFAULT FALSE,
    emoji JSONB,
    placeholder VARCHAR(150),
    min_values INTEGER,
    max_values INTEGER,
    required BOOLEAN DEFAULT FALSE,
    value TEXT,
    min_length INTEGER,
    max_length INTEGER,
    options JSONB DEFAULT '[]',
    parent_id UUID REFERENCES message_components(id) ON DELETE CASCADE,
    position INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create component_interactions table
CREATE TABLE IF NOT EXISTS component_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    component_id UUID NOT NULL REFERENCES message_components(id) ON DELETE CASCADE,
    custom_id VARCHAR(100) NOT NULL,
    type VARCHAR(30) NOT NULL CHECK (type IN ('button_click', 'select_menu_select', 'text_input_submit')),
    values JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for message_components
CREATE INDEX IF NOT EXISTS idx_message_components_message_id ON message_components(message_id);
CREATE INDEX IF NOT EXISTS idx_message_components_parent_id ON message_components(parent_id);
CREATE INDEX IF NOT EXISTS idx_message_components_custom_id ON message_components(custom_id);

-- Indexes for component_interactions
CREATE INDEX IF NOT EXISTS idx_component_interactions_message_id ON component_interactions(message_id);
CREATE INDEX IF NOT EXISTS idx_component_interactions_user_id ON component_interactions(user_id);
CREATE INDEX IF NOT EXISTS idx_component_interactions_component_id ON component_interactions(component_id);
CREATE INDEX IF NOT EXISTS idx_component_interactions_channel_id ON component_interactions(channel_id);
CREATE INDEX IF NOT EXISTS idx_component_interactions_custom_id ON component_interactions(custom_id);

COMMENT ON TABLE message_components IS 'Interactive components attached to messages (buttons, select menus, etc.)';
COMMENT ON TABLE component_interactions IS 'User interactions with message components';
