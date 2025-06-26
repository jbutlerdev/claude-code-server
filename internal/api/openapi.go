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
						"name":       map[string]string{"type": "string"},
						"path":       map[string]string{"type": "string"},
						"remote_url": map[string]string{"type": "string"},
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
		},
	}
}
