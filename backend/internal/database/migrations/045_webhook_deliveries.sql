-- Migration: Add webhook_deliveries table for delivery tracking and auditing
-- Created: 2025-04-06

-- Webhook delivery log for tracking and auditing
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    status_code INTEGER,
    response_body TEXT,
    error_message TEXT,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    delivered_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    request_payload JSONB,
    response_headers JSONB,
    duration_ms INTEGER
);

-- Indexes for common queries
CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at DESC);
CREATE INDEX idx_webhook_deliveries_status_code ON webhook_deliveries(status_code);
CREATE INDEX idx_webhook_deliveries_webhook_created ON webhook_deliveries(webhook_id, created_at DESC);

-- Partial index for failed deliveries (for retry logic and monitoring)
CREATE INDEX idx_webhook_deliveries_failed ON webhook_deliveries(webhook_id, attempt_number) 
    WHERE status_code < 200 OR status_code >= 300;
