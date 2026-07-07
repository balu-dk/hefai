package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock-DAWA med realistiske svar-strukturer.
func mockDAWA(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/adgangsadresser/autocomplete", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`[
			{"tekst":"Strandvejen 12, 4873 Væggerløse","adgangsadresse":{"id":"abc-123"}}
		]`))
	})
	mux.HandleFunc("/adgangsadresser/abc-123", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("srid") == "25832" {
			w.Write([]byte(`{"betegnelse":"Strandvejen 12, 4873 Væggerløse",
				"kommunekode":"0376","x":690123.4,"y":6072345.6}`))
			return
		}
		w.Write([]byte(`{"betegnelse":"Strandvejen 12, 4873 Væggerløse",
			"kommunekode":"0376","x":11.9702,"y":54.7639}`))
	})
	mux.HandleFunc("/kommuner/376", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"navn":"Guldborgsund"}`))
	})
	mux.HandleFunc("/jordstykker", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`[{"registreretareal":1214,"matrikelnr":"12b","ejerlav":{"navn":"Marielyst"}}]`))
	})
	return httptest.NewServer(mux)
}

func TestSearchAddress(t *testing.T) {
	dawa := mockDAWA(t)
	defer dawa.Close()
	lookup := NewLookup(dawa.URL, "")

	results, err := lookup.SearchAddress(context.Background(), "strandvejen 12")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != "abc-123" {
		t.Errorf("unexpected results: %+v", results)
	}
	// Korte søgninger rammer aldrig nettet.
	short, err := lookup.SearchAddress(context.Background(), "ab")
	if err != nil || len(short) != 0 {
		t.Error("short queries must return empty without calling DAWA")
	}
}

func TestAddressDetails(t *testing.T) {
	dawa := mockDAWA(t)
	defer dawa.Close()
	lookup := NewLookup(dawa.URL, "")

	d, err := lookup.AddressDetails(context.Background(), "abc-123")
	if err != nil {
		t.Fatal(err)
	}
	if d.Address != "Strandvejen 12, 4873 Væggerløse" {
		t.Errorf("address = %q", d.Address)
	}
	if d.Municipality != "Guldborgsund" {
		t.Errorf("municipality = %q", d.Municipality)
	}
	if d.CadastralID != "12b Marielyst" {
		t.Errorf("matrikel = %q (skal komme fra jordstykke-opslaget)", d.CadastralID)
	}
	if d.Lat != 54.7639 || d.Lon != 11.9702 {
		t.Errorf("coords = %v,%v", d.Lat, d.Lon)
	}
	if d.PlotAreaM2 == nil || *d.PlotAreaM2 != 1214 {
		t.Errorf("plot area = %v, want 1214", d.PlotAreaM2)
	}
	if d.UTMX != 690123.4 || d.UTMY != 6072345.6 {
		t.Errorf("utm = %v,%v", d.UTMX, d.UTMY)
	}
}

func TestLocalPlans(t *testing.T) {
	wfs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("CQL_FILTER") == "" {
			t.Error("CQL point filter missing")
		}
		w.Write([]byte(`{"features":[{"properties":{
			"planid":"1234567","plannavn":"Lokalplan 42 — Sommerhusområde Marielyst",
			"status":"V","doklink":"https://dokument.plandata.dk/1234567.pdf"}}]}`))
	}))
	defer wfs.Close()
	lookup := NewLookup("", wfs.URL)

	plans, err := lookup.LocalPlans(context.Background(), 690123.4, 6072345.6)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Name != "Lokalplan 42 — Sommerhusområde Marielyst" {
		t.Errorf("plans = %+v", plans)
	}
	if plans[0].DocLink == "" {
		t.Error("doc link missing")
	}
	if _, err := lookup.LocalPlans(context.Background(), 0, 0); err == nil {
		t.Error("missing coordinates must be rejected")
	}
}
