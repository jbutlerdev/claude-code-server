-- Add git identity fields to projects
ALTER TABLE projects ADD COLUMN git_user_name TEXT;
ALTER TABLE projects ADD COLUMN git_user_email TEXT;