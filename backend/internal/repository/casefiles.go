package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type CaseFiles struct {
	db *pgxpool.Pool
}

func NewCaseFiles(db *pgxpool.Pool) *CaseFiles { return &CaseFiles{db: db} }

const caseFileColumns = `id, project_id, title, description, case_type, status,
	municipal_case_number, submitted_at, decided_at, created_at, updated_at`

func scanCaseFile(row pgx.Row) (*domain.CaseFile, error) {
	var c domain.CaseFile
	err := row.Scan(&c.ID, &c.ProjectID, &c.Title, &c.Description, &c.CaseType, &c.Status,
		&c.MunicipalCaseNumber, &c.SubmittedAt, &c.DecidedAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &c, nil
}

func (r *CaseFiles) Create(ctx context.Context, c *domain.CaseFile) (*domain.CaseFile, error) {
	return scanCaseFile(r.db.QueryRow(ctx, `
		INSERT INTO case_files (project_id, title, description, case_type, status, municipal_case_number)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+caseFileColumns,
		c.ProjectID, c.Title, c.Description, c.CaseType, c.Status, c.MunicipalCaseNumber))
}

func (r *CaseFiles) Get(ctx context.Context, id uuid.UUID) (*domain.CaseFile, error) {
	return scanCaseFile(r.db.QueryRow(ctx,
		`SELECT `+caseFileColumns+` FROM case_files WHERE id = $1`, id))
}

func (r *CaseFiles) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.CaseFile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+caseFileColumns+` FROM case_files
		WHERE project_id = $1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	cases := []*domain.CaseFile{}
	for rows.Next() {
		c, err := scanCaseFile(rows)
		if err != nil {
			return nil, err
		}
		cases = append(cases, c)
	}
	return cases, mapErr(rows.Err())
}

func (r *CaseFiles) Update(ctx context.Context, c *domain.CaseFile) (*domain.CaseFile, error) {
	return scanCaseFile(r.db.QueryRow(ctx, `
		UPDATE case_files SET title = $2, description = $3, case_type = $4, status = $5,
			municipal_case_number = $6, submitted_at = $7, decided_at = $8
		WHERE id = $1
		RETURNING `+caseFileColumns,
		c.ID, c.Title, c.Description, c.CaseType, c.Status,
		c.MunicipalCaseNumber, c.SubmittedAt, c.DecidedAt))
}

func (r *CaseFiles) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM case_files WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const caseEventColumns = `id, case_file_id, event_type, direction, occurred_at, summary, body,
	document_id, created_by, created_at`

func (r *CaseFiles) CreateEvent(ctx context.Context, e *domain.CaseEvent) (*domain.CaseEvent, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO case_events (case_file_id, event_type, direction, occurred_at, summary, body,
			document_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+caseEventColumns,
		e.CaseFileID, e.EventType, e.Direction, e.OccurredAt, e.Summary, e.Body,
		e.DocumentID, e.CreatedBy)
	var out domain.CaseEvent
	err := row.Scan(&out.ID, &out.CaseFileID, &out.EventType, &out.Direction, &out.OccurredAt,
		&out.Summary, &out.Body, &out.DocumentID, &out.CreatedBy, &out.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &out, nil
}

func (r *CaseFiles) ListEvents(ctx context.Context, caseFileID uuid.UUID) ([]*domain.CaseEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+caseEventColumns+` FROM case_events
		WHERE case_file_id = $1 ORDER BY occurred_at DESC, created_at DESC`, caseFileID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	events := []*domain.CaseEvent{}
	for rows.Next() {
		var e domain.CaseEvent
		if err := rows.Scan(&e.ID, &e.CaseFileID, &e.EventType, &e.Direction, &e.OccurredAt,
			&e.Summary, &e.Body, &e.DocumentID, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		events = append(events, &e)
	}
	return events, mapErr(rows.Err())
}
