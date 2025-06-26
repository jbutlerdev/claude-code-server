-- Remove status and error tracking from projects
DROP INDEX IF EXISTS idx_projects_status;
ALTER TABLE projects DROP COLUMN IF EXISTS last_sync_at;
ALTER TABLE projects DROP COLUMN IF EXISTS error_message;
ALTER TABLE projects DROP COLUMN IF EXISTS status;
ALTER TABLE projects DROP COLUMN IF EXISTS description;