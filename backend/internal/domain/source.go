package domain

import (
	"time"

	"github.com/google/uuid"
)

type SourceKind string

const (
	SourceBR18       SourceKind = "br18"
	SourceEurocode   SourceKind = "eurocode"
	SourceLocalPlan  SourceKind = "local_plan"
	SourceMunicipal  SourceKind = "municipal_guidance"
	SourceOtherKind  SourceKind = "other"
)

func (k SourceKind) Valid() bool {
	switch k {
	case SourceBR18, SourceEurocode, SourceLocalPlan, SourceMunicipal, SourceOtherKind:
		return true
	}
	return false
}

type SourceStatus string

const (
	SourceProcessing SourceStatus = "processing"
	SourceReady      SourceStatus = "ready"
	SourceFailed     SourceStatus = "failed"
)

// SourceDocument is reference material (BR18, lokalplan, kommunens krav)
// that the assistant is grounded in. ProjectID nil = shared library.
type SourceDocument struct {
	ID           uuid.UUID    `json:"id"`
	ProjectID    *uuid.UUID   `json:"projectId"`
	DocumentID   *uuid.UUID   `json:"documentId"`
	Title        string       `json:"title"`
	Kind         SourceKind   `json:"kind"`
	VersionLabel string       `json:"versionLabel"`
	URL          string       `json:"url"`
	Status       SourceStatus `json:"status"`
	AddedBy      *uuid.UUID   `json:"addedBy"`
	ChunkCount   int          `json:"chunkCount"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type SourceChunk struct {
	ID               uuid.UUID `json:"id"`
	SourceDocumentID uuid.UUID `json:"sourceDocumentId"`
	ChunkIndex       int       `json:"chunkIndex"`
	Content          string    `json:"content"`
	SectionRef       string    `json:"sectionRef"`
	PageNo           *int      `json:"pageNo"`
}

// SourceHit is one retrieval result with provenance for citation.
type SourceHit struct {
	ChunkID     uuid.UUID  `json:"chunkId"`
	SourceID    uuid.UUID  `json:"sourceId"`
	SourceTitle string     `json:"sourceTitle"`
	SourceKind  SourceKind `json:"sourceKind"`
	SectionRef  string     `json:"sectionRef"`
	Content     string     `json:"content"`
	Rank        float64    `json:"rank"`
}
