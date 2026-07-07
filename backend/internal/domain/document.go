package domain

import (
	"time"

	"github.com/google/uuid"
)

type DocumentKind string

const (
	DocArchitectDrawing    DocumentKind = "architect_drawing"
	DocConstructionDrawing DocumentKind = "construction_drawing"
	DocReceipt             DocumentKind = "receipt"
	DocPhoto               DocumentKind = "photo"
	DocWarranty            DocumentKind = "warranty"
	DocDatasheet           DocumentKind = "datasheet"
	DocPermit              DocumentKind = "permit"
	DocCorrespondence      DocumentKind = "correspondence"
	DocGenerated           DocumentKind = "generated"
	DocOther               DocumentKind = "other"
)

func (k DocumentKind) Valid() bool {
	switch k {
	case DocArchitectDrawing, DocConstructionDrawing, DocReceipt, DocPhoto, DocWarranty,
		DocDatasheet, DocPermit, DocCorrespondence, DocGenerated, DocOther:
		return true
	}
	return false
}

type Document struct {
	ID          uuid.UUID    `json:"id"`
	ProjectID   uuid.UUID    `json:"projectId"`
	UploadedBy  *uuid.UUID   `json:"uploadedBy"`
	Kind        DocumentKind `json:"kind"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Filename    string       `json:"filename"`
	StorageKey  string       `json:"-"`
	MimeType    string       `json:"mimeType"`
	SizeBytes   int64        `json:"sizeBytes"`
	CapturedAt  *time.Time   `json:"capturedAt"`
	Tags        []string     `json:"tags"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// LinkTargetType names the entity kinds a document can attach to. Each maps
// to its own FK column in document_links.
type LinkTargetType string

const (
	LinkPhase             LinkTargetType = "phase"
	LinkTask              LinkTargetType = "task"
	LinkRoom              LinkTargetType = "room"
	LinkExpense           LinkTargetType = "expense"
	LinkMaterial          LinkTargetType = "material"
	LinkSupplier          LinkTargetType = "supplier"
	LinkCaseFile          LinkTargetType = "case_file"
	LinkStructuralElement LinkTargetType = "structural_element"
)

func (t LinkTargetType) Valid() bool {
	switch t {
	case LinkPhase, LinkTask, LinkRoom, LinkExpense, LinkMaterial, LinkSupplier,
		LinkCaseFile, LinkStructuralElement:
		return true
	}
	return false
}

type DocumentLink struct {
	ID         uuid.UUID      `json:"id"`
	DocumentID uuid.UUID      `json:"documentId"`
	TargetType LinkTargetType `json:"targetType"`
	TargetID   uuid.UUID      `json:"targetId"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// DocumentFilter narrows document listings.
type DocumentFilter struct {
	Kind       DocumentKind
	Query      string // matches title/description/filename, case-insensitive
	Tag        string
	TargetType LinkTargetType // with TargetID: only documents linked to that entity
	TargetID   uuid.UUID
}
