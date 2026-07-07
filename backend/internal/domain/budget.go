package domain

import (
	"time"

	"github.com/google/uuid"
)

// Monetary amounts are integer øre throughout Go and the JSON API; the
// repository layer converts to/from NUMERIC(12,2) kroner in SQL. Integer
// arithmetic keeps sums exact.

type BudgetItem struct {
	ID                 uuid.UUID  `json:"id"`
	ProjectID          uuid.UUID  `json:"projectId"`
	PhaseID            *uuid.UUID `json:"phaseId"`
	Category           string     `json:"category"`
	Description        string     `json:"description"`
	EstimatedAmountOre int64      `json:"estimatedAmountOre"`
	Currency           string     `json:"currency"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type Expense struct {
	ID           uuid.UUID  `json:"id"`
	ProjectID    uuid.UUID  `json:"projectId"`
	BudgetItemID *uuid.UUID `json:"budgetItemId"`
	PhaseID      *uuid.UUID `json:"phaseId"`
	SupplierID   *uuid.UUID `json:"supplierId"`
	Description  string     `json:"description"`
	AmountOre    int64      `json:"amountOre"`
	Currency     string     `json:"currency"`
	IncurredOn   time.Time  `json:"incurredOn"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// BudgetSummary answers "estimated vs. spent vs. remaining" — total and per
// phase/category.
type BudgetSummary struct {
	EstimatedOre int64              `json:"estimatedOre"`
	SpentOre     int64              `json:"spentOre"`
	RemainingOre int64              `json:"remainingOre"`
	ByPhase      []BudgetGroupTotal `json:"byPhase"`
	ByCategory   []BudgetGroupTotal `json:"byCategory"`
}

type BudgetGroupTotal struct {
	Key          string     `json:"key"`           // phase name or category
	PhaseID      *uuid.UUID `json:"phaseId"`       // set for phase groups
	EstimatedOre int64      `json:"estimatedOre"`
	SpentOre     int64      `json:"spentOre"`
	RemainingOre int64      `json:"remainingOre"`
}
