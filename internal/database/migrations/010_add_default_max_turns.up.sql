-- Add default_max_turns column to settings table (nullable like other optional settings)
ALTER TABLE settings ADD COLUMN default_max_turns INTEGER;