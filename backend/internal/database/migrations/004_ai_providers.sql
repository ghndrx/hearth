-- +migrate Up

-- AI Providers table - stores configured AI provider configurations
CREATE TABLE ai_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_type VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(200) NOT NULL,
    base_url TEXT,
    is_enabled BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    priority INT DEFAULT 100,
    config TEXT, -- Encrypted JSON containing credentials
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for ai_providers
CREATE INDEX idx_ai_providers_type ON ai_providers(provider_type);
CREATE INDEX idx_ai_providers_enabled ON ai_providers(is_enabled) WHERE is_enabled = TRUE;
CREATE INDEX idx_ai_providers_default ON ai_providers(is_default) WHERE is_default = TRUE;
CREATE INDEX idx_ai_providers_priority ON ai_providers(priority);

-- Add comment for documentation
COMMENT ON TABLE ai_providers IS 'Configured AI providers (OpenAI, Anthropic, Ollama, etc.)';
COMMENT ON COLUMN ai_providers.config IS 'Encrypted JSON containing API keys and other credentials';

-- User AI Credentials table - stores user-specific AI provider credentials
CREATE TABLE user_ai_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    provider_type VARCHAR(50) NOT NULL,
    credentials TEXT NOT NULL, -- Encrypted JSON containing user's API keys
    is_enabled BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, provider_id)
);

-- Indexes for user_ai_credentials
CREATE INDEX idx_user_ai_credentials_user ON user_ai_credentials(user_id);
CREATE INDEX idx_user_ai_credentials_provider ON user_ai_credentials(provider_id);
CREATE INDEX idx_user_ai_credentials_enabled ON user_ai_credentials(is_enabled) WHERE is_enabled = TRUE;

-- Add comment for documentation
COMMENT ON TABLE user_ai_credentials IS 'User-specific AI provider credentials (bring your own key)';
COMMENT ON COLUMN user_ai_credentials.credentials IS 'Encrypted JSON containing user API keys';

-- AI Model Routing table - configures which models to use for different features
CREATE TABLE ai_model_routing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE, -- NULL for global routing
    user_id UUID REFERENCES users(id) ON DELETE CASCADE, -- NULL for server/global routing
    feature VARCHAR(50) NOT NULL, -- summary, search, chat, embed, moderate, translate
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    model_id VARCHAR(200) NOT NULL,
    priority INT DEFAULT 100,
    is_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for ai_model_routing
CREATE INDEX idx_ai_model_routing_feature ON ai_model_routing(feature);
CREATE INDEX idx_ai_model_routing_server ON ai_model_routing(server_id) WHERE server_id IS NOT NULL;
CREATE INDEX idx_ai_model_routing_user ON ai_model_routing(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_ai_model_routing_enabled ON ai_model_routing(is_enabled) WHERE is_enabled = TRUE;
CREATE INDEX idx_ai_model_routing_priority ON ai_model_routing(priority);

-- Compound index for efficient routing lookups
CREATE INDEX idx_ai_model_routing_lookup ON ai_model_routing(feature, is_enabled, priority)
    WHERE is_enabled = TRUE;

-- Add comment for documentation
COMMENT ON TABLE ai_model_routing IS 'Per-feature model routing configuration (cheap for summaries, smart for search)';
COMMENT ON COLUMN ai_model_routing.feature IS 'Feature type: summary, search, chat, embed, moderate, translate';

-- AI Usage Tracking table - tracks AI usage for billing/quotas
CREATE TABLE ai_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    feature VARCHAR(50) NOT NULL,
    model_id VARCHAR(200) NOT NULL,
    prompt_tokens INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    latency_ms INT, -- Request latency in milliseconds
    error BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for ai_usage
CREATE INDEX idx_ai_usage_user ON ai_usage(user_id);
CREATE INDEX idx_ai_usage_server ON ai_usage(server_id) WHERE server_id IS NOT NULL;
CREATE INDEX idx_ai_usage_provider ON ai_usage(provider_id);
CREATE INDEX idx_ai_usage_feature ON ai_usage(feature);
CREATE INDEX idx_ai_usage_created ON ai_usage(created_at);

-- Index for usage aggregation queries
CREATE INDEX idx_ai_usage_aggregation ON ai_usage(user_id, created_at, provider_id);

-- Partitioning hint (for future optimization with large datasets)
COMMENT ON TABLE ai_usage IS 'AI usage tracking for billing and quotas. Consider partitioning by created_at for large deployments.';

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_ai_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ai_providers_updated_at
    BEFORE UPDATE ON ai_providers
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_updated_at();

CREATE TRIGGER user_ai_credentials_updated_at
    BEFORE UPDATE ON user_ai_credentials
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_updated_at();

CREATE TRIGGER ai_model_routing_updated_at
    BEFORE UPDATE ON ai_model_routing
    FOR EACH ROW
    EXECUTE FUNCTION update_ai_updated_at();

-- +migrate Down

DROP TRIGGER IF EXISTS ai_model_routing_updated_at ON ai_model_routing;
DROP TRIGGER IF EXISTS user_ai_credentials_updated_at ON user_ai_credentials;
DROP TRIGGER IF EXISTS ai_providers_updated_at ON ai_providers;
DROP FUNCTION IF EXISTS update_ai_updated_at();

DROP TABLE IF EXISTS ai_usage;
DROP TABLE IF EXISTS ai_model_routing;
DROP TABLE IF EXISTS user_ai_credentials;
DROP TABLE IF EXISTS ai_providers;
