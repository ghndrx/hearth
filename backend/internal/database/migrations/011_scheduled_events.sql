-- Migration: Scheduled Events
-- Description: Add scheduled events table for server events (stage, voice, external)

-- Create events table
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    creator_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    description TEXT DEFAULT '',
    image_url TEXT,
    scheduled_start TIMESTAMPTZ NOT NULL,
    scheduled_end TIMESTAMPTZ,
    entity_type INT NOT NULL CHECK (entity_type IN (1, 2, 3)), -- 1=stage, 2=voice, 3=external
    location TEXT DEFAULT '',
    status INT NOT NULL DEFAULT 1 CHECK (status IN (1, 2, 3, 4)), -- 1=scheduled, 2=active, 3=completed, 4=cancelled
    user_count INT NOT NULL DEFAULT 0,
    recurrence_rule JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create event_rsvps table for user RSVPs
CREATE TABLE IF NOT EXISTS event_rsvps (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status INT NOT NULL DEFAULT 1 CHECK (status IN (1, 2)), -- 1=interested, 2=going
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, user_id)
);

-- Create indexes for events
CREATE INDEX IF NOT EXISTS idx_events_server_id ON events(server_id);
CREATE INDEX IF NOT EXISTS idx_events_channel_id ON events(channel_id);
CREATE INDEX IF NOT EXISTS idx_events_creator_id ON events(creator_id);
CREATE INDEX IF NOT EXISTS idx_events_scheduled_start ON events(scheduled_start);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);
CREATE INDEX IF NOT EXISTS idx_events_server_status ON events(server_id, status);

-- Create indexes for event_rsvps
CREATE INDEX IF NOT EXISTS idx_event_rsvps_user_id ON event_rsvps(user_id);
CREATE INDEX IF NOT EXISTS idx_event_rsvps_status ON event_rsvps(status);

-- Add comment
COMMENT ON TABLE events IS 'Scheduled events for servers (stage, voice, or external)';
COMMENT ON TABLE event_rsvps IS 'User RSVPs for events';
