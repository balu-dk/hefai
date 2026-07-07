package domain

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

type DrawingKind string

const (
	DrawingSitePlan  DrawingKind = "site_plan"
	DrawingFloorPlan DrawingKind = "floor_plan"
	DrawingElevation DrawingKind = "elevation"
	DrawingSection   DrawingKind = "section"
	DrawingDetail    DrawingKind = "detail"
	DrawingOther     DrawingKind = "other"
)

func (k DrawingKind) Valid() bool {
	switch k {
	case DrawingSitePlan, DrawingFloorPlan, DrawingElevation, DrawingSection,
		DrawingDetail, DrawingOther:
		return true
	}
	return false
}

type Drawing struct {
	ID         uuid.UUID   `json:"id"`
	ProjectID  uuid.UUID   `json:"projectId"`
	CaseFileID *uuid.UUID  `json:"caseFileId"`
	Kind       DrawingKind `json:"kind"`
	Title      string      `json:"title"`
	CreatedBy  *uuid.UUID  `json:"createdBy"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

type DrawingVersion struct {
	ID        uuid.UUID   `json:"id"`
	DrawingID uuid.UUID   `json:"drawingId"`
	VersionNo int         `json:"versionNo"`
	Data      DrawingData `json:"data"`
	Scale     string      `json:"scale"`
	Note      string      `json:"note"`
	CreatedBy *uuid.UUID  `json:"createdBy"`
	CreatedAt time.Time   `json:"createdAt"`
}

// DrawingData is the measured 2D model produced by the drawing canvas. All
// coordinates are millimetres in a project-local coordinate system.
type DrawingData struct {
	Walls    []Wall      `json:"walls"`
	Rooms    []RoomShape `json:"rooms"`
	Openings []Opening   `json:"openings"`
	Plot     *Plot       `json:"plot,omitempty"`
	Trees    []Tree      `json:"trees,omitempty"`
	// Building envelope for the 3D view (and later for calculations).
	WallHeightMM float64 `json:"wallHeightMm,omitempty"` // default 2500 when 0
	RoofAngleDeg float64 `json:"roofAngleDeg,omitempty"` // 0 = flat roof
	// Geo anchors the drawing to the real world so an orthophoto
	// (satellit-/luftfoto) can be draped under plot and building.
	Geo *GeoAnchor `json:"geo,omitempty"`
}

// GeoAnchor is the real-world point at the plot centroid (or drawing origin
// when no plot is drawn), plus the photo window size in metres.
type GeoAnchor struct {
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	SizeM float64 `json:"sizeM"` // default 150 when 0
}

// Tree is planted in plot coordinates (same system as Plot.Boundary).
type Tree struct {
	Position      Point   `json:"position"`
	HeightMM      float64 `json:"heightMm"`
	CrownDiameterMM float64 `json:"crownDiameterMm"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Wall struct {
	ID            string  `json:"id"`
	From          Point   `json:"from"`
	To            Point   `json:"to"`
	ThicknessMM   float64 `json:"thicknessMm"`
	IsLoadBearing bool    `json:"isLoadBearing"`
}

type RoomShape struct {
	Name    string  `json:"name"`
	Polygon []Point `json:"polygon"`
}

type OpeningType string

const (
	OpeningDoor   OpeningType = "door"
	OpeningWindow OpeningType = "window"
)

type Opening struct {
	WallID   string      `json:"wallId"`
	Type     OpeningType `json:"type"`
	OffsetMM float64     `json:"offsetMm"` // along the wall from its start point
	WidthMM  float64     `json:"widthMm"`
	HeightMM float64     `json:"heightMm"`
}

// Plot describes the site: boundary polygon and where the building footprint
// sits on it (offset of the drawing origin, rotation in degrees).
type Plot struct {
	Boundary    []Point `json:"boundary"`
	Offset      Point   `json:"offset"`
	RotationDeg float64 `json:"rotationDeg"`
}

// Validate checks geometric sanity so downstream calculations (areas,
// generated drawings) can trust the data.
func (d *DrawingData) Validate() error {
	wallIDs := map[string]bool{}
	for i, w := range d.Walls {
		if w.ID == "" {
			return Validation(fmt.Sprintf("væg %d mangler id", i))
		}
		if wallIDs[w.ID] {
			return Validation("væg-id " + w.ID + " er ikke unikt")
		}
		wallIDs[w.ID] = true
		if w.From == w.To {
			return Validation("væg " + w.ID + " har nul længde")
		}
		if w.ThicknessMM <= 0 || w.ThicknessMM > 2000 {
			return Validation("væg " + w.ID + " har urealistisk tykkelse")
		}
	}
	for _, r := range d.Rooms {
		if r.Name == "" {
			return Validation("alle rum skal have navn")
		}
		if len(r.Polygon) < 3 {
			return Validation("rummet " + r.Name + " skal have mindst 3 punkter")
		}
	}
	for _, o := range d.Openings {
		if !wallIDs[o.WallID] {
			return Validation("åbning refererer til ukendt væg " + o.WallID)
		}
		if o.Type != OpeningDoor && o.Type != OpeningWindow {
			return Validation("ugyldig åbningstype")
		}
		if o.WidthMM <= 0 || o.HeightMM <= 0 {
			return Validation("åbninger skal have positiv bredde og højde")
		}
	}
	if d.Plot != nil && len(d.Plot.Boundary) < 3 {
		return Validation("grundens skel skal have mindst 3 punkter")
	}
	for i, t := range d.Trees {
		if t.HeightMM <= 0 || t.HeightMM > 40000 {
			return Validation(fmt.Sprintf("træ %d skal have en højde mellem 0 og 40 m", i+1))
		}
		if t.CrownDiameterMM <= 0 || t.CrownDiameterMM > 30000 {
			return Validation(fmt.Sprintf("træ %d skal have en kronediameter mellem 0 og 30 m", i+1))
		}
	}
	if d.WallHeightMM < 0 || d.WallHeightMM > 12000 {
		return Validation("væghøjden skal være mellem 0 og 12 m")
	}
	if d.Geo != nil {
		if d.Geo.Lat < -90 || d.Geo.Lat > 90 || d.Geo.Lon < -180 || d.Geo.Lon > 180 {
			return Validation("geokoordinaterne er ugyldige")
		}
		if d.Geo.SizeM < 0 || d.Geo.SizeM > 2000 {
			return Validation("fotoudsnittet skal være mellem 0 og 2000 m")
		}
	}
	if d.RoofAngleDeg < 0 || d.RoofAngleDeg >= 90 {
		return Validation("taghældningen skal være mellem 0 og 90 grader")
	}
	return nil
}

// PolygonAreaM2 computes the area of a polygon given in millimetres, in m²,
// via the shoelace formula.
func PolygonAreaM2(polygon []Point) float64 {
	if len(polygon) < 3 {
		return 0
	}
	sum := 0.0
	for i := range polygon {
		j := (i + 1) % len(polygon)
		sum += polygon[i].X*polygon[j].Y - polygon[j].X*polygon[i].Y
	}
	return math.Abs(sum) / 2 / 1e6 // mm² -> m²
}

// RoomAreas returns each named room's floor area in m².
func (d *DrawingData) RoomAreas() map[string]float64 {
	areas := make(map[string]float64, len(d.Rooms))
	for _, r := range d.Rooms {
		areas[r.Name] = PolygonAreaM2(r.Polygon)
	}
	return areas
}

// TotalRoomAreaM2 sums all room areas (indvendigt areal).
func (d *DrawingData) TotalRoomAreaM2() float64 {
	total := 0.0
	for _, a := range d.RoomAreas() {
		total += a
	}
	return total
}
