-- Remove JSON data column from session_logs
DROP INDEX IF EXISTS idx_session_logs_json_data;
ALTER TABLE session_logs DROP COLUMN IF EXISTS json_data;