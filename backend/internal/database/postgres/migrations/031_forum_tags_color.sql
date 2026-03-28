-- Migration 031: Add color field to forum tags
-- Allows tags to have an associated color for visual distinction

ALTER TABLE forum_tags ADD COLUMN color VARCHAR(7) DEFAULT NULL;
