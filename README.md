# Claude Code Server

A Go-based web service that provides a RESTful API and web interface for managing headless Claude Code sessions. This enables AI tool libraries to leverage Claude Code's capabilities programmatically while providing a user-friendly interface for monitoring and managing sessions.

## Features

- **RESTful API** with OpenAPI 3.0 specification and interactive Swagger UI
- **Modern Web UI** built with HTMX and Tailwind CSS
- **Session Management** with git worktree isolation for safe concurrent sessions
- **Conversational Interface** with JSON-structured Claude Code output parsing
- **Real-time Updates** via WebSockets and Server-Sent Events
- **Project Management** for git repositories with validation and error tracking
- **GitHub Integration** with push-to-remote functionality and token management
- **Git Identity Management** with system-level and project-level configuration
- **PostgreSQL Storage** with automatic migrations for session history and structured logs
- **Terminal Access** with WebSocket-based PTY for Claude Code authentication
- **Docker Support** with volume persistence for Claude credentials
- **Settings Management** with JSON file persistence
- **Concurrent Sessions** with goroutine-based architecture and configurable limits

## Quick Start

### Using Docker Compose

```bash
# Clone the repository
git clone https://github.com/yourusername/claude-code-server.git
cd claude-code-server

# Set your Claude Code API key (optional - can use Claude auth in terminal)
export CLAUDE_CODE_API_KEY="your-api-key"

# Start the server
docker compose up -d

# Access the web UI
open http://localhost:8080

# Authenticate with Claude Code (if no API key set)
# Go to Terminal tab and run:
# claude /login
```

### Using Pre-built Docker Image

```bash
# Run with external PostgreSQL
docker run -d \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/dbname" \
  -e CLAUDE_CODE_API_KEY="your-api-key" \
  --name claude-code-server \
  claude-code-server:latest
```

### Building from Source

```bash
# Prerequisites: Go 1.21+, Node.js 18+, PostgreSQL

# Install Claude Code CLI
npm install -g @anthropic-ai/claude-code

# Build the server
make build

# Run database migrations
make migrate-up

# Start the server
./bin/claude-code-server
```

## API Usage

### Create a Session

```bash
curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "prompt": "Help me refactor this code",
    "project_id": "project-uuid"
  }'
```

### List Sessions

```bash
curl http://localhost:8080/api/sessions \
  -H "Authorization: Bearer your-api-key"
```

### Stream Session Logs

```bash
curl http://localhost:8080/api/sessions/{id}/logs \
  -H "Authorization: Bearer your-api-key" \
  -H "Accept: text/event-stream"
```

### Push Changes to GitHub

```bash
curl -X POST http://localhost:8080/api/sessions/{id}/push \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "commit_message": "Changes from Claude Code session"
  }'
```

## Web Interface

The web UI provides a comprehensive interface for managing Claude Code sessions:

### Key Pages
- **Dashboard** - Overview of recent sessions and projects
- **Sessions** - List and manage all Claude Code sessions with real-time status
- **Session Detail** - Conversational view of Claude Code interactions with collapsible tool calls
- **Projects** - Manage git repositories with validation and error tracking
- **Settings** - Configure system-level GitHub tokens and git identity
- **Terminal** - WebSocket-based terminal for Claude Code authentication (`claude /login`)
- **API Docs** - Interactive Swagger UI for exploring the REST API

### Session Features
- **Real-time conversation view** with JSON-parsed Claude Code output
- **Auto-scroll with user control** - automatically follows conversation, disables on user scroll
- **Collapsible tool calls and results** for better readability
- **Push to GitHub** with automatic branch creation and commit management
- **Resume completed sessions** with additional prompts
- **Session cancellation and deletion**

### Project Management
- **Repository validation** during creation with immediate feedback
- **Automatic cloning** with error tracking and status updates
- **GitHub token configuration** at project or system level
- **Git identity management** for proper commit authorship
- **Error display and troubleshooting** with detailed error messages

## Configuration

See [docs/TECHNICAL_SPEC.md](docs/TECHNICAL_SPEC.md#environment-variables) for all configuration options.

Key environment variables:
- `CLAUDE_CODE_API_KEY` - Your Anthropic API key (optional, can auth via terminal)
- `DATABASE_URL` - PostgreSQL connection string
- `SERVER_PORT` - HTTP server port (default: 8080)
- `MAX_CONCURRENT_SESSIONS` - Limit concurrent Claude Code sessions
- `CLAUDE_CODE_BINARY` - Path to Claude Code binary (default: claude)
- `DEFAULT_GITHUB_TOKEN` - Default GitHub token for pushing changes

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run linter
make lint

# Start development server with hot reload
make dev

# Generate OpenAPI types
make generate
```

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed system design.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.