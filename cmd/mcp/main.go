package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Get server URL from environment or use default
	serverURL := os.Getenv("CLAUDE_CODE_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	// Get auth token from environment
	authToken := os.Getenv("CLAUDE_CODE_SERVER_TOKEN")
	if authToken == "" {
		log.Fatal("CLAUDE_CODE_SERVER_TOKEN environment variable is required")
	}

	// Create client for claude-code-server API
	client := NewClaudeCodeClient(serverURL, authToken)

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "claude-code-server-mcp",
		Version: "1.0.0",
	}, nil)

	// Add session management tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_session",
		Description: "Create a new Claude Code session",
	}, createSessionHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_sessions",
		Description: "List all Claude Code sessions",
	}, listSessionsHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_session",
		Description: "Get details of a specific Claude Code session",
	}, getSessionHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "cancel_session",
		Description: "Cancel a running Claude Code session",
	}, cancelSessionHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "push_session",
		Description: "Push session changes to GitHub repository",
	}, pushSessionHandler(client))

	// Add project management tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_project",
		Description: "Create a new project",
	}, createProjectHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_projects",
		Description: "List all projects",
	}, listProjectsHandler(client))

	// Add settings management tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_settings",
		Description: "Get system settings",
	}, getSettingsHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_settings",
		Description: "Update system settings",
	}, updateSettingsHandler(client))

	// Run the server over stdin/stdout, until the client disconnects
	if err := server.Run(context.Background(), mcp.NewStdioTransport()); err != nil {
		log.Fatal(err)
	}
}