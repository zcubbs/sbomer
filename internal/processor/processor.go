package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CycloneDX/cyclonedx-go"
	"github.com/zcubbs/sbomer/internal/db"
	"github.com/zcubbs/sbomer/internal/gitlab"
	"github.com/zcubbs/sbomer/internal/models"
	"github.com/zcubbs/sbomer/internal/syft"

	"github.com/zcubbs/sbomer/internal/rabbitmq"
)

type Message struct {
	ProjectID int `json:"project_id"`
}

type Processor struct {
	db     *db.DB
	gitlab *gitlab.Client
	syft   *syft.Generator
}

type SbomScanRequestEvent struct {
	Metadata Metadata       `json:"metadata"`
	SBOM     *cyclonedx.BOM `json:"sbom"`
}

type Metadata struct {
	ProjectId     string   `json:"projectId"`
	ProjectTitle  string   `json:"projectTitle"`
	ProjectUrl    string   `json:"projectUrl"`
	JobId         string   `json:"jobId"`
	CommitBranch  string   `json:"commitBranch"`
	Source        string   `json:"source"`
	GeneratedDate string   `json:"generatedDate"`
	SbomFormat    string   `json:"sbomFormat"`
	Version       string   `json:"version"`
	TopicsId      []string `json:"topicsId"`
}

func New(database *db.DB, gitlabClient *gitlab.Client, syftGenerator *syft.Generator) *Processor {
	return &Processor{
		db:     database,
		gitlab: gitlabClient,
		syft:   syftGenerator,
	}
}

func parseSBOM(sbomData []byte) (*cyclonedx.BOM, error) {
	decoder := cyclonedx.NewBOMDecoder(bytes.NewReader(sbomData), cyclonedx.BOMFileFormatJSON)

	var bom cyclonedx.BOM

	if err := decoder.Decode(&bom); err != nil {
		return nil, err
	}

	return &bom, nil
}

func (p *Processor) ProcessMessage(ctx context.Context, data []byte, workerScannerConsumer *rabbitmq.Consumer) error {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Log operation start
	if err := p.db.LogOperation(ctx, msg.ProjectID, "clone", "started", ""); err != nil {
		log.Printf("Failed to log operation start: %v", err)
	}

	// Clone repository
	repoPath, cloneUrl, details, err := p.gitlab.CloneProject(msg.ProjectID)
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

	// Create metadata
	metadata := Metadata{
		ProjectId:     strconv.Itoa(msg.ProjectID),
		ProjectTitle:  details.Name,
		ProjectUrl:    strings.TrimSuffix(cloneUrl, ".git"),
		JobId:         "1",
		CommitBranch:  details.CommitBranch,
		Source:        "sbomer",
		GeneratedDate: time.Now().Format("2006-01-02"),
		SbomFormat:    "cyclonedx-json",
		Version:       "1.0",
		TopicsId:      details.Topics,
	}

	bom, err := parseSBOM(sbomData)

	// Create SBOM scan request event
	sbomScanRequestEvent := SbomScanRequestEvent{
		Metadata: metadata,
		SBOM:     bom,
	}

	// Publish metadata to RabbitMQ
	sbomScanRequestEventBytes, err := json.Marshal(sbomScanRequestEvent)

	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := workerScannerConsumer.Publish(ctx, sbomScanRequestEventBytes); err != nil {
		return fmt.Errorf("failed to publish metadata: %w", err)
	}

	fmt.Printf("🚀 Published SBOM scan request event for project %d\n", msg.ProjectID)

	return nil
}
