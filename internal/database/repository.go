package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jbutlerdev/claude-code-server/internal/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateProject(ctx context.Context, req *models.CreateProjectRequest) (*models.Project, error) {
	project := &models.Project{
		ID:            uuid.New(),
		Name:          req.Name,
		Path:          req.Path,
		Description:   req.Description,
		RemoteURL:     req.RemoteURL,
		DefaultBranch: "main",
		Status:        "pending",
		GitHubToken:   req.GitHubToken,
		GitUserName:   req.GitUserName,
		GitUserEmail:  req.GitUserEmail,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (id, name, path, description, remote_url, default_branch, status, github_token, git_user_name, git_user_email, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		project.ID, project.Name, project.Path, project.Description, project.RemoteURL,
		project.DefaultBranch, project.Status, req.GitHubToken, req.GitUserName, req.GitUserEmail, project.CreatedAt, project.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

func (r *Repository) GetProject(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, path, description, remote_url, default_branch, status, error_message, last_sync_at, github_token, git_user_name, git_user_email, created_at, updated_at
		 FROM projects WHERE id = $1`,
		id).Scan(&project.ID, &project.Name, &project.Path, &project.Description, &project.RemoteURL,
		&project.DefaultBranch, &project.Status, &project.ErrorMessage, &project.LastSyncAt, &project.GitHubToken, &project.GitUserName, &project.GitUserEmail, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return &project, nil
}

func (r *Repository) ListProjects(ctx context.Context) ([]*models.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, path, description, remote_url, default_branch, status, error_message, last_sync_at, github_token, git_user_name, git_user_email, created_at, updated_at
		 FROM projects ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		var project models.Project
		err := rows.Scan(&project.ID, &project.Name, &project.Path, &project.Description, &project.RemoteURL,
			&project.DefaultBranch, &project.Status, &project.ErrorMessage, &project.LastSyncAt, &project.GitHubToken, &project.GitUserName, &project.GitUserEmail, &project.CreatedAt, &project.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, &project)
	}

	return projects, nil
}

func (r *Repository) UpdateProject(ctx context.Context, id uuid.UUID, req *models.UpdateProjectRequest) (*models.Project, error) {
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Path != nil {
		updates = append(updates, fmt.Sprintf("path = $%d", argIndex))
		args = append(args, *req.Path)
		argIndex++
	}
	if req.RemoteURL != nil {
		updates = append(updates, fmt.Sprintf("remote_url = $%d", argIndex))
		args = append(args, *req.RemoteURL)
		argIndex++
	}
	if req.DefaultBranch != nil {
		updates = append(updates, fmt.Sprintf("default_branch = $%d", argIndex))
		args = append(args, *req.DefaultBranch)
		argIndex++
	}
	if req.GitHubToken != nil {
		updates = append(updates, fmt.Sprintf("github_token = $%d", argIndex))
		args = append(args, *req.GitHubToken)
		argIndex++
	}
	if req.GitUserName != nil {
		updates = append(updates, fmt.Sprintf("git_user_name = $%d", argIndex))
		args = append(args, *req.GitUserName)
		argIndex++
	}
	if req.GitUserEmail != nil {
		updates = append(updates, fmt.Sprintf("git_user_email = $%d", argIndex))
		args = append(args, *req.GitUserEmail)
		argIndex++
	}

	if len(updates) == 0 {
		return r.GetProject(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE projects SET %s WHERE id = $%d",
		fmt.Sprintf("%s", updates), argIndex)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return r.GetProject(ctx, id)
}

func (r *Repository) DeleteProject(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM projects WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

func (r *Repository) UpdateProjectStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET status = $1, error_message = $2, last_sync_at = $3, updated_at = $4 WHERE id = $5`,
		status, errorMessage, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update project status: %w", err)
	}
	return nil
}

func (r *Repository) GetSession(ctx context.Context, id uuid.UUID) (*models.Session, error) {
	var session models.Session
	err := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, initial_prompt, model, max_turns, status,
		 worktree_id, worktree_path, conversation_id, started_at, completed_at, error, metadata,
		 push_branch, push_commit_hash, last_pushed_at, created_at, updated_at 
		 FROM sessions WHERE id = $1`,
		id).Scan(&session.ID, &session.ProjectID, &session.InitialPrompt,
		&session.Model, &session.MaxTurns, &session.Status, &session.WorktreeID,
		&session.WorktreePath, &session.ConversationID, &session.StartedAt, &session.CompletedAt,
		&session.Error, &session.Metadata, &session.PushBranch, &session.PushCommitHash,
		&session.LastPushedAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

func (r *Repository) ListSessions(ctx context.Context, query *models.ListSessionsQuery) (*models.ListResponse[models.Session], error) {
	baseQuery := `
		SELECT id, project_id, initial_prompt, model, max_turns, status,
		       worktree_id, worktree_path, conversation_id, started_at, completed_at, error, metadata,
		       push_branch, push_commit_hash, last_pushed_at, created_at, updated_at
		FROM sessions`

	countQuery := "SELECT COUNT(*) FROM sessions"
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if query.Status != "" {
		whereClause += fmt.Sprintf(" status = $%d", argIndex)
		args = append(args, query.Status)
		argIndex++
	}

	if query.ProjectID != "" {
		if whereClause != "" {
			whereClause += " AND"
		}
		whereClause += fmt.Sprintf(" project_id = $%d", argIndex)
		projectID, _ := uuid.Parse(query.ProjectID)
		args = append(args, projectID)
		argIndex++
	}

	if whereClause != "" {
		baseQuery += " WHERE" + whereClause
		countQuery += " WHERE" + whereClause
	}

	var totalItems int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, fmt.Errorf("failed to count sessions: %w", err)
	}

	offset := (query.Page - 1) * query.PageSize
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, query.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var session models.Session
		err := rows.Scan(&session.ID, &session.ProjectID, &session.InitialPrompt,
			&session.Model, &session.MaxTurns, &session.Status, &session.WorktreeID,
			&session.WorktreePath, &session.ConversationID, &session.StartedAt, &session.CompletedAt,
			&session.Error, &session.Metadata, &session.PushBranch, &session.PushCommitHash,
			&session.LastPushedAt, &session.CreatedAt, &session.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	totalPages := (totalItems + query.PageSize - 1) / query.PageSize

	return &models.ListResponse[models.Session]{
		Items:      sessions,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

func (r *Repository) GetSessionLogs(ctx context.Context, sessionID uuid.UUID, limit int) ([]*models.SessionLog, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, session_id, log_type, content, json_data, created_at
		 FROM session_logs WHERE session_id = $1
		 ORDER BY created_at ASC LIMIT $2`,
		sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get session logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.SessionLog
	for rows.Next() {
		var log models.SessionLog
		err := rows.Scan(&log.ID, &log.SessionID, &log.LogType, &log.Content, &log.JSONData, &log.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *Repository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	// First get the session to clean up worktree
	session, err := r.GetSession(ctx, id)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get session for cleanup: %w", err)
	}

	// Delete the session (this will cascade delete session_logs due to FK constraint)
	_, err = r.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Clean up worktree directory if session was found
	if session != nil && session.WorktreePath != "" {
		// Note: We don't return an error if worktree cleanup fails,
		// as the database deletion is more important
		if err := os.RemoveAll(session.WorktreePath); err != nil {
			log.Printf("Warning: failed to remove worktree %s: %v", session.WorktreePath, err)
		}
	}

	return nil
}

// UpdateSessionPush updates the session with push information
func (r *Repository) UpdateSessionPush(ctx context.Context, id uuid.UUID, branch, commitHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions 
		 SET push_branch = $2, push_commit_hash = $3, last_pushed_at = NOW(), updated_at = NOW()
		 WHERE id = $1`,
		id, branch, commitHash)
	if err != nil {
		return fmt.Errorf("failed to update session push info: %w", err)
	}
	return nil
}

// GetSettings retrieves the global settings from the database
func (r *Repository) GetSettings(ctx context.Context) (*models.Settings, error) {
	var settings models.Settings
	err := r.db.QueryRowContext(ctx,
		`SELECT id, default_github_token, claude_code_binary, claude_code_api_key, max_concurrent_sessions, 
		 default_max_turns, git_user_name, git_user_email, created_at, updated_at
		 FROM settings LIMIT 1`).Scan(
		&settings.ID, &settings.DefaultGitHubToken, &settings.ClaudeCodeBinary,
		&settings.ClaudeCodeAPIKey, &settings.MaxConcurrentSessions, &settings.DefaultMaxTurns, &settings.GitUserName, &settings.GitUserEmail,
		&settings.CreatedAt, &settings.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// No settings exist, return defaults
			return &models.Settings{
				ID:                    uuid.New(),
				ClaudeCodeBinary:      "/usr/local/bin/claude",
				MaxConcurrentSessions: 10,
				CreatedAt:             time.Now(),
				UpdatedAt:             time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	return &settings, nil
}

// UpdateSettings updates the global settings in the database
func (r *Repository) UpdateSettings(ctx context.Context, req *models.UpdateSettingsRequest) (*models.Settings, error) {
	// First, get the current settings to see if they exist
	settings, err := r.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current settings: %w", err)
	}

	// Check if we need to insert or update
	var exists bool
	err = r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM settings LIMIT 1)").Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check if settings exist: %w", err)
	}

	if !exists {
		// Insert new settings row
		settings.ID = uuid.New()
		settings.CreatedAt = time.Now()
		settings.UpdatedAt = time.Now()
		
		_, err = r.db.ExecContext(ctx,
			`INSERT INTO settings (id, default_github_token, claude_code_binary, claude_code_api_key, max_concurrent_sessions,
			 default_max_turns, git_user_name, git_user_email, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			settings.ID, req.DefaultGitHubToken, settings.ClaudeCodeBinary, req.ClaudeCodeAPIKey,
			settings.MaxConcurrentSessions, req.DefaultMaxTurns, req.GitUserName, req.GitUserEmail,
			settings.CreatedAt, settings.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to insert settings: %w", err)
		}
	} else {
		// Update existing settings
		updates := []string{}
		args := []interface{}{}
		argIndex := 1

		if req.DefaultGitHubToken != nil {
			updates = append(updates, fmt.Sprintf("default_github_token = $%d", argIndex))
			args = append(args, *req.DefaultGitHubToken)
			argIndex++
		}
		if req.ClaudeCodeBinary != nil {
			updates = append(updates, fmt.Sprintf("claude_code_binary = $%d", argIndex))
			args = append(args, *req.ClaudeCodeBinary)
			argIndex++
		}
		if req.ClaudeCodeAPIKey != nil {
			updates = append(updates, fmt.Sprintf("claude_code_api_key = $%d", argIndex))
			args = append(args, *req.ClaudeCodeAPIKey)
			argIndex++
		}
		if req.MaxConcurrentSessions != nil {
			updates = append(updates, fmt.Sprintf("max_concurrent_sessions = $%d", argIndex))
			args = append(args, *req.MaxConcurrentSessions)
			argIndex++
		}
		if req.DefaultMaxTurns != nil {
			updates = append(updates, fmt.Sprintf("default_max_turns = $%d", argIndex))
			args = append(args, *req.DefaultMaxTurns)
			argIndex++
		}
		if req.GitUserName != nil {
			updates = append(updates, fmt.Sprintf("git_user_name = $%d", argIndex))
			args = append(args, *req.GitUserName)
			argIndex++
		}
		if req.GitUserEmail != nil {
			updates = append(updates, fmt.Sprintf("git_user_email = $%d", argIndex))
			args = append(args, *req.GitUserEmail)
			argIndex++
		}

		if len(updates) > 0 {
			updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
			args = append(args, time.Now())
			argIndex++

			query := fmt.Sprintf("UPDATE settings SET %s", strings.Join(updates, ", "))
			_, err = r.db.ExecContext(ctx, query, args...)
			if err != nil {
				return nil, fmt.Errorf("failed to update settings: %w", err)
			}
		}
	}

	// Return updated settings
	return r.GetSettings(ctx)
}
