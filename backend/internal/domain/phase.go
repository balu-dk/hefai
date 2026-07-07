package domain

import (
	"time"

	"github.com/google/uuid"
)

type PhaseStatus string

const (
	PhaseNotStarted PhaseStatus = "not_started"
	PhaseInProgress PhaseStatus = "in_progress"
	PhaseCompleted  PhaseStatus = "completed"
)

func (s PhaseStatus) Valid() bool {
	switch s {
	case PhaseNotStarted, PhaseInProgress, PhaseCompleted:
		return true
	}
	return false
}

type Phase struct {
	ID           uuid.UUID   `json:"id"`
	ProjectID    uuid.UUID   `json:"projectId"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	SortOrder    int         `json:"sortOrder"`
	Status       PhaseStatus `json:"status"`
	PlannedStart *time.Time  `json:"plannedStart"`
	PlannedEnd   *time.Time  `json:"plannedEnd"`
	ActualStart  *time.Time  `json:"actualStart"`
	ActualEnd    *time.Time  `json:"actualEnd"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

// DefaultPhaseNames seeds new projects with the standard building phases.
var DefaultPhaseNames = []string{
	"Grund & fundament",
	"Råhus",
	"Tag",
	"Lukning",
	"Installationer",
	"Indvendig",
	"Finish",
}
