package models

import (
	"encoding/json"
	"time"
)

type SBOM struct {
	ProjectUID int             `db:"project_uid"`
	Name       string          `db:"name"`
	Path       string          `db:"path"`
	Topics     []string        `db:"topics"`
	SBOMData   json.RawMessage `db:"sbom_data"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}
