-- Migration 051: Federation Event and State Storage
-- Adds tables for persisting federation events, transactions, and room mappings
-- Part of Phase 3 (HRT-)

-- Track inbound transactions for idempotency
CREATE TABLE IF NOT EXISTS federation_transactions (
    origin VARCHAR(255) NOT NULL,
    txn_id VARCHAR(255) NOT NULL,
    received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    pdu_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (origin, txn_id)
);

CREATE INDEX IF NOT EXISTS idx_federation_transactions_received_at ON federation_transactions(received_at);

-- Persist federated events
CREATE TABLE IF NOT EXISTS federation_events (
    event_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    type TEXT NOT NULL,
    sender TEXT NOT NULL,
    origin_server VARCHAR(255) NOT NULL,
    origin_ts BIGINT NOT NULL,
    content JSONB NOT NULL,
    auth_events TEXT[] NOT NULL DEFAULT '{}',
    prev_events TEXT[] NOT NULL DEFAULT '{}',
    signatures JSONB NOT NULL DEFAULT '{}',
    hashes JSONB NOT NULL DEFAULT '{}',
    depth BIGINT NOT NULL,
    redacts TEXT,
    rejected BOOLEAN NOT NULL DEFAULT FALSE,
    reject_reason TEXT,
    received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id)
);

CREATE INDEX IF NOT EXISTS idx_federation_events_room_id ON federation_events(room_id);
CREATE INDEX IF NOT EXISTS idx_federation_events_room_ts ON federation_events(room_id, origin_ts);
CREATE INDEX IF NOT EXISTS idx_federation_events_sender ON federation_events(sender);
CREATE INDEX IF NOT EXISTS idx_federation_events_type ON federation_events(type);

-- Outbound transaction queue for federation
CREATE TABLE IF NOT EXISTS federation_outbound_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    destination_server VARCHAR(255) NOT NULL,
    pdu JSONB NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_error TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sending', 'sent', 'failed')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_federation_outbound_queue_destination ON federation_outbound_queue(destination_server);
CREATE INDEX IF NOT EXISTS idx_federation_outbound_queue_due ON federation_outbound_queue(next_retry_at) WHERE status IN ('pending', 'failed');
CREATE INDEX IF NOT EXISTS idx_federation_outbound_queue_status ON federation_outbound_queue(status);

-- Map Matrix room IDs to Hearth channel IDs for federation
CREATE TABLE IF NOT EXISTS room_channel_map (
    room_id TEXT NOT NULL,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id)
);

CREATE INDEX IF NOT EXISTS idx_room_channel_map_channel_id ON room_channel_map(channel_id);
CREATE INDEX IF NOT EXISTS idx_room_channel_map_server_id ON room_channel_map(server_id);

-- Persist room state for federation (current state snapshot)
CREATE TABLE IF NOT EXISTS federation_room_state (
    room_id TEXT NOT NULL,
    state_key TEXT NOT NULL,
    sender TEXT NOT NULL,
    type TEXT NOT NULL,
    content JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, state_key, type)
);

CREATE INDEX IF NOT EXISTS idx_federation_room_state_room_id ON federation_room_state(room_id);
CREATE INDEX IF NOT EXISTS idx_federation_room_state_type ON federation_room_state(type);
