package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jbutlerdev/claude-code-server/internal/config"
	"github.com/jbutlerdev/claude-code-server/internal/database"
	"github.com/jbutlerdev/claude-code-server/internal/models"
	"github.com/jbutlerdev/claude-code-server/internal/session"
)

type mockRepository struct {
	projects []*models.Project
	sessions []*models.Session
}

func (m *mockRepository) CreateProject(ctx context.Context, req *models.CreateProjectRequest) (*models.Project, error) {
	project := &models.Project{
		ID:            uuid.New(),
		Name:          req.Name,
		Path:          req.Path,
		RemoteURL:     req.RemoteURL,
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	m.projects = append(m.projects, project)
	return project, nil
}

func (m *mockRepository) GetProject(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	for _, p := range m.projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockRepository) ListProjects(ctx context.Context) ([]*models.Project, error) {
	return m.projects, nil
}

func (m *mockRepository) UpdateProject(ctx context.Context, id uuid.UUID, req *models.UpdateProjectRequest) (*models.Project, error) {
	for _, p := range m.projects {
		if p.ID == id {
			if req.Name != nil {
				p.Name = *req.Name
			}
			if req.Path != nil {
				p.Path = *req.Path
			}
			p.UpdatedAt = time.Now()
			return p, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockRepository) DeleteProject(ctx context.Context, id uuid.UUID) error {
	for i, p := range m.projects {
		if p.ID == id {
			m.projects = append(m.projects[:i], m.projects[i+1:]...)
			return nil
		}
	}
	return sql.ErrNoRows
}

func (m *mockRepository) GetSession(ctx context.Context, id uuid.UUID) (*models.Session, error) {
	for _, s := range m.sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockRepository) ListSessions(ctx context.Context, query *models.ListSessionsQuery) (*models.ListResponse[models.Session], error) {
	items := make([]models.Session, 0)
	for _, s := range m.sessions {
		if query.Status != "" && string(s.Status) != query.Status {
			continue
		}
		items = append(items, *s)
	}

	totalItems := len(items)
	totalPages := (totalItems + query.PageSize - 1) / query.PageSize

	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if end > len(items) {
		end = len(items)
	}
	if start > len(items) {
		start = len(items)
	}

	return &models.ListResponse[models.Session]{
		Items:      items[start:end],
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

func (m *mockRepository) GetSessionLogs(ctx context.Context, sessionID uuid.UUID, limit int) ([]*models.SessionLog, error) {
	return []*models.SessionLog{}, nil
}

func setupTestAPI() *API {
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}

	return &API{
		config:         cfg,
		repo:           &mockRepository{},
		sessionManager: nil, // Mock if needed
	}
}

func TestHealthHandler(t *testing.T) {
	api := setupTestAPI()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	api.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestCreateProjectHandler(t *testing.T) {
	api := setupTestAPI()

	reqBody := models.CreateProjectRequest{
		Name: "Test Project",
		Path: "/path/to/project",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.createProjectHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var project models.Project
	if err := json.NewDecoder(w.Body).Decode(&project); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if project.Name != "Test Project" {
		t.Errorf("Expected project name 'Test Project', got '%s'", project.Name)
	}
}

func TestListProjectsHandler(t *testing.T) {
	api := setupTestAPI()

	// Add a test project
	api.repo.(*mockRepository).projects = []*models.Project{
		{
			ID:   uuid.New(),
			Name: "Test Project",
			Path: "/path/to/project",
		},
	}

	req := httptest.NewRequest("GET", "/projects", nil)
	w := httptest.NewRecorder()

	api.listProjectsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var projects []*models.Project
	if err := json.NewDecoder(w.Body).Decode(&projects); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(projects))
	}

	if projects[0].Name != "Test Project" {
		t.Errorf("Expected project name 'Test Project', got '%s'", projects[0].Name)
	}
}

func TestGetProjectHandler(t *testing.T) {
	api := setupTestAPI()

	projectID := uuid.New()
	api.repo.(*mockRepository).projects = []*models.Project{
		{
			ID:   projectID,
			Name: "Test Project",
			Path: "/path/to/project",
		},
	}

	req := httptest.NewRequest("GET", "/projects/"+projectID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": projectID.String()})
	w := httptest.NewRecorder()

	api.getProjectHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var project models.Project
	if err := json.NewDecoder(w.Body).Decode(&project); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if project.ID != projectID {
		t.Errorf("Expected project ID %s, got %s", projectID, project.ID)
	}
}

func TestGetProjectHandlerNotFound(t *testing.T) {
	api := setupTestAPI()

	projectID := uuid.New()
	req := httptest.NewRequest("GET", "/projects/"+projectID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": projectID.String()})
	w := httptest.NewRecorder()

	api.getProjectHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestListSessionsHandler(t *testing.T) {
	api := setupTestAPI()

	sessionID := uuid.New()
	projectID := uuid.New()
	api.repo.(*mockRepository).sessions = []*models.Session{
		{
			ID:        sessionID,
			ProjectID: projectID,
			Status:    models.SessionStatusRunning,
		},
	}

	req := httptest.NewRequest("GET", "/sessions?page=1&page_size=10", nil)
	w := httptest.NewRecorder()

	api.listSessionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.ListResponse[models.Session]
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Items) != 1 {
		t.Errorf("Expected 1 session, got %d", len(response.Items))
	}

	if response.Items[0].ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, response.Items[0].ID)
	}
}

func TestWriteError(t *testing.T) {
	api := setupTestAPI()
	w := httptest.NewRecorder()

	api.writeError(w, http.StatusBadRequest, "Test error", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error != "Test error" {
		t.Errorf("Expected error 'Test error', got '%s'", response.Error)
	}
}
