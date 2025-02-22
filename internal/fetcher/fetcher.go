package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/zcubbs/sbomer/internal/db"
	"github.com/zcubbs/sbomer/internal/db/models"
	"gitlab.com/gitlab-org/api/client-go"
)

type Service struct {
	gitlabClient *gitlab.Client
	publisher    Publisher
	db           *db.DB
	schedule     string
	batchSize    int
	coolOffSecs  int
	groupIDs     []string
	cron         *cron.Cron
}

type Publisher interface {
	Publish(ctx context.Context, body []byte) error
}

type Config struct {
	GitLabToken string
	GitLabURL   string
	Schedule    string
	BatchSize   int
	CoolOffSecs int
	GroupIDs    []string
	Publisher   Publisher
	DB          *db.DB
}

func New(config Config) (*Service, error) {
	// Initialize GitLab client
	gitlabClient, err := gitlab.NewClient(config.GitLabToken, gitlab.WithBaseURL(config.GitLabURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &Service{
		gitlabClient: gitlabClient,
		publisher:    config.Publisher,
		db:           config.DB,
		schedule:     config.Schedule,
		batchSize:    config.BatchSize,
		coolOffSecs:  config.CoolOffSecs,
		groupIDs:     config.GroupIDs,
		cron:         cron.New(cron.WithSeconds()),
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	// Special case for "once" schedule
	if s.schedule == "once" {
		fmt.Printf("Running fetch and publish job once at %v\n", time.Now())
		if err := s.fetchAndPublish(ctx); err != nil {
			return fmt.Errorf("error in fetch and publish: %w", err)
		}
		return nil
	}

	// Regular cron schedule
	_, err := s.cron.AddFunc(s.schedule, func() {
		fmt.Printf("Running fetch and publish job at %v\n", time.Now())
		if err := s.fetchAndPublish(ctx); err != nil {
			log.Printf("Error in fetch and publish: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// Start the cron scheduler
	s.cron.Start()
	return nil
}

func (s *Service) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}

func (s *Service) fetchAndPublish(ctx context.Context) error {
	log.Printf("Starting fetch and publish cycle")
	startTime := time.Now()
	totalProjects := 0

	if len(s.groupIDs) > 0 {
		// Fetch projects from specified groups
		for _, groupID := range s.groupIDs {
			projectCount, err := s.fetchGroupProjects(ctx, groupID, startTime)
			if err != nil {
				log.Printf("Error fetching projects for group %s: %v", groupID, err)
				continue
			}
			totalProjects += projectCount
		}
	} else {
		// Fetch all projects
		projectCount, err := s.fetchAllProjects(ctx, startTime)
		if err != nil {
			return fmt.Errorf("error fetching all projects: %w", err)
		}
		totalProjects = projectCount
	}

	log.Printf("Completed fetch and publish cycle. Processed %d projects in %.2f seconds", totalProjects, time.Since(startTime).Seconds())
	return nil
}

func (s *Service) fetchGroupProjects(ctx context.Context, groupID string, startTime time.Time) (int, error) {
	totalProjects := 0
	page := 1

	for {
		// List projects in the group with pagination
		opt := &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: s.batchSize,
			},
			IncludeSubGroups: gitlab.Bool(true), // Include projects from subgroups
		}

		projects, resp, err := s.gitlabClient.Groups.ListGroupProjects(groupID, opt)
		if err != nil {
			return totalProjects, fmt.Errorf("failed to list group projects: %w", err)
		}

		batchCount := len(projects)
		totalProjects += batchCount

		// Process each project in the batch
		for _, project := range projects {
			if err := s.publishProject(ctx, project.ID); err != nil {
				log.Printf("Error publishing project %d: %v", project.ID, err)
				continue
			}
		}

		// Save fetch statistics for this batch
		stats := &models.FetchStats{
			ProjectsCount: totalProjects,
			BatchSize:     s.batchSize,
			Duration:      time.Since(startTime).Seconds(),
			CreatedAt:     time.Now(),
		}
		if err := s.db.SaveFetchStats(ctx, stats); err != nil {
			log.Printf("Error saving fetch stats: %v", err)
		}

		// Check if we've processed all pages
		if resp.NextPage == 0 {
			break
		}

		// Move to next page
		page = resp.NextPage

		// Cool off between batches
		select {
		case <-ctx.Done():
			return totalProjects, ctx.Err()
		case <-time.After(time.Duration(s.coolOffSecs) * time.Second):
		}
	}

	return totalProjects, nil
}

func (s *Service) fetchAllProjects(ctx context.Context, startTime time.Time) (int, error) {
	totalProjects := 0
	page := 1

	for {
		// List all projects with pagination
		opt := &gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: s.batchSize,
			},
		}

		projects, resp, err := s.gitlabClient.Projects.ListProjects(opt)
		if err != nil {
			return totalProjects, fmt.Errorf("failed to list projects: %w", err)
		}

		batchCount := len(projects)
		totalProjects += batchCount

		// Process each project in the batch
		for _, project := range projects {
			if err := s.publishProject(ctx, project.ID); err != nil {
				log.Printf("Error publishing project %d: %v", project.ID, err)
				continue
			}
		}

		// Save fetch statistics for this batch
		stats := &models.FetchStats{
			ProjectsCount: totalProjects,
			BatchSize:     s.batchSize,
			Duration:      time.Since(startTime).Seconds(),
			CreatedAt:     time.Now(),
		}
		if err := s.db.SaveFetchStats(ctx, stats); err != nil {
			log.Printf("Error saving fetch stats: %v", err)
		}

		// Check if we've processed all pages
		if resp.NextPage == 0 {
			break
		}

		// Move to next page
		page = resp.NextPage

		// Cool off between batches
		select {
		case <-ctx.Done():
			return totalProjects, ctx.Err()
		case <-time.After(time.Duration(s.coolOffSecs) * time.Second):
		}
	}

	return totalProjects, nil
}

func (s *Service) publishProject(ctx context.Context, projectID int) error {
	message := struct {
		ProjectID int `json:"project_id"`
	}{
		ProjectID: projectID,
	}

	// Marshal message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	// Publish message
	if err := s.publisher.Publish(ctx, messageBytes); err != nil {
		return fmt.Errorf("error publishing message: %w", err)
	}

	return nil
}
