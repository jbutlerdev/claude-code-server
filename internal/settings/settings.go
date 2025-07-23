package settings

import (
	"context"
	"sync"

	"github.com/jbutlerdev/claude-code-server/internal/config"
	"github.com/jbutlerdev/claude-code-server/internal/models"
)

type Repository interface {
	GetSettings(ctx context.Context) (*models.Settings, error)
	UpdateSettings(ctx context.Context, req *models.UpdateSettingsRequest) (*models.Settings, error)
}

type Service struct {
	config     *config.Config
	repo       Repository
	mutex      sync.RWMutex
}

func NewService(cfg *config.Config, repo Repository) *Service {
	return &Service{
		config: cfg,
		repo:   repo,
	}
}

func (s *Service) GetSettings() *models.Settings {
	ctx := context.Background()
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		// Return defaults with config values if database fails
		return &models.Settings{
			ClaudeCodeBinary:      s.config.ClaudeCodeBinary,
			MaxConcurrentSessions: s.config.MaxConcurrentSessions,
		}
	}
	
	// Merge with config defaults if database values are empty
	if settings.ClaudeCodeBinary == "" {
		settings.ClaudeCodeBinary = s.config.ClaudeCodeBinary
	}
	if settings.MaxConcurrentSessions == 0 {
		settings.MaxConcurrentSessions = s.config.MaxConcurrentSessions
	}
	if settings.DefaultGitHubToken == nil && s.config.DefaultGitHubToken != "" {
		settings.DefaultGitHubToken = &s.config.DefaultGitHubToken
	}
	if settings.ClaudeCodeAPIKey == nil && s.config.ClaudeCodeAPIKey != "" {
		settings.ClaudeCodeAPIKey = &s.config.ClaudeCodeAPIKey
	}
	
	return settings
}

func (s *Service) UpdateSettings(req *models.UpdateSettingsRequest) error {
	ctx := context.Background()
	_, err := s.repo.UpdateSettings(ctx, req)
	return err
}

// GetDefaultGitHubToken returns the current default GitHub token
func (s *Service) GetDefaultGitHubToken() string {
	settings := s.GetSettings()
	if settings.DefaultGitHubToken != nil {
		return *settings.DefaultGitHubToken
	}
	return ""
}

// GetGitIdentity returns the current git user name and email
func (s *Service) GetGitIdentity() (string, string) {
	settings := s.GetSettings()
	userName := ""
	userEmail := ""
	
	if settings.GitUserName != nil {
		userName = *settings.GitUserName
	}
	if settings.GitUserEmail != nil {
		userEmail = *settings.GitUserEmail
	}
	
	return userName, userEmail
}


// GetClaudeAPIKey returns the current Claude API key
func (s *Service) GetClaudeAPIKey() string {
	settings := s.GetSettings()
	if settings.ClaudeCodeAPIKey != nil {
		return *settings.ClaudeCodeAPIKey
	}
	return ""
}

// GetDefaultMaxTurns returns the current default max turns setting
func (s *Service) GetDefaultMaxTurns() int {
	settings := s.GetSettings()
	if settings.DefaultMaxTurns != nil {
		return *settings.DefaultMaxTurns
	}
	return 25 // Default fallback value
}