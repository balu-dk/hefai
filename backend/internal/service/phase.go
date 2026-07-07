package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type PhaseRepo interface {
	Create(ctx context.Context, p *domain.Phase) (*domain.Phase, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Phase, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Phase, error)
	Update(ctx context.Context, p *domain.Phase) (*domain.Phase, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Phases struct {
	repo   PhaseRepo
	access ProjectAccess
}

func NewPhases(repo PhaseRepo, access ProjectAccess) *Phases {
	return &Phases{repo: repo, access: access}
}

// PhasePatch uses pointers so PATCH requests only touch provided fields.
type PhasePatch struct {
	Name         *string    `json:"name"`
	Description  *string    `json:"description"`
	SortOrder    *int       `json:"sortOrder"`
	Status       *string    `json:"status"`
	PlannedStart *time.Time `json:"plannedStart"`
	PlannedEnd   *time.Time `json:"plannedEnd"`
	ActualStart  *time.Time `json:"actualStart"`
	ActualEnd    *time.Time `json:"actualEnd"`
}

func (s *Phases) Create(ctx context.Context, userID, projectID uuid.UUID, patch PhasePatch) (*domain.Phase, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	p := &domain.Phase{ProjectID: projectID, Status: domain.PhaseNotStarted}
	if err := applyPhasePatch(p, patch); err != nil {
		return nil, err
	}
	if p.Name == "" {
		return nil, domain.Validation("fasenavn kræves")
	}
	return s.repo.Create(ctx, p)
}

func (s *Phases) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.Phase, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Phases) Update(ctx context.Context, userID, phaseID uuid.UUID, patch PhasePatch) (*domain.Phase, error) {
	p, err := s.repo.Get(ctx, phaseID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, p.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := applyPhasePatch(p, patch); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, p)
}

func (s *Phases) Delete(ctx context.Context, userID, phaseID uuid.UUID) error {
	p, err := s.repo.Get(ctx, phaseID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, p.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, phaseID)
}

func applyPhasePatch(p *domain.Phase, patch PhasePatch) error {
	if patch.Name != nil {
		p.Name = strings.TrimSpace(*patch.Name)
		if p.Name == "" {
			return domain.Validation("fasenavn kan ikke være tomt")
		}
	}
	if patch.Description != nil {
		p.Description = *patch.Description
	}
	if patch.SortOrder != nil {
		p.SortOrder = *patch.SortOrder
	}
	if patch.Status != nil {
		status := domain.PhaseStatus(*patch.Status)
		if !status.Valid() {
			return domain.Validation("ugyldig fasestatus")
		}
		p.Status = status
	}
	if patch.PlannedStart != nil {
		p.PlannedStart = patch.PlannedStart
	}
	if patch.PlannedEnd != nil {
		p.PlannedEnd = patch.PlannedEnd
	}
	if patch.ActualStart != nil {
		p.ActualStart = patch.ActualStart
	}
	if patch.ActualEnd != nil {
		p.ActualEnd = patch.ActualEnd
	}
	return nil
}
