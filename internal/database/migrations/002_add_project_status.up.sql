-- Add status and error tracking to projects
ALTER TABLE projects ADD COLUMN description TEXT;
ALTER TABLE projects ADD COLUMN status VARCHAR(50) DEFAULT 'pending';
ALTER TABLE projects ADD COLUMN error_message TEXT;
ALTER TABLE projects ADD COLUMN last_sync_at TIMESTAMP;

-- Create index for status queries
CREATE INDEX idx_projects_status ON projects(status);

-- Update existing projects to 'pending' status
UPDATE projects SET status = 'pending' WHERE status IS NULL;