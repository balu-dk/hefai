package httpapi

import (
	"net/http"
	"strconv"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// orthoImage proxyer et luftfoto-udsnit (JPEG) omkring de angivne
// koordinater. Query: lat, lon, sizeM (valgfri, standard 150).
func (s *Server) orthoImage(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	q := r.URL.Query()
	lat, err1 := strconv.ParseFloat(q.Get("lat"), 64)
	lon, err2 := strconv.ParseFloat(q.Get("lon"), 64)
	if err1 != nil || err2 != nil {
		writeError(w, domain.Validation("lat og lon kræves som decimaltal"))
		return
	}
	sizeM, _ := strconv.ParseFloat(q.Get("sizeM"), 64)

	img, contentType, err := s.svc.Ortho.Fetch(r.Context(), userID(r), projectID, lat, lon, sizeM)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(img)
}

// orthoStatus lader frontenden vise/skjule funktionen.
func (s *Server) orthoStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"configured": s.svc.Ortho.Configured()})
}
