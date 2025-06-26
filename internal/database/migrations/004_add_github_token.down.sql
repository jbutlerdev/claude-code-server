-- Remove GitHub token support and push tracking
DROP INDEX IF EXISTS idx_sessions_push_branch;
ALTER TABLE sessions DROP COLUMN IF EXISTS last_pushed_at;
ALTER TABLE sessions DROP COLUMN IF EXISTS push_commit_hash;
ALTER TABLE sessions DROP COLUMN IF EXISTS push_branch;
ALTER TABLE projects DROP COLUMN IF EXISTS github_token;