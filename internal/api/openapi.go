package api

func (api *API) generateOpenAPISpec() map[string]interface{} {
	return map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Claude Code Server API",
			"description": "API for managing headless Claude Code sessions",
			"version":     "1.0.0",
		},
		"servers": []map[string]interface{}{
			{
				"url":         "/api",
				"description": "API server",
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":   "http",
					"scheme": "bearer",
				},
			},
			"schemas": map[string]interface{}{
				"Session": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id":             map[string]string{"type": "string", "format": "uuid"},
						"project_id":     map[string]string{"type": "string", "format": "uuid"},
						"initial_prompt": map[string]string{"type": "string"},
						"model":          map[string]string{"type": "string"},
						"max_turns":      map[string]string{"type": "integer"},
						"status":         map[string]interface{}{"type": "string", "enum": []string{"starting", "running", "completed", "failed"}},
						"worktree_id":    map[string]string{"type": "string"},
						"worktree_path":  map[string]string{"type": "string"},
						"started_at":     map[string]string{"type": "string", "format": "date-time"},
						"completed_at":   map[string]string{"type": "string", "format": "date-time"},
						"error":          map[string]string{"type": "string"},
						"created_at":     map[string]string{"type": "string", "format": "date-time"},
						"updated_at":     map[string]string{"type": "string", "format": "date-time"},
					},
				},
				"Project": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id":             map[string]string{"type": "string", "format": "uuid"},
						"name":           map[string]string{"type": "string"},
						"path":           map[string]string{"type": "string"},
						"remote_url":     map[string]string{"type": "string"},
						"default_branch": map[string]string{"type": "string"},
						"created_at":     map[string]string{"type": "string", "format": "date-time"},
						"updated_at":     map[string]string{"type": "string", "format": "date-time"},
					},
				},
				"CreateSessionRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"prompt", "project_id"},
					"properties": map[string]interface{}{
						"prompt":     map[string]string{"type": "string"},
						"project_id": map[string]string{"type": "string", "format": "uuid"},
						"model":      map[string]string{"type": "string"},
						"max_turns":  map[string]string{"type": "integer"},
					},
				},
				"CreateProjectRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"name", "path"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
							"description": "Human-readable project name",
							"example": "My Web App",
						},
						"path": map[string]interface{}{
							"type": "string",
							"description": "Local path where the project will be stored",
							"example": "/var/lib/claude-code-server/worktrees/my-web-app",
						},
						"description": map[string]interface{}{
							"type": "string",
							"description": "Optional project description",
							"example": "A sample web application built with React",
						},
						"remote_url": map[string]interface{}{
							"type": "string",
							"description": "Git repository URL (HTTPS or SSH)",
							"example": "https://github.com/user/repo.git",
						},
						"github_token": map[string]interface{}{
							"type": "string",
							"description": "Project-specific GitHub token for push operations",
							"example": "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
						},
						"git_user_name": map[string]interface{}{
							"type": "string",
							"description": "Git author name for commits in this project",
							"example": "John Doe",
						},
						"git_user_email": map[string]interface{}{
							"type": "string",
							"description": "Git author email for commits in this project",
							"format": "email",
							"example": "john@example.com",
						},
					},
				},
				"PushToRemoteRequest": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"commit_message": map[string]interface{}{
							"type": "string",
							"description": "Custom commit message for the push. If not provided, a default message will be used.",
							"example": "Updates from Claude Code session",
						},
					},
				},
				"PushToRemoteResponse": map[string]interface{}{
					"type": "object",
					"required": []string{"branch", "commit_hash", "pushed_at"},
					"properties": map[string]interface{}{
						"branch": map[string]interface{}{
							"type": "string",
							"description": "Name of the branch that was created/updated",
							"example": "claude-session-abc123-20240625-143000",
						},
						"commit_hash": map[string]interface{}{
							"type": "string",
							"description": "SHA hash of the commit that was pushed",
							"example": "a1b2c3d4e5f6789012345678901234567890abcd",
						},
						"pushed_at": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "ISO 8601 timestamp when the push was completed",
							"example": "2024-06-25T14:30:45Z",
						},
					},
				},
				"Settings": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"default_github_token": map[string]interface{}{
							"type": "string",
							"description": "Default GitHub token used for push operations when projects don't have their own token",
							"example": "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
						},
						"claude_binary": map[string]interface{}{
							"type": "string",
							"description": "Path to the Claude Code binary",
							"example": "/usr/local/bin/claude",
						},
						"max_concurrent_sessions": map[string]interface{}{
							"type": "integer",
							"description": "Maximum number of concurrent Claude Code sessions",
							"example": 10,
						},
						"git_user_name": map[string]interface{}{
							"type": "string",
							"description": "Default git author name for commits",
							"example": "Claude Code Server",
						},
						"git_user_email": map[string]interface{}{
							"type": "string",
							"description": "Default git author email for commits",
							"example": "claude@example.com",
						},
					},
				},
				"UpdateSettingsRequest": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"default_github_token": map[string]interface{}{
							"type": "string",
							"description": "Default GitHub token used for push operations",
						},
						"claude_binary": map[string]interface{}{
							"type": "string",
							"description": "Path to the Claude Code binary",
						},
						"max_concurrent_sessions": map[string]interface{}{
							"type": "integer",
							"description": "Maximum number of concurrent Claude Code sessions",
							"minimum": 1,
							"maximum": 100,
						},
						"git_user_name": map[string]interface{}{
							"type": "string",
							"description": "Default git author name for commits",
						},
						"git_user_email": map[string]interface{}{
							"type": "string",
							"description": "Default git author email for commits",
							"format": "email",
						},
					},
				},
				"Error": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"error":   map[string]string{"type": "string"},
						"message": map[string]string{"type": "string"},
						"code":    map[string]string{"type": "string"},
					},
				},
			},
		},
		"security": []map[string]interface{}{
			{"bearerAuth": []string{}},
		},
		"paths": map[string]interface{}{
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "OK",
						},
					},
					"security": []interface{}{},
				},
			},
			"/sessions": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Create a new session",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{"$ref": "#/components/schemas/CreateSessionRequest"},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Session created",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Session"},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Bad request",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Error"},
								},
							},
						},
					},
				},
				"get": map[string]interface{}{
					"summary": "List sessions",
					"parameters": []map[string]interface{}{
						{
							"name":        "page",
							"in":          "query",
							"description": "Page number",
							"schema":      map[string]interface{}{"type": "integer", "default": 1},
						},
						{
							"name":        "page_size",
							"in":          "query",
							"description": "Items per page",
							"schema":      map[string]interface{}{"type": "integer", "default": 10},
						},
						{
							"name":        "status",
							"in":          "query",
							"description": "Filter by status",
							"schema":      map[string]string{"type": "string"},
						},
						{
							"name":        "project_id",
							"in":          "query",
							"description": "Filter by project",
							"schema":      map[string]string{"type": "string", "format": "uuid"},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Sessions list",
						},
					},
				},
			},
			"/sessions/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get session details",
					"parameters": []map[string]interface{}{
						{
							"name":     "id",
							"in":       "path",
							"required": true,
							"schema":   map[string]string{"type": "string", "format": "uuid"},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Session details",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Session"},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Session not found",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary": "Cancel session",
					"parameters": []map[string]interface{}{
						{
							"name":     "id",
							"in":       "path",
							"required": true,
							"schema":   map[string]string{"type": "string", "format": "uuid"},
						},
					},
					"responses": map[string]interface{}{
						"204": map[string]interface{}{
							"description": "Session cancelled",
						},
					},
				},
			},
			"/sessions/{id}/push": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Push session changes to GitHub",
					"description": "Commits all changes in the session worktree and pushes them to a new branch on GitHub",
					"parameters": []map[string]interface{}{
						{
							"name":     "id",
							"in":       "path",
							"required": true,
							"schema":   map[string]string{"type": "string", "format": "uuid"},
							"description": "Session ID",
						},
					},
					"requestBody": map[string]interface{}{
						"required": false,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{"$ref": "#/components/schemas/PushToRemoteRequest"},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successfully pushed to GitHub",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/PushToRemoteResponse"},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Bad request - missing remote URL or GitHub token",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Error"},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Session not found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Error"},
								},
							},
						},
						"500": map[string]interface{}{
							"description": "Internal server error - git operation failed",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Error"},
								},
							},
						},
					},
				},
			},
			"/projects": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Create a new project",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{"$ref": "#/components/schemas/CreateProjectRequest"},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Project created",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Project"},
								},
							},
						},
					},
				},
				"get": map[string]interface{}{
					"summary": "List projects",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Projects list",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type":  "array",
										"items": map[string]string{"$ref": "#/components/schemas/Project"},
									},
								},
							},
						},
					},
				},
			},
			"/settings": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get system settings",
					"description": "Retrieve current system configuration including GitHub tokens and git identity",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "System settings",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Settings"},
								},
							},
						},
					},
				},
				"put": map[string]interface{}{
					"summary": "Update system settings",
					"description": "Update system configuration including GitHub tokens and git identity",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{"$ref": "#/components/schemas/UpdateSettingsRequest"},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Settings updated successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Settings"},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Error"},
								},
							},
						},
					},
				},
			},
		},
	}
}
