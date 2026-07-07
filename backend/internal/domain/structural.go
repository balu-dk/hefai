package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type StructuralElementType string

const (
	ElementBeam       StructuralElementType = "beam"
	ElementColumn     StructuralElementType = "column"
	ElementWall       StructuralElementType = "wall"
	ElementFoundation StructuralElementType = "foundation"
	ElementRoof       StructuralElementType = "roof"
	ElementSlab       StructuralElementType = "slab"
	ElementOther      StructuralElementType = "other"
)

func (t StructuralElementType) Valid() bool {
	switch t {
	case ElementBeam, ElementColumn, ElementWall, ElementFoundation, ElementRoof,
		ElementSlab, ElementOther:
		return true
	}
	return false
}

type StructuralMaterial string

const (
	MaterialTimber   StructuralMaterial = "timber"
	MaterialSteel    StructuralMaterial = "steel"
	MaterialConcrete StructuralMaterial = "concrete"
	MaterialMasonry  StructuralMaterial = "masonry"
	MaterialOther    StructuralMaterial = "other"
)

func (m StructuralMaterial) Valid() bool {
	switch m {
	case MaterialTimber, MaterialSteel, MaterialConcrete, MaterialMasonry, MaterialOther:
		return true
	}
	return false
}

type StructuralElement struct {
	ID            uuid.UUID             `json:"id"`
	ProjectID     uuid.UUID             `json:"projectId"`
	RoomID        *uuid.UUID            `json:"roomId"`
	DrawingID     *uuid.UUID            `json:"drawingId"`
	ElementType   StructuralElementType `json:"elementType"`
	Name          string                `json:"name"`
	IsLoadBearing bool                  `json:"isLoadBearing"`
	Material      StructuralMaterial    `json:"material"`
	MaterialSpec  string                `json:"materialSpec"` // C24, S235, C25/30 …
	Geometry      json.RawMessage       `json:"geometry"`     // type-dependent, validated per element type
	Notes         string                `json:"notes"`
	CreatedAt     time.Time             `json:"createdAt"`
	UpdatedAt     time.Time             `json:"updatedAt"`
}

type LoadType string

const (
	LoadDead  LoadType = "dead"
	LoadLive  LoadType = "live"
	LoadSnow  LoadType = "snow"
	LoadWind  LoadType = "wind"
	LoadPoint LoadType = "point"
	LoadLine  LoadType = "line"
	LoadOther LoadType = "other"
)

func (t LoadType) Valid() bool {
	switch t {
	case LoadDead, LoadLive, LoadSnow, LoadWind, LoadPoint, LoadLine, LoadOther:
		return true
	}
	return false
}

type LoadStatus string

const (
	LoadAssumed          LoadStatus = "assumed"
	LoadEngineerConfirmed LoadStatus = "engineer_confirmed"
	LoadEngineerChanged  LoadStatus = "engineer_changed"
)

func (s LoadStatus) Valid() bool {
	switch s {
	case LoadAssumed, LoadEngineerConfirmed, LoadEngineerChanged:
		return true
	}
	return false
}

type Load struct {
	ID                  uuid.UUID       `json:"id"`
	ProjectID           uuid.UUID       `json:"projectId"`
	StructuralElementID *uuid.UUID      `json:"structuralElementId"` // nil = project-level (site) load
	LoadType            LoadType        `json:"loadType"`
	Value               float64         `json:"value"`
	Unit                string          `json:"unit"`
	StandardReference   string          `json:"standardReference"`
	Derivation          json.RawMessage `json:"derivation"` // zone, terrain, factors …
	Status              LoadStatus      `json:"status"`
	Notes               string          `json:"notes"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
}

type EstimateStatus string

const (
	EstimateAdvisory   EstimateStatus = "advisory"
	EstimateVerified   EstimateStatus = "verified"
	EstimateSuperseded EstimateStatus = "superseded"
	EstimateRejected   EstimateStatus = "rejected"
)

func (s EstimateStatus) Valid() bool {
	switch s {
	case EstimateAdvisory, EstimateVerified, EstimateSuperseded, EstimateRejected:
		return true
	}
	return false
}

// CalculationEstimate is one immutable run of a calc method. Re-running
// creates a new row; the old one becomes superseded.
type CalculationEstimate struct {
	ID                  uuid.UUID       `json:"id"`
	ProjectID           uuid.UUID       `json:"projectId"`
	StructuralElementID *uuid.UUID      `json:"structuralElementId"`
	Method              string          `json:"method"`
	MethodVersion       string          `json:"methodVersion"`
	StandardReference   string          `json:"standardReference"`
	Inputs              json.RawMessage `json:"inputs"`
	Assumptions         json.RawMessage `json:"assumptions"`
	Results             json.RawMessage `json:"results"`
	Status              EstimateStatus  `json:"status"`
	Notes               string          `json:"notes"`
	CreatedAt           time.Time       `json:"createdAt"`
}

type PackageStatus string

const (
	PackageDraft    PackageStatus = "draft"
	PackageSent     PackageStatus = "sent"
	PackageReviewed PackageStatus = "reviewed"
)

// StructuralPackage is a versioned export of the structural basis for the
// engineer: elements + loads + estimates frozen in a snapshot with a PDF.
type StructuralPackage struct {
	ID         uuid.UUID       `json:"id"`
	ProjectID  uuid.UUID       `json:"projectId"`
	VersionNo  int             `json:"versionNo"`
	Title      string          `json:"title"`
	Snapshot   json.RawMessage `json:"snapshot"`
	DocumentID *uuid.UUID      `json:"documentId"`
	Status     PackageStatus   `json:"status"`
	SentAt     *time.Time      `json:"sentAt"`
	CreatedAt  time.Time       `json:"createdAt"`
}

type ReviewStatus string

const (
	ReviewApproved            ReviewStatus = "approved"
	ReviewApprovedWithChanges ReviewStatus = "approved_with_changes"
	ReviewRejected            ReviewStatus = "rejected"
	ReviewPartial             ReviewStatus = "partial"
)

func (s ReviewStatus) Valid() bool {
	switch s {
	case ReviewApproved, ReviewApprovedWithChanges, ReviewRejected, ReviewPartial:
		return true
	}
	return false
}

type ReviewVerdict string

const (
	VerdictApproved ReviewVerdict = "approved"
	VerdictChanged  ReviewVerdict = "changed"
	VerdictRejected ReviewVerdict = "rejected"
	VerdictComment  ReviewVerdict = "comment"
)

func (v ReviewVerdict) Valid() bool {
	switch v {
	case VerdictApproved, VerdictChanged, VerdictRejected, VerdictComment:
		return true
	}
	return false
}

// EngineerReview records the engineer's response to a package (entered
// manually in v1's offline loop).
type EngineerReview struct {
	ID                  uuid.UUID    `json:"id"`
	StructuralPackageID uuid.UUID    `json:"structuralPackageId"`
	ReviewerName        string       `json:"reviewerName"`
	ReviewerCompany     string       `json:"reviewerCompany"`
	ReviewerCredentials string       `json:"reviewerCredentials"`
	ReceivedAt          time.Time    `json:"receivedAt"`
	OverallStatus       ReviewStatus `json:"overallStatus"`
	Summary             string       `json:"summary"`
	ResponseDocumentID  *uuid.UUID   `json:"responseDocumentId"`
	Items               []*EngineerReviewItem `json:"items,omitempty"`
	CreatedAt           time.Time    `json:"createdAt"`
	UpdatedAt           time.Time    `json:"updatedAt"`
}

// EngineerReviewItem is one point-by-point verdict. At most one target is
// set; all nil means a general comment.
type EngineerReviewItem struct {
	ID                    uuid.UUID       `json:"id"`
	EngineerReviewID      uuid.UUID       `json:"engineerReviewId"`
	StructuralElementID   *uuid.UUID      `json:"structuralElementId"`
	LoadID                *uuid.UUID      `json:"loadId"`
	CalculationEstimateID *uuid.UUID      `json:"calculationEstimateId"`
	DrawingID             *uuid.UUID      `json:"drawingId"`
	Verdict               ReviewVerdict   `json:"verdict"`
	Comment               string          `json:"comment"`
	CorrectedValues       json.RawMessage `json:"correctedValues"`
	CreatedAt             time.Time       `json:"createdAt"`
}
