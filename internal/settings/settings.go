package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jbutlerdev/claude-code-server/internal/config"
	"github.com/jbutlerdev/claude-code-server/internal/models"
)

type Service struct {
	config     *config.Config
	filePath   string
	settings   *models.Settings
	mutex      sync.RWMutex
}

func NewService(cfg *config.Config) *Service {
	settingsPath := "/var/lib/claude-code-server/settings.json"
	
	service := &Service{
		config:   cfg,
		filePath: settingsPath,
		settings: &models.Settings{
			DefaultGitHubToken:    cfg.DefaultGitHubToken,
			ClaudeCodeBinary:      cfg.ClaudeCodeBinary,
			MaxConcurrentSessions: cfg.MaxConcurrentSessions,
			GitUserName:           "Claude Code Server",
			GitUserEmail:          "claude@example.com",
		},
	}
	
	// Try to load existing settings
	service.load()
	
	return service
}

func (s *Service) GetSettings() *models.Settings {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Return a copy to prevent modification
	return &models.Settings{
		DefaultGitHubToken:    s.settings.DefaultGitHubToken,
		ClaudeCodeBinary:      s.settings.ClaudeCodeBinary,
		MaxConcurrentSessions: s.settings.MaxConcurrentSessions,
		GitUserName:           s.settings.GitUserName,
		GitUserEmail:          s.settings.GitUserEmail,
	}
}

func (s *Service) UpdateSettings(req *models.UpdateSettingsRequest) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Update fields if provided
	if req.DefaultGitHubToken != nil {
		s.settings.DefaultGitHubToken = *req.DefaultGitHubToken
	}
	if req.ClaudeCodeBinary != nil {
		s.settings.ClaudeCodeBinary = *req.ClaudeCodeBinary
	}
	if req.MaxConcurrentSessions != nil {
		s.settings.MaxConcurrentSessions = *req.MaxConcurrentSessions
	}
	if req.GitUserName != nil {
		s.settings.GitUserName = *req.GitUserName
	}
	if req.GitUserEmail != nil {
		s.settings.GitUserEmail = *req.GitUserEmail
	}
	
	// Save to file
	return s.save()
}

func (s *Service) load() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// File doesn't exist, use defaults from config
		return nil
	}
	
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read settings file: %w", err)
	}
	
	var fileSettings models.Settings
	if err := json.Unmarshal(data, &fileSettings); err != nil {
		return fmt.Errorf("failed to parse settings file: %w", err)
	}
	
	// Merge with config defaults (file takes precedence)
	if fileSettings.DefaultGitHubToken != "" {
		s.settings.DefaultGitHubToken = fileSettings.DefaultGitHubToken
	}
	if fileSettings.ClaudeCodeBinary != "" {
		s.settings.ClaudeCodeBinary = fileSettings.ClaudeCodeBinary
	}
	if fileSettings.MaxConcurrentSessions > 0 {
		s.settings.MaxConcurrentSessions = fileSettings.MaxConcurrentSessions
	}
	if fileSettings.GitUserName != "" {
		s.settings.GitUserName = fileSettings.GitUserName
	}
	if fileSettings.GitUserEmail != "" {
		s.settings.GitUserEmail = fileSettings.GitUserEmail
	}
	
	return nil
}

func (s *Service) save() error {
	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}
	
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	
	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}
	
	return nil
}

// GetDefaultGitHubToken returns the current default GitHub token
func (s *Service) GetDefaultGitHubToken() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.settings.DefaultGitHubToken
}

// GetGitIdentity returns the current git user name and email
func (s *Service) GetGitIdentity() (string, string) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.settings.GitUserName, s.settings.GitUserEmail
}