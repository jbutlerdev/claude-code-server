-- Drop triggers
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS session_logs;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS projects;