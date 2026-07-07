package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Projects struct {
	db *pgxpool.Pool
}

func NewProjects(db *pgxpool.Pool) *Projects { return &Projects{db: db} }

const projectColumns = `id, name, description, kind, status, address, municipality,
	cadastral_id, plot_area_m2, latitude, longitude, utm_x, utm_y, created_by, created_at, updated_at`

// projectColumnsQualified is for queries joining other tables.
const projectColumnsQualified = `p.id, p.name, p.description, p.kind, p.status, p.address,
	p.municipality, p.cadastral_id, p.plot_area_m2, p.latitude, p.longitude, p.utm_x, p.utm_y,
	p.created_by, p.created_at, p.updated_at`

func scanProject(row pgx.Row) (*domain.Project, error) {
	var p domain.Project
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Kind, &p.Status, &p.Address,
		&p.Municipality, &p.CadastralID, &p.PlotAreaM2, &p.Latitude, &p.Longitude,
		&p.UTMX, &p.UTMY, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

// Create inserts the project, its owner membership, and the default phases in
// one transaction.
func (r *Projects) Create(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	created, err := scanProject(tx.QueryRow(ctx, `
		INSERT INTO projects (name, description, kind, status, address, municipality,
			cadastral_id, plot_area_m2, latitude, longitude, utm_x, utm_y, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING `+projectColumns,
		p.Name, p.Description, p.Kind, p.Status, p.Address, p.Municipality,
		p.CadastralID, p.PlotAreaM2, p.Latitude, p.Longitude, p.UTMX, p.UTMY, p.CreatedBy))
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, 'owner')`, created.ID, p.CreatedBy); err != nil {
		return nil, mapErr(err)
	}

	for i, name := range domain.DefaultPhaseNames {
		if _, err := tx.Exec(ctx, `
			INSERT INTO phases (project_id, name, sort_order)
			VALUES ($1, $2, $3)`, created.ID, name, i); err != nil {
			return nil, mapErr(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *Projects) Get(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return scanProject(r.db.QueryRow(ctx,
		`SELECT `+projectColumns+` FROM projects WHERE id = $1`, id))
}

func (r *Projects) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+projectColumnsQualified+` FROM projects p
		JOIN project_members m ON m.project_id = p.id
		WHERE m.user_id = $1
		ORDER BY p.created_at`, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	projects := []*domain.Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, mapErr(rows.Err())
}

func (r *Projects) Update(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	return scanProject(r.db.QueryRow(ctx, `
		UPDATE projects SET name = $2, description = $3, kind = $4, status = $5,
			address = $6, municipality = $7, cadastral_id = $8, plot_area_m2 = $9,
			latitude = $10, longitude = $11, utm_x = $12, utm_y = $13
		WHERE id = $1
		RETURNING `+projectColumns,
		p.ID, p.Name, p.Description, p.Kind, p.Status, p.Address,
		p.Municipality, p.CadastralID, p.PlotAreaM2, p.Latitude, p.Longitude,
		p.UTMX, p.UTMY))
}

func (r *Projects) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetMemberRole returns the caller's role on a project, or ErrForbidden if
// they are not a member.
func (r *Projects) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (domain.ProjectRole, error) {
	var role domain.ProjectRole
	err := r.db.QueryRow(ctx, `
		SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID).Scan(&role)
	if err != nil {
		if mapErr(err) == domain.ErrNotFound {
			return "", domain.ErrForbidden
		}
		return "", mapErr(err)
	}
	return role, nil
}

func (r *Projects) ListMembers(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.project_id, m.user_id, m.role, u.email, u.display_name, m.created_at
		FROM project_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.project_id = $1
		ORDER BY m.created_at`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	members := []*domain.ProjectMember{}
	for rows.Next() {
		var m domain.ProjectMember
		if err := rows.Scan(&m.ProjectID, &m.UserID, &m.Role, &m.Email, &m.DisplayName, &m.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		members = append(members, &m)
	}
	return members, mapErr(rows.Err())
}

func (r *Projects) AddMember(ctx context.Context, projectID, userID uuid.UUID, role domain.ProjectRole) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, $3)`,
		projectID, userID, role)
	return mapErr(err)
}

func (r *Projects) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
