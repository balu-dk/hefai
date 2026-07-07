package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskTodo       TaskStatus = "todo"
	TaskBlocked    TaskStatus = "blocked"
	TaskInProgress TaskStatus = "in_progress"
	TaskDone       TaskStatus = "done"
	TaskCancelled  TaskStatus = "cancelled"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskTodo, TaskBlocked, TaskInProgress, TaskDone, TaskCancelled:
		return true
	}
	return false
}

// Terminal reports whether the status ends the task (it no longer blocks
// dependents).
func (s TaskStatus) Terminal() bool { return s == TaskDone || s == TaskCancelled }

type Task struct {
	ID                    uuid.UUID  `json:"id"`
	ProjectID             uuid.UUID  `json:"projectId"`
	PhaseID               *uuid.UUID `json:"phaseId"`
	RoomID                *uuid.UUID `json:"roomId"`
	Title                 string     `json:"title"`
	Description           string     `json:"description"`
	Status                TaskStatus `json:"status"`
	ResponsibleUserID     *uuid.UUID `json:"responsibleUserId"`
	ResponsibleSupplierID *uuid.UUID `json:"responsibleSupplierId"`
	PlannedStart          *time.Time `json:"plannedStart"`
	PlannedEnd            *time.Time `json:"plannedEnd"`
	ActualStart           *time.Time `json:"actualStart"`
	ActualEnd             *time.Time `json:"actualEnd"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type TaskDependency struct {
	TaskID          uuid.UUID `json:"taskId"`
	DependsOnTaskID uuid.UUID `json:"dependsOnTaskId"`
}

// BoardTask is a task annotated with dependency-derived state for the
// "what can I do now / what blocks what" overview.
type BoardTask struct {
	Task
	DependsOn  []uuid.UUID `json:"dependsOn"`  // tasks this one waits for
	Blocks     []uuid.UUID `json:"blocks"`     // tasks waiting for this one
	Actionable bool        `json:"actionable"` // not started and nothing unfinished ahead of it
	WaitingFor []uuid.UUID `json:"waitingFor"` // the unfinished subset of DependsOn
}
