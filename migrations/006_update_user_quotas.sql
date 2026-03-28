-- Migration: Update user quotas and role
ALTER TABLE users ADD COLUMN quiz_quota INT DEFAULT 1;
ALTER TABLE users ADD COLUMN summarize_quota INT DEFAULT 1;

-- Seed existing users if needed (default handles it)
