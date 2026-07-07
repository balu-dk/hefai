package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
	"github.com/balu-dk/hefai/backend/internal/pdftext"
	"github.com/balu-dk/hefai/backend/internal/rag"
)

// PlanImport henter en fundet lokalplan-PDF, arkiverer den som dokument og
// udtrækker teksten direkte ind i kildematerialet — så assistenten og
// Lovtjek kan bruge planen uden manuel copy-paste. Skannede PDF'er uden
// tekstlag arkiveres stadig, men brugeren får ærlig besked om at teksten
// skal indsættes manuelt.
type PlanImport struct {
	files     FileStore
	documents DocumentRepo
	sources   SourceRepo
	access    ProjectAccess
	client    *http.Client
	// allowedHosts afgrænser hvilke værter der må hentes fra (SSRF-værn).
	allowedHosts []string
}

func NewPlanImport(files FileStore, documents DocumentRepo, sources SourceRepo, access ProjectAccess) *PlanImport {
	return &PlanImport{
		files:        files,
		documents:    documents,
		sources:      sources,
		access:       access,
		client:       &http.Client{Timeout: 60 * time.Second},
		allowedHosts: []string{"plandata.dk"}, // inkl. subdomæner, fx dokument.plandata.dk
	}
}

type PlanImportInput struct {
	Name    string `json:"name"`
	DocLink string `json:"docLink"`
}

type PlanImportResult struct {
	DocumentID uuid.UUID  `json:"documentId"`
	SourceID   *uuid.UUID `json:"sourceId"`
	ChunkCount int        `json:"chunkCount"`
	Notice     string     `json:"notice"`
}

// minExtractedRunes: under denne grænse regnes PDF'en for skannet/uden
// brugbart tekstlag, og der oprettes ingen kilde.
const minExtractedRunes = 400

func (s *PlanImport) Import(ctx context.Context, userID, projectID uuid.UUID, in PlanImportInput) (*PlanImportResult, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Lokalplan"
	}
	link, err := url.Parse(strings.TrimSpace(in.DocLink))
	if err != nil || link.Scheme != "https" {
		return nil, domain.Validation("docLink skal være en https-adresse")
	}
	if !s.hostAllowed(link.Hostname()) {
		return nil, domain.Validation("af sikkerhedshensyn hentes der kun dokumenter fra plandata.dk — upload andre PDF'er manuelt under Dokumenter")
	}

	// Hent PDF'en (maks. 50 MB).
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kunne ikke hente dokumentet: %w", err)
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	if readErr != nil && !(errors.Is(readErr, io.ErrUnexpectedEOF) && len(data) > 0) {
		return nil, readErr
	}
	if resp.StatusCode != http.StatusOK {
		return nil, domain.Validation(fmt.Sprintf("plandata svarede %d for dokumentet", resp.StatusCode))
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		return nil, domain.Validation("linket peger ikke på en PDF")
	}

	// Arkivér PDF'en som dokument.
	key := fmt.Sprintf("%s/%s.pdf", projectID, uuid.New())
	size, err := s.files.Save(key, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	doc, err := s.documents.Create(ctx, &domain.Document{
		ProjectID:   projectID,
		UploadedBy:  &userID,
		Kind:        domain.DocPermit,
		Title:       name,
		Description: "Hentet automatisk fra plandata.dk: " + link.String(),
		Filename:    safeFilename(name) + ".pdf",
		StorageKey:  key,
		MimeType:    "application/pdf",
		SizeBytes:   size,
	})
	if err != nil {
		_ = s.files.Delete(key)
		return nil, err
	}

	result := &PlanImportResult{DocumentID: doc.ID}

	// Udtræk teksten og læg den i kildematerialet.
	text, extractErr := pdftext.Extract(data)
	if extractErr != nil || len([]rune(text)) < minExtractedRunes {
		result.Notice = "PDF'en er arkiveret under Dokumenter, men teksten kunne ikke udtrækkes " +
			"(sandsynligvis en skannet plan uden tekstlag). Indsæt de relevante bestemmelser " +
			"manuelt under Kildemateriale, så Lovtjek og assistenten kan bruge dem."
		return result, nil
	}

	chunks := rag.Split(text)
	source, err := s.sources.CreateWithChunks(ctx, &domain.SourceDocument{
		ProjectID:  &projectID,
		DocumentID: &doc.ID,
		Title:      name,
		Kind:       domain.SourceLocalPlan,
		URL:        link.String(),
		AddedBy:    &userID,
	}, chunks)
	if err != nil {
		return nil, err
	}
	result.SourceID = &source.ID
	result.ChunkCount = source.ChunkCount
	result.Notice = fmt.Sprintf("Lokalplanen er arkiveret og teksten indlæst som kilde (%d afsnit). "+
		"Kør \"Foreslå regler fra kildematerialet\" under Lovtjek for at udtrække grænseværdierne.",
		source.ChunkCount)
	return result, nil
}

func (s *PlanImport) hostAllowed(host string) bool {
	host = strings.ToLower(host)
	for _, allowed := range s.allowedHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

func safeFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}, name)
	if name == "" {
		name = "lokalplan"
	}
	if len(name) > 60 {
		name = name[:60]
	}
	return name
}
