-- Add conversation_id column to sessions table for Claude Code conversation tracking
ALTER TABLE sessions ADD COLUMN conversation_id VARCHAR(36);

-- Create index for efficient conversation lookups
CREATE INDEX idx_sessions_conversation_id ON sessions(conversation_id);