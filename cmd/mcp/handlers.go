package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Parameter structures for type safety
type CreateSessionParams struct {
	Prompt    string `json:"prompt" jsonschema:"required,description=The initial prompt for the Claude Code session"`
	ProjectID string `json:"project_id" jsonschema:"required,description=UUID of the project to run the session in"`
	Model     string `json:"model,omitempty" jsonschema:"description=The model to use (optional)"`
	MaxTurns  int    `json:"max_turns,omitempty" jsonschema:"description=Maximum number of turns (optional)"`
}

type GetSessionParams struct {
	SessionID string `json:"session_id" jsonschema:"required,description=UUID of the session to retrieve"`
}

type CancelSessionParams struct {
	SessionID string `json:"session_id" jsonschema:"required,description=UUID of the session to cancel"`
}

type PushSessionParams struct {
	SessionID     string `json:"session_id" jsonschema:"required,description=UUID of the session to push"`
	CommitMessage string `json:"commit_message,omitempty" jsonschema:"description=Custom commit message (optional)"`
}

type CreateProjectParams struct {
	Name         string `json:"name" jsonschema:"required,description=Human-readable project name"`
	Path         string `json:"path" jsonschema:"required,description=Local path where the project will be stored"`
	Description  string `json:"description,omitempty" jsonschema:"description=Optional project description"`
	RemoteURL    string `json:"remote_url,omitempty" jsonschema:"description=Git repository URL (HTTPS or SSH)"`
	GitHubToken  string `json:"github_token,omitempty" jsonschema:"description=Project-specific GitHub token for push operations"`
	GitUserName  string `json:"git_user_name,omitempty" jsonschema:"description=Git author name for commits in this project"`
	GitUserEmail string `json:"git_user_email,omitempty" jsonschema:"description=Git author email for commits in this project"`
}

type UpdateSettingsParams struct {
	DefaultGitHubToken     *string `json:"default_github_token,omitempty" jsonschema:"description=Default GitHub token used for push operations"`
	ClaudeBinary          *string `json:"claude_binary,omitempty" jsonschema:"description=Path to the Claude Code binary"`
	MaxConcurrentSessions *int    `json:"max_concurrent_sessions,omitempty" jsonschema:"description=Maximum number of concurrent Claude Code sessions"`
	GitUserName           *string `json:"git_user_name,omitempty" jsonschema:"description=Default git author name for commits"`
	GitUserEmail          *string `json:"git_user_email,omitempty" jsonschema:"description=Default git author email for commits"`
}

// createSessionHandler creates a new Claude Code session
func createSessionHandler(client *ClaudeCodeClient) mcp.ToolHandlerFor[CreateSessionParams, any] {
	return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[CreateSessionParams]) (*mcp.CallToolResultFor[any], error) {
		req := CreateSessionRequest{
			Prompt:    params.Arguments.Prompt,
			ProjectID: params.Arguments.ProjectID,
			Model:     params.Arguments.Model,
			MaxTurns:  params.Arguments.MaxTurns,
		}

		session, err := client.CreateSession(req)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to create session: %v", err)}},
			}, nil
		}

		sessionJSON, _ := json.MarshalIndent(session, "", "  ")
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session created successfully:\n%s", string(sessionJSON))}},
		}, nil
	}
}

// listSessionsHandler lists all Claude Code sessions
func listSessionsHandler(client *ClaudeCodeClient) mcp.ToolHandlerFor[struct{}, any] {
	return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[struct{}]) (*mcp.CallToolResultFor[any], error) {
		sessions, err := client.ListSessions()
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to list sessions: %v", err)}},
			}, nil
		}

		if len(sessions) == 0 {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "No sessions found"}},
			}, nil
		}

		sessionsJSON, _ := json.MarshalIndent(sessions, "", "  ")
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Found %d sessions:\n%s", len(sessions), string(sessionsJSON))}},
		}, nil
	}
}

// getSessionHandler gets details of a specific session
func getSessionHandler(client *ClaudeCodeClient) mcp.ToolHandlerFor[GetSessionParams, any] {
	return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetSessionParams]) (*mcp.CallToolResultFor[any], error) {
		session, err := client.GetSession(params.Arguments.SessionID)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to get session: %v", err)}},
			}, nil
		}

		sessionJSON, _ := json.MarshalIndent(session, "", "  ")
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session details:\n%s", string(sessionJSON))}},
		}, nil
	}
}

// cancelSessionHandler cancels a running session
func cancelSessionHandler(client *ClaudeCodeClient) mcp.ToolHandlerFor[CancelSessionParams, any] {
	return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[CancelSessionParams]) (*mcp.CallToolResultFor[any], error) {
		err := client.CancelSession(params.Arguments.SessionID)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to cancel session: %v", err)}},
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session %s cancelled successfully", params.Arguments.SessionID)}},
		}, nil
	}
}

// pushSessionHandler pushes session changes to GitHub
func pushSessionHandler(client *ClaudeCodeClient) mcp.ToolHandlerFor[PushSessionParams, any] {
	return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[PushSessionParams]) (*mcp.CallToolResultFor[any], error) {
		req := PushToRemoteRequest{
			CommitMessage: params.Arguments.CommitMessage,
		}

		response, err := client.PushSession(params.Arguments.SessionID, req)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to push session: %v", err)}},
			}, nil
		}

		responseJSON, _ := json.MarshalIndent(response, "", "  ")
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session pushed successfully:\n%s", string(responseJSON))}},
		}, nil
	}
}

// createProjectHandler creates a new project
func createProjectHandler(client *ClaudeCodeClient) mcp.ToolHandlerFor[CreateProjectParams, any] {
	return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[CreateProjectParams]) (*mcp.CallToolResultFor[any], error) {
		req := CreateProjectRequest{
			Name:         params.Arguments.Name,
			Path:         params.Arguments.Path,
			Description:  params.Arguments.Description,
			RemoteURL:    params.Arguments.RemoteURL,
			GitHubToken:  params.Arguments.GitHubToken,
			GitUserName:  params.Arguments.GitUserName,
			GitUserEmail: params.Arguments.GitUserEmail,
		}

		project, err := client.CreateProject(req)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to create project: %v", err)}},
			}, nil
		}

		projectJSON, _ := json.MarshalIndent(project, "", "  ")
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Project created successfully:\n%s", string(projectJSON))}},
		}, nil
	}
}

// listProjectsHandler lists all projects
func listProjectsHandler(client *ClaudeCodeClient) mcp.ToolHandlerFor[struct{}, any] {
	return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[struct{}]) (*mcp.CallToolResultFor[any], error) {
		projects, err := client.ListProjects()
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to list projects: %v", err)}},
			}, nil
		}

		if len(projects) == 0 {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "No projects found"}},
			}, nil
		}

		projectsJSON, _ := json.MarshalIndent(projects, "", "  ")
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Found %d projects:\n%s", len(projects), string(projectsJSON))}},
		}, nil
	}
}

// getSettingsHandler gets system settings
func getSettingsHandler(client *ClaudeCodeClient) mcp.ToolHandlerFor[struct{}, any] {
	return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[struct{}]) (*mcp.CallToolResultFor[any], error) {
		settings, err := client.GetSettings()
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to get settings: %v", err)}},
			}, nil
		}

		settingsJSON, _ := json.MarshalIndent(settings, "", "  ")
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Current settings:\n%s", string(settingsJSON))}},
		}, nil
	}
}

// updateSettingsHandler updates system settings
func updateSettingsHandler(client *ClaudeCodeClient) mcp.ToolHandlerFor[UpdateSettingsParams, any] {
	return func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[UpdateSettingsParams]) (*mcp.CallToolResultFor[any], error) {
		req := UpdateSettingsRequest{
			DefaultGitHubToken:     params.Arguments.DefaultGitHubToken,
			ClaudeBinary:          params.Arguments.ClaudeBinary,
			MaxConcurrentSessions: params.Arguments.MaxConcurrentSessions,
			GitUserName:           params.Arguments.GitUserName,
			GitUserEmail:          params.Arguments.GitUserEmail,
		}

		settings, err := client.UpdateSettings(req)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to update settings: %v", err)}},
			}, nil
		}

		settingsJSON, _ := json.MarshalIndent(settings, "", "  ")
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Settings updated successfully:\n%s", string(settingsJSON))}},
		}, nil
	}
}