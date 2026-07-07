package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
	"github.com/balu-dk/hefai/backend/internal/pdfgen"
)

type GeneratedRepo interface {
	Create(ctx context.Context, g *domain.GeneratedDocument) (*domain.GeneratedDocument, error)
	ListByCaseFile(ctx context.Context, caseFileID uuid.UUID) ([]*domain.GeneratedDocument, error)
}

// Generator renders application PDFs from case data and files them in the
// document archive. Output is always marked as draft.
type Generator struct {
	generated GeneratedRepo
	caseFiles CaseFileRepo
	drawings  DrawingRepo
	checklist ComplianceRepo
	projects  ProjectRepo
	documents DocumentRepo
	files     FileStore
	access    ProjectAccess
}

func NewGenerator(generated GeneratedRepo, caseFiles CaseFileRepo, drawings DrawingRepo,
	checklist ComplianceRepo, projects ProjectRepo, documents DocumentRepo, files FileStore,
	access ProjectAccess) *Generator {
	return &Generator{
		generated: generated, caseFiles: caseFiles, drawings: drawings,
		checklist: checklist, projects: projects, documents: documents,
		files: files, access: access,
	}
}

type GenerateInput struct {
	Kind      string     `json:"kind"`
	DrawingID *uuid.UUID `json:"drawingId"` // for site_plan/floor_plan/area_statement
}

// Generate renders one application document for the case and returns the
// generated_documents record (the PDF itself lands in the archive).
func (s *Generator) Generate(ctx context.Context, userID, caseFileID uuid.UUID, in GenerateInput) (*domain.GeneratedDocument, error) {
	caseFile, err := s.caseFiles.Get(ctx, caseFileID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, caseFile.ProjectID, userID); err != nil {
		return nil, err
	}
	project, err := s.projects.Get(ctx, caseFile.ProjectID)
	if err != nil {
		return nil, err
	}

	kind := domain.GeneratedDocumentKind(in.Kind)
	if !kind.Valid() {
		return nil, domain.Validation("ugyldig dokumenttype")
	}

	meta := pdfgen.Meta{Project: project, CaseFile: caseFile, GeneratedAt: time.Now()}
	snapshot := map[string]any{
		"caseFileId": caseFileID,
		"caseStatus": caseFile.Status,
		"kind":       kind,
	}

	var pdf []byte
	var title string

	switch kind {
	case domain.GenProjectDescription:
		title = "Projektbeskrivelse"
		pdf, err = pdfgen.ProjectDescription(meta)

	case domain.GenApplicationSummary:
		title = "Ansøgningsoversigt"
		items, listErr := s.checklist.ListByCaseFile(ctx, caseFileID)
		if listErr != nil {
			return nil, listErr
		}
		attachments, listErr := s.attachmentTitles(ctx, caseFileID)
		if listErr != nil {
			return nil, listErr
		}
		snapshot["checklistItems"] = len(items)
		pdf, err = pdfgen.ApplicationSummary(meta, items, attachments)

	case domain.GenAreaStatement, domain.GenFloorPlan, domain.GenSitePlan:
		drawing, version, verErr := s.resolveDrawing(ctx, caseFile, in.DrawingID)
		if verErr != nil {
			return nil, verErr
		}
		snapshot["drawingId"] = drawing.ID
		snapshot["drawingVersion"] = version.VersionNo
		switch kind {
		case domain.GenAreaStatement:
			title = "Arealopgørelse"
			pdf, err = pdfgen.AreaStatement(meta, &version.Data, drawing.Title)
		case domain.GenFloorPlan:
			title = "Plantegning"
			pdf, err = pdfgen.FloorPlan(meta, &version.Data, drawing.Title, version.Scale)
		case domain.GenSitePlan:
			title = "Situationsplan"
			pdf, err = pdfgen.SitePlan(meta, &version.Data, drawing.Title)
		}

	case domain.GenElevation:
		return nil, domain.Validation("opstalt/facade kræver arkitektens tegninger — upload dem som dokument " +
			"i arkivet i stedet; Hefai genererer ikke facadetegninger")

	default:
		return nil, domain.Validation("dokumenttypen genereres ikke fra byggesagen")
	}
	if err != nil {
		return nil, err
	}

	// File the PDF in the archive so it is viewable and attachable like any
	// other document.
	filename := fmt.Sprintf("%s-v%s.pdf", kind, time.Now().Format("20060102-150405"))
	key := fmt.Sprintf("%s/%s.pdf", project.ID, uuid.New())
	size, err := s.files.Save(key, bytes.NewReader(pdf))
	if err != nil {
		return nil, err
	}
	doc, err := s.documents.Create(ctx, &domain.Document{
		ProjectID:   project.ID,
		UploadedBy:  &userID,
		Kind:        domain.DocGenerated,
		Title:       title + " (kladde) — " + caseFile.Title,
		Description: "Genereret af Hefai. Kladde — kræver kontrol og godkendelse.",
		Filename:    filename,
		StorageKey:  key,
		MimeType:    "application/pdf",
		SizeBytes:   size,
	})
	if err != nil {
		_ = s.files.Delete(key)
		return nil, err
	}

	snapshotJSON, _ := json.Marshal(snapshot)
	return s.generated.Create(ctx, &domain.GeneratedDocument{
		ProjectID:     project.ID,
		CaseFileID:    &caseFileID,
		Kind:          kind,
		Status:        domain.GeneratedDraft,
		InputSnapshot: snapshotJSON,
		DocumentID:    &doc.ID,
	})
}

func (s *Generator) List(ctx context.Context, userID, caseFileID uuid.UUID) ([]*domain.GeneratedDocument, error) {
	caseFile, err := s.caseFiles.Get(ctx, caseFileID)
	if err != nil {
		return nil, err
	}
	if _, err := requireRead(ctx, s.access, caseFile.ProjectID, userID); err != nil {
		return nil, err
	}
	return s.generated.ListByCaseFile(ctx, caseFileID)
}

// resolveDrawing picks the requested drawing or the case's only drawing,
// and loads its latest version.
func (s *Generator) resolveDrawing(ctx context.Context, caseFile *domain.CaseFile, drawingID *uuid.UUID) (*domain.Drawing, *domain.DrawingVersion, error) {
	var drawing *domain.Drawing
	if drawingID != nil {
		d, err := s.drawings.Get(ctx, *drawingID)
		if err != nil {
			return nil, nil, err
		}
		if d.ProjectID != caseFile.ProjectID {
			return nil, nil, domain.Validation("tegningen tilhører et andet projekt")
		}
		drawing = d
	} else {
		all, err := s.drawings.ListByProject(ctx, caseFile.ProjectID)
		if err != nil {
			return nil, nil, err
		}
		for _, d := range all {
			if d.CaseFileID != nil && *d.CaseFileID == caseFile.ID {
				drawing = d
				break
			}
		}
		if drawing == nil {
			return nil, nil, domain.Validation("ingen tegning er knyttet til sagen — angiv drawingId eller knyt en tegning")
		}
	}
	version, err := s.drawings.LatestVersion(ctx, drawing.ID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, nil, domain.Validation("tegningen har ingen versioner endnu — gem den i tegnefladen først")
		}
		return nil, nil, err
	}
	return drawing, version, nil
}

// attachmentTitles lists documents linked to the case for the summary's
// bilag section.
func (s *Generator) attachmentTitles(ctx context.Context, caseFileID uuid.UUID) ([]string, error) {
	caseFile, err := s.caseFiles.Get(ctx, caseFileID)
	if err != nil {
		return nil, err
	}
	docs, err := s.documents.List(ctx, caseFile.ProjectID, domain.DocumentFilter{
		TargetType: domain.LinkCaseFile,
		TargetID:   caseFileID,
	})
	if err != nil {
		return nil, err
	}
	titles := make([]string, 0, len(docs))
	for _, d := range docs {
		titles = append(titles, d.Title)
	}
	return titles, nil
}
