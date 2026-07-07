package calc

import (
	"math"
	"sort"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// ProjectFacts er de målbare kendsgerninger lovlighedstjekket sammenligner
// med reglerne. Alt beregnes deterministisk af tegningens geometri; hvad der
// ikke kan måles, er nil (aldrig et gæt). Assumptions forklarer hvordan
// hvert tal er fremkommet.
type ProjectFacts struct {
	FootprintM2          *float64 `json:"footprintM2"`          // bebygget areal (fodaftryk)
	RoomAreaM2           *float64 `json:"roomAreaM2"`           // indvendigt areal
	PlotAreaM2           *float64 `json:"plotAreaM2"`           // grundareal
	BebyggelsesprocentPct *float64 `json:"bebyggelsesprocentPct"`
	MinSkelafstandM      *float64 `json:"minSkelafstandM"`
	BygningshoejdeM      *float64 `json:"bygningshoejdeM"`
	TaghaeldningDeg      *float64 `json:"taghaeldningDeg"`
	Assumptions          []string `json:"assumptions"`
}

// ComputeFacts måler tegningen op. registeredPlotAreaM2 er projektets
// registrerede grundareal (0 = ukendt); det foretrækkes frem for den
// tegnede skelpolygons areal.
func ComputeFacts(data *domain.DrawingData, registeredPlotAreaM2 float64) *ProjectFacts {
	f := &ProjectFacts{Assumptions: []string{}}

	// --- fodaftryk -----------------------------------------------------------
	if len(data.Walls) >= 3 {
		hull := convexHull(wallEndpoints(data.Walls))
		if len(hull) >= 3 {
			area := polygonArea(hull) / 1e6 // mm² -> m²
			// Vægge tegnes som centerlinjer; halv vægtykkelse lægges til hele
			// vejen rundt via omkredsen.
			maxThickness := 0.0
			for _, w := range data.Walls {
				maxThickness = math.Max(maxThickness, w.ThicknessMM)
			}
			perimeter := polygonPerimeter(hull) / 1000 // m
			area += perimeter * (maxThickness / 2 / 1000)
			f.FootprintM2 = &area
			f.Assumptions = append(f.Assumptions,
				"Fodaftryk beregnet som konveks omkreds af væggenes centerlinjer plus halv vægtykkelse — udhæng og udestuer indgår ikke.")
		}
	}

	if total := data.TotalRoomAreaM2(); total > 0 {
		f.RoomAreaM2 = &total
	}

	// --- grundareal og bebyggelsesprocent -------------------------------------
	if registeredPlotAreaM2 > 0 {
		f.PlotAreaM2 = &registeredPlotAreaM2
	} else if data.Plot != nil && len(data.Plot.Boundary) >= 3 {
		area := polygonArea(data.Plot.Boundary) / 1e6
		f.PlotAreaM2 = &area
		f.Assumptions = append(f.Assumptions,
			"Grundareal målt på den tegnede skelpolygon — brug det registrerede areal fra BBR/skødet når det kendes.")
	}
	if f.FootprintM2 != nil && f.PlotAreaM2 != nil && *f.PlotAreaM2 > 0 {
		pct := *f.FootprintM2 / *f.PlotAreaM2 * 100
		f.BebyggelsesprocentPct = &pct
		f.Assumptions = append(f.Assumptions,
			"Bebyggelsesprocent = fodaftryk/grundareal. Den officielle beregning efter BR18 kan medregne yderligere etageareal — KRÆVER BEKRÆFTELSE.")
	}

	// --- skelafstand ------------------------------------------------------------
	if data.Plot != nil && len(data.Plot.Boundary) >= 3 && len(data.Walls) > 0 {
		transformed := transformPoints(wallSamplePoints(data.Walls), data.Plot)
		minDist := math.Inf(1)
		boundary := data.Plot.Boundary
		for _, p := range transformed {
			for i := range boundary {
				seg := distancePointToSegment(p, boundary[i], boundary[(i+1)%len(boundary)])
				minDist = math.Min(minDist, seg)
			}
		}
		if !math.IsInf(minDist, 1) {
			m := minDist / 1000
			f.MinSkelafstandM = &m
			f.Assumptions = append(f.Assumptions,
				"Skelafstand målt fra væggenes centerlinjer til den tegnede skelpolygon — den faktiske afstand er ca. en halv vægtykkelse kortere.")
		}
	}

	// --- højde og taghældning ------------------------------------------------------
	if len(data.Walls) > 0 {
		wallH := data.WallHeightMM
		if wallH == 0 {
			wallH = 2500
		}
		height := wallH / 1000
		if data.RoofAngleDeg > 0 {
			minX, minY := math.Inf(1), math.Inf(1)
			maxX, maxY := math.Inf(-1), math.Inf(-1)
			for _, p := range wallEndpoints(data.Walls) {
				minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
				minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
			}
			shortSpan := math.Min(maxX-minX, maxY-minY) / 1000 // m
			ridge := shortSpan / 2 * math.Tan(data.RoofAngleDeg*math.Pi/180)
			height += ridge
			f.Assumptions = append(f.Assumptions,
				"Bygningshøjde = væghøjde + tagrejsning over den korte side (saddeltag med ryg langs den lange side). Terrænets niveau indgår ikke — KRÆVER BEKRÆFTELSE.")
		} else {
			f.Assumptions = append(f.Assumptions,
				"Bygningshøjde = væghøjde (fladt tag). Terrænets niveau indgår ikke.")
		}
		f.BygningshoejdeM = &height
		angle := data.RoofAngleDeg
		f.TaghaeldningDeg = &angle
	}

	return f
}

// --- geometri-hjælpere -------------------------------------------------------

func wallEndpoints(walls []domain.Wall) []domain.Point {
	pts := make([]domain.Point, 0, len(walls)*2)
	for _, w := range walls {
		pts = append(pts, w.From, w.To)
	}
	return pts
}

// wallSamplePoints giver endepunkter + midtpunkter, så skelafstand også
// fanges midt på lange vægge.
func wallSamplePoints(walls []domain.Wall) []domain.Point {
	pts := wallEndpoints(walls)
	for _, w := range walls {
		pts = append(pts, domain.Point{X: (w.From.X + w.To.X) / 2, Y: (w.From.Y + w.To.Y) / 2})
	}
	return pts
}

func transformPoints(pts []domain.Point, plot *domain.Plot) []domain.Point {
	rad := plot.RotationDeg * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	out := make([]domain.Point, len(pts))
	for i, p := range pts {
		out[i] = domain.Point{
			X: plot.Offset.X + p.X*cos - p.Y*sin,
			Y: plot.Offset.Y + p.X*sin + p.Y*cos,
		}
	}
	return out
}

func polygonArea(polygon []domain.Point) float64 {
	sum := 0.0
	for i := range polygon {
		j := (i + 1) % len(polygon)
		sum += polygon[i].X*polygon[j].Y - polygon[j].X*polygon[i].Y
	}
	return math.Abs(sum) / 2
}

func polygonPerimeter(polygon []domain.Point) float64 {
	sum := 0.0
	for i := range polygon {
		j := (i + 1) % len(polygon)
		sum += math.Hypot(polygon[j].X-polygon[i].X, polygon[j].Y-polygon[i].Y)
	}
	return sum
}

func distancePointToSegment(p, a, b domain.Point) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	len2 := dx*dx + dy*dy
	if len2 == 0 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / len2
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(p.X-(a.X+t*dx), p.Y-(a.Y+t*dy))
}

// convexHull (Andrew's monotone chain).
func convexHull(pts []domain.Point) []domain.Point {
	if len(pts) < 3 {
		return pts
	}
	sorted := append([]domain.Point{}, pts...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].X != sorted[j].X {
			return sorted[i].X < sorted[j].X
		}
		return sorted[i].Y < sorted[j].Y
	})
	cross := func(o, a, b domain.Point) float64 {
		return (a.X-o.X)*(b.Y-o.Y) - (a.Y-o.Y)*(b.X-o.X)
	}
	var hull []domain.Point
	for _, p := range sorted {
		for len(hull) >= 2 && cross(hull[len(hull)-2], hull[len(hull)-1], p) <= 0 {
			hull = hull[:len(hull)-1]
		}
		hull = append(hull, p)
	}
	lower := len(hull) + 1
	for i := len(sorted) - 2; i >= 0; i-- {
		p := sorted[i]
		for len(hull) >= lower && cross(hull[len(hull)-2], hull[len(hull)-1], p) <= 0 {
			hull = hull[:len(hull)-1]
		}
		hull = append(hull, p)
	}
	return hull[:len(hull)-1]
}
