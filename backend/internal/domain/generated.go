package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type GeneratedDocumentKind string

const (
	GenSitePlan           GeneratedDocumentKind = "site_plan"
	GenFloorPlan          GeneratedDocumentKind = "floor_plan"
	GenElevation          GeneratedDocumentKind = "elevation"
	GenAreaStatement      GeneratedDocumentKind = "area_statement"
	GenProjectDescription GeneratedDocumentKind = "project_description"
	GenApplicationSummary GeneratedDocumentKind = "application_summary"
	GenStructuralPackage  GeneratedDocumentKind = "structural_package"
	GenOther              GeneratedDocumentKind = "other"
)

func (k GeneratedDocumentKind) Valid() bool {
	switch k {
	case GenSitePlan, GenFloorPlan, GenElevation, GenAreaStatement,
		GenProjectDescription, GenApplicationSummary, GenStructuralPackage, GenOther:
		return true
	}
	return false
}

type GeneratedDocumentStatus string

const (
	GeneratedDraft GeneratedDocumentStatus = "draft"
	GeneratedFinal GeneratedDocumentStatus = "final"
)

// GeneratedDocument records one immutable PDF generation: what was made,
// from which input snapshot, and where the rendered file lives.
type GeneratedDocument struct {
	ID            uuid.UUID               `json:"id"`
	ProjectID     uuid.UUID               `json:"projectId"`
	CaseFileID    *uuid.UUID              `json:"caseFileId"`
	Kind          GeneratedDocumentKind   `json:"kind"`
	Status        GeneratedDocumentStatus `json:"status"`
	VersionNo     int                     `json:"versionNo"`
	InputSnapshot json.RawMessage         `json:"inputSnapshot"`
	DocumentID    *uuid.UUID              `json:"documentId"`
	CreatedAt     time.Time               `json:"createdAt"`
}
