# Claude Code Server MCP

A Model Context Protocol (MCP) server that provides tools for interacting with a Claude Code Server instance. This allows MCP clients to manage Claude Code sessions, projects, and settings through standardized MCP tools.

## Features

The MCP server provides the following tools:

### Session Management
- `create_session` - Create a new Claude Code session
- `list_sessions` - List all Claude Code sessions
- `get_session` - Get details of a specific session
- `cancel_session` - Cancel a running session
- `push_session` - Push session changes to GitHub

### Project Management
- `create_project` - Create a new project
- `list_projects` - List all projects

### Settings Management
- `get_settings` - Get system settings
- `update_settings` - Update system settings

## Installation

### From Source

```bash
# Clone the repository (if you haven't already)
git clone https://github.com/jbutlerdev/claude-code-server
cd claude-code-server/cmd/mcp

# Download dependencies
make deps

# Build statically linked binary
make build-current-static
```

The binary will be built to `../../build/claude-code-mcp-server`.

### Cross-compilation

To build for multiple platforms:

```bash
make build-static
```

This will create binaries for:
- Linux AMD64
- macOS AMD64
- macOS ARM64 (Apple Silicon)
- Windows AMD64

## Configuration

The MCP server requires the following environment variables:

- `CLAUDE_CODE_SERVER_URL` - URL of the Claude Code Server (default: `http://localhost:8080`)
- `CLAUDE_CODE_SERVER_TOKEN` - Authentication token for the Claude Code Server (required)

## Usage

### Running the MCP Server

```bash
export CLAUDE_CODE_SERVER_URL="http://localhost:8080"
export CLAUDE_CODE_SERVER_TOKEN="your-auth-token"
./claude-code-mcp-server
```

The server communicates over stdin/stdout using the MCP protocol.

### Using with Claude Desktop

Add the following to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "claude-code-server": {
      "command": "/path/to/claude-code-mcp-server",
      "env": {
        "CLAUDE_CODE_SERVER_URL": "http://localhost:8080",
        "CLAUDE_CODE_SERVER_TOKEN": "your-auth-token"
      }
    }
  }
}
```

### Using with Other MCP Clients

The server follows the standard MCP protocol and can be used with any MCP-compatible client. It communicates via JSON-RPC over stdin/stdout.

## Tool Examples

### Create a Session

```json
{
  "name": "create_session",
  "arguments": {
    "prompt": "Help me build a web application",
    "project_id": "123e4567-e89b-12d3-a456-426614174000"
  }
}
```

### List Sessions

```json
{
  "name": "list_sessions",
  "arguments": {}
}
```

### Create a Project

```json
{
  "name": "create_project",
  "arguments": {
    "name": "My Web App",
    "path": "/var/lib/claude-code-server/projects/my-web-app",
    "remote_url": "https://github.com/user/my-web-app.git"
  }
}
```

### Push Session Changes

```json
{
  "name": "push_session",
  "arguments": {
    "session_id": "123e4567-e89b-12d3-a456-426614174001",
    "commit_message": "Updates from Claude Code session"
  }
}
```

## Development

### Building

```bash
# Build for development
make build

# Build statically linked binary
make build-current-static

# Build for all platforms
make build-static
```

### Testing

```bash
make test
```

### Cleaning

```bash
make clean
```

## Architecture

The MCP server consists of:

1. **main.go** - Entry point and MCP server setup
2. **client.go** - HTTP client for Claude Code Server API
3. **handlers.go** - MCP tool handlers
4. **implementation.go** - MCP server metadata

The server acts as a bridge between MCP clients and the Claude Code Server REST API, translating MCP tool calls into HTTP API requests.

## License

This project is part of the Claude Code Server and follows the same license terms.