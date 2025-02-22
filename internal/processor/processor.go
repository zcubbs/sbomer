package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zcubbs/sbomer/internal/db"
	"github.com/zcubbs/sbomer/internal/gitlab"
	"github.com/zcubbs/sbomer/internal/syft"
	"log"
	"os"
	"path/filepath"
)

type Message struct {
	ProjectID int `json:"project_id"`
}

type Processor struct {
	db       *db.DB
	gitlab   *gitlab.Client
	syft     *syft.Generator
	sbomPath string
}

func New(database *db.DB, gitlabClient *gitlab.Client, sbomGenerator *syft.Generator, sbomOutputPath string) *Processor {
	return &Processor{
		db:       database,
		gitlab:   gitlabClient,
		syft:     sbomGenerator,
		sbomPath: sbomOutputPath,
	}
}

func (p *Processor) ProcessMessage(ctx context.Context, data []byte) error {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Log operation start
	if err := p.db.LogOperation(ctx, msg.ProjectID, "clone", "started", ""); err != nil {
		log.Printf("Failed to log operation start: %v", err)
	}

	// Clone repository
	repoPath, err := p.gitlab.CloneProject(msg.ProjectID)
	if err != nil {
		p.db.LogOperation(ctx, msg.ProjectID, "clone", "failed", err.Error())
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	defer p.gitlab.CleanupRepository(repoPath)

	// Log clone success
	if err := p.db.LogOperation(ctx, msg.ProjectID, "clone", "success", ""); err != nil {
		log.Printf("Failed to log clone success: %v", err)
	}

	// make sure the output directory exists
	if err := os.MkdirAll(p.sbomPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate SBOM
	outputPath := filepath.Join(p.sbomPath, fmt.Sprintf("sbom-%d.json", msg.ProjectID))
	if err := p.syft.GenerateSBOM(repoPath, outputPath); err != nil {
		p.db.LogOperation(ctx, msg.ProjectID, "sbom", "failed", err.Error())
		return fmt.Errorf("failed to generate SBOM: %w", err)
	}

	// Log SBOM generation success
	if err := p.db.LogOperation(ctx, msg.ProjectID, "sbom", "success", ""); err != nil {
		log.Printf("Failed to log SBOM success: %v", err)
	}

	return nil
}
