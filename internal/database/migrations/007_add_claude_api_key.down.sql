-- Remove Claude API key field from settings table
ALTER TABLE settings DROP COLUMN claude_code_api_key;