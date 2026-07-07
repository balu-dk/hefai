package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/calc"
	"github.com/balu-dk/hefai/backend/internal/domain"
)

type StructuralRepo interface {
	CreateElement(ctx context.Context, e *domain.StructuralElement) (*domain.StructuralElement, error)
	GetElement(ctx context.Context, id uuid.UUID) (*domain.StructuralElement, error)
	ListElements(ctx context.Context, projectID uuid.UUID) ([]*domain.StructuralElement, error)
	UpdateElement(ctx context.Context, e *domain.StructuralElement) (*domain.StructuralElement, error)
	DeleteElement(ctx context.Context, id uuid.UUID) error
	CreateLoad(ctx context.Context, l *domain.Load) (*domain.Load, error)
	GetLoad(ctx context.Context, id uuid.UUID) (*domain.Load, error)
	ListLoads(ctx context.Context, projectID uuid.UUID) ([]*domain.Load, error)
	UpdateLoad(ctx context.Context, l *domain.Load) (*domain.Load, error)
	DeleteLoad(ctx context.Context, id uuid.UUID) error
	CreateEstimate(ctx context.Context, e *domain.CalculationEstimate) (*domain.CalculationEstimate, error)
	GetEstimate(ctx context.Context, id uuid.UUID) (*domain.CalculationEstimate, error)
	ListEstimates(ctx context.Context, projectID uuid.UUID) ([]*domain.CalculationEstimate, error)
	SetEstimateStatus(ctx context.Context, id uuid.UUID, status domain.EstimateStatus) error
	SetLoadStatus(ctx context.Context, id uuid.UUID, status domain.LoadStatus) error
}

type Structural struct {
	repo   StructuralRepo
	access ProjectAccess
}

func NewStructural(repo StructuralRepo, access ProjectAccess) *Structural {
	return &Structural{repo: repo, access: access}
}

// --- elements ---------------------------------------------------------------

type ElementPatch struct {
	RoomID        *uuid.UUID       `json:"roomId"`
	DrawingID     *uuid.UUID       `json:"drawingId"`
	ElementType   *string          `json:"elementType"`
	Name          *string          `json:"name"`
	IsLoadBearing *bool            `json:"isLoadBearing"`
	Material      *string          `json:"material"`
	MaterialSpec  *string          `json:"materialSpec"`
	Geometry      *json.RawMessage `json:"geometry"`
	Notes         *string          `json:"notes"`
}

func (s *Structural) CreateElement(ctx context.Context, userID, projectID uuid.UUID, patch ElementPatch) (*domain.StructuralElement, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	e := &domain.StructuralElement{
		ProjectID:     projectID,
		ElementType:   domain.ElementBeam,
		IsLoadBearing: true,
		Material:      domain.MaterialTimber,
		Geometry:      json.RawMessage(`{}`),
	}
	if err := applyElementPatch(e, patch); err != nil {
		return nil, err
	}
	if e.Name == "" {
		return nil, domain.Validation("elementnavn kræves")
	}
	return s.repo.CreateElement(ctx, e)
}

func (s *Structural) ListElements(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.StructuralElement, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListElements(ctx, projectID)
}

func (s *Structural) UpdateElement(ctx context.Context, userID, elementID uuid.UUID, patch ElementPatch) (*domain.StructuralElement, error) {
	e, err := s.repo.GetElement(ctx, elementID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, e.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := applyElementPatch(e, patch); err != nil {
		return nil, err
	}
	return s.repo.UpdateElement(ctx, e)
}

func (s *Structural) DeleteElement(ctx context.Context, userID, elementID uuid.UUID) error {
	e, err := s.repo.GetElement(ctx, elementID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, e.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.DeleteElement(ctx, elementID)
}

func applyElementPatch(e *domain.StructuralElement, patch ElementPatch) error {
	if patch.RoomID != nil {
		e.RoomID = nilIfZero(patch.RoomID)
	}
	if patch.DrawingID != nil {
		e.DrawingID = nilIfZero(patch.DrawingID)
	}
	if patch.ElementType != nil {
		t := domain.StructuralElementType(*patch.ElementType)
		if !t.Valid() {
			return domain.Validation("ugyldig elementtype")
		}
		e.ElementType = t
	}
	if patch.Name != nil {
		e.Name = strings.TrimSpace(*patch.Name)
		if e.Name == "" {
			return domain.Validation("elementnavn kan ikke være tomt")
		}
	}
	if patch.IsLoadBearing != nil {
		e.IsLoadBearing = *patch.IsLoadBearing
	}
	if patch.Material != nil {
		m := domain.StructuralMaterial(*patch.Material)
		if !m.Valid() {
			return domain.Validation("ugyldigt materiale")
		}
		e.Material = m
	}
	if patch.MaterialSpec != nil {
		e.MaterialSpec = strings.TrimSpace(*patch.MaterialSpec)
	}
	if patch.Geometry != nil {
		if !json.Valid(*patch.Geometry) {
			return domain.Validation("geometri skal være gyldig JSON")
		}
		e.Geometry = *patch.Geometry
	}
	if patch.Notes != nil {
		e.Notes = *patch.Notes
	}
	return nil
}

// --- loads -------------------------------------------------------------------

type LoadPatch struct {
	StructuralElementID *uuid.UUID       `json:"structuralElementId"`
	LoadType            *string          `json:"loadType"`
	Value               *float64         `json:"value"`
	Unit                *string          `json:"unit"`
	StandardReference   *string          `json:"standardReference"`
	Derivation          *json.RawMessage `json:"derivation"`
	Status              *string          `json:"status"`
	Notes               *string          `json:"notes"`
}

func (s *Structural) CreateLoad(ctx context.Context, userID, projectID uuid.UUID, patch LoadPatch) (*domain.Load, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	l := &domain.Load{
		ProjectID:  projectID,
		LoadType:   domain.LoadOther,
		Status:     domain.LoadAssumed,
		Derivation: json.RawMessage(`{}`),
	}
	if err := s.applyLoadPatch(ctx, l, patch); err != nil {
		return nil, err
	}
	if l.Unit == "" {
		return nil, domain.Validation("enhed kræves (fx kN/m²)")
	}
	return s.repo.CreateLoad(ctx, l)
}

func (s *Structural) ListLoads(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.Load, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListLoads(ctx, projectID)
}

func (s *Structural) UpdateLoad(ctx context.Context, userID, loadID uuid.UUID, patch LoadPatch) (*domain.Load, error) {
	l, err := s.repo.GetLoad(ctx, loadID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, l.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := s.applyLoadPatch(ctx, l, patch); err != nil {
		return nil, err
	}
	return s.repo.UpdateLoad(ctx, l)
}

func (s *Structural) DeleteLoad(ctx context.Context, userID, loadID uuid.UUID) error {
	l, err := s.repo.GetLoad(ctx, loadID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, l.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.DeleteLoad(ctx, loadID)
}

func (s *Structural) applyLoadPatch(ctx context.Context, l *domain.Load, patch LoadPatch) error {
	if patch.StructuralElementID != nil {
		if *patch.StructuralElementID == uuid.Nil {
			l.StructuralElementID = nil
		} else {
			e, err := s.repo.GetElement(ctx, *patch.StructuralElementID)
			if err != nil {
				return err
			}
			if e.ProjectID != l.ProjectID {
				return domain.Validation("elementet tilhører et andet projekt")
			}
			l.StructuralElementID = patch.StructuralElementID
		}
	}
	if patch.LoadType != nil {
		t := domain.LoadType(*patch.LoadType)
		if !t.Valid() {
			return domain.Validation("ugyldig lasttype")
		}
		l.LoadType = t
	}
	if patch.Value != nil {
		l.Value = *patch.Value
	}
	if patch.Unit != nil {
		l.Unit = strings.TrimSpace(*patch.Unit)
	}
	if patch.StandardReference != nil {
		l.StandardReference = strings.TrimSpace(*patch.StandardReference)
	}
	if patch.Derivation != nil {
		if !json.Valid(*patch.Derivation) {
			return domain.Validation("derivation skal være gyldig JSON")
		}
		l.Derivation = *patch.Derivation
	}
	if patch.Status != nil {
		st := domain.LoadStatus(*patch.Status)
		if !st.Valid() {
			return domain.Validation("ugyldig laststatus")
		}
		l.Status = st
	}
	if patch.Notes != nil {
		l.Notes = *patch.Notes
	}
	return nil
}

// --- estimates ----------------------------------------------------------------

type EstimateInput struct {
	Method              string          `json:"method"`
	StructuralElementID *uuid.UUID      `json:"structuralElementId"`
	Inputs              json.RawMessage `json:"inputs"`
	Notes               string          `json:"notes"`
}

// RunEstimate executes a deterministic calc method and stores the immutable
// result as an advisory estimate.
func (s *Structural) RunEstimate(ctx context.Context, userID, projectID uuid.UUID, in EstimateInput) (*domain.CalculationEstimate, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	if in.StructuralElementID != nil {
		e, err := s.repo.GetElement(ctx, *in.StructuralElementID)
		if err != nil {
			return nil, err
		}
		if e.ProjectID != projectID {
			return nil, domain.Validation("elementet tilhører et andet projekt")
		}
	}

	outcome, err := calc.Run(in.Method, in.Inputs)
	if err != nil {
		return nil, err
	}

	assumptions, _ := json.Marshal(outcome.Assumptions)
	results, _ := json.Marshal(map[string]any{
		"results": outcome.Results,
		"notice":  outcome.Notice,
	})
	return s.repo.CreateEstimate(ctx, &domain.CalculationEstimate{
		ProjectID:           projectID,
		StructuralElementID: in.StructuralElementID,
		Method:              outcome.Method,
		MethodVersion:       outcome.MethodVersion,
		StandardReference:   outcome.StandardReference,
		Inputs:              outcome.Inputs,
		Assumptions:         assumptions,
		Results:             results,
		Status:              domain.EstimateAdvisory,
		Notes:               in.Notes,
	})
}

func (s *Structural) ListEstimates(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.CalculationEstimate, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListEstimates(ctx, projectID)
}

// Methods documents the available calculation methods.
func (s *Structural) Methods() []calc.MethodInfo { return calc.Methods() }
