package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// Lookup slår adresser og plangrundlag op i Danmarks frie, nøglefri
// datakilder: DAWA (adresser, matrikel, grundareal, koordinater) og
// Plandata (gældende lokalplan for et punkt). Ingen data gemmes her —
// resultaterne bruges til at udfylde projektet og pege på dokumenter.
type Lookup struct {
	dawaBase    string
	plandataWFS string
	client      *http.Client
}

func NewLookup(dawaBase, plandataWFS string) *Lookup {
	return &Lookup{
		dawaBase:    strings.TrimSuffix(dawaBase, "/"),
		plandataWFS: plandataWFS,
		client:      &http.Client{Timeout: 15 * time.Second},
	}
}

// --- adresse-autocomplete --------------------------------------------------------

type AddressSuggestion struct {
	Text string `json:"text"`
	ID   string `json:"id"`
}

func (s *Lookup) SearchAddress(ctx context.Context, query string) ([]AddressSuggestion, error) {
	query = strings.TrimSpace(query)
	if len(query) < 3 {
		return []AddressSuggestion{}, nil
	}
	var raw []struct {
		Tekst          string `json:"tekst"`
		Adgangsadresse struct {
			ID string `json:"id"`
		} `json:"adgangsadresse"`
	}
	err := s.getJSON(ctx, s.dawaBase+"/adgangsadresser/autocomplete?per_side=8&q="+url.QueryEscape(query), &raw)
	if err != nil {
		return nil, err
	}
	out := make([]AddressSuggestion, 0, len(raw))
	for _, r := range raw {
		out = append(out, AddressSuggestion{Text: r.Tekst, ID: r.Adgangsadresse.ID})
	}
	return out, nil
}

// --- adresse-detaljer ---------------------------------------------------------------

// AddressDetails er alt hvad ét adresse-opslag kan udfylde i projektet.
type AddressDetails struct {
	Address      string   `json:"address"`
	Municipality string   `json:"municipality"`
	CadastralID  string   `json:"cadastralId"` // matrikelnr + ejerlav
	PlotAreaM2   *float64 `json:"plotAreaM2"`  // registreret areal fra matriklen
	Lat          float64  `json:"lat"`
	Lon          float64  `json:"lon"`
	UTMX         float64  `json:"utmX"` // ETRS89/UTM32 til plandata-opslag
	UTMY         float64  `json:"utmY"`
}

// dawaMini er DAWA's flade "mini"-struktur for en adgangsadresse.
type dawaMini struct {
	Betegnelse  string  `json:"betegnelse"`
	Kommunekode string  `json:"kommunekode"`
	X           float64 `json:"x"` // lon i WGS84 / øst i UTM alt efter srid
	Y           float64 `json:"y"`
}

func (s *Lookup) AddressDetails(ctx context.Context, accessAddressID string) (*AddressDetails, error) {
	if accessAddressID == "" {
		return nil, domain.Validation("adresse-id mangler")
	}
	base := s.dawaBase + "/adgangsadresser/" + url.PathEscape(accessAddressID)

	var addr dawaMini
	if err := s.getJSON(ctx, base+"?struktur=mini", &addr); err != nil {
		return nil, err
	}
	var utm dawaMini
	if err := s.getJSON(ctx, base+"?struktur=mini&srid=25832", &utm); err != nil {
		return nil, err
	}

	details := &AddressDetails{
		Address: addr.Betegnelse,
		Lon:     addr.X,
		Lat:     addr.Y,
		UTMX:    utm.X,
		UTMY:    utm.Y,
	}
	// Kommunenavn slås op via koden (mini-strukturen har kun koden).
	if addr.Kommunekode != "" {
		var kommune struct {
			Navn string `json:"navn"`
		}
		if err := s.getJSON(ctx, s.dawaBase+"/kommuner/"+url.PathEscape(strings.TrimLeft(addr.Kommunekode, "0")), &kommune); err == nil {
			details.Municipality = kommune.Navn
		}
	}

	// Jordstykkets registrerede areal (grundareal) fra matriklen.
	if details.Lat != 0 && details.Lon != 0 {
		var jord []struct {
			Registreretareal *float64 `json:"registreretareal"`
			Matrikelnr       string   `json:"matrikelnr"`
			Ejerlav          struct {
				Navn string `json:"navn"`
			} `json:"ejerlav"`
		}
		jordErr := s.getJSON(ctx, fmt.Sprintf("%s/jordstykker?x=%f&y=%f", s.dawaBase, details.Lon, details.Lat), &jord)
		if jordErr == nil && len(jord) > 0 {
			details.PlotAreaM2 = jord[0].Registreretareal
			if details.CadastralID == "" && jord[0].Matrikelnr != "" {
				details.CadastralID = strings.TrimSpace(jord[0].Matrikelnr + " " + jord[0].Ejerlav.Navn)
			}
		}
	}
	return details, nil
}

// --- lokalplan-opslag -----------------------------------------------------------------

type LocalPlan struct {
	PlanID   string `json:"planId"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	DocLink  string `json:"docLink"` // PDF hos plandata.dk
}

// LocalPlans finder vedtagne lokalplaner der dækker punktet (ETRS89/UTM32).
func (s *Lookup) LocalPlans(ctx context.Context, utmX, utmY float64) ([]LocalPlan, error) {
	if utmX == 0 || utmY == 0 {
		return nil, domain.Validation("koordinater mangler — slå adressen op først")
	}
	params := url.Values{
		"SERVICE":      {"WFS"},
		"VERSION":      {"2.0.0"},
		"REQUEST":      {"GetFeature"},
		"TYPENAMES":    {"pdk:theme_pdk_lokalplan_vedtaget"},
		"OUTPUTFORMAT": {"application/json"},
		"SRSNAME":      {"EPSG:25832"},
		"CQL_FILTER":   {fmt.Sprintf("INTERSECTS(geometri, POINT(%f %f))", utmX, utmY)},
	}
	var fc struct {
		Features []struct {
			Properties map[string]any `json:"properties"`
		} `json:"features"`
	}
	if err := s.getJSON(ctx, s.plandataWFS+"?"+params.Encode(), &fc); err != nil {
		return nil, fmt.Errorf("plandata-opslag fejlede: %w", err)
	}
	plans := []LocalPlan{}
	for _, f := range fc.Features {
		plan := LocalPlan{
			PlanID:  str(f.Properties["planid"]),
			Name:    firstStr(f.Properties, "plannavn", "titel", "navn"),
			Status:  firstStr(f.Properties, "status", "plst"),
			DocLink: firstStr(f.Properties, "doklink", "dokumentlink"),
		}
		if plan.Name != "" || plan.DocLink != "" {
			plans = append(plans, plan)
		}
	}
	return plans, nil
}

func (s *Lookup) getJSON(ctx context.Context, rawURL string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("opslag fejlede: %w", err)
	}
	defer resp.Body.Close()
	payload, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	// Nogle proxier lukker forbindelsen uden afsluttende chunk; er der
	// modtaget data, afgør JSON-parseren om svaret var komplet.
	if readErr != nil && !(errors.Is(readErr, io.ErrUnexpectedEOF) && len(payload) > 0) {
		return readErr
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("opslagstjenesten svarede %d: %.150s", resp.StatusCode, payload)
	}
	return json.Unmarshal(payload, into)
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%.0f", f)
	}
	return ""
}

func firstStr(props map[string]any, keys ...string) string {
	for _, k := range keys {
		if v := str(props[k]); v != "" {
			return v
		}
	}
	return ""
}
