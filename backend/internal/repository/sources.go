package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
	"github.com/balu-dk/hefai/backend/internal/rag"
)

type Sources struct {
	db *pgxpool.Pool
}

func NewSources(db *pgxpool.Pool) *Sources { return &Sources{db: db} }

const sourceColumns = `s.id, s.project_id, s.document_id, s.title, s.kind, s.version_label,
	s.url, s.status, s.added_by, s.created_at, s.updated_at,
	(SELECT count(*) FROM source_chunks c WHERE c.source_document_id = s.id)`

func scanSource(row pgx.Row) (*domain.SourceDocument, error) {
	var s domain.SourceDocument
	err := row.Scan(&s.ID, &s.ProjectID, &s.DocumentID, &s.Title, &s.Kind, &s.VersionLabel,
		&s.URL, &s.Status, &s.AddedBy, &s.CreatedAt, &s.UpdatedAt, &s.ChunkCount)
	if err != nil {
		return nil, mapErr(err)
	}
	return &s, nil
}

// CreateWithChunks stores the source document and its chunks in one
// transaction and marks it ready.
func (r *Sources) CreateWithChunks(ctx context.Context, s *domain.SourceDocument, chunks []rag.Chunk) (*domain.SourceDocument, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO source_documents (project_id, document_id, title, kind, version_label, url, status, added_by)
		VALUES ($1, $2, $3, $4, $5, $6, 'ready', $7)
		RETURNING id`,
		s.ProjectID, s.DocumentID, s.Title, s.Kind, s.VersionLabel, s.URL, s.AddedBy).Scan(&id)
	if err != nil {
		return nil, mapErr(err)
	}

	for _, c := range chunks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO source_chunks (source_document_id, chunk_index, content, section_ref)
			VALUES ($1, $2, $3, $4)`,
			id, c.Index, c.Content, c.SectionRef); err != nil {
			return nil, mapErr(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *Sources) Get(ctx context.Context, id uuid.UUID) (*domain.SourceDocument, error) {
	return scanSource(r.db.QueryRow(ctx,
		`SELECT `+sourceColumns+` FROM source_documents s WHERE s.id = $1`, id))
}

// ListForProject returns the project's sources plus the shared library
// (project_id IS NULL).
func (r *Sources) ListForProject(ctx context.Context, projectID uuid.UUID) ([]*domain.SourceDocument, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+sourceColumns+` FROM source_documents s
		WHERE s.project_id = $1 OR s.project_id IS NULL
		ORDER BY s.created_at`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	sources := []*domain.SourceDocument{}
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, mapErr(rows.Err())
}

func (r *Sources) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM source_documents WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Search runs Danish full-text search over the project's and shared chunks.
// Falls back to substring match when websearch parsing yields nothing
// (e.g. queries that are only stop words).
func (r *Sources) Search(ctx context.Context, projectID uuid.UUID, query string, limit int) ([]*domain.SourceHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 8
	}
	// Terms are OR'ed for recall; ts_rank still rewards chunks matching
	// more of them. Exact substring works as a fallback for short queries.
	orQuery := strings.Join(strings.Fields(query), " OR ")
	rows, err := r.db.Query(ctx, `
		WITH q AS (SELECT websearch_to_tsquery('danish', $2) AS tsq)
		SELECT c.id, s.id, s.title, s.kind, c.section_ref, c.content,
		       ts_rank(to_tsvector('danish', c.content), q.tsq) AS rank
		FROM source_chunks c
		JOIN source_documents s ON s.id = c.source_document_id
		CROSS JOIN q
		WHERE (s.project_id = $1 OR s.project_id IS NULL)
		  AND s.status = 'ready'
		  AND (to_tsvector('danish', c.content) @@ q.tsq OR c.content ILIKE '%' || $3 || '%')
		ORDER BY rank DESC, c.chunk_index
		LIMIT $4`, projectID, orQuery, query, limit)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	hits := []*domain.SourceHit{}
	for rows.Next() {
		var h domain.SourceHit
		if err := rows.Scan(&h.ChunkID, &h.SourceID, &h.SourceTitle, &h.SourceKind,
			&h.SectionRef, &h.Content, &h.Rank); err != nil {
			return nil, mapErr(err)
		}
		hits = append(hits, &h)
	}
	return hits, mapErr(rows.Err())
}

// ChunkProject resolves which project a chunk's source belongs to (nil for
// the shared library), for grounding validation.
func (r *Sources) ChunkProject(ctx context.Context, chunkID uuid.UUID) (*uuid.UUID, string, error) {
	var projectID *uuid.UUID
	var sectionRef string
	err := r.db.QueryRow(ctx, `
		SELECT s.project_id, c.section_ref
		FROM source_chunks c
		JOIN source_documents s ON s.id = c.source_document_id
		WHERE c.id = $1`, chunkID).Scan(&projectID, &sectionRef)
	if err != nil {
		return nil, "", mapErr(err)
	}
	return projectID, sectionRef, nil
}
