package session

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jbutlerdev/claude-code-server/internal/config"
	"github.com/jbutlerdev/claude-code-server/internal/models"
)

type SettingsService interface {
	GetDefaultMaxTurns() int
}

type Manager struct {
	config    *config.Config
	db        *sql.DB
	settings  SettingsService
	sessions  map[uuid.UUID]*ActiveSession
	mu        sync.RWMutex
	semaphore chan struct{}
}

type ActiveSession struct {
	Session      *models.Session
	Process      *exec.Cmd
	LogsChan     chan models.SessionLog
	Cancel       context.CancelFunc
	ResumeFromID *string // Claude Code conversation ID to resume from
}

func NewManager(cfg *config.Config, db *sql.DB, settings SettingsService) *Manager {
	return &Manager{
		config:    cfg,
		db:        db,
		settings:  settings,
		sessions:  make(map[uuid.UUID]*ActiveSession),
		semaphore: make(chan struct{}, cfg.MaxConcurrentSessions),
	}
}

func (m *Manager) Start() error {
	if err := os.MkdirAll(m.config.WorktreeBasePath, 0755); err != nil {
		return fmt.Errorf("failed to create worktree base path: %w", err)
	}

	go m.cleanupLoop()
	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		session.Cancel()
	}
}

func (m *Manager) CreateSession(ctx context.Context, req *models.CreateSessionRequest) (*models.Session, error) {
	select {
	case m.semaphore <- struct{}{}:
	default:
		return nil, fmt.Errorf("maximum concurrent sessions reached")
	}

	projectID, _ := uuid.Parse(req.ProjectID)
	var project models.Project
	err := m.db.QueryRowContext(ctx,
		"SELECT id, name, path, description, remote_url, default_branch, status, error_message, last_sync_at, created_at, updated_at FROM projects WHERE id = $1",
		projectID).Scan(&project.ID, &project.Name, &project.Path, &project.Description, &project.RemoteURL, &project.DefaultBranch, &project.Status, &project.ErrorMessage, &project.LastSyncAt, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		<-m.semaphore
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Check if project is ready
	if project.Status != "ready" {
		<-m.semaphore
		if project.Status == "error" && project.ErrorMessage != nil {
			return nil, fmt.Errorf("project has error: %s", *project.ErrorMessage)
		}
		return nil, fmt.Errorf("project is not ready (status: %s)", project.Status)
	}

	worktreeID := generateWorktreeID()
	worktreePath := filepath.Join(m.config.WorktreeBasePath, worktreeID)

	if err := m.createWorktree(project, worktreePath); err != nil {
		<-m.semaphore
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	session := &models.Session{
		ID:            uuid.New(),
		ProjectID:     projectID,
		InitialPrompt: req.Prompt,
		Model:         "claude-3-5-sonnet-20241022", // Fixed model for Claude Code CLI
		MaxTurns:      req.MaxTurns,
		Status:        models.SessionStatusStarting,
		WorktreeID:    worktreeID,
		WorktreePath:  worktreePath,
		StartedAt:     time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if session.MaxTurns == 0 {
		session.MaxTurns = m.settings.GetDefaultMaxTurns()
	}

	_, err = m.db.ExecContext(ctx,
		`INSERT INTO sessions (id, project_id, initial_prompt, model, max_turns, 
		 status, worktree_id, worktree_path, started_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		session.ID, session.ProjectID, session.InitialPrompt, session.Model,
		session.MaxTurns, session.Status, session.WorktreeID, session.WorktreePath,
		session.StartedAt, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		<-m.semaphore
		os.RemoveAll(worktreePath)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	activeCtx, cancel := context.WithCancel(context.Background())
	active := &ActiveSession{
		Session:  session,
		LogsChan: make(chan models.SessionLog, 100),
		Cancel:   cancel,
	}

	m.mu.Lock()
	m.sessions[session.ID] = active
	m.mu.Unlock()

	go m.runSession(activeCtx, active)

	return session, nil
}

func (m *Manager) createWorktree(project models.Project, worktreePath string) error {
	// Check if the project repository exists
	if _, err := os.Stat(filepath.Join(project.Path, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("repository not found at %s", project.Path)
	}

	// Create worktree base directory if it doesn't exist
	if err := os.MkdirAll(m.config.WorktreeBasePath, 0755); err != nil {
		return fmt.Errorf("failed to create worktree base path: %w", err)
	}

	// Generate unique branch name for this session
	branchName := fmt.Sprintf("claude-session-%s", generateWorktreeID())

	// Create git worktree using native git command
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	cmd.Dir = project.Path
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If branch already exists, try creating worktree without new branch
		cmd = exec.Command("git", "worktree", "add", worktreePath, project.DefaultBranch)
		cmd.Dir = project.Path
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to create worktree: %s", string(output))
		}
	}

	return nil
}

func (m *Manager) runSession(ctx context.Context, active *ActiveSession) {
	defer func() {
		<-m.semaphore
		m.mu.Lock()
		delete(m.sessions, active.Session.ID)
		m.mu.Unlock()
		close(active.LogsChan)
	}()

	// Log session start
	m.logToSession(active, "system", fmt.Sprintf("Starting Claude Code session %s", active.Session.ID))
	m.logToSession(active, "system", fmt.Sprintf("Working directory: %s", active.Session.WorktreePath))
	// Model is now fixed to sonnet for Claude Code CLI
	m.logToSession(active, "system", fmt.Sprintf("Max turns: %d", active.Session.MaxTurns))
	m.logToSession(active, "system", fmt.Sprintf("Claude binary: %s", m.config.ClaudeCodeBinary))

	// Check if worktree directory exists
	if _, err := os.Stat(active.Session.WorktreePath); os.IsNotExist(err) {
		m.handleSessionError(active, fmt.Errorf("worktree directory does not exist: %s", active.Session.WorktreePath))
		return
	}

	// Check if Claude binary exists
	if _, err := os.Stat(m.config.ClaudeCodeBinary); os.IsNotExist(err) {
		m.handleSessionError(active, fmt.Errorf("claude binary not found: %s", m.config.ClaudeCodeBinary))
		return
	}

	m.updateSessionStatus(active.Session.ID, models.SessionStatusRunning)

	args := []string{
		"-p", active.Session.InitialPrompt,
		"--max-turns", fmt.Sprintf("%d", active.Session.MaxTurns),
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
	}

	m.logToSession(active, "system", fmt.Sprintf("Command: %s %s", m.config.ClaudeCodeBinary, strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, m.config.ClaudeCodeBinary, args...)
	cmd.Dir = active.Session.WorktreePath

	// Set up environment
	env := append(os.Environ(),
		fmt.Sprintf("ANTHROPIC_API_KEY=%s", m.config.ClaudeCodeAPIKey),
		"SHELL=/bin/bash",
		"HOME=/home/appuser",
		"USER=appuser",
		"NODE_NO_WARNINGS=1",
	)
	cmd.Env = env

	m.logToSession(active, "system", fmt.Sprintf("Environment variables set: %d total", len(env)))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.handleSessionError(active, fmt.Errorf("failed to create stdout pipe: %w", err))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.handleSessionError(active, fmt.Errorf("failed to create stderr pipe: %w", err))
		return
	}

	m.logToSession(active, "system", "Starting Claude process...")

	if err := cmd.Start(); err != nil {
		m.handleSessionError(active, fmt.Errorf("failed to start claude: %w", err))
		return
	}

	m.logToSession(active, "system", fmt.Sprintf("Claude process started with PID: %d", cmd.Process.Pid))
	active.Process = cmd

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		m.streamOutput(active, stdout, "stdout")
	}()

	go func() {
		defer wg.Done()
		m.streamOutput(active, stderr, "stderr")
	}()

	m.logToSession(active, "system", "Waiting for Claude process to complete...")
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == nil {
			// Get exit code for better error reporting
			exitCode := "unknown"
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = fmt.Sprintf("%d", exitError.ExitCode())
			}
			m.handleSessionError(active, fmt.Errorf("claude process failed with exit code %s: %w", exitCode, err))
			return
		} else {
			m.logToSession(active, "system", "Claude process cancelled by context")
		}
	}

	m.logToSession(active, "system", "Claude process completed successfully")
	m.updateSessionStatus(active.Session.ID, models.SessionStatusCompleted)
}

func (m *Manager) streamOutput(active *ActiveSession, reader io.Reader, logType string) {
	scanner := bufio.NewScanner(reader)
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		content := scanner.Text()

		sessionLog := models.SessionLog{
			SessionID: active.Session.ID,
			LogType:   logType,
			Content:   content,
			CreatedAt: time.Now(),
		}

		// Try to parse JSON from stdout (Claude Code output)
		if logType == "stdout" && strings.TrimSpace(content) != "" {
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
				// Successfully parsed JSON
				jsonb := models.JSONB(jsonData)
				sessionLog.JSONData = &jsonb
				
				// Extract conversation ID and store it in the session
				if sessionID, exists := jsonData["session_id"].(string); exists && sessionID != "" {
					m.updateSessionConversationID(active.Session.ID, sessionID)
				}
			}
		}

		// Log to server console for debugging (but truncate very long JSON)
		logContent := content
		if len(logContent) > 200 {
			logContent = logContent[:200] + "..."
		}
		log.Printf("Session %s [%s]: %s", active.Session.ID, logType, logContent)

		select {
		case active.LogsChan <- sessionLog:
		default:
			log.Printf("Warning: log channel full for session %s", active.Session.ID)
		}

		// Store in database with JSON data if available
		if sessionLog.JSONData != nil {
			_, err := m.db.Exec(
				"INSERT INTO session_logs (session_id, log_type, content, json_data, created_at) VALUES ($1, $2, $3, $4, $5)",
				sessionLog.SessionID, sessionLog.LogType, sessionLog.Content, sessionLog.JSONData, sessionLog.CreatedAt)
			if err != nil {
				log.Printf("Failed to save log with JSON for session %s: %v", active.Session.ID, err)
			}
		} else {
			_, err := m.db.Exec(
				"INSERT INTO session_logs (session_id, log_type, content, created_at) VALUES ($1, $2, $3, $4)",
				sessionLog.SessionID, sessionLog.LogType, sessionLog.Content, sessionLog.CreatedAt)
			if err != nil {
				log.Printf("Failed to save log for session %s: %v", active.Session.ID, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		m.logToSession(active, "system", fmt.Sprintf("Error reading %s: %v", logType, err))
	}

	m.logToSession(active, "system", fmt.Sprintf("Finished reading %s (%d lines)", logType, lineCount))
}

func (m *Manager) logToSession(active *ActiveSession, logType, message string) {
	log.Printf("Session %s [%s]: %s", active.Session.ID, logType, message)

	sessionLog := models.SessionLog{
		SessionID: active.Session.ID,
		LogType:   logType,
		Content:   message,
		CreatedAt: time.Now(),
	}

	// Store log in database (no JSON data for system logs)
	_, err := m.db.Exec(
		"INSERT INTO session_logs (session_id, log_type, content, created_at) VALUES ($1, $2, $3, $4)",
		sessionLog.SessionID, sessionLog.LogType, sessionLog.Content, sessionLog.CreatedAt)
	if err != nil {
		log.Printf("Failed to store session log: %v", err)
	}

	// Send to live stream
	select {
	case active.LogsChan <- sessionLog:
	default:
	}
}

func (m *Manager) handleSessionError(active *ActiveSession, err error) {
	errStr := err.Error()
	log.Printf("Session %s failed: %s", active.Session.ID, errStr)

	_, dbErr := m.db.Exec(
		"UPDATE sessions SET status = $1, error = $2, completed_at = $3 WHERE id = $4",
		models.SessionStatusFailed, errStr, time.Now(), active.Session.ID)
	if dbErr != nil {
		log.Printf("Failed to update session error: %v", dbErr)
	}

	m.logToSession(active, "error", fmt.Sprintf("Session failed: %s", errStr))
}

func (m *Manager) updateSessionStatus(sessionID uuid.UUID, status models.SessionStatus) {
	query := "UPDATE sessions SET status = $1"
	args := []interface{}{status}

	if status == models.SessionStatusCompleted || status == models.SessionStatusFailed {
		query += ", completed_at = $2 WHERE id = $3"
		args = append(args, time.Now(), sessionID)
	} else {
		query += " WHERE id = $2"
		args = append(args, sessionID)
	}

	_, err := m.db.Exec(query, args...)
	if err != nil {
		log.Printf("Failed to update session status: %v", err)
	}
}

func (m *Manager) updateSessionConversationID(sessionID uuid.UUID, conversationID string) {
	_, err := m.db.Exec(
		"UPDATE sessions SET conversation_id = $1 WHERE id = $2",
		conversationID, sessionID)
	if err != nil {
		log.Printf("Failed to update session conversation ID: %v", err)
	}
}

func (m *Manager) GetActiveSession(sessionID uuid.UUID) (*ActiveSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[sessionID]
	return session, ok
}

func (m *Manager) ResumeSession(ctx context.Context, sessionID uuid.UUID, prompt string) (*models.Session, error) {
	var session models.Session
	err := m.db.QueryRowContext(ctx,
		`SELECT id, project_id, worktree_id, worktree_path, max_turns, conversation_id 
		 FROM sessions WHERE id = $1 AND status = 'completed'`,
		sessionID).Scan(&session.ID, &session.ProjectID, &session.WorktreeID, &session.WorktreePath,
		&session.MaxTurns, &session.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("session not found or not completed: %w", err)
	}

	if session.ConversationID == nil || *session.ConversationID == "" {
		return nil, fmt.Errorf("session does not have a conversation ID for proper resumption")
	}

	// Create new session for resumption tracking
	newSession := &models.Session{
		ID:             uuid.New(),
		ProjectID:      session.ProjectID,
		InitialPrompt:  prompt,
		Model:          "claude-3-5-sonnet-20241022", // Fixed model for Claude Code CLI
		MaxTurns:       session.MaxTurns,
		Status:         models.SessionStatusStarting,
		WorktreeID:     session.WorktreeID, // Reuse same worktree ID since constraint removed
		WorktreePath:   session.WorktreePath,
		StartedAt:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Insert new session with reference to original
	_, err = m.db.ExecContext(ctx,
		`INSERT INTO sessions (id, project_id, initial_prompt, model, max_turns, 
		 status, worktree_id, worktree_path, started_at, created_at, updated_at, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		newSession.ID, newSession.ProjectID, newSession.InitialPrompt, newSession.Model,
		newSession.MaxTurns, newSession.Status, newSession.WorktreeID, newSession.WorktreePath,
		newSession.StartedAt, newSession.CreatedAt, newSession.UpdatedAt,
		fmt.Sprintf(`{"resumed_from": "%s", "original_conversation_id": "%s"}`, sessionID.String(), *session.ConversationID))
	if err != nil {
		return nil, fmt.Errorf("failed to create resume session: %w", err)
	}

	// Start session with resume flag
	activeCtx, cancel := context.WithCancel(context.Background())
	active := &ActiveSession{
		Session:       newSession,
		LogsChan:      make(chan models.SessionLog, 100),
		Cancel:        cancel,
		ResumeFromID:  session.ConversationID, // Store conversation ID for resumption
	}

	m.mu.Lock()
	m.sessions[newSession.ID] = active
	m.mu.Unlock()

	go m.runResumedSession(activeCtx, active, prompt)

	return newSession, nil
}

func (m *Manager) runResumedSession(ctx context.Context, active *ActiveSession, prompt string) {
	defer func() {
		<-m.semaphore
		m.mu.Lock()
		delete(m.sessions, active.Session.ID)
		m.mu.Unlock()
		close(active.LogsChan)
	}()

	// Log session start
	m.logToSession(active, "system", fmt.Sprintf("Resuming Claude Code session %s", active.Session.ID))
	m.logToSession(active, "system", fmt.Sprintf("Original conversation ID: %s", *active.ResumeFromID))
	m.logToSession(active, "system", fmt.Sprintf("Working directory: %s", active.Session.WorktreePath))

	// Check if worktree directory exists
	if _, err := os.Stat(active.Session.WorktreePath); os.IsNotExist(err) {
		m.handleSessionError(active, fmt.Errorf("worktree directory does not exist: %s", active.Session.WorktreePath))
		return
	}

	m.updateSessionStatus(active.Session.ID, models.SessionStatusRunning)

	// Use Claude Code's resume functionality
	args := []string{
		"--resume", *active.ResumeFromID, // Resume from specific conversation ID
		"--max-turns", fmt.Sprintf("%d", active.Session.MaxTurns),
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
	}

	// If user provided additional prompt, add it
	if strings.TrimSpace(prompt) != "" {
		args = append(args, "-p", prompt)
	}

	m.logToSession(active, "system", fmt.Sprintf("Command: %s %s", m.config.ClaudeCodeBinary, strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, m.config.ClaudeCodeBinary, args...)
	cmd.Dir = active.Session.WorktreePath

	// Set up environment
	env := append(os.Environ(),
		fmt.Sprintf("ANTHROPIC_API_KEY=%s", m.config.ClaudeCodeAPIKey),
		"SHELL=/bin/bash",
		"HOME=/home/appuser",
		"USER=appuser",
		"NODE_NO_WARNINGS=1",
	)
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.handleSessionError(active, fmt.Errorf("failed to create stdout pipe: %w", err))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.handleSessionError(active, fmt.Errorf("failed to create stderr pipe: %w", err))
		return
	}

	if err := cmd.Start(); err != nil {
		m.handleSessionError(active, fmt.Errorf("failed to start claude: %w", err))
		return
	}

	m.logToSession(active, "system", fmt.Sprintf("Claude resume process started with PID: %d", cmd.Process.Pid))
	active.Process = cmd

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		m.streamOutput(active, stdout, "stdout")
	}()

	go func() {
		defer wg.Done()
		m.streamOutput(active, stderr, "stderr")
	}()

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == nil {
			exitCode := "unknown"
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = fmt.Sprintf("%d", exitError.ExitCode())
			}
			m.handleSessionError(active, fmt.Errorf("claude resume process failed with exit code %s: %w", exitCode, err))
			return
		} else {
			m.logToSession(active, "system", "Claude resume process cancelled by context")
		}
	}

	m.logToSession(active, "system", "Claude resume process completed successfully")
	m.updateSessionStatus(active.Session.ID, models.SessionStatusCompleted)
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupOldSessions()
	}
}

func (m *Manager) cleanupOldSessions() {
	cutoff := time.Now().AddDate(0, 0, -m.config.LogRetentionDays)

	rows, err := m.db.Query(
		"SELECT worktree_path FROM sessions WHERE created_at < $1 AND status IN ('completed', 'failed')",
		cutoff)
	if err != nil {
		log.Printf("Failed to query old sessions: %v", err)
		return
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		paths = append(paths, path)
	}

	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			log.Printf("Failed to remove worktree: %v", err)
		}
	}

	_, err = m.db.Exec("DELETE FROM sessions WHERE created_at < $1", cutoff)
	if err != nil {
		log.Printf("Failed to delete old sessions: %v", err)
	}
}

func generateWorktreeID() string {
	return fmt.Sprintf("%d-%s", time.Now().Unix(), uuid.New().String()[:8])
}
