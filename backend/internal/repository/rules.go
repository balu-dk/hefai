package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type ComplianceRules struct {
	db *pgxpool.Pool
}

func NewComplianceRules(db *pgxpool.Pool) *ComplianceRules { return &ComplianceRules{db: db} }

const ruleColumns = `r.id, r.project_id, r.parameter, r.value, r.source_chunk_id,
	COALESCE(c.section_ref, ''), r.quote, r.status, r.note, r.created_at, r.updated_at`

const ruleFrom = ` FROM compliance_rules r
	LEFT JOIN source_chunks c ON c.id = r.source_chunk_id`

func scanRule(row pgx.Row) (*domain.ComplianceRule, error) {
	var r domain.ComplianceRule
	err := row.Scan(&r.ID, &r.ProjectID, &r.Parameter, &r.Value, &r.SourceChunkID,
		&r.SourceRef, &r.Quote, &r.Status, &r.Note, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &r, nil
}

// Upsert opretter reglen eller opdaterer en eksisterende for samme
// parameter — men rører aldrig en regel brugeren allerede har bekræftet.
func (r *ComplianceRules) Upsert(ctx context.Context, rule *domain.ComplianceRule) (*domain.ComplianceRule, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO compliance_rules (project_id, parameter, value, source_chunk_id, quote, status, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (project_id, parameter) DO UPDATE
		SET value = EXCLUDED.value, source_chunk_id = EXCLUDED.source_chunk_id,
		    quote = EXCLUDED.quote, status = EXCLUDED.status, note = EXCLUDED.note
		WHERE compliance_rules.status <> 'confirmed'
		RETURNING id`,
		rule.ProjectID, rule.Parameter, rule.Value, rule.SourceChunkID,
		rule.Quote, rule.Status, rule.Note).Scan(&id)
	if err != nil {
		if mapErr(err) == domain.ErrNotFound {
			// Konflikt med bekræftet regel: behold den og returnér den.
			return r.GetByParameter(ctx, rule.ProjectID, rule.Parameter)
		}
		return nil, mapErr(err)
	}
	return r.Get(ctx, id)
}

func (r *ComplianceRules) Get(ctx context.Context, id uuid.UUID) (*domain.ComplianceRule, error) {
	return scanRule(r.db.QueryRow(ctx, `SELECT `+ruleColumns+ruleFrom+` WHERE r.id = $1`, id))
}

func (r *ComplianceRules) GetByParameter(ctx context.Context, projectID uuid.UUID, parameter string) (*domain.ComplianceRule, error) {
	return scanRule(r.db.QueryRow(ctx,
		`SELECT `+ruleColumns+ruleFrom+` WHERE r.project_id = $1 AND r.parameter = $2`,
		projectID, parameter))
}

func (r *ComplianceRules) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.ComplianceRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+ruleColumns+ruleFrom+`
		WHERE r.project_id = $1 ORDER BY r.parameter`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	rules := []*domain.ComplianceRule{}
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, mapErr(rows.Err())
}

func (r *ComplianceRules) Update(ctx context.Context, rule *domain.ComplianceRule) (*domain.ComplianceRule, error) {
	_, err := r.db.Exec(ctx, `
		UPDATE compliance_rules SET value = $2, source_chunk_id = $3, quote = $4,
			status = $5, note = $6
		WHERE id = $1`,
		rule.ID, rule.Value, rule.SourceChunkID, rule.Quote, rule.Status, rule.Note)
	if err != nil {
		return nil, mapErr(err)
	}
	return r.Get(ctx, rule.ID)
}

func (r *ComplianceRules) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM compliance_rules WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
