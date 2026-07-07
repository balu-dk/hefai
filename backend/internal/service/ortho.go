package service

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// Ortho henter luft-/satellitfotos fra en WMS-tjeneste og proxyer dem til
// frontenden, så API-tokenet aldrig forlader serveren. Standardopsætningen
// er Dataforsyningens frie danske ortofoto (kræver gratis token fra
// dataforsyningen.dk), men enhver WMS kan konfigureres.
type Ortho struct {
	wmsURL string
	layer  string
	token  string
	access ProjectAccess
	client *http.Client
}

func NewOrtho(wmsURL, layer, token string, access ProjectAccess) *Ortho {
	return &Ortho{
		wmsURL: wmsURL,
		layer:  layer,
		token:  token,
		access: access,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Configured melder om tjenesten kan bruges (token sat).
func (s *Ortho) Configured() bool { return s.token != "" && s.wmsURL != "" }

// degreeWindow omregner et kvadratisk vindue i meter omkring (lat, lon) til
// et grader-bbox. For vinduer op til et par kilometer er den sfæriske
// tilnærmelse (1° bredde = 111 320 m · cos(lat)) rigeligt præcis.
func degreeWindow(lat, lon, sizeM float64) (minLon, minLat, maxLon, maxLat float64) {
	const metersPerDegree = 111320.0
	dLat := sizeM / 2 / metersPerDegree
	dLon := sizeM / 2 / (metersPerDegree * math.Cos(lat*math.Pi/180))
	return lon - dLon, lat - dLat, lon + dLon, lat + dLat
}

// Fetch henter fotoudsnittet som JPEG. Kvadratisk, 1024×1024 px.
func (s *Ortho) Fetch(ctx context.Context, userID, projectID uuid.UUID, lat, lon, sizeM float64) ([]byte, string, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, "", err
	}
	if !s.Configured() {
		return nil, "", domain.Validation("luftfoto er ikke sat op — få et gratis token på dataforsyningen.dk " +
			"og sæt ORTHO_TOKEN i serverens miljø")
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return nil, "", domain.Validation("ugyldige koordinater")
	}
	if sizeM <= 0 {
		sizeM = 150
	}
	if sizeM < 20 || sizeM > 2000 {
		return nil, "", domain.Validation("fotoudsnittet skal være mellem 20 og 2000 m")
	}

	minLon, minLat, maxLon, maxLat := degreeWindow(lat, lon, sizeM)

	// WMS 1.1.1: bbox i lon,lat-rækkefølge for EPSG:4326.
	params := url.Values{
		"SERVICE":     {"WMS"},
		"VERSION":     {"1.1.1"},
		"REQUEST":     {"GetMap"},
		"LAYERS":      {s.layer},
		"STYLES":      {""},
		"SRS":         {"EPSG:4326"},
		"BBOX":        {fmt.Sprintf("%f,%f,%f,%f", minLon, minLat, maxLon, maxLat)},
		"WIDTH":       {"1024"},
		"HEIGHT":      {"1024"},
		"FORMAT":      {"image/jpeg"},
		"TRANSPARENT": {"false"},
		"token":       {s.token},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.wmsURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("kunne ikke hente luftfoto: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, "", err
	}
	contentType := resp.Header.Get("Content-Type")
	if resp.StatusCode != http.StatusOK || !isImage(contentType) {
		return nil, "", domain.Validation(fmt.Sprintf(
			"fototjenesten svarede %d (%s) — tjek token og lag: %.200s",
			resp.StatusCode, contentType, payload))
	}
	return payload, contentType, nil
}

func isImage(contentType string) bool {
	return len(contentType) >= 6 && contentType[:6] == "image/"
}
