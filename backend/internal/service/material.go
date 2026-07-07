package service

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type MaterialRepo interface {
	Create(ctx context.Context, m *domain.Material) (*domain.Material, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Material, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Material, error)
	Update(ctx context.Context, m *domain.Material) (*domain.Material, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Materials struct {
	repo      MaterialRepo
	suppliers SupplierRepo
	access    ProjectAccess
}

func NewMaterials(repo MaterialRepo, suppliers SupplierRepo, access ProjectAccess) *Materials {
	return &Materials{repo: repo, suppliers: suppliers, access: access}
}

type MaterialPatch struct {
	PhaseID      *uuid.UUID `json:"phaseId"`
	TaskID       *uuid.UUID `json:"taskId"`
	RoomID       *uuid.UUID `json:"roomId"`
	SupplierID   *uuid.UUID `json:"supplierId"`
	Name         *string    `json:"name"`
	Spec         *string    `json:"spec"`
	Quantity     *float64   `json:"quantity"`
	Unit         *string    `json:"unit"`
	UnitPriceOre *int64     `json:"unitPriceOre"`
	Status       *string    `json:"status"`
	Notes        *string    `json:"notes"`
}

func (s *Materials) Create(ctx context.Context, userID, projectID uuid.UUID, patch MaterialPatch) (*domain.Material, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	m := &domain.Material{
		ProjectID: projectID,
		Status:    domain.MaterialNeeded,
		Currency:  "DKK",
		Unit:      "stk",
		Quantity:  1,
	}
	if err := s.applyMaterialPatch(ctx, m, patch); err != nil {
		return nil, err
	}
	if m.Name == "" {
		return nil, domain.Validation("materialenavn kræves")
	}
	return s.repo.Create(ctx, m)
}

func (s *Materials) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.Material, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Materials) Update(ctx context.Context, userID, materialID uuid.UUID, patch MaterialPatch) (*domain.Material, error) {
	m, err := s.repo.Get(ctx, materialID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, m.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := s.applyMaterialPatch(ctx, m, patch); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, m)
}

func (s *Materials) Delete(ctx context.Context, userID, materialID uuid.UUID) error {
	m, err := s.repo.Get(ctx, materialID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, m.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, materialID)
}

// ShoppingList groups materials that still need buying (status needed) by
// supplier, with a line total where unit price is known.
func (s *Materials) ShoppingList(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.ShoppingListGroup, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	materials, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	suppliers, err := s.suppliers.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	names := map[uuid.UUID]string{}
	for _, sup := range suppliers {
		names[sup.ID] = sup.CompanyName
	}
	return buildShoppingList(materials, names), nil
}

// buildShoppingList is pure for testability.
func buildShoppingList(materials []*domain.Material, supplierNames map[uuid.UUID]string) []*domain.ShoppingListGroup {
	groups := map[string]*domain.ShoppingListGroup{}
	for _, m := range materials {
		if m.Status != domain.MaterialNeeded {
			continue
		}
		key := ""
		name := "Uden leverandør"
		if m.SupplierID != nil {
			key = m.SupplierID.String()
			if n, ok := supplierNames[*m.SupplierID]; ok {
				name = n
			} else {
				name = "Ukendt leverandør"
			}
		}
		g, ok := groups[key]
		if !ok {
			g = &domain.ShoppingListGroup{SupplierID: m.SupplierID, SupplierName: name}
			groups[key] = g
		}
		g.Materials = append(g.Materials, m)
		if m.UnitPriceOre != nil {
			g.TotalOre += int64(math.Round(m.Quantity * float64(*m.UnitPriceOre)))
		}
	}
	result := make([]*domain.ShoppingListGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, g)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SupplierName < result[j].SupplierName })
	return result
}

func (s *Materials) applyMaterialPatch(ctx context.Context, m *domain.Material, patch MaterialPatch) error {
	if patch.SupplierID != nil {
		if *patch.SupplierID == uuid.Nil {
			m.SupplierID = nil
		} else {
			sup, err := s.suppliers.Get(ctx, *patch.SupplierID)
			if err != nil {
				return err
			}
			if sup.ProjectID != m.ProjectID {
				return domain.Validation("leverandøren tilhører et andet projekt")
			}
			m.SupplierID = patch.SupplierID
		}
	}
	// Phase/task/room references share the project via FK; cross-project
	// misuse is prevented at the same level as tasks (service checks above,
	// FK integrity below).
	if patch.PhaseID != nil {
		m.PhaseID = nilIfZero(patch.PhaseID)
	}
	if patch.TaskID != nil {
		m.TaskID = nilIfZero(patch.TaskID)
	}
	if patch.RoomID != nil {
		m.RoomID = nilIfZero(patch.RoomID)
	}
	if patch.Name != nil {
		m.Name = strings.TrimSpace(*patch.Name)
		if m.Name == "" {
			return domain.Validation("materialenavn kan ikke være tomt")
		}
	}
	if patch.Spec != nil {
		m.Spec = *patch.Spec
	}
	if patch.Quantity != nil {
		if *patch.Quantity < 0 {
			return domain.Validation("antal kan ikke være negativt")
		}
		m.Quantity = *patch.Quantity
	}
	if patch.Unit != nil {
		m.Unit = strings.TrimSpace(*patch.Unit)
	}
	if patch.UnitPriceOre != nil {
		if *patch.UnitPriceOre < 0 {
			return domain.Validation("pris kan ikke være negativ")
		}
		m.UnitPriceOre = patch.UnitPriceOre
	}
	if patch.Status != nil {
		status := domain.MaterialStatus(*patch.Status)
		if !status.Valid() {
			return domain.Validation("ugyldig materialestatus")
		}
		m.Status = status
	}
	if patch.Notes != nil {
		m.Notes = *patch.Notes
	}
	return nil
}

func nilIfZero(id *uuid.UUID) *uuid.UUID {
	if id == nil || *id == uuid.Nil {
		return nil
	}
	return id
}
