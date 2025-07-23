package project

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jbutlerdev/claude-code-server/internal/database"
	"github.com/jbutlerdev/claude-code-server/internal/git"
	"github.com/jbutlerdev/claude-code-server/internal/models"
	"github.com/jbutlerdev/claude-code-server/internal/settings"
)

type Service struct {
	repo            *database.Repository
	gitService      *git.Service
	settingsService *settings.Service
}

func NewService(repo *database.Repository, settingsService *settings.Service) *Service {
	return &Service{
		repo:            repo,
		gitService:      git.NewService(),
		settingsService: settingsService,
	}
}

// CreateProject creates a new project and attempts to clone its repository
func (s *Service) CreateProject(ctx context.Context, req *models.CreateProjectRequest) (*models.Project, error) {
	// Create the project in the database
	project, err := s.repo.CreateProject(ctx, req)
	if err != nil {
		return nil, err
	}

	// If there's a remote URL, attempt to clone it
	if project.RemoteURL != nil && *project.RemoteURL != "" {
		var token string
		if project.GitHubToken != nil && *project.GitHubToken != "" {
			token = *project.GitHubToken
			log.Printf("CreateProject: Using project-specific GitHub token for project %s", project.Name)
		} else {
			token = s.settingsService.GetDefaultGitHubToken()
			log.Printf("CreateProject: Using default GitHub token for project %s, token length: %d, token provided: %v", project.Name, len(token), token != "")
		}
		go s.cloneProjectAsync(project.ID, *project.RemoteURL, project.Path, project.DefaultBranch, token)
	} else {
		// For local projects, mark as ready
		s.repo.UpdateProjectStatus(ctx, project.ID, "ready", nil)
	}

	return project, nil
}

// cloneProjectAsync clones a project repository in the background
func (s *Service) cloneProjectAsync(projectID uuid.UUID, remoteURL, localPath, branch, token string) {
	ctx := context.Background()

	// Debug logging
	log.Printf("cloneProjectAsync called for project %s, remote URL: %s, branch: %s, token provided: %v", 
		projectID, remoteURL, branch, token != "")

	// First validate the repository
	if err := s.gitService.ValidateRepository(remoteURL, branch, token); err != nil {
		log.Printf("Failed to validate repository %s: %v", remoteURL, err)
		errorMsg := err.Error()
		s.repo.UpdateProjectStatus(ctx, projectID, "error", &errorMsg)
		return
	}

	// Clone the repository
	if err := s.gitService.CloneRepository(remoteURL, localPath, branch, token); err != nil {
		log.Printf("Failed to clone repository %s: %v", remoteURL, err)
		errorMsg := err.Error()
		s.repo.UpdateProjectStatus(ctx, projectID, "error", &errorMsg)
		return
	}

	// Mark as ready
	log.Printf("Successfully cloned repository %s to %s", remoteURL, localPath)
	s.repo.UpdateProjectStatus(ctx, projectID, "ready", nil)
}

// SyncProject ensures a project's repository is up to date
func (s *Service) SyncProject(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	if project.RemoteURL == nil || *project.RemoteURL == "" {
		// Local project, just mark as ready
		return s.repo.UpdateProjectStatus(ctx, projectID, "ready", nil)
	}

	// Check if repository exists locally
	if !s.gitService.RepositoryExists(project.Path) {
		// Repository doesn't exist, clone it
		var token string
		if project.GitHubToken != nil && *project.GitHubToken != "" {
			token = *project.GitHubToken
			log.Printf("Using project-specific GitHub token for project %s", project.Name)
		} else {
			token = s.settingsService.GetDefaultGitHubToken()
			log.Printf("Using default GitHub token for project %s, token length: %d, token provided: %v", project.Name, len(token), token != "")
		}
		s.repo.UpdateProjectStatus(ctx, projectID, "syncing", nil)
		go s.cloneProjectAsync(projectID, *project.RemoteURL, project.Path, project.DefaultBranch, token)
		return nil
	}

	// Repository exists, mark as ready
	return s.repo.UpdateProjectStatus(ctx, projectID, "ready", nil)
}

// SyncAllProjects ensures all projects have their repositories available
func (s *Service) SyncAllProjects(ctx context.Context) error {
	projects, err := s.repo.ListProjects(ctx)
	if err != nil {
		return err
	}

	for _, project := range projects {
		if project.Status == "ready" && s.gitService.RepositoryExists(project.Path) {
			continue // Already ready and exists
		}

		log.Printf("Syncing project %s (%s)", project.Name, project.ID)
		if err := s.SyncProject(ctx, project.ID); err != nil {
			log.Printf("Failed to sync project %s: %v", project.Name, err)
		}
	}

	return nil
}
