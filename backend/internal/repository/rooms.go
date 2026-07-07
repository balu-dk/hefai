package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Rooms struct {
	db *pgxpool.Pool
}

func NewRooms(db *pgxpool.Pool) *Rooms { return &Rooms{db: db} }

const roomColumns = `id, project_id, name, kind, description, area_m2, created_at, updated_at`

func scanRoom(row pgx.Row) (*domain.Room, error) {
	var m domain.Room
	err := row.Scan(&m.ID, &m.ProjectID, &m.Name, &m.Kind, &m.Description, &m.AreaM2,
		&m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

func (r *Rooms) Create(ctx context.Context, m *domain.Room) (*domain.Room, error) {
	return scanRoom(r.db.QueryRow(ctx, `
		INSERT INTO rooms (project_id, name, kind, description, area_m2)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+roomColumns,
		m.ProjectID, m.Name, m.Kind, m.Description, m.AreaM2))
}

func (r *Rooms) Get(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	return scanRoom(r.db.QueryRow(ctx, `SELECT `+roomColumns+` FROM rooms WHERE id = $1`, id))
}

func (r *Rooms) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Room, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+roomColumns+` FROM rooms WHERE project_id = $1 ORDER BY name`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	rooms := []*domain.Room{}
	for rows.Next() {
		m, err := scanRoom(rows)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, m)
	}
	return rooms, mapErr(rows.Err())
}

func (r *Rooms) Update(ctx context.Context, m *domain.Room) (*domain.Room, error) {
	return scanRoom(r.db.QueryRow(ctx, `
		UPDATE rooms SET name = $2, kind = $3, description = $4, area_m2 = $5
		WHERE id = $1
		RETURNING `+roomColumns,
		m.ID, m.Name, m.Kind, m.Description, m.AreaM2))
}

func (r *Rooms) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM rooms WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
