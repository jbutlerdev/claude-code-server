package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jbutlerdev/claude-code-server/internal/config"
	"github.com/jbutlerdev/claude-code-server/internal/database"
	"github.com/jbutlerdev/claude-code-server/internal/git"
	"github.com/jbutlerdev/claude-code-server/internal/models"
	"github.com/jbutlerdev/claude-code-server/internal/project"
	"github.com/jbutlerdev/claude-code-server/internal/session"
	"github.com/jbutlerdev/claude-code-server/internal/settings"
)

type API struct {
	config          *config.Config
	db              *sql.DB
	repo            *database.Repository
	sessionManager  *session.Manager
	projectService  *project.Service
	gitService      *git.Service
	settingsService *settings.Service
}

func (api *API) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (api *API) readyHandler(w http.ResponseWriter, r *http.Request) {
	if err := api.db.Ping(); err != nil {
		http.Error(w, "Database not ready", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (api *API) openAPIHandler(w http.ResponseWriter, r *http.Request) {
	spec := api.generateOpenAPISpec()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}

func (api *API) createSessionHandler(w http.ResponseWriter, r *http.Request) {
	var req models.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Prompt == "" {
		api.writeError(w, http.StatusBadRequest, "Prompt is required", nil)
		return
	}

	if req.ProjectID == "" {
		api.writeError(w, http.StatusBadRequest, "Project ID is required", nil)
		return
	}

	session, err := api.sessionManager.CreateSession(r.Context(), &req)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to create session", err)
		return
	}

	response := models.CreateSessionResponse{
		ID:           session.ID.String(),
		Status:       string(session.Status),
		CreatedAt:    session.CreatedAt.Format(time.RFC3339),
		WebSocketURL: fmt.Sprintf("ws://%s/api/sessions/%s/logs", r.Host, session.ID),
		WorktreePath: session.WorktreePath,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (api *API) listSessionsHandler(w http.ResponseWriter, r *http.Request) {
	query := &models.ListSessionsQuery{
		Page:      getIntQuery(r, "page", 1),
		PageSize:  getIntQuery(r, "page_size", 10),
		Status:    r.URL.Query().Get("status"),
		ProjectID: r.URL.Query().Get("project_id"),
	}

	if query.PageSize > 100 {
		query.PageSize = 100
	}

	result, err := api.repo.ListSessions(r.Context(), query)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to list sessions", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (api *API) getSessionHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid session ID", err)
		return
	}

	session, err := api.repo.GetSession(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			api.writeError(w, http.StatusNotFound, "Session not found", nil)
			return
		}
		api.writeError(w, http.StatusInternalServerError, "Failed to get session", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (api *API) deleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid session ID", err)
		return
	}

	// Cancel if it's an active session
	if active, ok := api.sessionManager.GetActiveSession(id); ok {
		active.Cancel()
	}

	// Delete the session from database
	err = api.repo.DeleteSession(r.Context(), id)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to delete session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *API) resumeSessionHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid session ID", err)
		return
	}

	var req models.ResumeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Prompt == "" {
		api.writeError(w, http.StatusBadRequest, "Prompt is required", nil)
		return
	}

	err = api.sessionManager.ResumeSession(r.Context(), id, req.Prompt)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to resume session", err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (api *API) getSessionLogsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid session ID", err)
		return
	}

	if active, ok := api.sessionManager.GetActiveSession(id); ok {
		api.streamLiveLogs(w, r, active)
		return
	}

	logs, err := api.repo.GetSessionLogs(r.Context(), id, 1000)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get logs", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (api *API) createProjectHandler(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Name == "" || req.Path == "" {
		api.writeError(w, http.StatusBadRequest, "Name and path are required", nil)
		return
	}

	project, err := api.projectService.CreateProject(r.Context(), &req)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to create project", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

func (api *API) listProjectsHandler(w http.ResponseWriter, r *http.Request) {
	projects, err := api.repo.ListProjects(r.Context())
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to list projects", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func (api *API) getProjectHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	project, err := api.repo.GetProject(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			api.writeError(w, http.StatusNotFound, "Project not found", nil)
			return
		}
		api.writeError(w, http.StatusInternalServerError, "Failed to get project", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func (api *API) updateProjectHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	var req models.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	project, err := api.repo.UpdateProject(r.Context(), id, &req)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to update project", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func (api *API) deleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid project ID", err)
		return
	}

	err = api.repo.DeleteProject(r.Context(), id)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to delete project", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *API) streamLiveLogs(w http.ResponseWriter, r *http.Request, active *session.ActiveSession) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case log, ok := <-active.LogsChan:
			if !ok {
				return
			}

			data, _ := json.Marshal(log)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

func (api *API) writeError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := models.ErrorResponse{
		Error:   message,
		Message: message,
	}

	if err != nil {
		response.Message = err.Error()
	}

	json.NewEncoder(w).Encode(response)
}

func getIntQuery(r *http.Request, key string, defaultValue int) int {
	if value := r.URL.Query().Get(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func (api *API) pushSessionToRemoteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid session ID", err)
		return
	}

	var req models.PushToRemoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get session and project information
	session, err := api.repo.GetSession(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			api.writeError(w, http.StatusNotFound, "Session not found", nil)
			return
		}
		api.writeError(w, http.StatusInternalServerError, "Failed to get session", err)
		return
	}

	project, err := api.repo.GetProject(r.Context(), session.ProjectID)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get project", err)
		return
	}

	// Check if project has a remote URL
	if project.RemoteURL == nil || *project.RemoteURL == "" {
		api.writeError(w, http.StatusBadRequest, "Project has no remote URL configured", nil)
		return
	}

	// Determine which GitHub token to use (project-specific or default)
	var githubToken string
	if project.GitHubToken != nil && *project.GitHubToken != "" {
		githubToken = *project.GitHubToken
	} else {
		githubToken = api.settingsService.GetDefaultGitHubToken()
	}

	if githubToken == "" {
		api.writeError(w, http.StatusBadRequest, "No GitHub token configured", nil)
		return
	}

	// Determine which git identity to use (project-specific or default)
	var gitUserName, gitUserEmail string
	if project.GitUserName != nil && *project.GitUserName != "" {
		gitUserName = *project.GitUserName
	} else {
		gitUserName, _ = api.settingsService.GetGitIdentity()
	}
	
	if project.GitUserEmail != nil && *project.GitUserEmail != "" {
		gitUserEmail = *project.GitUserEmail
	} else {
		_, gitUserEmail = api.settingsService.GetGitIdentity()
	}

	// Set git identity if we have one
	if gitUserName != "" || gitUserEmail != "" {
		err = api.gitService.SetGitIdentity(session.WorktreePath, gitUserName, gitUserEmail)
		if err != nil {
			api.writeError(w, http.StatusInternalServerError, "Failed to set git identity", err)
			return
		}
	}

	// Check if there are any changes to push
	hasChanges, err := api.gitService.HasUncommittedChanges(session.WorktreePath)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to check for changes", err)
		return
	}

	// Determine branch name
	var branchName string
	if session.PushBranch != nil && *session.PushBranch != "" {
		// Use existing branch for subsequent pushes
		branchName = *session.PushBranch
	} else {
		// Generate new branch name for first push
		branchName = api.gitService.GenerateBranchName(session.ID.String())
	}

	// Create/switch to branch
	err = api.gitService.CreateAndSwitchBranch(session.WorktreePath, branchName)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to create/switch branch", err)
		return
	}

	var commitHash string
	if hasChanges {
		// Set default commit message if not provided
		commitMessage := req.CommitMessage
		if commitMessage == "" {
			if session.PushBranch != nil {
				commitMessage = "Additional changes from Claude Code session"
			} else {
				commitMessage = "Changes from Claude Code session"
			}
		}

		// Commit changes
		commitHash, err = api.gitService.CommitChanges(session.WorktreePath, commitMessage)
		if err != nil {
			api.writeError(w, http.StatusInternalServerError, "Failed to commit changes", err)
			return
		}
	} else {
		// No new changes, get current commit hash
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = session.WorktreePath
		output, err := cmd.CombinedOutput()
		if err != nil {
			api.writeError(w, http.StatusInternalServerError, "Failed to get current commit", err)
			return
		}
		commitHash = strings.TrimSpace(string(output))
	}

	// Push to remote
	err = api.gitService.PushToRemote(session.WorktreePath, branchName, *project.RemoteURL, githubToken)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to push to remote", err)
		return
	}

	// Update session with push information
	err = api.repo.UpdateSessionPush(r.Context(), session.ID, branchName, commitHash)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to update session", err)
		return
	}

	// Return success response
	response := models.PushToRemoteResponse{
		Branch:     branchName,
		CommitHash: commitHash,
		PushedAt:   time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *API) getSettingsHandler(w http.ResponseWriter, r *http.Request) {
	settings := api.settingsService.GetSettings()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (api *API) updateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	if err := api.settingsService.UpdateSettings(&req); err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to update settings", err)
		return
	}
	
	// Return updated settings
	settings := api.settingsService.GetSettings()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}
