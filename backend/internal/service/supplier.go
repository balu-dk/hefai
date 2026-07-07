package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type SupplierRepo interface {
	Create(ctx context.Context, s *domain.Supplier) (*domain.Supplier, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Supplier, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Supplier, error)
	Update(ctx context.Context, s *domain.Supplier) (*domain.Supplier, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Suppliers struct {
	repo   SupplierRepo
	access ProjectAccess
}

func NewSuppliers(repo SupplierRepo, access ProjectAccess) *Suppliers {
	return &Suppliers{repo: repo, access: access}
}

type SupplierPatch struct {
	CompanyName   *string `json:"companyName"`
	ContactPerson *string `json:"contactPerson"`
	Trade         *string `json:"trade"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Notes         *string `json:"notes"`
}

func (s *Suppliers) Create(ctx context.Context, userID, projectID uuid.UUID, patch SupplierPatch) (*domain.Supplier, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	sup := &domain.Supplier{ProjectID: projectID}
	applySupplierPatch(sup, patch)
	if sup.CompanyName == "" {
		return nil, domain.Validation("firmanavn kræves")
	}
	return s.repo.Create(ctx, sup)
}

func (s *Suppliers) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.Supplier, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Suppliers) Update(ctx context.Context, userID, supplierID uuid.UUID, patch SupplierPatch) (*domain.Supplier, error) {
	sup, err := s.repo.Get(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, sup.ProjectID, userID); err != nil {
		return nil, err
	}
	applySupplierPatch(sup, patch)
	if sup.CompanyName == "" {
		return nil, domain.Validation("firmanavn kan ikke være tomt")
	}
	return s.repo.Update(ctx, sup)
}

func (s *Suppliers) Delete(ctx context.Context, userID, supplierID uuid.UUID) error {
	sup, err := s.repo.Get(ctx, supplierID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, sup.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, supplierID)
}

func applySupplierPatch(s *domain.Supplier, patch SupplierPatch) {
	if patch.CompanyName != nil {
		s.CompanyName = strings.TrimSpace(*patch.CompanyName)
	}
	if patch.ContactPerson != nil {
		s.ContactPerson = *patch.ContactPerson
	}
	if patch.Trade != nil {
		s.Trade = *patch.Trade
	}
	if patch.Phone != nil {
		s.Phone = *patch.Phone
	}
	if patch.Email != nil {
		s.Email = *patch.Email
	}
	if patch.Notes != nil {
		s.Notes = *patch.Notes
	}
}
