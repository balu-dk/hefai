package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Generated struct {
	db *pgxpool.Pool
}

func NewGenerated(db *pgxpool.Pool) *Generated { return &Generated{db: db} }

const generatedColumns = `id, project_id, case_file_id, kind, status, version_no,
	input_snapshot, document_id, created_at`

func scanGenerated(row pgx.Row) (*domain.GeneratedDocument, error) {
	var g domain.GeneratedDocument
	var snapshot []byte
	err := row.Scan(&g.ID, &g.ProjectID, &g.CaseFileID, &g.Kind, &g.Status, &g.VersionNo,
		&snapshot, &g.DocumentID, &g.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	g.InputSnapshot = json.RawMessage(snapshot)
	return &g, nil
}

// Create assigns the next version number for this case file + kind.
func (r *Generated) Create(ctx context.Context, g *domain.GeneratedDocument) (*domain.GeneratedDocument, error) {
	return scanGenerated(r.db.QueryRow(ctx, `
		INSERT INTO generated_documents (project_id, case_file_id, kind, status, version_no,
			input_snapshot, document_id)
		VALUES ($1, $2, $3, $4,
			(SELECT COALESCE(max(version_no), 0) + 1 FROM generated_documents
			 WHERE case_file_id IS NOT DISTINCT FROM $2 AND kind = $3),
			$5, $6)
		RETURNING `+generatedColumns,
		g.ProjectID, g.CaseFileID, g.Kind, g.Status, []byte(g.InputSnapshot), g.DocumentID))
}

func (r *Generated) ListByCaseFile(ctx context.Context, caseFileID uuid.UUID) ([]*domain.GeneratedDocument, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+generatedColumns+` FROM generated_documents
		WHERE case_file_id = $1 ORDER BY kind, version_no DESC`, caseFileID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	docs := []*domain.GeneratedDocument{}
	for rows.Next() {
		g, err := scanGenerated(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, g)
	}
	return docs, mapErr(rows.Err())
}
