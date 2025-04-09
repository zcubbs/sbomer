package gitlab

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	gc "gitlab.com/gitlab-org/api/client-go"
)

type Client struct {
	token   string
	host    string
	scheme  string
	tempDir string
	client  *gc.Client
}

type ProjectDetails struct {
	ID           int
	Name         string
	Path         string
	Topics       []string
	ClonePath    string
	CommitBranch string
}

// New creates a new GitLab client
func New(token, host, scheme, tempDir string) (*Client, error) {
	// Create GitLab API client
	client, err := gc.NewClient(token,
		gc.WithBaseURL(fmt.Sprintf("%s://%s/api/v4", scheme, host)))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &Client{
		token:   token,
		host:    host,
		scheme:  scheme,
		tempDir: tempDir,
		client:  client,
	}, nil
}

// GetProjectDetails fetches project details from GitLab API
func (c *Client) GetProjectDetails(projectID int) (*ProjectDetails, error) {
	project, _, err := c.client.Projects.GetProject(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project details: %w", err)
	}

	return &ProjectDetails{
		ID:           project.ID,
		Name:         project.Name,
		Path:         project.PathWithNamespace,
		Topics:       project.Topics,
		ClonePath:    project.PathWithNamespace,
		CommitBranch: project.DefaultBranch,
	}, nil
}

// CloneProject clones the specified GitLab project into a temporary directory
func (c *Client) CloneProject(projectID int) (string, string, *ProjectDetails, error) {
	// Get project details
	details, err := c.GetProjectDetails(projectID)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get project details: %w", err)
	}

	// Create temp directory for the project
	localPath := filepath.Join(c.tempDir, fmt.Sprintf("project-%d", projectID))
	if err := os.MkdirAll(c.tempDir, 0755); err != nil {
		return "", "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clean existing directory if it exists
	if err := os.RemoveAll(localPath); err != nil {
		return "", "", nil, fmt.Errorf("failed to clean existing project directory: %w", err)
	}

	// Build correct clone URL using standard GitLab repository format
	cloneURL := fmt.Sprintf("%s://%s@%s/%s.git",
		c.scheme,
		fmt.Sprintf("oauth2:%s", c.token),
		c.host,
		details.ClonePath,
	)

	cloneUrlWithoutToken := fmt.Sprintf("%s://%s/%s.git",
		c.scheme,
		c.host,
		details.ClonePath,
	)

	// Set up git command
	fmt.Printf("Cloning repository %s from %s...\n", cloneUrlWithoutToken, c.host)
	cmd := exec.Command("git", "clone", "--depth", "1", cloneURL, localPath)
	cmd.Stderr = os.Stderr // Show git errors in console for debugging

	// Run git clone command
	if err := cmd.Run(); err != nil {
		return "", "", nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	return localPath, cloneUrlWithoutToken, details, nil
}

// getAPIBaseURL returns the base URL for API requests
func (c *Client) getAPIBaseURL() string {
	return fmt.Sprintf("%s://%s/api/v4", c.scheme, c.host)
}

// getBaseURL returns the base URL for API requests with the token
func (c *Client) getBaseURL() string {
	return fmt.Sprintf("%s://oauth2:%s@%s", c.scheme, c.token, c.host)
}

// getAPIBaseURLWithToken returns the base URL for API requests with the token
func (c *Client) getAPIBaseURLWithToken() string {
	return fmt.Sprintf("%s://%s@%s/api/v4", c.scheme, c.token, c.host)
}

// CleanupRepository cleans up the cloned project directory
func (c *Client) CleanupRepository(projectPath string) error {
	if err := os.RemoveAll(projectPath); err != nil {
		return fmt.Errorf("failed to cleanup repository: %w", err)
	}
	return nil
}
