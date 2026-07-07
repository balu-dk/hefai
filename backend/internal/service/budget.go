package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type BudgetRepo interface {
	CreateItem(ctx context.Context, b *domain.BudgetItem) (*domain.BudgetItem, error)
	GetItem(ctx context.Context, id uuid.UUID) (*domain.BudgetItem, error)
	ListItemsByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.BudgetItem, error)
	UpdateItem(ctx context.Context, b *domain.BudgetItem) (*domain.BudgetItem, error)
	DeleteItem(ctx context.Context, id uuid.UUID) error
	CreateExpense(ctx context.Context, e *domain.Expense) (*domain.Expense, error)
	GetExpense(ctx context.Context, id uuid.UUID) (*domain.Expense, error)
	ListExpensesByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Expense, error)
	UpdateExpense(ctx context.Context, e *domain.Expense) (*domain.Expense, error)
	DeleteExpense(ctx context.Context, id uuid.UUID) error
	Summary(ctx context.Context, projectID uuid.UUID) (*domain.BudgetSummary, error)
}

type Budget struct {
	repo   BudgetRepo
	phases PhaseRepo
	access ProjectAccess
}

func NewBudget(repo BudgetRepo, phases PhaseRepo, access ProjectAccess) *Budget {
	return &Budget{repo: repo, phases: phases, access: access}
}

type BudgetItemPatch struct {
	PhaseID            *uuid.UUID `json:"phaseId"`
	Category           *string    `json:"category"`
	Description        *string    `json:"description"`
	EstimatedAmountOre *int64     `json:"estimatedAmountOre"`
}

type ExpensePatch struct {
	BudgetItemID *uuid.UUID `json:"budgetItemId"`
	PhaseID      *uuid.UUID `json:"phaseId"`
	SupplierID   *uuid.UUID `json:"supplierId"`
	Description  *string    `json:"description"`
	AmountOre    *int64     `json:"amountOre"`
	IncurredOn   *time.Time `json:"incurredOn"`
}

func (s *Budget) CreateItem(ctx context.Context, userID, projectID uuid.UUID, patch BudgetItemPatch) (*domain.BudgetItem, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	b := &domain.BudgetItem{ProjectID: projectID, Currency: "DKK"}
	if err := s.applyItemPatch(ctx, b, patch); err != nil {
		return nil, err
	}
	if b.Description == "" {
		return nil, domain.Validation("beskrivelse kræves")
	}
	return s.repo.CreateItem(ctx, b)
}

func (s *Budget) ListItems(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.BudgetItem, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListItemsByProject(ctx, projectID)
}

func (s *Budget) UpdateItem(ctx context.Context, userID, itemID uuid.UUID, patch BudgetItemPatch) (*domain.BudgetItem, error) {
	b, err := s.repo.GetItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, b.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := s.applyItemPatch(ctx, b, patch); err != nil {
		return nil, err
	}
	return s.repo.UpdateItem(ctx, b)
}

func (s *Budget) DeleteItem(ctx context.Context, userID, itemID uuid.UUID) error {
	b, err := s.repo.GetItem(ctx, itemID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, b.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.DeleteItem(ctx, itemID)
}

func (s *Budget) CreateExpense(ctx context.Context, userID, projectID uuid.UUID, patch ExpensePatch) (*domain.Expense, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	e := &domain.Expense{ProjectID: projectID, Currency: "DKK", IncurredOn: time.Now()}
	if err := s.applyExpensePatch(ctx, e, patch); err != nil {
		return nil, err
	}
	if e.Description == "" {
		return nil, domain.Validation("beskrivelse kræves")
	}
	return s.repo.CreateExpense(ctx, e)
}

func (s *Budget) ListExpenses(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.Expense, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListExpensesByProject(ctx, projectID)
}

func (s *Budget) UpdateExpense(ctx context.Context, userID, expenseID uuid.UUID, patch ExpensePatch) (*domain.Expense, error) {
	e, err := s.repo.GetExpense(ctx, expenseID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, e.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := s.applyExpensePatch(ctx, e, patch); err != nil {
		return nil, err
	}
	return s.repo.UpdateExpense(ctx, e)
}

func (s *Budget) DeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error {
	e, err := s.repo.GetExpense(ctx, expenseID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, e.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.DeleteExpense(ctx, expenseID)
}

func (s *Budget) Summary(ctx context.Context, userID, projectID uuid.UUID) (*domain.BudgetSummary, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.Summary(ctx, projectID)
}

func (s *Budget) applyItemPatch(ctx context.Context, b *domain.BudgetItem, patch BudgetItemPatch) error {
	if patch.PhaseID != nil {
		if *patch.PhaseID == uuid.Nil {
			b.PhaseID = nil
		} else {
			phase, err := s.phases.Get(ctx, *patch.PhaseID)
			if err != nil {
				return err
			}
			if phase.ProjectID != b.ProjectID {
				return domain.Validation("fasen tilhører et andet projekt")
			}
			b.PhaseID = patch.PhaseID
		}
	}
	if patch.Category != nil {
		b.Category = strings.TrimSpace(*patch.Category)
	}
	if patch.Description != nil {
		b.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.EstimatedAmountOre != nil {
		if *patch.EstimatedAmountOre < 0 {
			return domain.Validation("beløb kan ikke være negativt")
		}
		b.EstimatedAmountOre = *patch.EstimatedAmountOre
	}
	return nil
}

func (s *Budget) applyExpensePatch(ctx context.Context, e *domain.Expense, patch ExpensePatch) error {
	if patch.BudgetItemID != nil {
		if *patch.BudgetItemID == uuid.Nil {
			e.BudgetItemID = nil
		} else {
			item, err := s.repo.GetItem(ctx, *patch.BudgetItemID)
			if err != nil {
				return err
			}
			if item.ProjectID != e.ProjectID {
				return domain.Validation("budgetposten tilhører et andet projekt")
			}
			e.BudgetItemID = patch.BudgetItemID
		}
	}
	if patch.PhaseID != nil {
		if *patch.PhaseID == uuid.Nil {
			e.PhaseID = nil
		} else {
			phase, err := s.phases.Get(ctx, *patch.PhaseID)
			if err != nil {
				return err
			}
			if phase.ProjectID != e.ProjectID {
				return domain.Validation("fasen tilhører et andet projekt")
			}
			e.PhaseID = patch.PhaseID
		}
	}
	if patch.SupplierID != nil {
		if *patch.SupplierID == uuid.Nil {
			e.SupplierID = nil
		} else {
			e.SupplierID = patch.SupplierID
		}
	}
	if patch.Description != nil {
		e.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.AmountOre != nil {
		e.AmountOre = *patch.AmountOre
	}
	if patch.IncurredOn != nil {
		e.IncurredOn = *patch.IncurredOn
	}
	return nil
}
