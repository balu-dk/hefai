package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Drawings struct {
	db *pgxpool.Pool
}

func NewDrawings(db *pgxpool.Pool) *Drawings { return &Drawings{db: db} }

const drawingColumns = `id, project_id, case_file_id, kind, title, created_by, created_at, updated_at`

func scanDrawing(row pgx.Row) (*domain.Drawing, error) {
	var d domain.Drawing
	err := row.Scan(&d.ID, &d.ProjectID, &d.CaseFileID, &d.Kind, &d.Title,
		&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &d, nil
}

func (r *Drawings) Create(ctx context.Context, d *domain.Drawing) (*domain.Drawing, error) {
	return scanDrawing(r.db.QueryRow(ctx, `
		INSERT INTO drawings (project_id, case_file_id, kind, title, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+drawingColumns,
		d.ProjectID, d.CaseFileID, d.Kind, d.Title, d.CreatedBy))
}

func (r *Drawings) Get(ctx context.Context, id uuid.UUID) (*domain.Drawing, error) {
	return scanDrawing(r.db.QueryRow(ctx,
		`SELECT `+drawingColumns+` FROM drawings WHERE id = $1`, id))
}

func (r *Drawings) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Drawing, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+drawingColumns+` FROM drawings
		WHERE project_id = $1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	drawings := []*domain.Drawing{}
	for rows.Next() {
		d, err := scanDrawing(rows)
		if err != nil {
			return nil, err
		}
		drawings = append(drawings, d)
	}
	return drawings, mapErr(rows.Err())
}

func (r *Drawings) Update(ctx context.Context, d *domain.Drawing) (*domain.Drawing, error) {
	return scanDrawing(r.db.QueryRow(ctx, `
		UPDATE drawings SET case_file_id = $2, kind = $3, title = $4
		WHERE id = $1
		RETURNING `+drawingColumns,
		d.ID, d.CaseFileID, d.Kind, d.Title))
}

func (r *Drawings) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM drawings WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const drawingVersionColumns = `id, drawing_id, version_no, data, scale, note, created_by, created_at`

func scanDrawingVersion(row pgx.Row) (*domain.DrawingVersion, error) {
	var v domain.DrawingVersion
	var data []byte
	err := row.Scan(&v.ID, &v.DrawingID, &v.VersionNo, &data, &v.Scale, &v.Note,
		&v.CreatedBy, &v.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	if err := json.Unmarshal(data, &v.Data); err != nil {
		return nil, err
	}
	return &v, nil
}

// CreateVersion assigns the next version number atomically.
func (r *Drawings) CreateVersion(ctx context.Context, v *domain.DrawingVersion) (*domain.DrawingVersion, error) {
	data, err := json.Marshal(v.Data)
	if err != nil {
		return nil, err
	}
	return scanDrawingVersion(r.db.QueryRow(ctx, `
		INSERT INTO drawing_versions (drawing_id, version_no, data, scale, note, created_by)
		VALUES ($1,
			(SELECT COALESCE(max(version_no), 0) + 1 FROM drawing_versions WHERE drawing_id = $1),
			$2, $3, $4, $5)
		RETURNING `+drawingVersionColumns,
		v.DrawingID, data, v.Scale, v.Note, v.CreatedBy))
}

func (r *Drawings) ListVersions(ctx context.Context, drawingID uuid.UUID) ([]*domain.DrawingVersion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+drawingVersionColumns+` FROM drawing_versions
		WHERE drawing_id = $1 ORDER BY version_no DESC`, drawingID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	versions := []*domain.DrawingVersion{}
	for rows.Next() {
		v, err := scanDrawingVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, mapErr(rows.Err())
}

// LatestVersion returns the newest version, or ErrNotFound if none exist.
func (r *Drawings) LatestVersion(ctx context.Context, drawingID uuid.UUID) (*domain.DrawingVersion, error) {
	return scanDrawingVersion(r.db.QueryRow(ctx, `
		SELECT `+drawingVersionColumns+` FROM drawing_versions
		WHERE drawing_id = $1 ORDER BY version_no DESC LIMIT 1`, drawingID))
}
