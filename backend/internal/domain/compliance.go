package domain

import (
	"time"

	"github.com/google/uuid"
)

type ComplianceStatus string

const (
	ComplianceNotChecked        ComplianceStatus = "not_checked"
	ComplianceOK                ComplianceStatus = "ok"
	ComplianceAttention         ComplianceStatus = "attention"
	ComplianceNeedsConfirmation ComplianceStatus = "needs_confirmation"
	ComplianceConfirmed         ComplianceStatus = "confirmed"
)

func (s ComplianceStatus) Valid() bool {
	switch s {
	case ComplianceNotChecked, ComplianceOK, ComplianceAttention,
		ComplianceNeedsConfirmation, ComplianceConfirmed:
		return true
	}
	return false
}

// ComplianceCheckItem is one row in the non-binding self-check list. Items
// grounded in source material carry a SourceChunkID; ungrounded items must
// be presented as requiring confirmation.
type ComplianceCheckItem struct {
	ID            uuid.UUID        `json:"id"`
	CaseFileID    uuid.UUID        `json:"caseFileId"`
	Category      string           `json:"category"`
	Requirement   string           `json:"requirement"`
	ExpectedValue string           `json:"expectedValue"`
	ActualValue   string           `json:"actualValue"`
	Status        ComplianceStatus `json:"status"`
	SourceChunkID *uuid.UUID       `json:"sourceChunkId"`
	SourceRef     string           `json:"sourceRef"` // resolved section ref for display
	Note          string           `json:"note"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}
