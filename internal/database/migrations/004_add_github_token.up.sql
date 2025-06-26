-- Add GitHub token support to projects
ALTER TABLE projects ADD COLUMN github_token TEXT;

-- Add session push tracking
ALTER TABLE sessions ADD COLUMN push_branch VARCHAR(255);
ALTER TABLE sessions ADD COLUMN push_commit_hash VARCHAR(64);
ALTER TABLE sessions ADD COLUMN last_pushed_at TIMESTAMP;

-- Create index for push tracking
CREATE INDEX idx_sessions_push_branch ON sessions(push_branch);