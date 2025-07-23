-- Remove conversation_id column and index
DROP INDEX IF EXISTS idx_sessions_conversation_id;
ALTER TABLE sessions DROP COLUMN conversation_id;