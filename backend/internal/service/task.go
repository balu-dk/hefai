package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type TaskRepo interface {
	Create(ctx context.Context, t *domain.Task) (*domain.Task, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Task, error)
	Update(ctx context.Context, t *domain.Task) (*domain.Task, error)
	Delete(ctx context.Context, id uuid.UUID) error
	AddDependency(ctx context.Context, taskID, dependsOnTaskID uuid.UUID) error
	RemoveDependency(ctx context.Context, taskID, dependsOnTaskID uuid.UUID) error
	ListDependenciesByProject(ctx context.Context, projectID uuid.UUID) ([]domain.TaskDependency, error)
}

type Tasks struct {
	repo   TaskRepo
	phases PhaseRepo
	access ProjectAccess
}

func NewTasks(repo TaskRepo, phases PhaseRepo, access ProjectAccess) *Tasks {
	return &Tasks{repo: repo, phases: phases, access: access}
}

type TaskPatch struct {
	PhaseID               *uuid.UUID `json:"phaseId"`
	RoomID                *uuid.UUID `json:"roomId"`
	Title                 *string    `json:"title"`
	Description           *string    `json:"description"`
	Status                *string    `json:"status"`
	ResponsibleUserID     *uuid.UUID `json:"responsibleUserId"`
	ResponsibleSupplierID *uuid.UUID `json:"responsibleSupplierId"`
	PlannedStart          *time.Time `json:"plannedStart"`
	PlannedEnd            *time.Time `json:"plannedEnd"`
	ActualStart           *time.Time `json:"actualStart"`
	ActualEnd             *time.Time `json:"actualEnd"`
}

func (s *Tasks) Create(ctx context.Context, userID, projectID uuid.UUID, patch TaskPatch) (*domain.Task, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	t := &domain.Task{ProjectID: projectID, Status: domain.TaskTodo}
	if err := s.applyTaskPatch(ctx, t, patch); err != nil {
		return nil, err
	}
	if t.Title == "" {
		return nil, domain.Validation("opgavetitel kræves")
	}
	return s.repo.Create(ctx, t)
}

func (s *Tasks) Get(ctx context.Context, userID, taskID uuid.UUID) (*domain.Task, error) {
	t, err := s.repo.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := requireRead(ctx, s.access, t.ProjectID, userID); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Tasks) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.Task, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Tasks) Update(ctx context.Context, userID, taskID uuid.UUID, patch TaskPatch) (*domain.Task, error) {
	t, err := s.repo.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, t.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := s.applyTaskPatch(ctx, t, patch); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, t)
}

func (s *Tasks) Delete(ctx context.Context, userID, taskID uuid.UUID) error {
	t, err := s.repo.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, t.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, taskID)
}

// AddDependency records "task waits for dependsOn" after verifying both tasks
// share a project and the new edge would not close a cycle.
func (s *Tasks) AddDependency(ctx context.Context, userID, taskID, dependsOnID uuid.UUID) error {
	if taskID == dependsOnID {
		return domain.Validation("en opgave kan ikke afhænge af sig selv")
	}
	task, err := s.repo.Get(ctx, taskID)
	if err != nil {
		return err
	}
	dependsOn, err := s.repo.Get(ctx, dependsOnID)
	if err != nil {
		return err
	}
	if task.ProjectID != dependsOn.ProjectID {
		return domain.Validation("opgaverne tilhører ikke samme projekt")
	}
	if err := requireWrite(ctx, s.access, task.ProjectID, userID); err != nil {
		return err
	}

	edges, err := s.repo.ListDependenciesByProject(ctx, task.ProjectID)
	if err != nil {
		return err
	}
	if createsCycle(edges, taskID, dependsOnID) {
		return domain.Validation("afhængigheden ville skabe en cirkel")
	}
	if err := s.repo.AddDependency(ctx, taskID, dependsOnID); err != nil {
		if err == domain.ErrConflict {
			return domain.Validation("afhængigheden findes allerede")
		}
		return err
	}
	return nil
}

func (s *Tasks) RemoveDependency(ctx context.Context, userID, taskID, dependsOnID uuid.UUID) error {
	task, err := s.repo.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, task.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.RemoveDependency(ctx, taskID, dependsOnID)
}

// Board returns every task annotated with dependency state: what it waits
// for, what it blocks, and whether it is actionable right now.
func (s *Tasks) Board(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.BoardTask, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	tasks, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	edges, err := s.repo.ListDependenciesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return buildBoard(tasks, edges), nil
}

// buildBoard is pure so it can be tested without a database.
func buildBoard(tasks []*domain.Task, edges []domain.TaskDependency) []*domain.BoardTask {
	byID := make(map[uuid.UUID]*domain.Task, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
	}
	dependsOn := map[uuid.UUID][]uuid.UUID{}
	blocks := map[uuid.UUID][]uuid.UUID{}
	for _, e := range edges {
		dependsOn[e.TaskID] = append(dependsOn[e.TaskID], e.DependsOnTaskID)
		blocks[e.DependsOnTaskID] = append(blocks[e.DependsOnTaskID], e.TaskID)
	}

	board := make([]*domain.BoardTask, 0, len(tasks))
	for _, t := range tasks {
		bt := &domain.BoardTask{
			Task:       *t,
			DependsOn:  orEmpty(dependsOn[t.ID]),
			Blocks:     orEmpty(blocks[t.ID]),
			WaitingFor: []uuid.UUID{},
		}
		for _, depID := range dependsOn[t.ID] {
			if dep, ok := byID[depID]; ok && !dep.Status.Terminal() {
				bt.WaitingFor = append(bt.WaitingFor, depID)
			}
		}
		notStarted := t.Status == domain.TaskTodo || t.Status == domain.TaskBlocked
		bt.Actionable = notStarted && len(bt.WaitingFor) == 0
		board = append(board, bt)
	}
	return board
}

func orEmpty(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

// createsCycle reports whether adding taskID -> dependsOnID closes a cycle,
// i.e. whether taskID is already reachable from dependsOnID via existing
// depends-on edges.
func createsCycle(edges []domain.TaskDependency, taskID, dependsOnID uuid.UUID) bool {
	adj := map[uuid.UUID][]uuid.UUID{}
	for _, e := range edges {
		adj[e.TaskID] = append(adj[e.TaskID], e.DependsOnTaskID)
	}
	seen := map[uuid.UUID]bool{}
	stack := []uuid.UUID{dependsOnID}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == taskID {
			return true
		}
		if seen[cur] {
			continue
		}
		seen[cur] = true
		stack = append(stack, adj[cur]...)
	}
	return false
}

func (s *Tasks) applyTaskPatch(ctx context.Context, t *domain.Task, patch TaskPatch) error {
	if patch.Title != nil {
		t.Title = strings.TrimSpace(*patch.Title)
		if t.Title == "" {
			return domain.Validation("opgavetitel kan ikke være tom")
		}
	}
	if patch.Description != nil {
		t.Description = *patch.Description
	}
	if patch.Status != nil {
		status := domain.TaskStatus(*patch.Status)
		if !status.Valid() {
			return domain.Validation("ugyldig opgavestatus")
		}
		t.Status = status
	}
	if patch.PhaseID != nil {
		if *patch.PhaseID == uuid.Nil {
			t.PhaseID = nil
		} else {
			phase, err := s.phases.Get(ctx, *patch.PhaseID)
			if err != nil {
				return err
			}
			if phase.ProjectID != t.ProjectID {
				return domain.Validation("fasen tilhører et andet projekt")
			}
			t.PhaseID = patch.PhaseID
		}
	}
	if patch.RoomID != nil {
		if *patch.RoomID == uuid.Nil {
			t.RoomID = nil
		} else {
			t.RoomID = patch.RoomID
		}
	}
	if patch.ResponsibleUserID != nil {
		if *patch.ResponsibleUserID == uuid.Nil {
			t.ResponsibleUserID = nil
		} else {
			t.ResponsibleUserID = patch.ResponsibleUserID
			t.ResponsibleSupplierID = nil
		}
	}
	if patch.ResponsibleSupplierID != nil {
		if *patch.ResponsibleSupplierID == uuid.Nil {
			t.ResponsibleSupplierID = nil
		} else {
			t.ResponsibleSupplierID = patch.ResponsibleSupplierID
			t.ResponsibleUserID = nil
		}
	}
	if patch.PlannedStart != nil {
		t.PlannedStart = patch.PlannedStart
	}
	if patch.PlannedEnd != nil {
		t.PlannedEnd = patch.PlannedEnd
	}
	if patch.ActualStart != nil {
		t.ActualStart = patch.ActualStart
	}
	if patch.ActualEnd != nil {
		t.ActualEnd = patch.ActualEnd
	}
	return nil
}
