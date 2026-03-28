-- +migrate Up

-- AI Chat Conversations table - stores user conversations with AI
CREATE TABLE ai_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL DEFAULT 'New Chat',
    model_id VARCHAR(200), -- Preferred model for this conversation
    provider_id UUID REFERENCES ai_providers(id) ON DELETE SET NULL,
    system_prompt TEXT, -- Custom system prompt for this conversation
    temperature REAL DEFAULT 0.7,
    max_tokens INT DEFAULT 2048,
    is_archived BOOLEAN DEFAULT FALSE,
    is_pinned BOOLEAN DEFAULT FALSE,
    message_count INT DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for ai_conversations
CREATE INDEX idx_ai_conversations_user ON ai_conversations(user_id);
CREATE INDEX idx_ai_conversations_user_active ON ai_conversations(user_id, is_archived) WHERE is_archived = FALSE;
CREATE INDEX idx_ai_conversations_user_pinned ON ai_conversations(user_id, is_pinned) WHERE is_pinned = TRUE;
CREATE INDEX idx_ai_conversations_last_message ON ai_conversations(last_message_at DESC);
CREATE INDEX idx_ai_conversations_created ON ai_conversations(created_at DESC);

-- Add comment for documentation
COMMENT ON TABLE ai_conversations IS 'User conversations with AI assistants';
COMMENT ON COLUMN ai_conversations.system_prompt IS 'Custom system prompt that defines AI behavior for this conversation';

-- AI Chat Messages table - stores individual messages in conversations
CREATE TABLE ai_chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content TEXT NOT NULL,
    
    -- Tool/function call support
    tool_calls JSONB, -- Array of tool calls made by assistant
    tool_call_id VARCHAR(100), -- ID of the tool call this message responds to
    name VARCHAR(100), -- Name for tool messages
    
    -- Token usage and metadata
    tokens_used INT,
    model_used VARCHAR(200),
    provider_used VARCHAR(50),
    finish_reason VARCHAR(50), -- stop, length, tool_calls, error
    
    -- For regeneration/editing
    is_edited BOOLEAN DEFAULT FALSE,
    parent_message_id UUID REFERENCES ai_chat_messages(id) ON DELETE SET NULL,
    
    -- Error tracking
    error_message TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for ai_chat_messages
CREATE INDEX idx_ai_chat_messages_conversation ON ai_chat_messages(conversation_id);
CREATE INDEX idx_ai_chat_messages_conversation_created ON ai_chat_messages(conversation_id, created_at ASC);
CREATE INDEX idx_ai_chat_messages_role ON ai_chat_messages(role);
CREATE INDEX idx_ai_chat_messages_parent ON ai_chat_messages(parent_message_id) WHERE parent_message_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON TABLE ai_chat_messages IS 'Individual messages within AI conversations';
COMMENT ON COLUMN ai_chat_messages.tool_calls IS 'JSON array of tool/function calls made by the assistant';
COMMENT ON COLUMN ai_chat_messages.finish_reason IS 'Reason for generation completion: stop, length, tool_calls, error';

-- AI Chat Templates table - reusable conversation starters/prompts
CREATE TABLE ai_chat_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE, -- NULL for system templates
    name VARCHAR(100) NOT NULL,
    description TEXT,
    system_prompt TEXT NOT NULL,
    initial_messages JSONB, -- Array of initial messages to seed conversation
    suggested_prompts JSONB, -- Array of suggested follow-up prompts
    icon VARCHAR(50), -- Emoji or icon identifier
    category VARCHAR(50), -- coding, writing, analysis, creative, etc.
    is_public BOOLEAN DEFAULT FALSE, -- Can other users use this template?
    usage_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for ai_chat_templates
CREATE INDEX idx_ai_chat_templates_user ON ai_chat_templates(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_ai_chat_templates_public ON ai_chat_templates(is_public) WHERE is_public = TRUE;
CREATE INDEX idx_ai_chat_templates_category ON ai_chat_templates(category);
CREATE INDEX idx_ai_chat_templates_usage ON ai_chat_templates(usage_count DESC);

-- Add comment for documentation
COMMENT ON TABLE ai_chat_templates IS 'Reusable conversation templates with system prompts and initial messages';

-- AI Conversation Shares table - share conversations with other users
CREATE TABLE ai_conversation_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    shared_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_code VARCHAR(32) NOT NULL UNIQUE, -- Public share link code
    is_public BOOLEAN DEFAULT FALSE, -- Accessible without auth
    can_continue BOOLEAN DEFAULT FALSE, -- Can recipients continue the conversation
    expires_at TIMESTAMPTZ, -- NULL for no expiry
    view_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for ai_conversation_shares
CREATE INDEX idx_ai_conversation_shares_conversation ON ai_conversation_shares(conversation_id);
CREATE INDEX idx_ai_conversation_shares_code ON ai_conversation_shares(share_code);
CREATE INDEX idx_ai_conversation_shares_user ON ai_conversation_shares(shared_by);

-- Add comment for documentation
COMMENT ON TABLE ai_conversation_shares IS 'Shareable links for AI conversations';

-- Update ai_conversations message_count and last_message_at on insert
CREATE OR REPLACE FUNCTION update_ai_conversation_message_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE ai_conversations
    SET 
        message_count = message_count + 1,
        last_message_at = NEW.created_at,
        updated_at = NOW()
    WHERE id = NEW.conversation_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ai_chat_messages_stats_trigger
    AFTER INSERT ON ai_chat_messages
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_conversation_message_stats();

-- Update conversation message_count on delete
CREATE OR REPLACE FUNCTION decrement_ai_conversation_message_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE ai_conversations
    SET 
        message_count = GREATEST(0, message_count - 1),
        updated_at = NOW()
    WHERE id = OLD.conversation_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ai_chat_messages_delete_trigger
    AFTER DELETE ON ai_chat_messages
    FOR EACH ROW
    EXECUTE FUNCTION decrement_ai_conversation_message_count();

-- Create the update_ai_updated_at function for timestamp management
CREATE OR REPLACE FUNCTION update_ai_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update updated_at timestamps
CREATE TRIGGER ai_conversations_updated_at
    BEFORE UPDATE ON ai_conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_updated_at();

CREATE TRIGGER ai_chat_messages_updated_at
    BEFORE UPDATE ON ai_chat_messages
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_updated_at();

CREATE TRIGGER ai_chat_templates_updated_at
    BEFORE UPDATE ON ai_chat_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_updated_at();

-- Insert some default system templates
INSERT INTO ai_chat_templates (id, name, description, system_prompt, icon, category, is_public) VALUES
(gen_random_uuid(), 'General Assistant', 'A helpful AI assistant for any task', 'You are a helpful, harmless, and honest AI assistant. Answer questions clearly and concisely. If you are unsure about something, say so.', '💬', 'general', true),
(gen_random_uuid(), 'Code Helper', 'Expert coding assistant for programming tasks', 'You are an expert software engineer. Help with coding questions, debugging, code review, and explaining programming concepts. Provide clear, well-documented code examples. Follow best practices and modern conventions.', '💻', 'coding', true),
(gen_random_uuid(), 'Creative Writer', 'Assistant for creative writing and storytelling', 'You are a creative writing assistant. Help with stories, poetry, scripts, and other creative content. Be imaginative, engaging, and adapt your style to the user''s requests. Offer constructive feedback on writing.', '✍️', 'creative', true),
(gen_random_uuid(), 'Data Analyst', 'Help with data analysis and visualization', 'You are a data analysis expert. Help analyze data, create visualizations, write SQL queries, and explain statistical concepts. Be precise with numbers and clearly explain your reasoning.', '📊', 'analysis', true),
(gen_random_uuid(), 'Translator', 'Multi-language translation assistant', 'You are a professional translator. Translate text between languages accurately while preserving meaning, tone, and context. Explain cultural nuances when relevant. Support both formal and informal translations.', '🌐', 'translation', true);

-- +migrate Down

DROP TRIGGER IF EXISTS ai_chat_messages_delete_trigger ON ai_chat_messages;
DROP TRIGGER IF EXISTS ai_chat_messages_stats_trigger ON ai_chat_messages;
DROP TRIGGER IF EXISTS ai_chat_templates_updated_at ON ai_chat_templates;
DROP TRIGGER IF EXISTS ai_chat_messages_updated_at ON ai_chat_messages;
DROP TRIGGER IF EXISTS ai_conversations_updated_at ON ai_conversations;

DROP FUNCTION IF EXISTS decrement_ai_conversation_message_count();
DROP FUNCTION IF EXISTS update_ai_conversation_message_stats();

DROP TABLE IF EXISTS ai_conversation_shares;
DROP TABLE IF EXISTS ai_chat_templates;
DROP TABLE IF EXISTS ai_chat_messages;
DROP TABLE IF EXISTS ai_conversations;
