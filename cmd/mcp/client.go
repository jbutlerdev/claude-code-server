package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ClaudeCodeClient is an HTTP client for the claude-code-server API
type ClaudeCodeClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewClaudeCodeClient creates a new client
func NewClaudeCodeClient(baseURL, authToken string) *ClaudeCodeClient {
	return &ClaudeCodeClient{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest makes an authenticated HTTP request
func (c *ClaudeCodeClient) makeRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+"/api"+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Session represents a Claude Code session
type Session struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	InitialPrompt string    `json:"initial_prompt"`
	Model         string    `json:"model"`
	MaxTurns      int       `json:"max_turns"`
	Status        string    `json:"status"`
	WorktreeID    string    `json:"worktree_id"`
	WorktreePath  string    `json:"worktree_path"`
	StartedAt     *time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	Error         string    `json:"error"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Project represents a project
type Project struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	RemoteURL     string    `json:"remote_url"`
	DefaultBranch string    `json:"default_branch"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Settings represents system settings
type Settings struct {
	DefaultGitHubToken     string `json:"default_github_token"`
	ClaudeBinary          string `json:"claude_binary"`
	MaxConcurrentSessions int    `json:"max_concurrent_sessions"`
	GitUserName           string `json:"git_user_name"`
	GitUserEmail          string `json:"git_user_email"`
}

// CreateSessionRequest represents a session creation request
type CreateSessionRequest struct {
	Prompt    string `json:"prompt"`
	ProjectID string `json:"project_id"`
	Model     string `json:"model,omitempty"`
	MaxTurns  int    `json:"max_turns,omitempty"`
}

// CreateProjectRequest represents a project creation request
type CreateProjectRequest struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Description  string `json:"description,omitempty"`
	RemoteURL    string `json:"remote_url,omitempty"`
	GitHubToken  string `json:"github_token,omitempty"`
	GitUserName  string `json:"git_user_name,omitempty"`
	GitUserEmail string `json:"git_user_email,omitempty"`
}

// PushToRemoteRequest represents a push request
type PushToRemoteRequest struct {
	CommitMessage string `json:"commit_message,omitempty"`
}

// PushToRemoteResponse represents a push response
type PushToRemoteResponse struct {
	Branch     string    `json:"branch"`
	CommitHash string    `json:"commit_hash"`
	PushedAt   time.Time `json:"pushed_at"`
}

// UpdateSettingsRequest represents a settings update request
type UpdateSettingsRequest struct {
	DefaultGitHubToken     *string `json:"default_github_token,omitempty"`
	ClaudeBinary          *string `json:"claude_binary,omitempty"`
	MaxConcurrentSessions *int    `json:"max_concurrent_sessions,omitempty"`
	GitUserName           *string `json:"git_user_name,omitempty"`
	GitUserEmail          *string `json:"git_user_email,omitempty"`
}

// CreateSession creates a new session
func (c *ClaudeCodeClient) CreateSession(req CreateSessionRequest) (*Session, error) {
	respBody, err := c.makeRequest("POST", "/sessions", req)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(respBody, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &session, nil
}

// ListSessions lists all sessions
func (c *ClaudeCodeClient) ListSessions() ([]Session, error) {
	respBody, err := c.makeRequest("GET", "/sessions", nil)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	if err := json.Unmarshal(respBody, &sessions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return sessions, nil
}

// GetSession gets a specific session
func (c *ClaudeCodeClient) GetSession(sessionID string) (*Session, error) {
	respBody, err := c.makeRequest("GET", fmt.Sprintf("/sessions/%s", sessionID), nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(respBody, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &session, nil
}

// CancelSession cancels a session
func (c *ClaudeCodeClient) CancelSession(sessionID string) error {
	_, err := c.makeRequest("DELETE", fmt.Sprintf("/sessions/%s", sessionID), nil)
	return err
}

// PushSession pushes session changes to GitHub
func (c *ClaudeCodeClient) PushSession(sessionID string, req PushToRemoteRequest) (*PushToRemoteResponse, error) {
	respBody, err := c.makeRequest("POST", fmt.Sprintf("/sessions/%s/push", sessionID), req)
	if err != nil {
		return nil, err
	}

	var response PushToRemoteResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// CreateProject creates a new project
func (c *ClaudeCodeClient) CreateProject(req CreateProjectRequest) (*Project, error) {
	respBody, err := c.makeRequest("POST", "/projects", req)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := json.Unmarshal(respBody, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &project, nil
}

// ListProjects lists all projects
func (c *ClaudeCodeClient) ListProjects() ([]Project, error) {
	respBody, err := c.makeRequest("GET", "/projects", nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(respBody, &projects); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return projects, nil
}

// GetSettings gets system settings
func (c *ClaudeCodeClient) GetSettings() (*Settings, error) {
	respBody, err := c.makeRequest("GET", "/settings", nil)
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(respBody, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &settings, nil
}

// UpdateSettings updates system settings
func (c *ClaudeCodeClient) UpdateSettings(req UpdateSettingsRequest) (*Settings, error) {
	respBody, err := c.makeRequest("PUT", "/settings", req)
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(respBody, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &settings, nil
}