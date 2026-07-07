package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Materials struct {
	db *pgxpool.Pool
}

func NewMaterials(db *pgxpool.Pool) *Materials { return &Materials{db: db} }

const materialColumns = `id, project_id, phase_id, task_id, room_id, supplier_id, name, spec,
	quantity, unit, (unit_price*100)::bigint, currency, status, notes, created_at, updated_at`

func scanMaterial(row pgx.Row) (*domain.Material, error) {
	var m domain.Material
	err := row.Scan(&m.ID, &m.ProjectID, &m.PhaseID, &m.TaskID, &m.RoomID, &m.SupplierID,
		&m.Name, &m.Spec, &m.Quantity, &m.Unit, &m.UnitPriceOre, &m.Currency,
		&m.Status, &m.Notes, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

func (r *Materials) Create(ctx context.Context, m *domain.Material) (*domain.Material, error) {
	return scanMaterial(r.db.QueryRow(ctx, `
		INSERT INTO materials (project_id, phase_id, task_id, room_id, supplier_id, name, spec,
			quantity, unit, unit_price, currency, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::bigint::numeric/100, $11, $12, $13)
		RETURNING `+materialColumns,
		m.ProjectID, m.PhaseID, m.TaskID, m.RoomID, m.SupplierID, m.Name, m.Spec,
		m.Quantity, m.Unit, m.UnitPriceOre, m.Currency, m.Status, m.Notes))
}

func (r *Materials) Get(ctx context.Context, id uuid.UUID) (*domain.Material, error) {
	return scanMaterial(r.db.QueryRow(ctx,
		`SELECT `+materialColumns+` FROM materials WHERE id = $1`, id))
}

func (r *Materials) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Material, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+materialColumns+` FROM materials
		WHERE project_id = $1 ORDER BY status, name`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	materials := []*domain.Material{}
	for rows.Next() {
		m, err := scanMaterial(rows)
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}
	return materials, mapErr(rows.Err())
}

func (r *Materials) Update(ctx context.Context, m *domain.Material) (*domain.Material, error) {
	return scanMaterial(r.db.QueryRow(ctx, `
		UPDATE materials SET phase_id = $2, task_id = $3, room_id = $4, supplier_id = $5,
			name = $6, spec = $7, quantity = $8, unit = $9,
			unit_price = $10::bigint::numeric/100, currency = $11, status = $12, notes = $13
		WHERE id = $1
		RETURNING `+materialColumns,
		m.ID, m.PhaseID, m.TaskID, m.RoomID, m.SupplierID, m.Name, m.Spec,
		m.Quantity, m.Unit, m.UnitPriceOre, m.Currency, m.Status, m.Notes))
}

func (r *Materials) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM materials WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
