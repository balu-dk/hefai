package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Documents struct {
	db *pgxpool.Pool
}

func NewDocuments(db *pgxpool.Pool) *Documents { return &Documents{db: db} }

// Tags are aggregated into the document row so listings need no N+1 queries.
const documentSelect = `
	SELECT d.id, d.project_id, d.uploaded_by, d.kind, d.title, d.description,
	       d.filename, d.storage_key, d.mime_type, d.size_bytes, d.captured_at,
	       COALESCE(array_agg(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL), '{}'),
	       d.created_at, d.updated_at
	FROM documents d
	LEFT JOIN document_tags dt ON dt.document_id = d.id
	LEFT JOIN tags t ON t.id = dt.tag_id`

func scanDocument(row pgx.Row) (*domain.Document, error) {
	var d domain.Document
	err := row.Scan(&d.ID, &d.ProjectID, &d.UploadedBy, &d.Kind, &d.Title, &d.Description,
		&d.Filename, &d.StorageKey, &d.MimeType, &d.SizeBytes, &d.CapturedAt,
		&d.Tags, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &d, nil
}

func (r *Documents) Create(ctx context.Context, d *domain.Document) (*domain.Document, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO documents (project_id, uploaded_by, kind, title, description,
			filename, storage_key, mime_type, size_bytes, captured_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		d.ProjectID, d.UploadedBy, d.Kind, d.Title, d.Description,
		d.Filename, d.StorageKey, d.MimeType, d.SizeBytes, d.CapturedAt).Scan(&id)
	if err != nil {
		return nil, mapErr(err)
	}
	return r.Get(ctx, id)
}

func (r *Documents) Get(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	return scanDocument(r.db.QueryRow(ctx,
		documentSelect+` WHERE d.id = $1 GROUP BY d.id`, id))
}

// List returns project documents narrowed by the filter.
func (r *Documents) List(ctx context.Context, projectID uuid.UUID, f domain.DocumentFilter) ([]*domain.Document, error) {
	query := documentSelect + ` WHERE d.project_id = $1`
	args := []any{projectID}

	if f.Kind != "" {
		args = append(args, f.Kind)
		query += fmt.Sprintf(` AND d.kind = $%d`, len(args))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		query += fmt.Sprintf(` AND (d.title ILIKE $%d OR d.description ILIKE $%d OR d.filename ILIKE $%d)`, n, n, n)
	}
	if f.Tag != "" {
		args = append(args, f.Tag)
		query += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM document_tags dt2
			JOIN tags t2 ON t2.id = dt2.tag_id
			WHERE dt2.document_id = d.id AND t2.name = $%d)`, len(args))
	}
	if f.TargetType != "" && f.TargetID != uuid.Nil {
		col, err := linkColumn(f.TargetType)
		if err != nil {
			return nil, err
		}
		args = append(args, f.TargetID)
		query += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM document_links dl
			WHERE dl.document_id = d.id AND dl.%s = $%d)`, col, len(args))
	}

	query += ` GROUP BY d.id ORDER BY d.created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	docs := []*domain.Document{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, mapErr(rows.Err())
}

func (r *Documents) Update(ctx context.Context, d *domain.Document) (*domain.Document, error) {
	_, err := r.db.Exec(ctx, `
		UPDATE documents SET kind = $2, title = $3, description = $4, captured_at = $5
		WHERE id = $1`,
		d.ID, d.Kind, d.Title, d.Description, d.CapturedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return r.Get(ctx, d.ID)
}

func (r *Documents) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// SetTags replaces the document's tags, creating unknown tag names in the
// project's tag set.
func (r *Documents) SetTags(ctx context.Context, docID, projectID uuid.UUID, names []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `DELETE FROM document_tags WHERE document_id = $1`, docID); err != nil {
		return mapErr(err)
	}
	for _, name := range names {
		var tagID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO tags (project_id, name) VALUES ($1, $2)
			ON CONFLICT (project_id, name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, projectID, name).Scan(&tagID)
		if err != nil {
			return mapErr(err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO document_tags (document_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, docID, tagID); err != nil {
			return mapErr(err)
		}
	}
	return tx.Commit(ctx)
}

// linkColumn maps a target type to its FK column in document_links.
func linkColumn(t domain.LinkTargetType) (string, error) {
	switch t {
	case domain.LinkPhase:
		return "phase_id", nil
	case domain.LinkTask:
		return "task_id", nil
	case domain.LinkRoom:
		return "room_id", nil
	case domain.LinkExpense:
		return "expense_id", nil
	case domain.LinkMaterial:
		return "material_id", nil
	case domain.LinkSupplier:
		return "supplier_id", nil
	case domain.LinkCaseFile:
		return "case_file_id", nil
	case domain.LinkStructuralElement:
		return "structural_element_id", nil
	}
	return "", domain.Validation("ugyldig link-type")
}

// targetTable maps a target type to the table holding its project_id.
func targetTable(t domain.LinkTargetType) (string, error) {
	switch t {
	case domain.LinkPhase:
		return "phases", nil
	case domain.LinkTask:
		return "tasks", nil
	case domain.LinkRoom:
		return "rooms", nil
	case domain.LinkExpense:
		return "expenses", nil
	case domain.LinkMaterial:
		return "materials", nil
	case domain.LinkSupplier:
		return "suppliers", nil
	case domain.LinkCaseFile:
		return "case_files", nil
	case domain.LinkStructuralElement:
		return "structural_elements", nil
	}
	return "", domain.Validation("ugyldig link-type")
}

// TargetProjectID resolves which project a link target belongs to, so the
// service can refuse cross-project links.
func (r *Documents) TargetProjectID(ctx context.Context, t domain.LinkTargetType, id uuid.UUID) (uuid.UUID, error) {
	table, err := targetTable(t)
	if err != nil {
		return uuid.Nil, err
	}
	var projectID uuid.UUID
	err = r.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT project_id FROM %s WHERE id = $1`, table), id).Scan(&projectID)
	if err != nil {
		return uuid.Nil, mapErr(err)
	}
	return projectID, nil
}

func (r *Documents) CreateLink(ctx context.Context, docID uuid.UUID, t domain.LinkTargetType, targetID uuid.UUID) (*domain.DocumentLink, error) {
	col, err := linkColumn(t)
	if err != nil {
		return nil, err
	}
	var link domain.DocumentLink
	link.DocumentID = docID
	link.TargetType = t
	link.TargetID = targetID
	err = r.db.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO document_links (document_id, %s) VALUES ($1, $2)
		RETURNING id, created_at`, col),
		docID, targetID).Scan(&link.ID, &link.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &link, nil
}

func (r *Documents) DeleteLink(ctx context.Context, linkID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM document_links WHERE id = $1`, linkID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Documents) GetLink(ctx context.Context, linkID uuid.UUID) (*domain.DocumentLink, error) {
	links, err := r.scanLinks(ctx, `WHERE dl.id = $1`, linkID)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, domain.ErrNotFound
	}
	return links[0], nil
}

func (r *Documents) ListLinks(ctx context.Context, docID uuid.UUID) ([]*domain.DocumentLink, error) {
	return r.scanLinks(ctx, `WHERE dl.document_id = $1 ORDER BY dl.created_at`, docID)
}

func (r *Documents) scanLinks(ctx context.Context, where string, arg any) ([]*domain.DocumentLink, error) {
	rows, err := r.db.Query(ctx, `
		SELECT dl.id, dl.document_id, dl.created_at,
		       dl.phase_id, dl.task_id, dl.room_id, dl.expense_id,
		       dl.material_id, dl.supplier_id, dl.case_file_id, dl.structural_element_id
		FROM document_links dl `+where, arg)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	links := []*domain.DocumentLink{}
	for rows.Next() {
		var l domain.DocumentLink
		targets := make([]*uuid.UUID, 8)
		if err := rows.Scan(&l.ID, &l.DocumentID, &l.CreatedAt,
			&targets[0], &targets[1], &targets[2], &targets[3],
			&targets[4], &targets[5], &targets[6], &targets[7]); err != nil {
			return nil, mapErr(err)
		}
		types := []domain.LinkTargetType{
			domain.LinkPhase, domain.LinkTask, domain.LinkRoom, domain.LinkExpense,
			domain.LinkMaterial, domain.LinkSupplier, domain.LinkCaseFile, domain.LinkStructuralElement,
		}
		for i, target := range targets {
			if target != nil {
				l.TargetType = types[i]
				l.TargetID = *target
				break
			}
		}
		links = append(links, &l)
	}
	return links, mapErr(rows.Err())
}
