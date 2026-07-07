package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type CaseFileRepo interface {
	Create(ctx context.Context, c *domain.CaseFile) (*domain.CaseFile, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.CaseFile, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.CaseFile, error)
	Update(ctx context.Context, c *domain.CaseFile) (*domain.CaseFile, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CreateEvent(ctx context.Context, e *domain.CaseEvent) (*domain.CaseEvent, error)
	ListEvents(ctx context.Context, caseFileID uuid.UUID) ([]*domain.CaseEvent, error)
}

type CaseFiles struct {
	repo   CaseFileRepo
	access ProjectAccess
}

func NewCaseFiles(repo CaseFileRepo, access ProjectAccess) *CaseFiles {
	return &CaseFiles{repo: repo, access: access}
}

type CaseFilePatch struct {
	Title               *string    `json:"title"`
	Description         *string    `json:"description"`
	CaseType            *string    `json:"caseType"`
	Status              *string    `json:"status"`
	MunicipalCaseNumber *string    `json:"municipalCaseNumber"`
	SubmittedAt         *time.Time `json:"submittedAt"`
	DecidedAt           *time.Time `json:"decidedAt"`
}

func (s *CaseFiles) Create(ctx context.Context, userID, projectID uuid.UUID, patch CaseFilePatch) (*domain.CaseFile, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	c := &domain.CaseFile{
		ProjectID: projectID,
		CaseType:  domain.CaseTypeUnknown,
		Status:    domain.CaseDraft,
	}
	if err := applyCaseFilePatch(c, patch); err != nil {
		return nil, err
	}
	if c.Title == "" {
		return nil, domain.Validation("sagstitel kræves")
	}
	return s.repo.Create(ctx, c)
}

func (s *CaseFiles) Get(ctx context.Context, userID, caseFileID uuid.UUID) (*domain.CaseFile, error) {
	c, err := s.repo.Get(ctx, caseFileID)
	if err != nil {
		return nil, err
	}
	if _, err := requireRead(ctx, s.access, c.ProjectID, userID); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CaseFiles) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.CaseFile, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

// Update patches the case; a status change is logged automatically on the
// case timeline.
func (s *CaseFiles) Update(ctx context.Context, userID, caseFileID uuid.UUID, patch CaseFilePatch) (*domain.CaseFile, error) {
	c, err := s.repo.Get(ctx, caseFileID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, c.ProjectID, userID); err != nil {
		return nil, err
	}
	oldStatus := c.Status
	if err := applyCaseFilePatch(c, patch); err != nil {
		return nil, err
	}
	if c.Status == domain.CaseSubmitted && oldStatus != domain.CaseSubmitted && c.SubmittedAt == nil {
		now := time.Now()
		c.SubmittedAt = &now
	}
	updated, err := s.repo.Update(ctx, c)
	if err != nil {
		return nil, err
	}
	if updated.Status != oldStatus {
		_, _ = s.repo.CreateEvent(ctx, &domain.CaseEvent{
			CaseFileID: caseFileID,
			EventType:  domain.CaseEventStatusChange,
			OccurredAt: time.Now(),
			Summary:    "Status ændret: " + string(oldStatus) + " → " + string(updated.Status),
			CreatedBy:  &userID,
		})
	}
	return updated, nil
}

func (s *CaseFiles) Delete(ctx context.Context, userID, caseFileID uuid.UUID) error {
	c, err := s.repo.Get(ctx, caseFileID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, c.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, caseFileID)
}

type CaseEventInput struct {
	EventType  string     `json:"eventType"`
	Direction  *string    `json:"direction"`
	OccurredAt *time.Time `json:"occurredAt"`
	Summary    string     `json:"summary"`
	Body       string     `json:"body"`
	DocumentID *uuid.UUID `json:"documentId"`
}

func (s *CaseFiles) AddEvent(ctx context.Context, userID, caseFileID uuid.UUID, in CaseEventInput) (*domain.CaseEvent, error) {
	c, err := s.repo.Get(ctx, caseFileID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, c.ProjectID, userID); err != nil {
		return nil, err
	}
	eventType := domain.CaseEventType(in.EventType)
	if !eventType.Valid() {
		return nil, domain.Validation("ugyldig hændelsestype")
	}
	if strings.TrimSpace(in.Summary) == "" {
		return nil, domain.Validation("resumé kræves")
	}
	e := &domain.CaseEvent{
		CaseFileID: caseFileID,
		EventType:  eventType,
		OccurredAt: time.Now(),
		Summary:    strings.TrimSpace(in.Summary),
		Body:       in.Body,
		DocumentID: in.DocumentID,
		CreatedBy:  &userID,
	}
	if in.OccurredAt != nil {
		e.OccurredAt = *in.OccurredAt
	}
	if in.Direction != nil {
		dir := domain.Direction(*in.Direction)
		if !dir.Valid() {
			return nil, domain.Validation("ugyldig retning")
		}
		e.Direction = &dir
	}
	return s.repo.CreateEvent(ctx, e)
}

func (s *CaseFiles) ListEvents(ctx context.Context, userID, caseFileID uuid.UUID) ([]*domain.CaseEvent, error) {
	if _, err := s.Get(ctx, userID, caseFileID); err != nil {
		return nil, err
	}
	return s.repo.ListEvents(ctx, caseFileID)
}

func applyCaseFilePatch(c *domain.CaseFile, patch CaseFilePatch) error {
	if patch.Title != nil {
		c.Title = strings.TrimSpace(*patch.Title)
		if c.Title == "" {
			return domain.Validation("sagstitel kan ikke være tom")
		}
	}
	if patch.Description != nil {
		c.Description = *patch.Description
	}
	if patch.CaseType != nil {
		t := domain.CaseType(*patch.CaseType)
		if !t.Valid() {
			return domain.Validation("ugyldig sagstype")
		}
		c.CaseType = t
	}
	if patch.Status != nil {
		st := domain.CaseStatus(*patch.Status)
		if !st.Valid() {
			return domain.Validation("ugyldig sagsstatus")
		}
		c.Status = st
	}
	if patch.MunicipalCaseNumber != nil {
		c.MunicipalCaseNumber = strings.TrimSpace(*patch.MunicipalCaseNumber)
	}
	if patch.SubmittedAt != nil {
		c.SubmittedAt = patch.SubmittedAt
	}
	if patch.DecidedAt != nil {
		c.DecidedAt = patch.DecidedAt
	}
	return nil
}
