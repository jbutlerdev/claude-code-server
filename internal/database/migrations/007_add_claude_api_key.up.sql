-- Add Claude API key field to settings table
ALTER TABLE settings ADD COLUMN claude_code_api_key TEXT;