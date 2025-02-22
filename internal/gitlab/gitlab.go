package gitlab

import (
	"fmt"
	gc "gitlab.com/gitlab-org/api/client-go"
	"os"
	"os/exec"
	"path/filepath"
)

type Client struct {
	token   string
	host    string
	scheme  string
	tempDir string
}

// New creates a new GitLab client and ensures the baseURL is trimmed of trailing slashes and has a proper scheme
func New(token, host, scheme, tempDir string) *Client {
	return &Client{
		token:   token,
		host:    host,
		scheme:  scheme,
		tempDir: tempDir,
	}
}

// getAPIBaseURL returns the base URL for API requests
func (c *Client) getAPIBaseURL() string {
	return fmt.Sprintf("%s://%s/api/v4", c.scheme, c.host)
}

func (c *Client) getBaseURL() string {
	return fmt.Sprintf("%s://oauth2:%s@%s", c.scheme, c.token, c.host)
}

// getProjectPath returns the project path for the specified project ID
func (c *Client) getProjectPath(projectID int) (string, error) {
	// Create GitLab client
	client, err := gc.NewClient(c.token, gc.WithBaseURL(c.getAPIBaseURL()))
	if err != nil {
		return "", fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Get project details
	project, _, err := client.Projects.GetProject(projectID, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get project details: %w", err)
	}

	return project.PathWithNamespace, nil
}

// getAPIBaseURLWithToken returns the base URL for API requests with the token
func (c *Client) getAPIBaseURLWithToken() string {
	return fmt.Sprintf("%s://%s@%s/api/v4", c.scheme, c.token, c.host)
}

// CloneProject clones the specified GitLab project into a temporary directory
func (c *Client) CloneProject(projectID int) (string, error) {
	// Create temp directory for the project
	localPath := filepath.Join(c.tempDir, fmt.Sprintf("project-%d", projectID))
	if err := os.MkdirAll(c.tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clean existing directory if it exists
	if err := os.RemoveAll(localPath); err != nil {
		return "", fmt.Errorf("failed to clean existing project directory: %w", err)
	}

	// Create GitLab client
	projectPath, err := c.getProjectPath(projectID)
	if err != nil {
		return "", fmt.Errorf("failed to get project path: %w", err)
	}

	// Build correct clone URL using standard GitLab repository format
	cloneURL := fmt.Sprintf("%s://%s@%s/%s.git",
		c.scheme,
		fmt.Sprintf("oauth2:%s", c.token),
		c.host,
		projectPath,
	)

	cloneUrlWithoutToken := fmt.Sprintf("%s://%s/%s.git",
		c.scheme,
		c.host,
		projectPath,
	)
	// Set up git command
	fmt.Printf("Cloning repository %s from %s...\n", cloneUrlWithoutToken, c.host) // Log cloning operation
	cmd := exec.Command("git", "clone", "--depth", "1", cloneURL, localPath)
	cmd.Stderr = os.Stderr // Show git errors in console for debugging

	// Run git clone command
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return localPath, nil
}

// CleanupRepository cleans up the cloned project directory
func (c *Client) CleanupRepository(projectPath string) error {
	if err := os.RemoveAll(projectPath); err != nil {
		return fmt.Errorf("failed to cleanup repository: %w", err)
	}
	return nil
}
