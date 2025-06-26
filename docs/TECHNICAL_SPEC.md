# Claude Code Server Technical Specification

## API Endpoints

### Sessions

#### POST /api/sessions
Creates a new Claude Code session.

**Request Body:**
```json
{
  "prompt": "string",
  "project_id": "string",
  "model": "string (optional, default: claude-3-5-sonnet-20241022)",
  "max_turns": "integer (optional, default: 25)"
}
```

**Response:**
```json
{
  "id": "string",
  "status": "starting|running|completed|failed",
  "created_at": "timestamp",
  "websocket_url": "string",
  "worktree_path": "string"
}
```

#### GET /api/sessions
List sessions with pagination.

**Query Parameters:**
- `page` (default: 1)
- `page_size` (default: 10)
- `status` (optional filter)
- `project_id` (optional filter)

#### GET /api/sessions/{id}
Get session details including logs.

#### POST /api/sessions/{id}/resume
Resume a completed session.

**Request Body:**
```json
{
  "prompt": "string"
}
```

#### DELETE /api/sessions/{id}
Cancel and cleanup a session.

#### GET /api/sessions/{id}/logs
Stream session logs via Server-Sent Events.

#### POST /api/sessions/{id}/push
Push session changes to GitHub.

**Request Body:**
```json
{
  "commit_message": "string (optional)"
}
```

**Response:**
```json
{
  "branch": "string",
  "commit_hash": "string",
  "pushed_at": "timestamp"
}
```

### Projects

#### POST /api/projects
Register a new git repository.

**Request Body:**
```json
{
  "name": "string",
  "path": "string",
  "remote_url": "string (optional)",
  "github_token": "string (optional)",
  "git_user_name": "string (optional)",
  "git_user_email": "string (optional)"
}
```

#### GET /api/projects
List all projects.

#### GET /api/projects/{id}
Get project details.

#### PUT /api/projects/{id}
Update project configuration.

#### DELETE /api/projects/{id}
Remove project (does not delete files).

### System

#### GET /api/health
Health check endpoint.

#### GET /api/ready
Readiness check endpoint.

#### GET /api/metrics
Prometheus metrics endpoint.

#### GET /api/openapi.json
OpenAPI 3.0 specification.

### Settings

#### GET /api/settings
Get system configuration.

#### PUT /api/settings
Update system configuration.

**Request Body:**
```json
{
  "default_github_token": "string (optional)",
  "claude_binary": "string (optional)",
  "max_concurrent_sessions": "integer (optional)",
  "git_user_name": "string (optional)",
  "git_user_email": "string (optional)"
}
```

## Database Schema

### sessions
```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id),
    initial_prompt TEXT NOT NULL,
    model VARCHAR(100) NOT NULL DEFAULT 'claude-3-5-sonnet-20241022',
    max_turns INTEGER NOT NULL DEFAULT 25,
    status VARCHAR(50) NOT NULL DEFAULT 'starting',
    worktree_id VARCHAR(100) NOT NULL UNIQUE,
    worktree_path TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    error TEXT,
    metadata JSONB,
    push_branch VARCHAR(255),
    push_commit_hash VARCHAR(40),
    last_pushed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_project_id ON sessions(project_id);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_created_at ON sessions(created_at DESC);
```

### session_logs
```sql
CREATE TABLE session_logs (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    log_type VARCHAR(50) NOT NULL, -- 'stdout', 'stderr', 'system'
    content TEXT NOT NULL,
    json_data JSONB, -- Structured Claude Code output
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_logs_session_id ON session_logs(session_id);
CREATE INDEX idx_session_logs_created_at ON session_logs(session_id, created_at);
```

### projects
```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    path TEXT NOT NULL,
    description TEXT,
    remote_url TEXT,
    default_branch VARCHAR(100) DEFAULT 'main',
    status VARCHAR(50) DEFAULT 'pending',
    error_message TEXT,
    last_sync_at TIMESTAMP,
    github_token TEXT,
    git_user_name TEXT,
    git_user_email TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_name ON projects(name);
```

### api_keys
```sql
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    permissions JSONB NOT NULL DEFAULT '["read", "write"]',
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
```

## Environment Variables

```bash
# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

# Database Configuration
DATABASE_URL=postgres://user:pass@localhost:5432/claude_code_server
DATABASE_MAX_CONNECTIONS=25
DATABASE_MAX_IDLE_CONNECTIONS=5

# Claude Code Configuration
CLAUDE_CODE_BINARY=/usr/local/bin/claude
CLAUDE_CODE_API_KEY=sk-ant-... # Optional, can auth via terminal
MAX_CONCURRENT_SESSIONS=10
SESSION_TIMEOUT=1h

# GitHub Integration
DEFAULT_GITHUB_TOKEN=ghp-... # Default token for push operations
GIT_USER_NAME=Claude Code Server # Default git author name
GIT_USER_EMAIL=claude@example.com # Default git author email

# Storage Configuration
WORKTREE_BASE_PATH=/var/lib/claude-code-server/worktrees
LOG_RETENTION_DAYS=30

# Security
JWT_SECRET=your-secret-key
API_RATE_LIMIT=100
API_RATE_LIMIT_WINDOW=1m

# Observability
LOG_LEVEL=info
LOG_FORMAT=json
METRICS_ENABLED=true
```

## Web UI Routes

### Pages
- `/` - Dashboard showing recent sessions and projects
- `/sessions` - Session list with filters and real-time status
- `/sessions/{id}` - Session detail with conversational view and push functionality
- `/projects` - Project management with validation and error tracking
- `/projects/new` - Add new project with GitHub integration
- `/settings` - System configuration for tokens and git identity
- `/terminal` - WebSocket terminal for Claude Code authentication
- `/api-docs` - Interactive Swagger UI for API exploration

### HTMX Endpoints
- `GET /htmx/sessions` - Session list partial
- `GET /htmx/sessions/{id}/logs` - Live log streaming
- `POST /htmx/sessions/{id}/resume` - Resume session form
- `GET /htmx/projects` - Project list partial

## Docker Configuration

### Dockerfile
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o claude-code-server ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ca-certificates nodejs npm git
RUN npm install -g @anthropic-ai/claude-code
RUN adduser -D -u 1000 appuser
WORKDIR /app
COPY --from=builder /app/claude-code-server .
COPY --from=builder /app/static ./static
COPY --from=builder /app/templates ./templates
USER appuser
EXPOSE 8080
CMD ["./claude-code-server"]
```

### docker-compose.yml
```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://claude:password@db:5432/claude_code_server
      CLAUDE_CODE_API_KEY: ${CLAUDE_CODE_API_KEY}
    volumes:
      - worktrees:/var/lib/claude-code-server/worktrees
    depends_on:
      - db

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: claude
      POSTGRES_PASSWORD: password
      POSTGRES_DB: claude_code_server
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

volumes:
  worktrees:
  postgres_data:
```

## Testing Strategy

### Unit Tests
- Model validation tests
- API handler tests with mocks
- Session manager logic tests
- Git operations tests

### Integration Tests
- Database operations
- Claude Code subprocess management
- API endpoint integration
- WebSocket communication

### End-to-End Tests
- Full session lifecycle
- Web UI interactions
- API client scenarios

## Performance Requirements

- Support 100+ concurrent users
- Session creation < 2 seconds
- API response time < 200ms (p95)
- Log streaming latency < 100ms
- Database query time < 50ms (p95)