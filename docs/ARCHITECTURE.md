# Claude Code Server Architecture

## Overview

Claude Code Server is a Go-based web service that provides a RESTful API and web interface for managing headless Claude Code sessions. It enables AI tool libraries to leverage Claude Code's capabilities programmatically while providing a user-friendly interface for monitoring and managing sessions.

## System Components

### 1. API Server (Go + Gorilla Mux)
- RESTful HTTP API with OpenAPI 3.0 specification
- Concurrent request handling using goroutines
- JWT-based authentication for API access
- WebSocket support for real-time session updates

### 2. Web UI (HTMX + Tailwind CSS)
- Server-side rendered pages using Go templates
- HTMX for dynamic updates without JavaScript
- Dark theme using Tailwind CSS
- Real-time session monitoring via Server-Sent Events (SSE)

### 3. Session Manager
- Manages Claude Code subprocess lifecycle
- Git worktree isolation for each session
- Session state tracking and persistence
- Log streaming and storage

### 4. Database (PostgreSQL)
- Sessions table: stores session metadata
- Logs table: stores Claude Code output
- Projects table: stores git repository configurations
- Migrations handled by golang-migrate

### 5. Git Integration
- Uses go-git library for repository operations
- Creates isolated worktrees for each session
- Supports both local and remote repositories

## Key Design Decisions

### 1. Using Go Standard Library
We prioritize the Go standard library over third-party dependencies where possible:
- `net/http` for HTTP server (with gorilla/mux for routing)
- `database/sql` with `lib/pq` for PostgreSQL
- `html/template` for server-side rendering
- `encoding/json` for API responses

### 2. Third-Party Libraries Justification
- **gorilla/mux**: Advanced routing features not available in standard library
- **lib/pq**: PostgreSQL driver required for database/sql
- **go-git**: Comprehensive git operations without shelling out
- **golang-migrate**: Database schema versioning
- **oapi-codegen**: OpenAPI code generation for type safety

### 3. Process Management
Claude Code runs as a subprocess using `os/exec` with:
- Proper signal handling for graceful shutdown
- Output streaming to capture logs in real-time
- Environment variable isolation
- Working directory management

### 4. Concurrency Model
- Each API request handled in separate goroutine
- Session manager uses channels for communication
- Database connection pooling for efficiency
- Rate limiting to prevent resource exhaustion

## Data Flow

1. **Session Creation**
   - Client sends POST /api/sessions with prompt and project
   - Server creates git worktree
   - Spawns Claude Code subprocess
   - Returns session ID and WebSocket URL

2. **Session Monitoring**
   - Client connects to WebSocket or SSE endpoint
   - Server streams Claude Code output in real-time
   - Logs persisted to database asynchronously

3. **Session Resumption**
   - Client sends POST /api/sessions/{id}/resume
   - Server restores session state
   - Resumes Claude Code with --resume flag

## Security Considerations

1. **Authentication**
   - API key authentication for programmatic access
   - Session tokens for web UI
   - Rate limiting per API key

2. **Isolation**
   - Each session runs in isolated git worktree
   - File system permissions restrict access
   - Container runs as non-root user

3. **Input Validation**
   - Sanitize all user inputs
   - Validate git URLs before cloning
   - Escape shell commands properly

## Deployment Architecture

### Single Container Mode
- Application container connects to external PostgreSQL
- Environment variables for configuration
- Health check endpoints for orchestration

### Docker Compose Mode
- Application + PostgreSQL containers
- Shared network for communication
- Volume mounts for data persistence

## Performance Considerations

1. **Database Optimization**
   - Indexed columns for common queries
   - Pagination for large result sets
   - Connection pooling

2. **Resource Management**
   - Limit concurrent Claude Code sessions
   - Implement session timeouts
   - Clean up abandoned worktrees

3. **Caching**
   - Cache project metadata
   - In-memory session state cache
   - Static asset caching headers

## Monitoring and Observability

1. **Logging**
   - Structured logging with log levels
   - Correlation IDs for request tracing
   - Separate application and Claude Code logs

2. **Metrics**
   - Prometheus-compatible metrics endpoint
   - Session duration and success rates
   - API response times

3. **Health Checks**
   - /health endpoint for liveness
   - /ready endpoint for readiness
   - Database connectivity checks