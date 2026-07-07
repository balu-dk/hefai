package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
	"github.com/balu-dk/hefai/backend/internal/pdfgen"
)

type PackageRepo interface {
	Create(ctx context.Context, p *domain.StructuralPackage) (*domain.StructuralPackage, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.StructuralPackage, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.StructuralPackage, error)
	SetStatus(ctx context.Context, id uuid.UUID, status domain.PackageStatus, sentAt bool) error
	CreateReview(ctx context.Context, rv *domain.EngineerReview) (*domain.EngineerReview, error)
	ListReviews(ctx context.Context, packageID uuid.UUID) ([]*domain.EngineerReview, error)
}

// Packages assembles the engineer hand-over and records the feedback loop.
type Packages struct {
	repo       PackageRepo
	structural StructuralRepo
	projects   ProjectRepo
	documents  DocumentRepo
	files      FileStore
	access     ProjectAccess
}

func NewPackages(repo PackageRepo, structural StructuralRepo, projects ProjectRepo,
	documents DocumentRepo, files FileStore, access ProjectAccess) *Packages {
	return &Packages{repo: repo, structural: structural, projects: projects,
		documents: documents, files: files, access: access}
}

type PackageInput struct {
	Title string `json:"title"`
}

// Create freezes the project's current elements, loads and estimates into a
// versioned package with a rendered PDF.
func (s *Packages) Create(ctx context.Context, userID, projectID uuid.UUID, in PackageInput) (*domain.StructuralPackage, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	elements, err := s.structural.ListElements(ctx, projectID)
	if err != nil {
		return nil, err
	}
	loads, err := s.structural.ListLoads(ctx, projectID)
	if err != nil {
		return nil, err
	}
	allEstimates, err := s.structural.ListEstimates(ctx, projectID)
	if err != nil {
		return nil, err
	}
	// Superseded runs stay out of the hand-over.
	estimates := make([]*domain.CalculationEstimate, 0, len(allEstimates))
	for _, e := range allEstimates {
		if e.Status != domain.EstimateSuperseded {
			estimates = append(estimates, e)
		}
	}
	if len(elements) == 0 && len(loads) == 0 && len(estimates) == 0 {
		return nil, domain.Validation("pakken ville være tom — registrér elementer, laster eller beregninger først")
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "Statisk grundlag — " + project.Name
	}

	pdf, err := pdfgen.StructuralPackage(pdfgen.Meta{Project: project, GeneratedAt: time.Now()},
		title, elements, loads, estimates)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s/%s.pdf", projectID, uuid.New())
	size, err := s.files.Save(key, bytes.NewReader(pdf))
	if err != nil {
		return nil, err
	}
	doc, err := s.documents.Create(ctx, &domain.Document{
		ProjectID:   projectID,
		UploadedBy:  &userID,
		Kind:        domain.DocGenerated,
		Title:       title + " (kladde — kræver statiker-godkendelse)",
		Description: "Statiker-pakke genereret af Hefai. Vejledende grundlag til verifikation.",
		Filename:    "statiker-pakke.pdf",
		StorageKey:  key,
		MimeType:    "application/pdf",
		SizeBytes:   size,
	})
	if err != nil {
		_ = s.files.Delete(key)
		return nil, err
	}

	snapshot, _ := json.Marshal(map[string]any{
		"elementIds":  ids(elements, func(e *domain.StructuralElement) uuid.UUID { return e.ID }),
		"loadIds":     ids(loads, func(l *domain.Load) uuid.UUID { return l.ID }),
		"estimateIds": ids(estimates, func(e *domain.CalculationEstimate) uuid.UUID { return e.ID }),
	})
	return s.repo.Create(ctx, &domain.StructuralPackage{
		ProjectID:  projectID,
		Title:      title,
		Snapshot:   snapshot,
		DocumentID: &doc.ID,
		Status:     domain.PackageDraft,
	})
}

func ids[T any](items []T, id func(T) uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, len(items))
	for i, item := range items {
		out[i] = id(item)
	}
	return out
}

func (s *Packages) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.StructuralPackage, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

type PackageStatusInput struct {
	Status string `json:"status"`
}

// SetStatus moves the package through draft → sent → reviewed.
func (s *Packages) SetStatus(ctx context.Context, userID, packageID uuid.UUID, in PackageStatusInput) (*domain.StructuralPackage, error) {
	p, err := s.repo.Get(ctx, packageID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, p.ProjectID, userID); err != nil {
		return nil, err
	}
	status := domain.PackageStatus(in.Status)
	switch status {
	case domain.PackageDraft, domain.PackageSent, domain.PackageReviewed:
	default:
		return nil, domain.Validation("ugyldig pakkestatus")
	}
	markSent := status == domain.PackageSent && p.SentAt == nil
	if err := s.repo.SetStatus(ctx, packageID, status, markSent); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, packageID)
}

type ReviewItemInput struct {
	StructuralElementID   *uuid.UUID      `json:"structuralElementId"`
	LoadID                *uuid.UUID      `json:"loadId"`
	CalculationEstimateID *uuid.UUID      `json:"calculationEstimateId"`
	DrawingID             *uuid.UUID      `json:"drawingId"`
	Verdict               string          `json:"verdict"`
	Comment               string          `json:"comment"`
	CorrectedValues       json.RawMessage `json:"correctedValues"`
}

type ReviewInput struct {
	ReviewerName        string            `json:"reviewerName"`
	ReviewerCompany     string            `json:"reviewerCompany"`
	ReviewerCredentials string            `json:"reviewerCredentials"`
	ReceivedAt          *time.Time        `json:"receivedAt"`
	OverallStatus       string            `json:"overallStatus"`
	Summary             string            `json:"summary"`
	ResponseDocumentID  *uuid.UUID        `json:"responseDocumentId"`
	Items               []ReviewItemInput `json:"items"`
}

// AddReview records the engineer's response and propagates the verdicts:
// approved estimates become verified, rejected ones rejected; approved loads
// become engineer_confirmed, changed ones engineer_changed.
func (s *Packages) AddReview(ctx context.Context, userID, packageID uuid.UUID, in ReviewInput) (*domain.EngineerReview, error) {
	p, err := s.repo.Get(ctx, packageID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, p.ProjectID, userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.ReviewerName) == "" {
		return nil, domain.Validation("statikerens navn kræves")
	}
	overall := domain.ReviewStatus(in.OverallStatus)
	if !overall.Valid() {
		return nil, domain.Validation("ugyldig samlet status")
	}

	review := &domain.EngineerReview{
		StructuralPackageID: packageID,
		ReviewerName:        strings.TrimSpace(in.ReviewerName),
		ReviewerCompany:     strings.TrimSpace(in.ReviewerCompany),
		ReviewerCredentials: strings.TrimSpace(in.ReviewerCredentials),
		ReceivedAt:          time.Now(),
		OverallStatus:       overall,
		Summary:             in.Summary,
		ResponseDocumentID:  in.ResponseDocumentID,
	}
	if in.ReceivedAt != nil {
		review.ReceivedAt = *in.ReceivedAt
	}

	for _, item := range in.Items {
		verdict := domain.ReviewVerdict(item.Verdict)
		if !verdict.Valid() {
			return nil, domain.Validation("ugyldig verdict: " + item.Verdict)
		}
		targets := 0
		for _, set := range []bool{item.StructuralElementID != nil, item.LoadID != nil,
			item.CalculationEstimateID != nil, item.DrawingID != nil} {
			if set {
				targets++
			}
		}
		if targets > 1 {
			return nil, domain.Validation("et review-punkt kan højst pege på ét mål")
		}
		if err := s.validateItemTarget(ctx, p.ProjectID, item); err != nil {
			return nil, err
		}
		review.Items = append(review.Items, &domain.EngineerReviewItem{
			StructuralElementID:   item.StructuralElementID,
			LoadID:                item.LoadID,
			CalculationEstimateID: item.CalculationEstimateID,
			DrawingID:             item.DrawingID,
			Verdict:               verdict,
			Comment:               item.Comment,
			CorrectedValues:       item.CorrectedValues,
		})
	}

	created, err := s.repo.CreateReview(ctx, review)
	if err != nil {
		return nil, err
	}

	// Propagate verdicts so the UI can show confirmed vs. changed
	// assumptions per element/load/estimate.
	for _, item := range created.Items {
		switch {
		case item.CalculationEstimateID != nil:
			var status domain.EstimateStatus
			switch item.Verdict {
			case domain.VerdictApproved:
				status = domain.EstimateVerified
			case domain.VerdictRejected, domain.VerdictChanged:
				status = domain.EstimateRejected
			default:
				continue
			}
			if err := s.structural.SetEstimateStatus(ctx, *item.CalculationEstimateID, status); err != nil {
				return nil, err
			}
		case item.LoadID != nil:
			var status domain.LoadStatus
			switch item.Verdict {
			case domain.VerdictApproved:
				status = domain.LoadEngineerConfirmed
			case domain.VerdictChanged, domain.VerdictRejected:
				status = domain.LoadEngineerChanged
			default:
				continue
			}
			if err := s.structural.SetLoadStatus(ctx, *item.LoadID, status); err != nil {
				return nil, err
			}
		}
	}

	if err := s.repo.SetStatus(ctx, packageID, domain.PackageReviewed, false); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Packages) ListReviews(ctx context.Context, userID, packageID uuid.UUID) ([]*domain.EngineerReview, error) {
	p, err := s.repo.Get(ctx, packageID)
	if err != nil {
		return nil, err
	}
	if _, err := requireRead(ctx, s.access, p.ProjectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListReviews(ctx, packageID)
}

func (s *Packages) validateItemTarget(ctx context.Context, projectID uuid.UUID, item ReviewItemInput) error {
	if item.StructuralElementID != nil {
		e, err := s.structural.GetElement(ctx, *item.StructuralElementID)
		if err != nil {
			return err
		}
		if e.ProjectID != projectID {
			return domain.Validation("review-punktet peger på et element i et andet projekt")
		}
	}
	if item.LoadID != nil {
		l, err := s.structural.GetLoad(ctx, *item.LoadID)
		if err != nil {
			return err
		}
		if l.ProjectID != projectID {
			return domain.Validation("review-punktet peger på en last i et andet projekt")
		}
	}
	if item.CalculationEstimateID != nil {
		e, err := s.structural.GetEstimate(ctx, *item.CalculationEstimateID)
		if err != nil {
			return err
		}
		if e.ProjectID != projectID {
			return domain.Validation("review-punktet peger på en beregning i et andet projekt")
		}
	}
	return nil
}
