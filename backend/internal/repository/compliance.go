package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Compliance struct {
	db *pgxpool.Pool
}

func NewCompliance(db *pgxpool.Pool) *Compliance { return &Compliance{db: db} }

const complianceColumns = `i.id, i.case_file_id, i.category, i.requirement, i.expected_value,
	i.actual_value, i.status, i.source_chunk_id, COALESCE(c.section_ref, ''), i.note,
	i.created_at, i.updated_at`

const complianceFrom = ` FROM compliance_check_items i
	LEFT JOIN source_chunks c ON c.id = i.source_chunk_id`

func scanComplianceItem(row pgx.Row) (*domain.ComplianceCheckItem, error) {
	var item domain.ComplianceCheckItem
	err := row.Scan(&item.ID, &item.CaseFileID, &item.Category, &item.Requirement,
		&item.ExpectedValue, &item.ActualValue, &item.Status, &item.SourceChunkID,
		&item.SourceRef, &item.Note, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &item, nil
}

func (r *Compliance) Create(ctx context.Context, item *domain.ComplianceCheckItem) (*domain.ComplianceCheckItem, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO compliance_check_items (case_file_id, category, requirement, expected_value,
			actual_value, status, source_chunk_id, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		item.CaseFileID, item.Category, item.Requirement, item.ExpectedValue,
		item.ActualValue, item.Status, item.SourceChunkID, item.Note).Scan(&id)
	if err != nil {
		return nil, mapErr(err)
	}
	return r.Get(ctx, id)
}

func (r *Compliance) Get(ctx context.Context, id uuid.UUID) (*domain.ComplianceCheckItem, error) {
	return scanComplianceItem(r.db.QueryRow(ctx,
		`SELECT `+complianceColumns+complianceFrom+` WHERE i.id = $1`, id))
}

func (r *Compliance) ListByCaseFile(ctx context.Context, caseFileID uuid.UUID) ([]*domain.ComplianceCheckItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+complianceColumns+complianceFrom+`
		WHERE i.case_file_id = $1 ORDER BY i.category, i.created_at`, caseFileID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	items := []*domain.ComplianceCheckItem{}
	for rows.Next() {
		item, err := scanComplianceItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, mapErr(rows.Err())
}

func (r *Compliance) Update(ctx context.Context, item *domain.ComplianceCheckItem) (*domain.ComplianceCheckItem, error) {
	_, err := r.db.Exec(ctx, `
		UPDATE compliance_check_items SET category = $2, requirement = $3, expected_value = $4,
			actual_value = $5, status = $6, source_chunk_id = $7, note = $8
		WHERE id = $1`,
		item.ID, item.Category, item.Requirement, item.ExpectedValue,
		item.ActualValue, item.Status, item.SourceChunkID, item.Note)
	if err != nil {
		return nil, mapErr(err)
	}
	return r.Get(ctx, item.ID)
}

func (r *Compliance) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM compliance_check_items WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
