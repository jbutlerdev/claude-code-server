package api

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jbutlerdev/claude-code-server/internal/models"
)

var templates *template.Template

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
	EnableCompression: false,
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
}

func init() {
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments")
			}
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"duration": func(start, end time.Time) string {
			if end.IsZero() {
				return "Running..."
			}
			d := end.Sub(start)
			if d < time.Minute {
				return fmt.Sprintf("%.0fs", d.Seconds())
			} else if d < time.Hour {
				return fmt.Sprintf("%.1fm", d.Minutes())
			} else {
				return fmt.Sprintf("%.1fh", d.Hours())
			}
		},
	}
	templates = template.New("").Funcs(funcMap)
	templates = template.Must(templates.ParseGlob(filepath.Join("templates", "*.html")))
	templates = template.Must(templates.ParseGlob(filepath.Join("templates", "partials", "*.html")))
}

func (api *API) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	query := &models.ListSessionsQuery{
		Page:     1,
		PageSize: 10,
	}

	sessions, err := api.repo.ListSessions(r.Context(), query)
	if err != nil {
		http.Error(w, "Failed to load sessions", http.StatusInternalServerError)
		return
	}

	projects, err := api.repo.ListProjects(r.Context())
	if err != nil {
		http.Error(w, "Failed to load projects", http.StatusInternalServerError)
		return
	}

	// Create a map of project ID to project name for easy lookup
	projectMap := make(map[uuid.UUID]string)
	for _, p := range projects {
		projectMap[p.ID] = p.Name
	}

	data := struct {
		Title      string
		Sessions   *models.ListResponse[models.Session]
		Projects   []*models.Project
		ProjectMap map[uuid.UUID]string
		Query      *models.ListSessionsQuery
	}{
		Title:      "Dashboard",
		Sessions:   sessions,
		Projects:   projects,
		ProjectMap: projectMap,
		Query:      query,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (api *API) sessionsPageHandler(w http.ResponseWriter, r *http.Request) {
	query := &models.ListSessionsQuery{
		Page:      getIntQuery(r, "page", 1),
		PageSize:  getIntQuery(r, "page_size", 20),
		Status:    r.URL.Query().Get("status"),
		ProjectID: r.URL.Query().Get("project_id"),
	}

	sessions, err := api.repo.ListSessions(r.Context(), query)
	if err != nil {
		http.Error(w, "Failed to load sessions", http.StatusInternalServerError)
		return
	}

	projects, err := api.repo.ListProjects(r.Context())
	if err != nil {
		http.Error(w, "Failed to load projects", http.StatusInternalServerError)
		return
	}

	// Create a map of project ID to project name for easy lookup
	projectMap := make(map[uuid.UUID]string)
	for _, p := range projects {
		projectMap[p.ID] = p.Name
	}

	data := struct {
		Title       string
		Sessions    *models.ListResponse[models.Session]
		Projects    []*models.Project
		ProjectMap  map[uuid.UUID]string
		Query       *models.ListSessionsQuery
	}{
		Title:       "Sessions",
		Sessions:    sessions,
		Projects:    projects,
		ProjectMap:  projectMap,
		Query:       query,
	}

	if err := templates.ExecuteTemplate(w, "sessions.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (api *API) sessionDetailHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	session, err := api.repo.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	logs, err := api.repo.GetSessionLogs(r.Context(), id, 1000)
	if err != nil {
		http.Error(w, "Failed to load logs", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title   string
		Session *models.Session
		Logs    []*models.SessionLog
		IsLive  bool
	}{
		Title:   "Session Details",
		Session: session,
		Logs:    logs,
		IsLive:  session.Status == models.SessionStatusRunning,
	}

	if err := templates.ExecuteTemplate(w, "session_detail.html", data); err != nil {
		log.Printf("Template error for session %s: %v", id, err)
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (api *API) projectsPageHandler(w http.ResponseWriter, r *http.Request) {
	projects, err := api.repo.ListProjects(r.Context())
	if err != nil {
		http.Error(w, "Failed to load projects", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title           string
		Projects        []*models.Project
		DefaultMaxTurns int
	}{
		Title:           "Projects",
		Projects:        projects,
		DefaultMaxTurns: api.settingsService.GetDefaultMaxTurns(),
	}

	if err := templates.ExecuteTemplate(w, "projects.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (api *API) newProjectPageHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title string
	}{
		Title: "New Project",
	}

	if err := templates.ExecuteTemplate(w, "new_project.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (api *API) settingsPageHandler(w http.ResponseWriter, r *http.Request) {
	settings := api.settingsService.GetSettings()
	
	data := struct {
		Title    string
		Settings *models.Settings
	}{
		Title:    "Settings",
		Settings: settings,
	}

	if err := templates.ExecuteTemplate(w, "settings.html", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (api *API) apiDocsHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title string
	}{
		Title: "API Documentation",
	}

	if err := templates.ExecuteTemplate(w, "api_docs.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (api *API) htmxSessionsHandler(w http.ResponseWriter, r *http.Request) {
	query := &models.ListSessionsQuery{
		Page:      getIntQuery(r, "page", 1),
		PageSize:  getIntQuery(r, "page_size", 20),
		Status:    r.URL.Query().Get("status"),
		ProjectID: r.URL.Query().Get("project_id"),
	}

	sessions, err := api.repo.ListSessions(r.Context(), query)
	if err != nil {
		http.Error(w, "Failed to load sessions", http.StatusInternalServerError)
		return
	}

	// Fetch projects to create a map for name lookup
	projects, err := api.repo.ListProjects(r.Context())
	if err != nil {
		http.Error(w, "Failed to load projects", http.StatusInternalServerError)
		return
	}

	// Create a map of project ID to project name for easy lookup
	projectMap := make(map[uuid.UUID]string)
	for _, p := range projects {
		projectMap[p.ID] = p.Name
	}

	data := struct {
		Items      []models.Session
		ProjectMap map[uuid.UUID]string
	}{
		Items:      sessions.Items,
		ProjectMap: projectMap,
	}

	if err := templates.ExecuteTemplate(w, "partials/sessions_list.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (api *API) htmxSessionLogsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	if active, ok := api.sessionManager.GetActiveSession(id); ok {
		api.streamLiveLogs(w, r, active)
		return
	}

	logs, err := api.repo.GetSessionLogs(r.Context(), id, 100)
	if err != nil {
		http.Error(w, "Failed to load logs", http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "partials/session_logs.html", logs); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (api *API) htmxResumeSessionHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	prompt := r.FormValue("prompt")
	if prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	_, err = api.sessionManager.ResumeSession(r.Context(), id, prompt)
	if err != nil {
		http.Error(w, "Failed to resume session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (api *API) htmxProjectsHandler(w http.ResponseWriter, r *http.Request) {
	projects, err := api.repo.ListProjects(r.Context())
	if err != nil {
		http.Error(w, "Failed to load projects", http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "partials/projects_list.html", projects); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (api *API) terminalPageHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title string
	}{
		Title: "Terminal",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "terminal.html", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (api *API) terminalWSHandler(w http.ResponseWriter, r *http.Request) {
	// Check if we can upgrade to WebSocket
	if !websocket.IsWebSocketUpgrade(r) {
		log.Printf("Not a WebSocket upgrade request")
		http.Error(w, "Expected WebSocket request", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Start a shell process with proper environment
	cmd := exec.Command("/bin/bash", "-l")

	// Set up proper environment for Claude Code
	env := os.Environ()
	env = append(env,
		"TERM=xterm-256color",
		"SHELL=/bin/bash",
		"HOME=/home/appuser",
		"USER=appuser",
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"PS1=$ ",
	)
	cmd.Env = env
	cmd.Dir = "/home/appuser"

	// Start the command with a pty
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Failed to start terminal: "+err.Error()+"\r\n"))
		return
	}
	defer func() {
		ptmx.Close()
		cmd.Process.Kill()
	}()

	// Copy terminal output to websocket
	go func() {
		defer conn.Close()
		io.Copy(&wsWriter{conn: conn}, ptmx)
	}()

	// Copy websocket input to terminal
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		if _, err := ptmx.Write(message); err != nil {
			log.Printf("PTY write error: %v", err)
			break
		}
	}
}

// wsWriter implements io.Writer for WebSocket
type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

