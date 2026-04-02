-- +migrate Up

-- Add icon column to channels table for group DM support
ALTER TABLE channels ADD COLUMN IF NOT EXISTS icon TEXT;

-- +migrate Down

ALTER TABLE channels DROP COLUMN IF EXISTS icon;
