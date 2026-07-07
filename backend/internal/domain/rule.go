package domain

import (
	"time"

	"github.com/google/uuid"
)

type RuleStatus string

const (
	RuleSuggested RuleStatus = "suggested"
	RuleConfirmed RuleStatus = "confirmed"
	RuleRejected  RuleStatus = "rejected"
)

func (s RuleStatus) Valid() bool {
	switch s {
	case RuleSuggested, RuleConfirmed, RuleRejected:
		return true
	}
	return false
}

// ComplianceRule er ét målbart lovkrav med kildehenvisning — fx maks.
// bebyggelsesprocent 15 med citat fra lokalplanen. Reglen er "suggested"
// indtil brugeren bekræfter den mod kilden.
type ComplianceRule struct {
	ID            uuid.UUID  `json:"id"`
	ProjectID     uuid.UUID  `json:"projectId"`
	Parameter     string     `json:"parameter"`
	Value         float64    `json:"value"`
	SourceChunkID *uuid.UUID `json:"sourceChunkId"`
	SourceRef     string     `json:"sourceRef"` // opslået sektionsreference
	Quote         string     `json:"quote"`
	Status        RuleStatus `json:"status"`
	Note          string     `json:"note"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
