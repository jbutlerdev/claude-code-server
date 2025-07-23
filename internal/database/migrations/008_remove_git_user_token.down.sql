-- Add back git_user_token field to settings table
ALTER TABLE settings ADD COLUMN git_user_token TEXT;