package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Suppliers struct {
	db *pgxpool.Pool
}

func NewSuppliers(db *pgxpool.Pool) *Suppliers { return &Suppliers{db: db} }

const supplierColumns = `id, project_id, company_name, contact_person, trade, phone, email,
	(hourly_rate*100)::bigint, notes, created_at, updated_at`

func scanSupplier(row pgx.Row) (*domain.Supplier, error) {
	var s domain.Supplier
	err := row.Scan(&s.ID, &s.ProjectID, &s.CompanyName, &s.ContactPerson, &s.Trade,
		&s.Phone, &s.Email, &s.HourlyRateOre, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &s, nil
}

func (r *Suppliers) Create(ctx context.Context, s *domain.Supplier) (*domain.Supplier, error) {
	return scanSupplier(r.db.QueryRow(ctx, `
		INSERT INTO suppliers (project_id, company_name, contact_person, trade, phone, email,
			hourly_rate, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7::bigint::numeric/100, $8)
		RETURNING `+supplierColumns,
		s.ProjectID, s.CompanyName, s.ContactPerson, s.Trade, s.Phone, s.Email,
		s.HourlyRateOre, s.Notes))
}

func (r *Suppliers) Get(ctx context.Context, id uuid.UUID) (*domain.Supplier, error) {
	return scanSupplier(r.db.QueryRow(ctx,
		`SELECT `+supplierColumns+` FROM suppliers WHERE id = $1`, id))
}

func (r *Suppliers) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Supplier, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+supplierColumns+` FROM suppliers
		WHERE project_id = $1 ORDER BY company_name`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	suppliers := []*domain.Supplier{}
	for rows.Next() {
		s, err := scanSupplier(rows)
		if err != nil {
			return nil, err
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, mapErr(rows.Err())
}

func (r *Suppliers) Update(ctx context.Context, s *domain.Supplier) (*domain.Supplier, error) {
	return scanSupplier(r.db.QueryRow(ctx, `
		UPDATE suppliers SET company_name = $2, contact_person = $3, trade = $4,
			phone = $5, email = $6, hourly_rate = $7::bigint::numeric/100, notes = $8
		WHERE id = $1
		RETURNING `+supplierColumns,
		s.ID, s.CompanyName, s.ContactPerson, s.Trade, s.Phone, s.Email,
		s.HourlyRateOre, s.Notes))
}

func (r *Suppliers) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM suppliers WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
