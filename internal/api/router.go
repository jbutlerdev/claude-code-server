package api

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jbutlerdev/claude-code-server/internal/config"
	"github.com/jbutlerdev/claude-code-server/internal/database"
	"github.com/jbutlerdev/claude-code-server/internal/git"
	"github.com/jbutlerdev/claude-code-server/internal/project"
	"github.com/jbutlerdev/claude-code-server/internal/session"
	"github.com/jbutlerdev/claude-code-server/internal/settings"
)

func NewRouter(cfg *config.Config, db *sql.DB, sessionManager *session.Manager) http.Handler {
	r := mux.NewRouter()
	repo := database.NewRepository(db)
	projectService := project.NewService(repo)
	gitService := git.NewService()
	settingsService := settings.NewService(cfg)

	api := &API{
		config:          cfg,
		db:              db,
		repo:            repo,
		sessionManager:  sessionManager,
		projectService:  projectService,
		gitService:      gitService,
		settingsService: settingsService,
	}

	r.Use(corsMiddleware)
	r.Use(loggingMiddleware)

	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(api.authMiddleware)
	apiRouter.Use(rateLimitMiddleware(cfg))

	apiRouter.HandleFunc("/health", api.healthHandler).Methods("GET")
	apiRouter.HandleFunc("/ready", api.readyHandler).Methods("GET")
	apiRouter.HandleFunc("/openapi.json", api.openAPIHandler).Methods("GET")

	sessionsRouter := apiRouter.PathPrefix("/sessions").Subrouter()
	sessionsRouter.HandleFunc("", api.createSessionHandler).Methods("POST")
	sessionsRouter.HandleFunc("", api.listSessionsHandler).Methods("GET")
	sessionsRouter.HandleFunc("/{id}", api.getSessionHandler).Methods("GET")
	sessionsRouter.HandleFunc("/{id}", api.deleteSessionHandler).Methods("DELETE")
	sessionsRouter.HandleFunc("/{id}/resume", api.resumeSessionHandler).Methods("POST")
	sessionsRouter.HandleFunc("/{id}/logs", api.getSessionLogsHandler).Methods("GET")
	sessionsRouter.HandleFunc("/{id}/push", api.pushSessionToRemoteHandler).Methods("POST")

	projectsRouter := apiRouter.PathPrefix("/projects").Subrouter()
	projectsRouter.HandleFunc("", api.createProjectHandler).Methods("POST")
	projectsRouter.HandleFunc("", api.listProjectsHandler).Methods("GET")
	projectsRouter.HandleFunc("/{id}", api.getProjectHandler).Methods("GET")
	projectsRouter.HandleFunc("/{id}", api.updateProjectHandler).Methods("PUT")
	projectsRouter.HandleFunc("/{id}", api.deleteProjectHandler).Methods("DELETE")

	// Settings routes
	apiRouter.HandleFunc("/settings", api.getSettingsHandler).Methods("GET")
	apiRouter.HandleFunc("/settings", api.updateSettingsHandler).Methods("PUT")

	// WebSocket route without middleware (has to be first to avoid conflicts)
	r.HandleFunc("/ws/terminal", api.terminalWSHandler)

	webRouter := r.PathPrefix("").Subrouter()
	webRouter.HandleFunc("/", api.dashboardHandler).Methods("GET")
	webRouter.HandleFunc("/sessions", api.sessionsPageHandler).Methods("GET")
	webRouter.HandleFunc("/sessions/{id}", api.sessionDetailHandler).Methods("GET")
	webRouter.HandleFunc("/projects", api.projectsPageHandler).Methods("GET")
	webRouter.HandleFunc("/projects/new", api.newProjectPageHandler).Methods("GET")
	webRouter.HandleFunc("/settings", api.settingsPageHandler).Methods("GET")
	webRouter.HandleFunc("/api-docs", api.apiDocsHandler).Methods("GET")
	webRouter.HandleFunc("/terminal", api.terminalPageHandler).Methods("GET")

	htmxRouter := r.PathPrefix("/htmx").Subrouter()
	htmxRouter.HandleFunc("/sessions", api.htmxSessionsHandler).Methods("GET")
	htmxRouter.HandleFunc("/sessions/{id}/logs", api.htmxSessionLogsHandler).Methods("GET")
	htmxRouter.HandleFunc("/sessions/{id}/resume", api.htmxResumeSessionHandler).Methods("POST")
	htmxRouter.HandleFunc("/projects", api.htmxProjectsHandler).Methods("GET")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	return r
}
