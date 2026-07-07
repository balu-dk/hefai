package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type Packages struct {
	db *pgxpool.Pool
}

func NewPackages(db *pgxpool.Pool) *Packages { return &Packages{db: db} }

const packageColumns = `id, project_id, version_no, title, snapshot, document_id, status,
	sent_at, created_at`

func scanPackage(row pgx.Row) (*domain.StructuralPackage, error) {
	var p domain.StructuralPackage
	var snapshot []byte
	err := row.Scan(&p.ID, &p.ProjectID, &p.VersionNo, &p.Title, &snapshot,
		&p.DocumentID, &p.Status, &p.SentAt, &p.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	p.Snapshot = json.RawMessage(snapshot)
	return &p, nil
}

func (r *Packages) Create(ctx context.Context, p *domain.StructuralPackage) (*domain.StructuralPackage, error) {
	return scanPackage(r.db.QueryRow(ctx, `
		INSERT INTO structural_packages (project_id, version_no, title, snapshot, document_id, status)
		VALUES ($1,
			(SELECT COALESCE(max(version_no), 0) + 1 FROM structural_packages WHERE project_id = $1),
			$2, $3, $4, $5)
		RETURNING `+packageColumns,
		p.ProjectID, p.Title, []byte(p.Snapshot), p.DocumentID, p.Status))
}

func (r *Packages) Get(ctx context.Context, id uuid.UUID) (*domain.StructuralPackage, error) {
	return scanPackage(r.db.QueryRow(ctx,
		`SELECT `+packageColumns+` FROM structural_packages WHERE id = $1`, id))
}

func (r *Packages) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.StructuralPackage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+packageColumns+` FROM structural_packages
		WHERE project_id = $1 ORDER BY version_no DESC`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	packages := []*domain.StructuralPackage{}
	for rows.Next() {
		p, err := scanPackage(rows)
		if err != nil {
			return nil, err
		}
		packages = append(packages, p)
	}
	return packages, mapErr(rows.Err())
}

func (r *Packages) SetStatus(ctx context.Context, id uuid.UUID, status domain.PackageStatus, sentAt bool) error {
	var err error
	if sentAt {
		_, err = r.db.Exec(ctx,
			`UPDATE structural_packages SET status = $2, sent_at = now() WHERE id = $1`, id, status)
	} else {
		_, err = r.db.Exec(ctx,
			`UPDATE structural_packages SET status = $2 WHERE id = $1`, id, status)
	}
	return mapErr(err)
}

// --- engineer reviews ---------------------------------------------------------

const reviewColumns = `id, structural_package_id, reviewer_name, reviewer_company,
	reviewer_credentials, received_at, overall_status, summary, response_document_id,
	created_at, updated_at`

func scanReview(row pgx.Row) (*domain.EngineerReview, error) {
	var rv domain.EngineerReview
	err := row.Scan(&rv.ID, &rv.StructuralPackageID, &rv.ReviewerName, &rv.ReviewerCompany,
		&rv.ReviewerCredentials, &rv.ReceivedAt, &rv.OverallStatus, &rv.Summary,
		&rv.ResponseDocumentID, &rv.CreatedAt, &rv.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &rv, nil
}

func (r *Packages) CreateReview(ctx context.Context, rv *domain.EngineerReview) (*domain.EngineerReview, error) {
	created, err := scanReview(r.db.QueryRow(ctx, `
		INSERT INTO engineer_reviews (structural_package_id, reviewer_name, reviewer_company,
			reviewer_credentials, received_at, overall_status, summary, response_document_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+reviewColumns,
		rv.StructuralPackageID, rv.ReviewerName, rv.ReviewerCompany, rv.ReviewerCredentials,
		rv.ReceivedAt, rv.OverallStatus, rv.Summary, rv.ResponseDocumentID))
	if err != nil {
		return nil, err
	}
	for _, item := range rv.Items {
		item.EngineerReviewID = created.ID
		var id uuid.UUID
		err := r.db.QueryRow(ctx, `
			INSERT INTO engineer_review_items (engineer_review_id, structural_element_id,
				load_id, calculation_estimate_id, drawing_id, verdict, comment, corrected_values)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`,
			created.ID, item.StructuralElementID, item.LoadID, item.CalculationEstimateID,
			item.DrawingID, item.Verdict, item.Comment, orEmptyJSON(item.CorrectedValues)).Scan(&id)
		if err != nil {
			return nil, mapErr(err)
		}
		item.ID = id
	}
	created.Items = rv.Items
	return created, nil
}

func (r *Packages) ListReviews(ctx context.Context, packageID uuid.UUID) ([]*domain.EngineerReview, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+reviewColumns+` FROM engineer_reviews
		WHERE structural_package_id = $1 ORDER BY received_at DESC`, packageID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	reviews := []*domain.EngineerReview{}
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, rv)
	}
	if err := rows.Err(); err != nil {
		return nil, mapErr(err)
	}

	for _, rv := range reviews {
		items, err := r.listReviewItems(ctx, rv.ID)
		if err != nil {
			return nil, err
		}
		rv.Items = items
	}
	return reviews, nil
}

func (r *Packages) listReviewItems(ctx context.Context, reviewID uuid.UUID) ([]*domain.EngineerReviewItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, engineer_review_id, structural_element_id, load_id, calculation_estimate_id,
		       drawing_id, verdict, comment, corrected_values, created_at
		FROM engineer_review_items
		WHERE engineer_review_id = $1 ORDER BY created_at`, reviewID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	items := []*domain.EngineerReviewItem{}
	for rows.Next() {
		var item domain.EngineerReviewItem
		var corrected []byte
		if err := rows.Scan(&item.ID, &item.EngineerReviewID, &item.StructuralElementID,
			&item.LoadID, &item.CalculationEstimateID, &item.DrawingID, &item.Verdict,
			&item.Comment, &corrected, &item.CreatedAt); err != nil {
			return nil, mapErr(err)
		}
		item.CorrectedValues = json.RawMessage(corrected)
		items = append(items, &item)
	}
	return items, mapErr(rows.Err())
}

func orEmptyJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte(`{}`)
	}
	return []byte(raw)
}
