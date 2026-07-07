package service

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type DocumentRepo interface {
	Create(ctx context.Context, d *domain.Document) (*domain.Document, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Document, error)
	List(ctx context.Context, projectID uuid.UUID, f domain.DocumentFilter) ([]*domain.Document, error)
	Update(ctx context.Context, d *domain.Document) (*domain.Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SetTags(ctx context.Context, docID, projectID uuid.UUID, names []string) error
	TargetProjectID(ctx context.Context, t domain.LinkTargetType, id uuid.UUID) (uuid.UUID, error)
	CreateLink(ctx context.Context, docID uuid.UUID, t domain.LinkTargetType, targetID uuid.UUID) (*domain.DocumentLink, error)
	DeleteLink(ctx context.Context, linkID uuid.UUID) error
	GetLink(ctx context.Context, linkID uuid.UUID) (*domain.DocumentLink, error)
	ListLinks(ctx context.Context, docID uuid.UUID) ([]*domain.DocumentLink, error)
}

// FileStore is the blob storage behind document content.
type FileStore interface {
	Save(key string, r io.Reader) (int64, error)
	Open(key string) (io.ReadSeekCloser, error)
	Delete(key string) error
}

type Documents struct {
	repo   DocumentRepo
	files  FileStore
	access ProjectAccess
}

func NewDocuments(repo DocumentRepo, files FileStore, access ProjectAccess) *Documents {
	return &Documents{repo: repo, files: files, access: access}
}

type UploadInput struct {
	Kind        string
	Title       string
	Description string
	Filename    string
	MimeType    string
	CapturedAt  *time.Time
	Content     io.Reader
}

func (s *Documents) Upload(ctx context.Context, userID, projectID uuid.UUID, in UploadInput) (*domain.Document, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	kind := domain.DocumentKind(in.Kind)
	if in.Kind == "" {
		kind = domain.DocOther
	}
	if !kind.Valid() {
		return nil, domain.Validation("ugyldig dokumenttype")
	}
	filename := filepath.Base(strings.TrimSpace(in.Filename))
	if filename == "" || filename == "." {
		return nil, domain.Validation("filnavn kræves")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = filename
	}
	mimeType := in.MimeType
	if mimeType == "" || mimeType == "application/octet-stream" {
		if byExt := mime.TypeByExtension(filepath.Ext(filename)); byExt != "" {
			mimeType = byExt
		} else if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	key := fmt.Sprintf("%s/%s%s", projectID, uuid.New(), strings.ToLower(filepath.Ext(filename)))
	size, err := s.files.Save(key, in.Content)
	if err != nil {
		return nil, err
	}

	doc, err := s.repo.Create(ctx, &domain.Document{
		ProjectID:   projectID,
		UploadedBy:  &userID,
		Kind:        kind,
		Title:       title,
		Description: in.Description,
		Filename:    filename,
		StorageKey:  key,
		MimeType:    mimeType,
		SizeBytes:   size,
		CapturedAt:  in.CapturedAt,
	})
	if err != nil {
		_ = s.files.Delete(key) // don't leave orphaned blobs behind
		return nil, err
	}
	return doc, nil
}

type DocumentListFilter struct {
	Kind       string
	Query      string
	Tag        string
	TargetType string
	TargetID   string
}

func (s *Documents) List(ctx context.Context, userID, projectID uuid.UUID, in DocumentListFilter) ([]*domain.Document, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	f := domain.DocumentFilter{Query: strings.TrimSpace(in.Query), Tag: strings.TrimSpace(in.Tag)}
	if in.Kind != "" {
		f.Kind = domain.DocumentKind(in.Kind)
		if !f.Kind.Valid() {
			return nil, domain.Validation("ugyldig dokumenttype")
		}
	}
	if in.TargetType != "" {
		f.TargetType = domain.LinkTargetType(in.TargetType)
		if !f.TargetType.Valid() {
			return nil, domain.Validation("ugyldig link-type")
		}
		id, err := uuid.Parse(in.TargetID)
		if err != nil {
			return nil, domain.Validation("targetId kræves sammen med targetType")
		}
		f.TargetID = id
	}
	return s.repo.List(ctx, projectID, f)
}

func (s *Documents) Get(ctx context.Context, userID, docID uuid.UUID) (*domain.Document, error) {
	doc, err := s.repo.Get(ctx, docID)
	if err != nil {
		return nil, err
	}
	if _, err := requireRead(ctx, s.access, doc.ProjectID, userID); err != nil {
		return nil, err
	}
	return doc, nil
}

// OpenContent returns the document metadata and a reader over its bytes.
// The caller must close the reader.
func (s *Documents) OpenContent(ctx context.Context, userID, docID uuid.UUID) (*domain.Document, io.ReadSeekCloser, error) {
	doc, err := s.Get(ctx, userID, docID)
	if err != nil {
		return nil, nil, err
	}
	f, err := s.files.Open(doc.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return doc, f, nil
}

type DocumentPatch struct {
	Kind        *string    `json:"kind"`
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	CapturedAt  *time.Time `json:"capturedAt"`
}

func (s *Documents) Update(ctx context.Context, userID, docID uuid.UUID, patch DocumentPatch) (*domain.Document, error) {
	doc, err := s.repo.Get(ctx, docID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, doc.ProjectID, userID); err != nil {
		return nil, err
	}
	if patch.Kind != nil {
		kind := domain.DocumentKind(*patch.Kind)
		if !kind.Valid() {
			return nil, domain.Validation("ugyldig dokumenttype")
		}
		doc.Kind = kind
	}
	if patch.Title != nil {
		doc.Title = strings.TrimSpace(*patch.Title)
		if doc.Title == "" {
			return nil, domain.Validation("titel kan ikke være tom")
		}
	}
	if patch.Description != nil {
		doc.Description = *patch.Description
	}
	if patch.CapturedAt != nil {
		doc.CapturedAt = patch.CapturedAt
	}
	return s.repo.Update(ctx, doc)
}

func (s *Documents) Delete(ctx context.Context, userID, docID uuid.UUID) error {
	doc, err := s.repo.Get(ctx, docID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, doc.ProjectID, userID); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, docID); err != nil {
		return err
	}
	return s.files.Delete(doc.StorageKey)
}

type TagsInput struct {
	Tags []string `json:"tags"`
}

func (s *Documents) SetTags(ctx context.Context, userID, docID uuid.UUID, in TagsInput) (*domain.Document, error) {
	doc, err := s.repo.Get(ctx, docID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, doc.ProjectID, userID); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	names := []string{}
	for _, name := range in.Tags {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	if err := s.repo.SetTags(ctx, docID, doc.ProjectID, names); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, docID)
}

type LinkInput struct {
	TargetType string    `json:"targetType"`
	TargetID   uuid.UUID `json:"targetId"`
}

func (s *Documents) AddLink(ctx context.Context, userID, docID uuid.UUID, in LinkInput) (*domain.DocumentLink, error) {
	doc, err := s.repo.Get(ctx, docID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, doc.ProjectID, userID); err != nil {
		return nil, err
	}
	targetType := domain.LinkTargetType(in.TargetType)
	if !targetType.Valid() {
		return nil, domain.Validation("ugyldig link-type")
	}
	targetProject, err := s.repo.TargetProjectID(ctx, targetType, in.TargetID)
	if err != nil {
		return nil, err
	}
	if targetProject != doc.ProjectID {
		return nil, domain.Validation("målet tilhører et andet projekt")
	}
	return s.repo.CreateLink(ctx, docID, targetType, in.TargetID)
}

func (s *Documents) ListLinks(ctx context.Context, userID, docID uuid.UUID) ([]*domain.DocumentLink, error) {
	if _, err := s.Get(ctx, userID, docID); err != nil {
		return nil, err
	}
	return s.repo.ListLinks(ctx, docID)
}

func (s *Documents) RemoveLink(ctx context.Context, userID, docID, linkID uuid.UUID) error {
	doc, err := s.repo.Get(ctx, docID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, doc.ProjectID, userID); err != nil {
		return err
	}
	link, err := s.repo.GetLink(ctx, linkID)
	if err != nil {
		return err
	}
	if link.DocumentID != docID {
		return domain.ErrNotFound
	}
	return s.repo.DeleteLink(ctx, linkID)
}
