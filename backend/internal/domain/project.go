package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProjectKind string

const (
	ProjectKindNewBuild   ProjectKind = "new_build"
	ProjectKindRenovation ProjectKind = "renovation"
	ProjectKindExtension  ProjectKind = "extension"
	ProjectKindOther      ProjectKind = "other"
)

func (k ProjectKind) Valid() bool {
	switch k {
	case ProjectKindNewBuild, ProjectKindRenovation, ProjectKindExtension, ProjectKindOther:
		return true
	}
	return false
}

type ProjectStatus string

const (
	ProjectStatusPlanning   ProjectStatus = "planning"
	ProjectStatusInProgress ProjectStatus = "in_progress"
	ProjectStatusOnHold     ProjectStatus = "on_hold"
	ProjectStatusCompleted  ProjectStatus = "completed"
	ProjectStatusArchived   ProjectStatus = "archived"
)

func (s ProjectStatus) Valid() bool {
	switch s {
	case ProjectStatusPlanning, ProjectStatusInProgress, ProjectStatusOnHold,
		ProjectStatusCompleted, ProjectStatusArchived:
		return true
	}
	return false
}

type ProjectRole string

const (
	RoleOwner  ProjectRole = "owner"
	RoleMember ProjectRole = "member"
	RoleViewer ProjectRole = "viewer"
)

func (r ProjectRole) Valid() bool {
	switch r {
	case RoleOwner, RoleMember, RoleViewer:
		return true
	}
	return false
}

// CanWrite reports whether the role may create/update project content.
func (r ProjectRole) CanWrite() bool { return r == RoleOwner || r == RoleMember }

// CanManage reports whether the role may manage members and delete the project.
func (r ProjectRole) CanManage() bool { return r == RoleOwner }

type Project struct {
	ID           uuid.UUID     `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Kind         ProjectKind   `json:"kind"`
	Status       ProjectStatus `json:"status"`
	Address      string        `json:"address"`
	Municipality string        `json:"municipality"`
	CadastralID  string        `json:"cadastralId"`
	PlotAreaM2   *float64      `json:"plotAreaM2"`
	CreatedBy    uuid.UUID     `json:"createdBy"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

type ProjectMember struct {
	ProjectID   uuid.UUID   `json:"projectId"`
	UserID      uuid.UUID   `json:"userId"`
	Role        ProjectRole `json:"role"`
	Email       string      `json:"email"`
	DisplayName string      `json:"displayName"`
	CreatedAt   time.Time   `json:"createdAt"`
}
