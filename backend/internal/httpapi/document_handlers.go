package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/balu-dk/hefai/backend/internal/domain"
	"github.com/balu-dk/hefai/backend/internal/service"
)

// maxUploadBytes caps document uploads (drawings, photos, PDFs).
const maxUploadBytes = 100 << 20 // 100 MiB

func (s *Server) uploadDocument(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, domain.Validation("ugyldig upload (maks. 100 MB): "+err.Error()))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, domain.Validation("feltet 'file' mangler"))
		return
	}
	defer file.Close()

	in := service.UploadInput{
		Kind:        r.FormValue("kind"),
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Filename:    header.Filename,
		MimeType:    header.Header.Get("Content-Type"),
		Content:     file,
	}
	if capturedAt := r.FormValue("capturedAt"); capturedAt != "" {
		t, err := time.Parse(time.RFC3339, capturedAt)
		if err != nil {
			writeError(w, domain.Validation("capturedAt skal være RFC3339"))
			return
		}
		in.CapturedAt = &t
	}

	doc, err := s.svc.Documents.Upload(r.Context(), userID(r), projectID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	q := r.URL.Query()
	docs, err := s.svc.Documents.List(r.Context(), userID(r), projectID, service.DocumentListFilter{
		Kind:       q.Get("kind"),
		Query:      q.Get("q"),
		Tag:        q.Get("tag"),
		TargetType: q.Get("targetType"),
		TargetID:   q.Get("targetId"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, docs)
}

// documentContent streams the file for in-browser viewing (PDF, images).
// http.ServeContent handles range requests, which PDF viewers rely on.
func (s *Server) documentContent(w http.ResponseWriter, r *http.Request) {
	docID, err := pathUUID(r, "documentID")
	if err != nil {
		writeError(w, err)
		return
	}
	doc, content, err := s.svc.Documents.OpenContent(r.Context(), userID(r), docID)
	if err != nil {
		writeError(w, err)
		return
	}
	defer content.Close()

	w.Header().Set("Content-Type", doc.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", doc.Filename))
	http.ServeContent(w, r, doc.Filename, doc.UpdatedAt, content)
}

func (s *Server) setDocumentTags(w http.ResponseWriter, r *http.Request) {
	docID, err := pathUUID(r, "documentID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in service.TagsInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	doc, err := s.svc.Documents.SetTags(r.Context(), userID(r), docID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) removeDocumentLink(w http.ResponseWriter, r *http.Request) {
	docID, err := pathUUID(r, "documentID")
	if err != nil {
		writeError(w, err)
		return
	}
	linkID, err := pathUUID(r, "linkID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.svc.Documents.RemoveLink(r.Context(), userID(r), docID, linkID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
