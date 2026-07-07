package domain

import (
	"time"

	"github.com/google/uuid"
)

type CaseType string

const (
	CaseTypeUnknown        CaseType = "unknown"
	CaseTypeNotification   CaseType = "notification"    // anmeldelse
	CaseTypeBuildingPermit CaseType = "building_permit" // byggetilladelse
)

func (t CaseType) Valid() bool {
	switch t {
	case CaseTypeUnknown, CaseTypeNotification, CaseTypeBuildingPermit:
		return true
	}
	return false
}

type CaseStatus string

const (
	CaseDraft        CaseStatus = "draft"
	CaseReady        CaseStatus = "ready_for_submission"
	CaseSubmitted    CaseStatus = "submitted"
	CaseAwaiting     CaseStatus = "awaiting_response"
	CaseQuestions    CaseStatus = "questions_from_municipality"
	CaseApproved     CaseStatus = "approved"
	CaseRejected     CaseStatus = "rejected"
	CaseClosed       CaseStatus = "closed"
)

func (s CaseStatus) Valid() bool {
	switch s {
	case CaseDraft, CaseReady, CaseSubmitted, CaseAwaiting, CaseQuestions,
		CaseApproved, CaseRejected, CaseClosed:
		return true
	}
	return false
}

type CaseFile struct {
	ID                  uuid.UUID  `json:"id"`
	ProjectID           uuid.UUID  `json:"projectId"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	CaseType            CaseType   `json:"caseType"`
	Status              CaseStatus `json:"status"`
	MunicipalCaseNumber string     `json:"municipalCaseNumber"`
	SubmittedAt         *time.Time `json:"submittedAt"`
	DecidedAt           *time.Time `json:"decidedAt"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type CaseEventType string

const (
	CaseEventStatusChange   CaseEventType = "status_change"
	CaseEventCorrespondence CaseEventType = "correspondence"
	CaseEventNote           CaseEventType = "note"
	CaseEventSubmission     CaseEventType = "submission"
)

func (t CaseEventType) Valid() bool {
	switch t {
	case CaseEventStatusChange, CaseEventCorrespondence, CaseEventNote, CaseEventSubmission:
		return true
	}
	return false
}

type Direction string

const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
	DirectionInternal Direction = "internal"
)

func (d Direction) Valid() bool {
	switch d {
	case DirectionIncoming, DirectionOutgoing, DirectionInternal:
		return true
	}
	return false
}

type CaseEvent struct {
	ID         uuid.UUID     `json:"id"`
	CaseFileID uuid.UUID     `json:"caseFileId"`
	EventType  CaseEventType `json:"eventType"`
	Direction  *Direction    `json:"direction"`
	OccurredAt time.Time     `json:"occurredAt"`
	Summary    string        `json:"summary"`
	Body       string        `json:"body"`
	DocumentID *uuid.UUID    `json:"documentId"`
	CreatedBy  *uuid.UUID    `json:"createdBy"`
	CreatedAt  time.Time     `json:"createdAt"`
}
