# Claude Code Server API Documentation

## Overview

The Claude Code Server provides a RESTful API for managing headless Claude Code sessions. This API enables AI tool libraries and other applications to leverage Claude Code's capabilities programmatically.

## Authentication

All API endpoints require authentication using Bearer tokens in the Authorization header:

```http
Authorization: Bearer your-api-key-here
```

For development, use `dev-key` as the API key when JWT_SECRET is set to the default value.

## Base URL

```
http://localhost:8080/api
```

## Content Type

All requests and responses use JSON format:

```http
Content-Type: application/json
```

## Rate Limiting

The API implements rate limiting with a default of 100 requests per minute per IP address. Rate limit headers are included in responses:

- `X-RateLimit-Limit`: Request limit per window
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Time when rate limit resets

## Error Responses

All errors follow a consistent format:

```json
{
  "error": "Short error description",
  "message": "Detailed error message",
  "code": "ERROR_CODE"
}
```

Common HTTP status codes:
- `400`: Bad Request - Invalid input
- `401`: Unauthorized - Missing or invalid API key
- `404`: Not Found - Resource doesn't exist
- `429`: Too Many Requests - Rate limit exceeded
- `500`: Internal Server Error - Server-side error

## Endpoints

### Health Check

#### GET /health

Check API health status.

**Response:**
```json
{
  "status": "ok"
}
```

### Sessions

#### POST /sessions

Create a new Claude Code session.

**Request Body:**
```json
{
  "prompt": "Help me refactor this code for better performance",
  "project_id": "123e4567-e89b-12d3-a456-426614174000",
  "model": "claude-3-5-sonnet-20241022",
  "max_turns": 25
}
```

**Parameters:**
- `prompt` (required): The initial prompt for Claude Code
- `project_id` (required): UUID of the project to run the session in
- `model` (optional): Claude model to use (default: claude-3-5-sonnet-20241022)
- `max_turns` (optional): Maximum number of turns (default: 25)

**Response (201):**
```json
{
  "id": "987fcdeb-51a2-43d7-8f9e-123456789abc",
  "status": "starting",
  "created_at": "2024-01-15T10:30:00Z",
  "websocket_url": "ws://localhost:8080/api/sessions/987fcdeb-51a2-43d7-8f9e-123456789abc/logs",
  "worktree_path": "/var/lib/claude-code-server/worktrees/session-123"
}
```

#### GET /sessions

List sessions with pagination and filtering.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 10, max: 100)
- `status` (optional): Filter by status (starting, running, completed, failed)
- `project_id` (optional): Filter by project UUID

**Response (200):**
```json
{
  "items": [
    {
      "id": "987fcdeb-51a2-43d7-8f9e-123456789abc",
      "project_id": "123e4567-e89b-12d3-a456-426614174000",
      "initial_prompt": "Help me refactor this code",
      "model": "claude-3-5-sonnet-20241022",
      "max_turns": 25,
      "status": "completed",
      "worktree_id": "session-123",
      "worktree_path": "/var/lib/claude-code-server/worktrees/session-123",
      "started_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T10:35:30Z",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:35:30Z"
    }
  ],
  "page": 1,
  "page_size": 10,
  "total_items": 1,
  "total_pages": 1
}
```

#### GET /sessions/{id}

Get details of a specific session.

**Response (200):**
```json
{
  "id": "987fcdeb-51a2-43d7-8f9e-123456789abc",
  "project_id": "123e4567-e89b-12d3-a456-426614174000",
  "initial_prompt": "Help me refactor this code",
  "model": "claude-3-5-sonnet-20241022",
  "max_turns": 25,
  "status": "completed",
  "worktree_id": "session-123",
  "worktree_path": "/var/lib/claude-code-server/worktrees/session-123",
  "started_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:35:30Z",
  "error": null,
  "metadata": {},
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:35:30Z"
}
```

#### POST /sessions/{id}/resume

Resume a completed session with a new prompt.

**Request Body:**
```json
{
  "prompt": "Now help me add unit tests"
}
```

**Response (202):** Accepted - New session will be created

#### DELETE /sessions/{id}

Cancel a running session.

**Response (204):** No Content

#### POST /sessions/{id}/push

Push session changes to GitHub repository.

**Request Body:**
```json
{
  "commit_message": "Updates from Claude Code session"
}
```

**Parameters:**
- `commit_message` (optional): Custom commit message

**Response (200):**
```json
{
  "branch": "claude-session-abc123-20240115-103000",
  "commit_hash": "a1b2c3d4e5f6...",
  "pushed_at": "2024-01-15T10:35:45Z"
}
```

#### GET /sessions/{id}/logs

Stream session logs in real-time.

**For active sessions:** Returns Server-Sent Events stream
**For completed sessions:** Returns JSON array of logs

**SSE Response:**
```
data: {"id": 1, "session_id": "987fcdeb-51a2-43d7-8f9e-123456789abc", "log_type": "stdout", "content": "Starting analysis...", "created_at": "2024-01-15T10:30:01Z"}

data: {"id": 2, "session_id": "987fcdeb-51a2-43d7-8f9e-123456789abc", "log_type": "stdout", "content": "Found 3 files to analyze", "created_at": "2024-01-15T10:30:02Z"}
```

**JSON Response:**
```json
[
  {
    "id": 1,
    "session_id": "987fcdeb-51a2-43d7-8f9e-123456789abc",
    "log_type": "stdout",
    "content": "Starting analysis...",
    "created_at": "2024-01-15T10:30:01Z"
  }
]
```

### Projects

#### POST /projects

Register a new git repository as a project.

**Request Body:**
```json
{
  "name": "My Web App",
  "path": "/path/to/repository",
  "description": "A sample web application",
  "remote_url": "https://github.com/user/repo.git",
  "github_token": "ghp_...",
  "git_user_name": "John Doe",
  "git_user_email": "john@example.com"
}
```

**Parameters:**
- `name` (required): Human-readable project name
- `path` (required): Local path to git repository
- `description` (optional): Project description
- `remote_url` (optional): Git remote URL
- `github_token` (optional): Project-specific GitHub token
- `git_user_name` (optional): Git author name for commits
- `git_user_email` (optional): Git author email for commits

**Response (201):**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "My Web App",
  "path": "/path/to/repository",
  "remote_url": "https://github.com/user/repo.git",
  "default_branch": "main",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

#### GET /projects

List all registered projects.

**Response (200):**
```json
[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "My Web App",
    "path": "/path/to/repository",
    "remote_url": "https://github.com/user/repo.git",
    "default_branch": "main",
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  }
]
```

#### GET /projects/{id}

Get details of a specific project.

**Response (200):**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "My Web App",
  "path": "/path/to/repository",
  "remote_url": "https://github.com/user/repo.git",
  "default_branch": "main",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

#### PUT /projects/{id}

Update project configuration.

**Request Body:**
```json
{
  "name": "Updated Project Name",
  "remote_url": "https://github.com/user/new-repo.git"
}
```

**Response (200):** Updated project object

#### DELETE /projects/{id}

Remove project from registry (does not delete files).

**Response (204):** No Content

### Settings

#### GET /settings

Get system configuration.

**Response (200):**
```json
{
  "default_github_token": "ghp_...",
  "claude_binary": "/usr/local/bin/claude",
  "max_concurrent_sessions": 10,
  "git_user_name": "Claude Code Server",
  "git_user_email": "claude@example.com"
}
```

#### PUT /settings

Update system configuration.

**Request Body:**
```json
{
  "default_github_token": "ghp_...",
  "claude_binary": "/usr/local/bin/claude",
  "max_concurrent_sessions": 10,
  "git_user_name": "Claude Code Server",
  "git_user_email": "claude@example.com"
}
```

**Parameters:**
- `default_github_token` (optional): Default GitHub token for push operations
- `claude_binary` (optional): Path to Claude Code binary
- `max_concurrent_sessions` (optional): Maximum concurrent sessions
- `git_user_name` (optional): Default git author name
- `git_user_email` (optional): Default git author email

**Response (200):** Updated settings object

## Session States

Sessions progress through the following states:

1. **starting**: Session is being initialized
2. **running**: Claude Code is actively processing
3. **completed**: Session finished successfully
4. **failed**: Session encountered an error

## Log Types

Session logs are categorized by type:

- **stdout**: Standard Claude Code output
- **stderr**: Error output from Claude Code
- **system**: Server-generated messages

## WebSocket Connection

For real-time session monitoring, connect to the WebSocket URL provided when creating a session:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/sessions/{id}/logs');
ws.onmessage = function(event) {
    const log = JSON.parse(event.data);
    console.log(`[${log.log_type}] ${log.content}`);
};
```

## Example Usage

### Create and Monitor a Session

```bash
# Create a session
curl -X POST http://localhost:8080/api/sessions \
  -H "Authorization: Bearer dev-key" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Analyze the performance of this codebase",
    "project_id": "123e4567-e89b-12d3-a456-426614174000"
  }'

# Stream logs
curl -N http://localhost:8080/api/sessions/{id}/logs \
  -H "Authorization: Bearer dev-key" \
  -H "Accept: text/event-stream"
```

### Project Management

```bash
# Register a project
curl -X POST http://localhost:8080/api/projects \
  -H "Authorization: Bearer dev-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Project",
    "path": "/path/to/repo"
  }'

# List projects
curl http://localhost:8080/api/projects \
  -H "Authorization: Bearer dev-key"
```