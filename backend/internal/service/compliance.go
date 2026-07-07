package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type ComplianceRepo interface {
	Create(ctx context.Context, item *domain.ComplianceCheckItem) (*domain.ComplianceCheckItem, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.ComplianceCheckItem, error)
	ListByCaseFile(ctx context.Context, caseFileID uuid.UUID) ([]*domain.ComplianceCheckItem, error)
	Update(ctx context.Context, item *domain.ComplianceCheckItem) (*domain.ComplianceCheckItem, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Compliance struct {
	repo      ComplianceRepo
	caseFiles CaseFileRepo
	sources   SourceRepo
	access    ProjectAccess
}

func NewCompliance(repo ComplianceRepo, caseFiles CaseFileRepo, sources SourceRepo, access ProjectAccess) *Compliance {
	return &Compliance{repo: repo, caseFiles: caseFiles, sources: sources, access: access}
}

type ComplianceItemPatch struct {
	Category      *string    `json:"category"`
	Requirement   *string    `json:"requirement"`
	ExpectedValue *string    `json:"expectedValue"`
	ActualValue   *string    `json:"actualValue"`
	Status        *string    `json:"status"`
	SourceChunkID *uuid.UUID `json:"sourceChunkId"`
	Note          *string    `json:"note"`
}

func (s *Compliance) Create(ctx context.Context, userID, caseFileID uuid.UUID, patch ComplianceItemPatch) (*domain.ComplianceCheckItem, error) {
	c, err := s.caseFiles.Get(ctx, caseFileID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, c.ProjectID, userID); err != nil {
		return nil, err
	}
	item := &domain.ComplianceCheckItem{CaseFileID: caseFileID, Status: domain.ComplianceNotChecked}
	if err := s.applyPatch(ctx, item, c.ProjectID, patch); err != nil {
		return nil, err
	}
	if item.Requirement == "" {
		return nil, domain.Validation("kravtekst kræves")
	}
	return s.repo.Create(ctx, item)
}

func (s *Compliance) List(ctx context.Context, userID, caseFileID uuid.UUID) ([]*domain.ComplianceCheckItem, error) {
	c, err := s.caseFiles.Get(ctx, caseFileID)
	if err != nil {
		return nil, err
	}
	if _, err := requireRead(ctx, s.access, c.ProjectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByCaseFile(ctx, caseFileID)
}

func (s *Compliance) Update(ctx context.Context, userID, itemID uuid.UUID, patch ComplianceItemPatch) (*domain.ComplianceCheckItem, error) {
	item, err := s.repo.Get(ctx, itemID)
	if err != nil {
		return nil, err
	}
	c, err := s.caseFiles.Get(ctx, item.CaseFileID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, c.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := s.applyPatch(ctx, item, c.ProjectID, patch); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, item)
}

func (s *Compliance) Delete(ctx context.Context, userID, itemID uuid.UUID) error {
	item, err := s.repo.Get(ctx, itemID)
	if err != nil {
		return err
	}
	c, err := s.caseFiles.Get(ctx, item.CaseFileID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, c.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, itemID)
}

func (s *Compliance) applyPatch(ctx context.Context, item *domain.ComplianceCheckItem, projectID uuid.UUID, patch ComplianceItemPatch) error {
	if patch.Category != nil {
		item.Category = strings.TrimSpace(*patch.Category)
	}
	if patch.Requirement != nil {
		item.Requirement = strings.TrimSpace(*patch.Requirement)
	}
	if patch.ExpectedValue != nil {
		item.ExpectedValue = strings.TrimSpace(*patch.ExpectedValue)
	}
	if patch.ActualValue != nil {
		item.ActualValue = strings.TrimSpace(*patch.ActualValue)
	}
	if patch.Status != nil {
		status := domain.ComplianceStatus(*patch.Status)
		if !status.Valid() {
			return domain.Validation("ugyldig tjekliste-status")
		}
		item.Status = status
	}
	if patch.SourceChunkID != nil {
		if *patch.SourceChunkID == uuid.Nil {
			item.SourceChunkID = nil
		} else {
			// Grounding must point at a chunk visible to this project
			// (own material or the shared library).
			chunkProject, _, err := s.sources.ChunkProject(ctx, *patch.SourceChunkID)
			if err != nil {
				return err
			}
			if chunkProject != nil && *chunkProject != projectID {
				return domain.Validation("kildehenvisningen tilhører et andet projekt")
			}
			item.SourceChunkID = patch.SourceChunkID
		}
	}
	if patch.Note != nil {
		item.Note = *patch.Note
	}
	return nil
}
