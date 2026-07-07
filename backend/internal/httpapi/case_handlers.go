package httpapi

import (
	"net/http"
	"strconv"
)

func (s *Server) searchSources(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	hits, err := s.svc.Sources.Search(r.Context(), userID(r), projectID, r.URL.Query().Get("q"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, hits)
}
