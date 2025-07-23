-- Remove duplicate git_user_token field from settings table
ALTER TABLE settings DROP COLUMN IF EXISTS git_user_token;