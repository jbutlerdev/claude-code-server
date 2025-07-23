-- Settings table to store global application settings
CREATE TABLE settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    default_github_token TEXT,
    claude_code_binary VARCHAR(255) NOT NULL DEFAULT '/usr/local/bin/claude',
    max_concurrent_sessions INTEGER NOT NULL DEFAULT 10,
    git_user_name VARCHAR(255),
    git_user_email VARCHAR(320), -- RFC 5321 max length for email
    git_user_token TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create trigger for updated_at
CREATE TRIGGER update_settings_updated_at BEFORE UPDATE ON settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default settings row
INSERT INTO settings (
    claude_code_binary,
    max_concurrent_sessions
) VALUES (
    '/usr/local/bin/claude',
    10
);