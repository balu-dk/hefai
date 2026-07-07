package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Structural struct {
	db *pgxpool.Pool
}

func NewStructural(db *pgxpool.Pool) *Structural { return &Structural{db: db} }

// --- elements ---------------------------------------------------------------

const elementColumns = `id, project_id, room_id, drawing_id, element_type, name,
	is_load_bearing, material, material_spec, geometry, notes, created_at, updated_at`

func scanElement(row pgx.Row) (*domain.StructuralElement, error) {
	var e domain.StructuralElement
	var geometry []byte
	err := row.Scan(&e.ID, &e.ProjectID, &e.RoomID, &e.DrawingID, &e.ElementType, &e.Name,
		&e.IsLoadBearing, &e.Material, &e.MaterialSpec, &geometry, &e.Notes,
		&e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	e.Geometry = json.RawMessage(geometry)
	return &e, nil
}

func (r *Structural) CreateElement(ctx context.Context, e *domain.StructuralElement) (*domain.StructuralElement, error) {
	return scanElement(r.db.QueryRow(ctx, `
		INSERT INTO structural_elements (project_id, room_id, drawing_id, element_type, name,
			is_load_bearing, material, material_spec, geometry, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+elementColumns,
		e.ProjectID, e.RoomID, e.DrawingID, e.ElementType, e.Name,
		e.IsLoadBearing, e.Material, e.MaterialSpec, []byte(e.Geometry), e.Notes))
}

func (r *Structural) GetElement(ctx context.Context, id uuid.UUID) (*domain.StructuralElement, error) {
	return scanElement(r.db.QueryRow(ctx,
		`SELECT `+elementColumns+` FROM structural_elements WHERE id = $1`, id))
}

func (r *Structural) ListElements(ctx context.Context, projectID uuid.UUID) ([]*domain.StructuralElement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+elementColumns+` FROM structural_elements
		WHERE project_id = $1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	elements := []*domain.StructuralElement{}
	for rows.Next() {
		e, err := scanElement(rows)
		if err != nil {
			return nil, err
		}
		elements = append(elements, e)
	}
	return elements, mapErr(rows.Err())
}

func (r *Structural) UpdateElement(ctx context.Context, e *domain.StructuralElement) (*domain.StructuralElement, error) {
	return scanElement(r.db.QueryRow(ctx, `
		UPDATE structural_elements SET room_id = $2, drawing_id = $3, element_type = $4,
			name = $5, is_load_bearing = $6, material = $7, material_spec = $8,
			geometry = $9, notes = $10
		WHERE id = $1
		RETURNING `+elementColumns,
		e.ID, e.RoomID, e.DrawingID, e.ElementType, e.Name, e.IsLoadBearing,
		e.Material, e.MaterialSpec, []byte(e.Geometry), e.Notes))
}

func (r *Structural) DeleteElement(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM structural_elements WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// --- loads -------------------------------------------------------------------

const loadColumns = `id, project_id, structural_element_id, load_type, value, unit,
	standard_reference, derivation, status, notes, created_at, updated_at`

func scanLoad(row pgx.Row) (*domain.Load, error) {
	var l domain.Load
	var derivation []byte
	err := row.Scan(&l.ID, &l.ProjectID, &l.StructuralElementID, &l.LoadType, &l.Value,
		&l.Unit, &l.StandardReference, &derivation, &l.Status, &l.Notes,
		&l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	l.Derivation = json.RawMessage(derivation)
	return &l, nil
}

func (r *Structural) CreateLoad(ctx context.Context, l *domain.Load) (*domain.Load, error) {
	return scanLoad(r.db.QueryRow(ctx, `
		INSERT INTO loads (project_id, structural_element_id, load_type, value, unit,
			standard_reference, derivation, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+loadColumns,
		l.ProjectID, l.StructuralElementID, l.LoadType, l.Value, l.Unit,
		l.StandardReference, []byte(l.Derivation), l.Status, l.Notes))
}

func (r *Structural) GetLoad(ctx context.Context, id uuid.UUID) (*domain.Load, error) {
	return scanLoad(r.db.QueryRow(ctx, `SELECT `+loadColumns+` FROM loads WHERE id = $1`, id))
}

func (r *Structural) ListLoads(ctx context.Context, projectID uuid.UUID) ([]*domain.Load, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+loadColumns+` FROM loads WHERE project_id = $1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	loads := []*domain.Load{}
	for rows.Next() {
		l, err := scanLoad(rows)
		if err != nil {
			return nil, err
		}
		loads = append(loads, l)
	}
	return loads, mapErr(rows.Err())
}

func (r *Structural) UpdateLoad(ctx context.Context, l *domain.Load) (*domain.Load, error) {
	return scanLoad(r.db.QueryRow(ctx, `
		UPDATE loads SET structural_element_id = $2, load_type = $3, value = $4, unit = $5,
			standard_reference = $6, derivation = $7, status = $8, notes = $9
		WHERE id = $1
		RETURNING `+loadColumns,
		l.ID, l.StructuralElementID, l.LoadType, l.Value, l.Unit,
		l.StandardReference, []byte(l.Derivation), l.Status, l.Notes))
}

func (r *Structural) DeleteLoad(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM loads WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// --- estimates ----------------------------------------------------------------

const estimateColumns = `id, project_id, structural_element_id, method, method_version,
	standard_reference, inputs, assumptions, results, status, notes, created_at`

func scanEstimate(row pgx.Row) (*domain.CalculationEstimate, error) {
	var e domain.CalculationEstimate
	var inputs, assumptions, results []byte
	err := row.Scan(&e.ID, &e.ProjectID, &e.StructuralElementID, &e.Method, &e.MethodVersion,
		&e.StandardReference, &inputs, &assumptions, &results, &e.Status, &e.Notes, &e.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	e.Inputs = json.RawMessage(inputs)
	e.Assumptions = json.RawMessage(assumptions)
	e.Results = json.RawMessage(results)
	return &e, nil
}

// CreateEstimate stores a new run and supersedes previous advisory runs of
// the same method on the same element.
func (r *Structural) CreateEstimate(ctx context.Context, e *domain.CalculationEstimate) (*domain.CalculationEstimate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		UPDATE calculation_estimates SET status = 'superseded'
		WHERE project_id = $1 AND method = $2 AND status = 'advisory'
		  AND structural_element_id IS NOT DISTINCT FROM $3`,
		e.ProjectID, e.Method, e.StructuralElementID); err != nil {
		return nil, mapErr(err)
	}

	created, err := scanEstimate(tx.QueryRow(ctx, `
		INSERT INTO calculation_estimates (project_id, structural_element_id, method,
			method_version, standard_reference, inputs, assumptions, results, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+estimateColumns,
		e.ProjectID, e.StructuralElementID, e.Method, e.MethodVersion, e.StandardReference,
		[]byte(e.Inputs), []byte(e.Assumptions), []byte(e.Results), e.Status, e.Notes))
	if err != nil {
		return nil, err
	}
	return created, tx.Commit(ctx)
}

func (r *Structural) GetEstimate(ctx context.Context, id uuid.UUID) (*domain.CalculationEstimate, error) {
	return scanEstimate(r.db.QueryRow(ctx,
		`SELECT `+estimateColumns+` FROM calculation_estimates WHERE id = $1`, id))
}

func (r *Structural) ListEstimates(ctx context.Context, projectID uuid.UUID) ([]*domain.CalculationEstimate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+estimateColumns+` FROM calculation_estimates
		WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	estimates := []*domain.CalculationEstimate{}
	for rows.Next() {
		e, err := scanEstimate(rows)
		if err != nil {
			return nil, err
		}
		estimates = append(estimates, e)
	}
	return estimates, mapErr(rows.Err())
}

func (r *Structural) SetEstimateStatus(ctx context.Context, id uuid.UUID, status domain.EstimateStatus) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE calculation_estimates SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Structural) SetLoadStatus(ctx context.Context, id uuid.UUID, status domain.LoadStatus) error {
	tag, err := r.db.Exec(ctx, `UPDATE loads SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
