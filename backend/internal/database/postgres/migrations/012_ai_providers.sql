-- Migration: AI Provider Support
-- Description: Adds tables for multi-provider AI backend support with encrypted credential storage,
--              user-level API keys, server-level defaults, and per-feature model routing.

-- ai_providers: Server-level provider configurations
CREATE TABLE IF NOT EXISTS ai_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_type VARCHAR(50) NOT NULL, -- openai, anthropic, openrouter, bedrock, vertex_ai, ollama, etc.
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    base_url TEXT, -- Custom endpoint for self-hosted models
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    priority INTEGER NOT NULL DEFAULT 100, -- Lower = higher priority
    config TEXT, -- Encrypted JSON credentials (API keys, etc.)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensure only one default provider
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_providers_default 
    ON ai_providers (is_default) WHERE is_default = TRUE;

CREATE INDEX IF NOT EXISTS idx_ai_providers_type ON ai_providers (provider_type);
CREATE INDEX IF NOT EXISTS idx_ai_providers_enabled ON ai_providers (is_enabled) WHERE is_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_ai_providers_priority ON ai_providers (priority);

-- user_ai_credentials: User-specific encrypted API key storage
CREATE TABLE IF NOT EXISTS user_ai_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    provider_type VARCHAR(50) NOT NULL,
    credentials TEXT NOT NULL, -- Encrypted JSON credentials
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_user_ai_credentials_user ON user_ai_credentials (user_id);
CREATE INDEX IF NOT EXISTS idx_user_ai_credentials_provider ON user_ai_credentials (provider_id);
CREATE INDEX IF NOT EXISTS idx_user_ai_credentials_enabled ON user_ai_credentials (is_enabled) WHERE is_enabled = TRUE;

-- ai_model_routing: Per-feature model routing configuration
-- Allows cheap models for summaries, smart models for search, etc.
CREATE TABLE IF NOT EXISTS ai_model_routing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE, -- NULL = global default
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,     -- NULL = server/global default
    feature VARCHAR(50) NOT NULL, -- summary, search, chat, embed, moderate, translate
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    model_id VARCHAR(255) NOT NULL, -- e.g., gpt-3.5-turbo, claude-3-haiku
    priority INTEGER NOT NULL DEFAULT 100,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure unique routing per scope
    CONSTRAINT unique_global_routing UNIQUE NULLS NOT DISTINCT (feature, server_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_ai_model_routing_feature ON ai_model_routing (feature);
CREATE INDEX IF NOT EXISTS idx_ai_model_routing_server ON ai_model_routing (server_id) WHERE server_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ai_model_routing_user ON ai_model_routing (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ai_model_routing_enabled ON ai_model_routing (is_enabled) WHERE is_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_ai_model_routing_lookup ON ai_model_routing (feature, server_id, user_id, is_enabled);

-- ai_usage_log: Track AI usage for billing/quotas (optional, but recommended)
CREATE TABLE IF NOT EXISTS ai_usage_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    provider_id UUID REFERENCES ai_providers(id) ON DELETE SET NULL,
    feature VARCHAR(50) NOT NULL,
    model_id VARCHAR(255) NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_type VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_usage_log_user ON ai_usage_log (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_usage_log_server ON ai_usage_log (server_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_usage_log_provider ON ai_usage_log (provider_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_usage_log_feature ON ai_usage_log (feature, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_usage_log_time ON ai_usage_log (created_at DESC);

-- Partition ai_usage_log by month for better performance (optional)
-- This requires PostgreSQL 10+ and proper partition maintenance

-- Comments for documentation
COMMENT ON TABLE ai_providers IS 'Server-level AI provider configurations with encrypted credentials';
COMMENT ON TABLE user_ai_credentials IS 'User-specific encrypted API keys for AI providers';
COMMENT ON TABLE ai_model_routing IS 'Per-feature model routing (cheap models for summaries, smart for search)';
COMMENT ON TABLE ai_usage_log IS 'AI usage tracking for billing, quotas, and analytics';

COMMENT ON COLUMN ai_providers.config IS 'Encrypted JSON containing API keys and provider-specific settings';
COMMENT ON COLUMN ai_providers.base_url IS 'Custom endpoint URL for self-hosted models or proxies';
COMMENT ON COLUMN ai_providers.priority IS 'Lower values = higher priority for fallback ordering';

COMMENT ON COLUMN user_ai_credentials.credentials IS 'AES-GCM encrypted JSON containing user API keys';

COMMENT ON COLUMN ai_model_routing.feature IS 'Feature type: summary, search, chat, embed, moderate, translate';
COMMENT ON COLUMN ai_model_routing.server_id IS 'NULL for global routing, set for server-specific routing';
COMMENT ON COLUMN ai_model_routing.user_id IS 'NULL for server/global routing, set for user-specific routing';
