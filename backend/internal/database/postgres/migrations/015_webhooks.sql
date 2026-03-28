-- Webhooks table for channel webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type SMALLINT NOT NULL DEFAULT 1,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(80) NOT NULL,
    avatar TEXT,
    token VARCHAR(128) NOT NULL,
    application_id UUID,
    source_server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    source_channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_webhooks_channel_id ON webhooks(channel_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_server_id ON webhooks(server_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_creator_id ON webhooks(creator_id);

-- Ensure unique tokens
CREATE UNIQUE INDEX IF NOT EXISTS idx_webhooks_token ON webhooks(token);
