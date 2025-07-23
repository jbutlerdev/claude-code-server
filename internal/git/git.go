package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// CloneRepository clones a git repository to the specified path
func (s *Service) CloneRepository(remoteURL, localPath, branch, token string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Remove existing directory if it exists
	if _, err := os.Stat(localPath); err == nil {
		if err := os.RemoveAll(localPath); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	// Use authenticated URL if token is provided
	cloneURL := remoteURL
	if token != "" {
		cloneURL = s.buildAuthenticatedURL(remoteURL, token)
	}

	// Clone the repository
	cmd := exec.Command("git", "clone", "--branch", branch, cloneURL, localPath)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0") // Disable interactive prompts

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s", string(output))
	}

	return nil
}

// ValidateRepository checks if a remote repository is accessible
func (s *Service) ValidateRepository(remoteURL, branch, token string) error {
	// Use authenticated URL if token is provided
	validateURL := remoteURL
	if token != "" {
		validateURL = s.buildAuthenticatedURL(remoteURL, token)
	}

	cmd := exec.Command("git", "ls-remote", "--heads", validateURL, branch)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("repository validation failed: %s", string(output))
	}

	// Check if branch exists in output
	if !strings.Contains(string(output), fmt.Sprintf("refs/heads/%s", branch)) {
		return fmt.Errorf("branch '%s' not found in repository", branch)
	}

	return nil
}

// RepositoryExists checks if a local repository exists and is valid
func (s *Service) RepositoryExists(localPath string) bool {
	gitDir := filepath.Join(localPath, ".git")
	if stat, err := os.Stat(gitDir); err != nil || !stat.IsDir() {
		return false
	}

	// Check if it's a valid git repository
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = localPath
	err := cmd.Run()
	return err == nil
}

// GetDefaultBranch attempts to determine the default branch of a remote repository
func (s *Service) GetDefaultBranch(remoteURL, token string) (string, error) {
	// Use authenticated URL if token is provided
	branchURL := remoteURL
	if token != "" {
		branchURL = s.buildAuthenticatedURL(remoteURL, token)
	}

	cmd := exec.Command("git", "ls-remote", "--symref", branchURL, "HEAD")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "main", nil // Default fallback
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ref: refs/heads/") {
			branch := strings.TrimPrefix(line, "ref: refs/heads/")
			branch = strings.Fields(branch)[0] // Remove any trailing content
			return branch, nil
		}
	}

	return "main", nil // Default fallback
}

// HasUncommittedChanges checks if there are uncommitted changes in the repository
func (s *Service) HasUncommittedChanges(localPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = localPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// CreateAndSwitchBranch creates a new branch or switches to existing one
func (s *Service) CreateAndSwitchBranch(localPath, branchName string) error {
	// Check if branch already exists
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	cmd.Dir = localPath
	err := cmd.Run()

	if err != nil {
		// Branch doesn't exist, create it
		cmd = exec.Command("git", "checkout", "-b", branchName)
	} else {
		// Branch exists, switch to it
		cmd = exec.Command("git", "checkout", branchName)
	}

	cmd.Dir = localPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create/switch branch: %s", string(output))
	}

	return nil
}

// SetGitIdentity configures git user name and email for the repository
func (s *Service) SetGitIdentity(localPath, userName, userEmail string) error {
	if userName != "" {
		cmd := exec.Command("git", "config", "user.name", userName)
		cmd.Dir = localPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set git user name: %s", string(output))
		}
	}

	if userEmail != "" {
		cmd := exec.Command("git", "config", "user.email", userEmail)
		cmd.Dir = localPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set git user email: %s", string(output))
		}
	}

	return nil
}

// CommitChanges adds all changes and commits them
func (s *Service) CommitChanges(localPath, message string) (string, error) {
	// Add all changes
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = localPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to add changes: %s", string(output))
	}

	// Commit changes
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = localPath
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to commit changes: %s", string(output))
	}

	// Get commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = localPath
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// PushToRemote pushes the current branch to remote with authentication
func (s *Service) PushToRemote(localPath, branchName, remoteURL, token string) error {
	// Configure git to use the token for authentication
	authenticatedURL := s.buildAuthenticatedURL(remoteURL, token)

	// Push to remote
	cmd := exec.Command("git", "push", "-u", authenticatedURL, branchName)
	cmd.Dir = localPath
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=echo", // Disable password prompts
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push to remote: %s", string(output))
	}

	return nil
}

// buildAuthenticatedURL builds a GitHub URL with token authentication
func (s *Service) buildAuthenticatedURL(remoteURL, token string) string {
	// Handle different GitHub URL formats
	if strings.HasPrefix(remoteURL, "https://github.com/") {
		// Replace https://github.com/ with https://TOKEN@github.com/
		return strings.Replace(remoteURL, "https://github.com/", fmt.Sprintf("https://%s@github.com/", token), 1)
	} else if strings.HasPrefix(remoteURL, "git@github.com:") {
		// Convert SSH to HTTPS with token
		repo := strings.TrimPrefix(remoteURL, "git@github.com:")
		return fmt.Sprintf("https://%s@github.com/%s", token, repo)
	}

	// Fallback - assume it's already a proper HTTPS URL
	return remoteURL
}

// GetCurrentBranch returns the current branch name
func (s *Service) GetCurrentBranch(localPath string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = localPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GenerateBranchName generates a unique branch name for a session
func (s *Service) GenerateBranchName(sessionID string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("claude-session-%s-%s", sessionID[:8], timestamp)
}
