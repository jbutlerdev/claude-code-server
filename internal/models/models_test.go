package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSessionStatus(t *testing.T) {
	tests := []struct {
		status SessionStatus
		valid  bool
	}{
		{SessionStatusStarting, true},
		{SessionStatusRunning, true},
		{SessionStatusCompleted, true},
		{SessionStatusFailed, true},
		{SessionStatus("invalid"), false},
	}

	validStatuses := map[SessionStatus]bool{
		SessionStatusStarting:  true,
		SessionStatusRunning:   true,
		SessionStatusCompleted: true,
		SessionStatusFailed:    true,
	}

	for _, test := range tests {
		_, exists := validStatuses[test.status]
		if exists != test.valid {
			t.Errorf("Status %s validity mismatch: expected %v, got %v", test.status, test.valid, exists)
		}
	}
}

func TestJSONBValue(t *testing.T) {
	metadata := JSONB{
		"key1": "value1",
		"key2": 123,
		"key3": []string{"a", "b", "c"},
	}

	value, err := metadata.Value()
	if err != nil {
		t.Fatalf("Failed to get JSONB value: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(value.([]byte), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSONB value: %v", err)
	}

	if result["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", result["key1"])
	}
}

func TestJSONBScan(t *testing.T) {
	jsonData := []byte(`{"key1":"value1","key2":123}`)
	var metadata JSONB

	if err := metadata.Scan(jsonData); err != nil {
		t.Fatalf("Failed to scan JSONB: %v", err)
	}

	if metadata["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", metadata["key1"])
	}

	if metadata["key2"].(float64) != 123 {
		t.Errorf("Expected key2=123, got %v", metadata["key2"])
	}
}

func TestCreateSessionRequestValidation(t *testing.T) {
	req := CreateSessionRequest{
		Prompt:    "Test prompt",
		ProjectID: uuid.New().String(),
		MaxTurns:  10,
	}

	if req.Prompt == "" {
		t.Error("Prompt should not be empty")
	}

	if req.ProjectID == "" {
		t.Error("ProjectID should not be empty")
	}

	if _, err := uuid.Parse(req.ProjectID); err != nil {
		t.Errorf("ProjectID should be valid UUID: %v", err)
	}
}

func TestSessionModel(t *testing.T) {
	session := Session{
		ID:            uuid.New(),
		ProjectID:     uuid.New(),
		InitialPrompt: "Test prompt",
		Model:         "claude-3-5-sonnet-20241022",
		MaxTurns:      25,
		Status:        SessionStatusStarting,
		WorktreeID:    "test-worktree",
		WorktreePath:  "/tmp/test",
		StartedAt:     time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if session.ID == uuid.Nil {
		t.Error("Session ID should not be nil")
	}

	if session.Status != SessionStatusStarting {
		t.Errorf("Expected status %s, got %s", SessionStatusStarting, session.Status)
	}

	if session.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model claude-3-5-sonnet-20241022, got %s", session.Model)
	}
}

func TestProjectModel(t *testing.T) {
	remoteURL := "https://github.com/test/repo.git"
	project := Project{
		ID:            uuid.New(),
		Name:          "Test Project",
		Path:          "/path/to/project",
		RemoteURL:     &remoteURL,
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if project.Name == "" {
		t.Error("Project name should not be empty")
	}

	if project.Path == "" {
		t.Error("Project path should not be empty")
	}

	if project.RemoteURL == nil || *project.RemoteURL != remoteURL {
		t.Error("Project remote URL not set correctly")
	}

	if project.DefaultBranch != "main" {
		t.Errorf("Expected default branch 'main', got %s", project.DefaultBranch)
	}
}

func TestListResponse(t *testing.T) {
	sessions := []Session{
		{ID: uuid.New(), Status: SessionStatusRunning},
		{ID: uuid.New(), Status: SessionStatusCompleted},
	}

	response := ListResponse[Session]{
		Items:      sessions,
		Page:       1,
		PageSize:   10,
		TotalItems: 2,
		TotalPages: 1,
	}

	if len(response.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(response.Items))
	}

	if response.TotalPages != 1 {
		t.Errorf("Expected 1 total page, got %d", response.TotalPages)
	}
}
