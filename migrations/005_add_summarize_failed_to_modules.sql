-- Migration: 005_add_summarize_failed_to_modules
-- Description: Add summarize_failed column to modules table

ALTER TABLE modules ADD COLUMN summarize_failed BOOLEAN DEFAULT FALSE;
