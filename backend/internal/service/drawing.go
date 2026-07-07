package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type DrawingRepo interface {
	Create(ctx context.Context, d *domain.Drawing) (*domain.Drawing, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Drawing, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Drawing, error)
	Update(ctx context.Context, d *domain.Drawing) (*domain.Drawing, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CreateVersion(ctx context.Context, v *domain.DrawingVersion) (*domain.DrawingVersion, error)
	ListVersions(ctx context.Context, drawingID uuid.UUID) ([]*domain.DrawingVersion, error)
	LatestVersion(ctx context.Context, drawingID uuid.UUID) (*domain.DrawingVersion, error)
}

type Drawings struct {
	repo      DrawingRepo
	caseFiles CaseFileRepo
	access    ProjectAccess
}

func NewDrawings(repo DrawingRepo, caseFiles CaseFileRepo, access ProjectAccess) *Drawings {
	return &Drawings{repo: repo, caseFiles: caseFiles, access: access}
}

type DrawingPatch struct {
	CaseFileID *uuid.UUID `json:"caseFileId"`
	Kind       *string    `json:"kind"`
	Title      *string    `json:"title"`
}

func (s *Drawings) Create(ctx context.Context, userID, projectID uuid.UUID, patch DrawingPatch) (*domain.Drawing, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	d := &domain.Drawing{ProjectID: projectID, Kind: domain.DrawingFloorPlan, CreatedBy: &userID}
	if err := s.applyDrawingPatch(ctx, d, patch); err != nil {
		return nil, err
	}
	if d.Title == "" {
		return nil, domain.Validation("tegningstitel kræves")
	}
	return s.repo.Create(ctx, d)
}

func (s *Drawings) Get(ctx context.Context, userID, drawingID uuid.UUID) (*domain.Drawing, error) {
	d, err := s.repo.Get(ctx, drawingID)
	if err != nil {
		return nil, err
	}
	if _, err := requireRead(ctx, s.access, d.ProjectID, userID); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Drawings) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.Drawing, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Drawings) Update(ctx context.Context, userID, drawingID uuid.UUID, patch DrawingPatch) (*domain.Drawing, error) {
	d, err := s.repo.Get(ctx, drawingID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, d.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := s.applyDrawingPatch(ctx, d, patch); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, d)
}

func (s *Drawings) Delete(ctx context.Context, userID, drawingID uuid.UUID) error {
	d, err := s.repo.Get(ctx, drawingID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, d.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, drawingID)
}

type DrawingVersionInput struct {
	Data  domain.DrawingData `json:"data"`
	Scale string             `json:"scale"`
	Note  string             `json:"note"`
}

// AddVersion validates the 2D model and stores it as the next version.
func (s *Drawings) AddVersion(ctx context.Context, userID, drawingID uuid.UUID, in DrawingVersionInput) (*domain.DrawingVersion, error) {
	d, err := s.repo.Get(ctx, drawingID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, d.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := in.Data.Validate(); err != nil {
		return nil, err
	}
	scale := strings.TrimSpace(in.Scale)
	if scale == "" {
		scale = "1:100"
	}
	return s.repo.CreateVersion(ctx, &domain.DrawingVersion{
		DrawingID: drawingID,
		Data:      in.Data,
		Scale:     scale,
		Note:      in.Note,
		CreatedBy: &userID,
	})
}

func (s *Drawings) ListVersions(ctx context.Context, userID, drawingID uuid.UUID) ([]*domain.DrawingVersion, error) {
	if _, err := s.Get(ctx, userID, drawingID); err != nil {
		return nil, err
	}
	return s.repo.ListVersions(ctx, drawingID)
}

func (s *Drawings) applyDrawingPatch(ctx context.Context, d *domain.Drawing, patch DrawingPatch) error {
	if patch.CaseFileID != nil {
		if *patch.CaseFileID == uuid.Nil {
			d.CaseFileID = nil
		} else {
			c, err := s.caseFiles.Get(ctx, *patch.CaseFileID)
			if err != nil {
				return err
			}
			if c.ProjectID != d.ProjectID {
				return domain.Validation("byggesagen tilhører et andet projekt")
			}
			d.CaseFileID = patch.CaseFileID
		}
	}
	if patch.Kind != nil {
		kind := domain.DrawingKind(*patch.Kind)
		if !kind.Valid() {
			return domain.Validation("ugyldig tegningstype")
		}
		d.Kind = kind
	}
	if patch.Title != nil {
		d.Title = strings.TrimSpace(*patch.Title)
		if d.Title == "" {
			return domain.Validation("tegningstitel kan ikke være tom")
		}
	}
	return nil
}
