package service

import (
	"testing"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

func edge(task, dependsOn uuid.UUID) domain.TaskDependency {
	return domain.TaskDependency{TaskID: task, DependsOnTaskID: dependsOn}
}

func TestCreatesCycle(t *testing.T) {
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	tests := []struct {
		name      string
		edges     []domain.TaskDependency
		task, dep uuid.UUID
		want      bool
	}{
		{"empty graph", nil, a, b, false},
		{"direct cycle", []domain.TaskDependency{edge(b, a)}, a, b, true},
		{"transitive cycle", []domain.TaskDependency{edge(b, a), edge(c, b)}, a, c, true},
		{"long chain no cycle", []domain.TaskDependency{edge(b, a), edge(c, b), edge(d, c)}, d, a, false},
		{"diamond no cycle", []domain.TaskDependency{edge(b, a), edge(c, a), edge(d, b), edge(d, c)}, d, a, false},
		{"self edge", nil, a, a, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createsCycle(tt.edges, tt.task, tt.dep); got != tt.want {
				t.Errorf("createsCycle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func task(id uuid.UUID, status domain.TaskStatus) *domain.Task {
	return &domain.Task{ID: id, Status: status}
}

func TestBuildBoard(t *testing.T) {
	dig := uuid.New()     // done
	found := uuid.New()   // todo, depends on dig -> actionable
	walls := uuid.New()   // todo, depends on found -> waiting
	roof := uuid.New()    // todo, depends on walls -> waiting
	permit := uuid.New()  // cancelled
	windows := uuid.New() // todo, depends on cancelled permit -> actionable

	tasks := []*domain.Task{
		task(dig, domain.TaskDone),
		task(found, domain.TaskTodo),
		task(walls, domain.TaskTodo),
		task(roof, domain.TaskTodo),
		task(permit, domain.TaskCancelled),
		task(windows, domain.TaskTodo),
	}
	edges := []domain.TaskDependency{
		edge(found, dig),
		edge(walls, found),
		edge(roof, walls),
		edge(windows, permit),
	}

	board := buildBoard(tasks, edges)
	byID := map[uuid.UUID]*domain.BoardTask{}
	for _, bt := range board {
		byID[bt.ID] = bt
	}

	if !byID[found].Actionable {
		t.Error("found should be actionable: its only dependency is done")
	}
	if byID[walls].Actionable {
		t.Error("walls should not be actionable: found is not done")
	}
	if len(byID[walls].WaitingFor) != 1 || byID[walls].WaitingFor[0] != found {
		t.Errorf("walls should be waiting for found, got %v", byID[walls].WaitingFor)
	}
	if !byID[windows].Actionable {
		t.Error("windows should be actionable: a cancelled dependency does not block")
	}
	if byID[dig].Actionable {
		t.Error("a done task is not actionable")
	}
	// dig blocks found; found blocks walls
	if len(byID[dig].Blocks) != 1 || byID[dig].Blocks[0] != found {
		t.Errorf("dig should block found, got %v", byID[dig].Blocks)
	}
}
