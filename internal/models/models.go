package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SessionStatus string

const (
	SessionStatusStarting  SessionStatus = "starting"
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusFailed    SessionStatus = "failed"
)

type Session struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	ProjectID      uuid.UUID     `json:"project_id" db:"project_id"`
	InitialPrompt  string        `json:"initial_prompt" db:"initial_prompt"`
	Model          string        `json:"model" db:"model"`
	MaxTurns       int           `json:"max_turns" db:"max_turns"`
	Status         SessionStatus `json:"status" db:"status"`
	WorktreeID     string        `json:"worktree_id" db:"worktree_id"`
	WorktreePath   string        `json:"worktree_path" db:"worktree_path"`
	ConversationID *string       `json:"conversation_id,omitempty" db:"conversation_id"`
	StartedAt      time.Time     `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
	Error          *string       `json:"error,omitempty" db:"error"`
	Metadata       JSONB         `json:"metadata,omitempty" db:"metadata"`
	PushBranch     *string       `json:"push_branch,omitempty" db:"push_branch"`
	PushCommitHash *string       `json:"push_commit_hash,omitempty" db:"push_commit_hash"`
	LastPushedAt   *time.Time    `json:"last_pushed_at,omitempty" db:"last_pushed_at"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
}

type SessionLog struct {
	ID        int64     `json:"id" db:"id"`
	SessionID uuid.UUID `json:"session_id" db:"session_id"`
	LogType   string    `json:"log_type" db:"log_type"`
	Content   string    `json:"content" db:"content"`
	JSONData  *JSONB    `json:"json_data,omitempty" db:"json_data"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Project struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Path          string     `json:"path" db:"path"`
	Description   *string    `json:"description,omitempty" db:"description"`
	RemoteURL     *string    `json:"remote_url,omitempty" db:"remote_url"`
	DefaultBranch string     `json:"default_branch" db:"default_branch"`
	Status        string     `json:"status" db:"status"`
	ErrorMessage  *string    `json:"error_message,omitempty" db:"error_message"`
	LastSyncAt    *time.Time `json:"last_sync_at,omitempty" db:"last_sync_at"`
	GitHubToken   *string    `json:"github_token,omitempty" db:"github_token"`
	GitUserName   *string    `json:"git_user_name,omitempty" db:"git_user_name"`
	GitUserEmail  *string    `json:"git_user_email,omitempty" db:"git_user_email"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type APIKey struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	KeyHash     string     `json:"-" db:"key_hash"`
	Name        string     `json:"name" db:"name"`
	Permissions []string   `json:"permissions" db:"permissions"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(value.(string)), j)
	}
	return json.Unmarshal(bytes, j)
}

type CreateSessionRequest struct {
	Prompt    string `json:"prompt" validate:"required"`
	ProjectID string `json:"project_id" validate:"required,uuid"`
	MaxTurns  int    `json:"max_turns,omitempty"`
}

type CreateSessionResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	WebSocketURL string `json:"websocket_url"`
	WorktreePath string `json:"worktree_path"`
}

type ResumeSessionRequest struct {
	Prompt string `json:"prompt" validate:"required"`
}

type CreateProjectRequest struct {
	Name        string  `json:"name" validate:"required"`
	Path        string  `json:"path" validate:"required"`
	Description *string `json:"description,omitempty"`
	RemoteURL   *string `json:"remote_url,omitempty"`
	GitHubToken *string `json:"github_token,omitempty"`
	GitUserName *string `json:"git_user_name,omitempty"`
	GitUserEmail *string `json:"git_user_email,omitempty"`
}

type UpdateProjectRequest struct {
	Name          *string `json:"name,omitempty"`
	Path          *string `json:"path,omitempty"`
	RemoteURL     *string `json:"remote_url,omitempty"`
	DefaultBranch *string `json:"default_branch,omitempty"`
	GitHubToken   *string `json:"github_token,omitempty"`
	GitUserName   *string `json:"git_user_name,omitempty"`
	GitUserEmail  *string `json:"git_user_email,omitempty"`
}

type ListSessionsQuery struct {
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	Status    string `json:"status,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

type ListResponse[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type PushToRemoteRequest struct {
	CommitMessage string `json:"commit_message,omitempty"`
}

type PushToRemoteResponse struct {
	Branch     string `json:"branch"`
	CommitHash string `json:"commit_hash"`
	PushedAt   string `json:"pushed_at"`
}

type Settings struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	DefaultGitHubToken    *string    `json:"default_github_token,omitempty" db:"default_github_token"`
	ClaudeCodeBinary      string     `json:"claude_code_binary" db:"claude_code_binary"`
	ClaudeCodeAPIKey      *string    `json:"claude_code_api_key,omitempty" db:"claude_code_api_key"`
	MaxConcurrentSessions int        `json:"max_concurrent_sessions" db:"max_concurrent_sessions"`
	DefaultMaxTurns       *int       `json:"default_max_turns,omitempty" db:"default_max_turns"`
	GitUserName           *string    `json:"git_user_name,omitempty" db:"git_user_name"`
	GitUserEmail          *string    `json:"git_user_email,omitempty" db:"git_user_email"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

type UpdateSettingsRequest struct {
	DefaultGitHubToken    *string `json:"default_github_token,omitempty"`
	ClaudeCodeBinary      *string `json:"claude_code_binary,omitempty"`
	ClaudeCodeAPIKey      *string `json:"claude_code_api_key,omitempty"`
	MaxConcurrentSessions *int    `json:"max_concurrent_sessions,omitempty"`
	DefaultMaxTurns       *int    `json:"default_max_turns,omitempty"`
	GitUserName           *string `json:"git_user_name,omitempty"`
	GitUserEmail          *string `json:"git_user_email,omitempty"`
}
