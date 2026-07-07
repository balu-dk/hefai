package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
	"github.com/balu-dk/hefai/backend/internal/rag"
)

type SourceRepo interface {
	CreateWithChunks(ctx context.Context, s *domain.SourceDocument, chunks []rag.Chunk) (*domain.SourceDocument, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.SourceDocument, error)
	ListForProject(ctx context.Context, projectID uuid.UUID) ([]*domain.SourceDocument, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Search(ctx context.Context, projectID uuid.UUID, query string, limit int) ([]*domain.SourceHit, error)
	ChunkProject(ctx context.Context, chunkID uuid.UUID) (*uuid.UUID, string, error)
}

type Sources struct {
	repo   SourceRepo
	access ProjectAccess
}

func NewSources(repo SourceRepo, access ProjectAccess) *Sources {
	return &Sources{repo: repo, access: access}
}

type IngestInput struct {
	Title        string `json:"title"`
	Kind         string `json:"kind"`
	VersionLabel string `json:"versionLabel"`
	URL          string `json:"url"`
	Content      string `json:"content"`
}

// Ingest chunks the pasted source text and stores it for retrieval. The
// assistant only ever works from material added this way.
func (s *Sources) Ingest(ctx context.Context, userID, projectID uuid.UUID, in IngestInput) (*domain.SourceDocument, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, domain.Validation("kildetitel kræves")
	}
	kind := domain.SourceKind(in.Kind)
	if in.Kind == "" {
		kind = domain.SourceOtherKind
	}
	if !kind.Valid() {
		return nil, domain.Validation("ugyldig kildetype")
	}
	chunks := rag.Split(in.Content)
	if len(chunks) == 0 {
		return nil, domain.Validation("kildeteksten er tom — indsæt selve teksten fra BR18/lokalplanen")
	}
	return s.repo.CreateWithChunks(ctx, &domain.SourceDocument{
		ProjectID:    &projectID,
		Title:        title,
		Kind:         kind,
		VersionLabel: strings.TrimSpace(in.VersionLabel),
		URL:          strings.TrimSpace(in.URL),
		AddedBy:      &userID,
	}, chunks)
}

func (s *Sources) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.SourceDocument, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListForProject(ctx, projectID)
}

func (s *Sources) Delete(ctx context.Context, userID, sourceID uuid.UUID) error {
	src, err := s.repo.Get(ctx, sourceID)
	if err != nil {
		return err
	}
	if src.ProjectID == nil {
		return domain.Validation("kilder i det fælles bibliotek kan ikke slettes fra et projekt")
	}
	if err := requireWrite(ctx, s.access, *src.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, sourceID)
}

func (s *Sources) Search(ctx context.Context, userID, projectID uuid.UUID, query string, limit int) ([]*domain.SourceHit, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, domain.Validation("søgetekst kræves")
	}
	return s.repo.Search(ctx, projectID, query, limit)
}
