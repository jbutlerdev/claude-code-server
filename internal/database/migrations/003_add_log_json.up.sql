-- Add JSON data column to session_logs for storing parsed Claude output
ALTER TABLE session_logs ADD COLUMN json_data JSONB;

-- Add index for querying JSON data (simplified for compatibility)
CREATE INDEX idx_session_logs_json_data ON session_logs USING GIN (json_data);