package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Phases struct {
	db *pgxpool.Pool
}

func NewPhases(db *pgxpool.Pool) *Phases { return &Phases{db: db} }

const phaseColumns = `id, project_id, name, description, sort_order, status,
	planned_start, planned_end, actual_start, actual_end, created_at, updated_at`

func scanPhase(row pgx.Row) (*domain.Phase, error) {
	var p domain.Phase
	err := row.Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.SortOrder, &p.Status,
		&p.PlannedStart, &p.PlannedEnd, &p.ActualStart, &p.ActualEnd, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

func (r *Phases) Create(ctx context.Context, p *domain.Phase) (*domain.Phase, error) {
	return scanPhase(r.db.QueryRow(ctx, `
		INSERT INTO phases (project_id, name, description, sort_order, status,
			planned_start, planned_end, actual_start, actual_end)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+phaseColumns,
		p.ProjectID, p.Name, p.Description, p.SortOrder, p.Status,
		p.PlannedStart, p.PlannedEnd, p.ActualStart, p.ActualEnd))
}

func (r *Phases) Get(ctx context.Context, id uuid.UUID) (*domain.Phase, error) {
	return scanPhase(r.db.QueryRow(ctx,
		`SELECT `+phaseColumns+` FROM phases WHERE id = $1`, id))
}

func (r *Phases) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Phase, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+phaseColumns+` FROM phases
		WHERE project_id = $1 ORDER BY sort_order, created_at`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	phases := []*domain.Phase{}
	for rows.Next() {
		p, err := scanPhase(rows)
		if err != nil {
			return nil, err
		}
		phases = append(phases, p)
	}
	return phases, mapErr(rows.Err())
}

func (r *Phases) Update(ctx context.Context, p *domain.Phase) (*domain.Phase, error) {
	return scanPhase(r.db.QueryRow(ctx, `
		UPDATE phases SET name = $2, description = $3, sort_order = $4, status = $5,
			planned_start = $6, planned_end = $7, actual_start = $8, actual_end = $9
		WHERE id = $1
		RETURNING `+phaseColumns,
		p.ID, p.Name, p.Description, p.SortOrder, p.Status,
		p.PlannedStart, p.PlannedEnd, p.ActualStart, p.ActualEnd))
}

func (r *Phases) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM phases WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
