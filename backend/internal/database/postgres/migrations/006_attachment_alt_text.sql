-- Migration 006: Add alt_text column to attachments for accessibility
-- A11Y-004: Image alt text support

ALTER TABLE attachments ADD COLUMN IF NOT EXISTS alt_text TEXT;

-- Add comment for documentation
COMMENT ON COLUMN attachments.alt_text IS 'Alt text description for accessibility (screen readers)';
