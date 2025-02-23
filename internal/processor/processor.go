package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/zcubbs/sbomer/internal/db"
	"github.com/zcubbs/sbomer/internal/gitlab"
	"github.com/zcubbs/sbomer/internal/models"
	"github.com/zcubbs/sbomer/internal/syft"
)

type Message struct {
	ProjectID int `json:"project_id"`
}

type Processor struct {
	db     *db.DB
	gitlab *gitlab.Client
	syft   *syft.Generator
}

func New(database *db.DB, gitlabClient *gitlab.Client, syftGenerator *syft.Generator) *Processor {
	return &Processor{
		db:     database,
		gitlab: gitlabClient,
		syft:   syftGenerator,
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
	repoPath, details, err := p.gitlab.CloneProject(msg.ProjectID)
	if err != nil {
		err := p.db.LogOperation(ctx, msg.ProjectID, "clone", "failed", err.Error())
		if err != nil {
			return fmt.Errorf("failed to log clone failure: %w", err)
		}
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	defer func() {
		if err := p.gitlab.CleanupRepository(repoPath); err != nil {
			log.Printf("Failed to cleanup repository: %v", err)
		}
	}()

	// Log clone success
	if err := p.db.LogOperation(ctx, msg.ProjectID, "clone", "success", ""); err != nil {
		log.Printf("Failed to log clone success: %v", err)
	}

	// Create output path for SBOM
	sbomPath := filepath.Join(repoPath, "sbom.json")

	// Generate SBOM
	err = p.syft.GenerateSBOM(repoPath, sbomPath)
	if err != nil {
		err := p.db.LogOperation(ctx, msg.ProjectID, "sbom", "failed", err.Error())
		if err != nil {
			return fmt.Errorf("failed to log SBOM failure: %w", err)
		}
		return fmt.Errorf("failed to generate SBOM: %w", err)
	}

	// Read generated SBOM file
	sbomData, err := os.ReadFile(sbomPath)
	if err != nil {
		return fmt.Errorf("failed to read SBOM file: %w", err)
	}

	// Store SBOM in database
	sbom := &models.SBOM{
		ProjectUID: details.ID,
		Name:       details.Name,
		Path:       details.Path,
		Topics:     details.Topics,
		SBOMData:   json.RawMessage(sbomData),
	}

	if err := p.db.SaveSBOM(ctx, sbom); err != nil {
		return fmt.Errorf("failed to save SBOM: %w", err)
	}

	// Log SBOM generation success
	if err := p.db.LogOperation(ctx, msg.ProjectID, "sbom", "success", ""); err != nil {
		log.Printf("Failed to log SBOM success: %v", err)
	}

	return nil
}
