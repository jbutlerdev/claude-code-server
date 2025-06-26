-- Remove git identity fields from projects
ALTER TABLE projects DROP COLUMN IF EXISTS git_user_email;
ALTER TABLE projects DROP COLUMN IF EXISTS git_user_name;