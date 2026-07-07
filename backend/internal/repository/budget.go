package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// Budget persists budget items and expenses. Amounts are NUMERIC(12,2)
// kroner in the database and integer øre in Go; the conversion happens in
// SQL ((amount*100)::bigint out, $n::numeric/100 in) so no floats are
// involved.
type Budget struct {
	db *pgxpool.Pool
}

func NewBudget(db *pgxpool.Pool) *Budget { return &Budget{db: db} }

const budgetItemColumns = `id, project_id, phase_id, category, description,
	(estimated_amount*100)::bigint, currency, created_at, updated_at`

func scanBudgetItem(row pgx.Row) (*domain.BudgetItem, error) {
	var b domain.BudgetItem
	err := row.Scan(&b.ID, &b.ProjectID, &b.PhaseID, &b.Category, &b.Description,
		&b.EstimatedAmountOre, &b.Currency, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &b, nil
}

func (r *Budget) CreateItem(ctx context.Context, b *domain.BudgetItem) (*domain.BudgetItem, error) {
	return scanBudgetItem(r.db.QueryRow(ctx, `
		INSERT INTO budget_items (project_id, phase_id, category, description, estimated_amount, currency)
		VALUES ($1, $2, $3, $4, $5::bigint::numeric/100, $6)
		RETURNING `+budgetItemColumns,
		b.ProjectID, b.PhaseID, b.Category, b.Description, b.EstimatedAmountOre, b.Currency))
}

func (r *Budget) GetItem(ctx context.Context, id uuid.UUID) (*domain.BudgetItem, error) {
	return scanBudgetItem(r.db.QueryRow(ctx,
		`SELECT `+budgetItemColumns+` FROM budget_items WHERE id = $1`, id))
}

func (r *Budget) ListItemsByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.BudgetItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+budgetItemColumns+` FROM budget_items
		WHERE project_id = $1 ORDER BY category, created_at`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	items := []*domain.BudgetItem{}
	for rows.Next() {
		b, err := scanBudgetItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, mapErr(rows.Err())
}

func (r *Budget) UpdateItem(ctx context.Context, b *domain.BudgetItem) (*domain.BudgetItem, error) {
	return scanBudgetItem(r.db.QueryRow(ctx, `
		UPDATE budget_items SET phase_id = $2, category = $3, description = $4,
			estimated_amount = $5::bigint::numeric/100, currency = $6
		WHERE id = $1
		RETURNING `+budgetItemColumns,
		b.ID, b.PhaseID, b.Category, b.Description, b.EstimatedAmountOre, b.Currency))
}

func (r *Budget) DeleteItem(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM budget_items WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const expenseColumns = `id, project_id, budget_item_id, phase_id, supplier_id, description,
	(amount*100)::bigint, currency, incurred_on, created_at, updated_at`

func scanExpense(row pgx.Row) (*domain.Expense, error) {
	var e domain.Expense
	err := row.Scan(&e.ID, &e.ProjectID, &e.BudgetItemID, &e.PhaseID, &e.SupplierID,
		&e.Description, &e.AmountOre, &e.Currency, &e.IncurredOn, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &e, nil
}

func (r *Budget) CreateExpense(ctx context.Context, e *domain.Expense) (*domain.Expense, error) {
	return scanExpense(r.db.QueryRow(ctx, `
		INSERT INTO expenses (project_id, budget_item_id, phase_id, supplier_id, description,
			amount, currency, incurred_on)
		VALUES ($1, $2, $3, $4, $5, $6::bigint::numeric/100, $7, $8)
		RETURNING `+expenseColumns,
		e.ProjectID, e.BudgetItemID, e.PhaseID, e.SupplierID, e.Description,
		e.AmountOre, e.Currency, e.IncurredOn))
}

func (r *Budget) GetExpense(ctx context.Context, id uuid.UUID) (*domain.Expense, error) {
	return scanExpense(r.db.QueryRow(ctx,
		`SELECT `+expenseColumns+` FROM expenses WHERE id = $1`, id))
}

func (r *Budget) ListExpensesByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Expense, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+expenseColumns+` FROM expenses
		WHERE project_id = $1 ORDER BY incurred_on DESC, created_at DESC`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	expenses := []*domain.Expense{}
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}
	return expenses, mapErr(rows.Err())
}

func (r *Budget) UpdateExpense(ctx context.Context, e *domain.Expense) (*domain.Expense, error) {
	return scanExpense(r.db.QueryRow(ctx, `
		UPDATE expenses SET budget_item_id = $2, phase_id = $3, supplier_id = $4,
			description = $5, amount = $6::bigint::numeric/100, currency = $7, incurred_on = $8
		WHERE id = $1
		RETURNING `+expenseColumns,
		e.ID, e.BudgetItemID, e.PhaseID, e.SupplierID, e.Description,
		e.AmountOre, e.Currency, e.IncurredOn))
}

func (r *Budget) DeleteExpense(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM expenses WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Summary aggregates estimated vs. spent, total and per phase/category.
// Phase grouping keys by the phase row (unassigned lands in the '' group);
// category grouping keys by the free-text category.
func (r *Budget) Summary(ctx context.Context, projectID uuid.UUID) (*domain.BudgetSummary, error) {
	s := &domain.BudgetSummary{ByPhase: []domain.BudgetGroupTotal{}, ByCategory: []domain.BudgetGroupTotal{}}

	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT sum(estimated_amount*100)::bigint FROM budget_items WHERE project_id = $1), 0),
			COALESCE((SELECT sum(amount*100)::bigint FROM expenses WHERE project_id = $1), 0)`,
		projectID).Scan(&s.EstimatedOre, &s.SpentOre)
	if err != nil {
		return nil, mapErr(err)
	}
	s.RemainingOre = s.EstimatedOre - s.SpentOre

	// FULL JOIN requires an equality condition, so NULL phase ids are mapped
	// to a sentinel UUID for the join and mapped back with NULLIF.
	rows, err := r.db.Query(ctx, `
		WITH est AS (
			SELECT COALESCE(phase_id, '00000000-0000-0000-0000-000000000000'::uuid) AS pid,
			       sum(estimated_amount*100)::bigint AS estimated
			FROM budget_items WHERE project_id = $1 GROUP BY 1
		), spent AS (
			SELECT COALESCE(phase_id, '00000000-0000-0000-0000-000000000000'::uuid) AS pid,
			       sum(amount*100)::bigint AS spent
			FROM expenses WHERE project_id = $1 GROUP BY 1
		)
		SELECT COALESCE(p.name, 'Uden fase'),
		       NULLIF(COALESCE(est.pid, spent.pid), '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(est.estimated, 0), COALESCE(spent.spent, 0)
		FROM est
		FULL OUTER JOIN spent ON est.pid = spent.pid
		LEFT JOIN phases p ON p.id = COALESCE(est.pid, spent.pid)
		ORDER BY p.sort_order NULLS LAST`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	for rows.Next() {
		var g domain.BudgetGroupTotal
		if err := rows.Scan(&g.Key, &g.PhaseID, &g.EstimatedOre, &g.SpentOre); err != nil {
			return nil, mapErr(err)
		}
		g.RemainingOre = g.EstimatedOre - g.SpentOre
		s.ByPhase = append(s.ByPhase, g)
	}
	if err := rows.Err(); err != nil {
		return nil, mapErr(err)
	}

	catRows, err := r.db.Query(ctx, `
		WITH est AS (
			SELECT category, sum(estimated_amount*100)::bigint AS estimated
			FROM budget_items WHERE project_id = $1 GROUP BY category
		), spent AS (
			SELECT COALESCE(b.category, '') AS category, sum(e.amount*100)::bigint AS spent
			FROM expenses e
			LEFT JOIN budget_items b ON b.id = e.budget_item_id
			WHERE e.project_id = $1 GROUP BY COALESCE(b.category, '')
		)
		SELECT COALESCE(NULLIF(COALESCE(est.category, spent.category), ''), 'Uden kategori'),
		       COALESCE(est.estimated, 0), COALESCE(spent.spent, 0)
		FROM est
		FULL OUTER JOIN spent ON est.category = spent.category
		ORDER BY 1`, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer catRows.Close()
	for catRows.Next() {
		var g domain.BudgetGroupTotal
		if err := catRows.Scan(&g.Key, &g.EstimatedOre, &g.SpentOre); err != nil {
			return nil, mapErr(err)
		}
		g.RemainingOre = g.EstimatedOre - g.SpentOre
		s.ByCategory = append(s.ByCategory, g)
	}
	return s, mapErr(catRows.Err())
}
