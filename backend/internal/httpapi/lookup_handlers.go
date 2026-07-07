package httpapi

import (
	"net/http"
	"strconv"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

func (s *Server) lookupAddress(w http.ResponseWriter, r *http.Request) {
	results, err := s.svc.Lookup.SearchAddress(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) lookupAddressDetails(w http.ResponseWriter, r *http.Request) {
	details, err := s.svc.Lookup.AddressDetails(r.Context(), r.PathValue("addressID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, details)
}

func (s *Server) lookupLocalPlans(w http.ResponseWriter, r *http.Request) {
	utmX, err1 := strconv.ParseFloat(r.URL.Query().Get("utmX"), 64)
	utmY, err2 := strconv.ParseFloat(r.URL.Query().Get("utmY"), 64)
	if err1 != nil || err2 != nil {
		writeError(w, domain.Validation("utmX og utmY kræves (ETRS89/UTM32)"))
		return
	}
	plans, err := s.svc.Lookup.LocalPlans(r.Context(), utmX, utmY)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plans)
}
