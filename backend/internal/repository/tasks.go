package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Tasks struct {
	db *pgxpool.Pool
}

func NewTasks(db *pgxpool.Pool) *Tasks { return &Tasks{db: db} }

const taskColumns = `id, project_id, phase_id, room_id, title, description, status,
	responsible_user_id, responsible_supplier_id,
	planned_start, planned_end, actual_start, actual_end, created_at, updated_at`

func scanTask(row pgx.Row) (*domain.Task, error) {
	var t domain.Task
	err := row.Scan(&t.ID, &t.ProjectID, &t.PhaseID, &t.RoomID, &t.Title, &t.Description,
		&t.Status, &t.ResponsibleUserID, &t.ResponsibleSupplierID,
		&t.PlannedStart, &t.PlannedEnd, &t.ActualStart, &t.ActualEnd, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &t, nil
}

func (r *Tasks) Create(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	return scanTask(r.db.QueryRow(ctx, `
		INSERT INTO tasks (project_id, phase_id, room_id, title, description, status,
			responsible_user_id, responsible_supplier_id,
			planned_start, planned_end, actual_start, actual_end)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING `+taskColumns,
		t.ProjectID, t.PhaseID, t.RoomID, t.Title, t.Description, t.Status,
		t.ResponsibleUserID, t.ResponsibleSupplierID,
		t.PlannedStart, t.PlannedEnd, t.ActualStart, t.ActualEnd))
}

func (r *Tasks) Get(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	return scanTask(r.db.QueryRow(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE id = $1`, id))
}

func (r *Tasks) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Task, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+taskColumns+` FROM tasks
		WHERE project_id = $1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	tasks := []*domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, mapErr(rows.Err())
}

func (r *Tasks) Update(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	return scanTask(r.db.QueryRow(ctx, `
		UPDATE tasks SET phase_id = $2, room_id = $3, title = $4, description = $5,
			status = $6, responsible_user_id = $7, responsible_supplier_id = $8,
			planned_start = $9, planned_end = $10, actual_start = $11, actual_end = $12
		WHERE id = $1
		RETURNING `+taskColumns,
		t.ID, t.PhaseID, t.RoomID, t.Title, t.Description, t.Status,
		t.ResponsibleUserID, t.ResponsibleSupplierID,
		t.PlannedStart, t.PlannedEnd, t.ActualStart, t.ActualEnd))
}

func (r *Tasks) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Tasks) AddDependency(ctx context.Context, taskID, dependsOnTaskID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO task_dependencies (task_id, depends_on_task_id) VALUES ($1, $2)`,
		taskID, dependsOnTaskID)
	return mapErr(err)
}

func (r *Tasks) RemoveDependency(ctx context.Context, taskID, dependsOnTaskID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM task_dependencies WHERE task_id = $1 AND depends_on_task_id = $2`,
		taskID, dependsOnTaskID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListDependenciesByProject returns every dependency edge within a project.
func (r *Tasks) ListDependenciesByProject(ctx context.Context, projectID uuid.UUID) ([]domain.TaskDependency, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.task_id, d.depends_on_task_id
		FROM task_dependencies d
		JOIN tasks t ON t.id = d.task_id
		WHERE t.project_id = $1`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	deps := []domain.TaskDependency{}
	for rows.Next() {
		var d domain.TaskDependency
		if err := rows.Scan(&d.TaskID, &d.DependsOnTaskID); err != nil {
			return nil, mapErr(err)
		}
		deps = append(deps, d)
	}
	return deps, mapErr(rows.Err())
}
