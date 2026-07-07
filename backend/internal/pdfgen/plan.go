package pdfgen

import (
	"fmt"
	"math"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// FloorPlan renders the measured 2D model to scale on A4 landscape. Walls
// are drawn with their real thickness, openings are marked, rooms labelled
// with name and area.
func FloorPlan(m Meta, data *domain.DrawingData, drawingTitle, scaleLabel string) ([]byte, error) {
	d := newDoc("L", "Plantegning", m.GeneratedAt)
	d.pdf.AddPage()
	d.h1("Plantegning — " + drawingTitle)

	pageW, pageH := d.pdf.GetPageSize()
	area := planArea{x: 15, y: 30, w: pageW - 30, h: pageH - 55}

	pts := wallPoints(data)
	if len(pts) == 0 {
		d.para("(Tegningen indeholder ingen vægge endnu.)")
		return d.output()
	}
	proj, scaleDenominator := fitProjection(pts, area)

	// Walls with real thickness.
	d.pdf.SetDrawColor(30, 30, 30)
	for _, w := range data.Walls {
		x1, y1 := proj.point(w.From)
		x2, y2 := proj.point(w.To)
		d.pdf.SetLineWidth(math.Max(w.ThicknessMM*proj.scale, 0.3))
		if !w.IsLoadBearing {
			d.pdf.SetDrawColor(120, 120, 120)
		}
		d.pdf.Line(x1, y1, x2, y2)
		d.pdf.SetDrawColor(30, 30, 30)
	}

	// Openings: tick across the wall, blue for windows, brown for doors.
	d.pdf.SetLineWidth(0.5)
	for _, o := range data.Openings {
		wall := findWall(data, o.WallID)
		if wall == nil {
			continue
		}
		cx, cy, nx, ny := openingCenter(*wall, o)
		px, py := proj.point(domain.Point{X: cx, Y: cy})
		half := o.WidthMM / 2 * proj.scale
		if o.Type == domain.OpeningWindow {
			d.pdf.SetDrawColor(40, 90, 200)
		} else {
			d.pdf.SetDrawColor(150, 90, 30)
		}
		d.pdf.Line(px-nx*half, py-ny*half, px+nx*half, py+ny*half)
	}
	d.pdf.SetDrawColor(0, 0, 0)

	// Room labels at polygon centroid.
	d.pdf.SetFont("Helvetica", "", 8)
	for _, room := range data.Rooms {
		cx, cy := centroid(room.Polygon)
		px, py := proj.point(domain.Point{X: cx, Y: cy})
		label := fmt.Sprintf("%s (%.1f m²)", room.Name, domain.PolygonAreaM2(room.Polygon))
		d.pdf.SetXY(px-30, py-2)
		d.pdf.CellFormat(60, 4, d.tr(label), "", 0, "C", false, 0, "")
	}

	// The note sits inside the bottom margin; auto page break must not fire.
	d.pdf.SetAutoPageBreak(false, 0)
	d.pdf.SetFont("Helvetica", "", 9)
	d.pdf.SetXY(15, pageH-22)
	note := fmt.Sprintf("Målestok ca. 1:%d (A4 liggende). Angivet målestok: %s. Mål i mm fra tegnefladen.",
		scaleDenominator, scaleLabel)
	d.pdf.CellFormat(0, 5, d.tr(note), "", 0, "L", false, 0, "")
	return d.output()
}

// SitePlan renders plot boundary and the building footprint placed on it.
func SitePlan(m Meta, data *domain.DrawingData, drawingTitle string) ([]byte, error) {
	d := newDoc("L", "Situationsplan", m.GeneratedAt)
	d.pdf.AddPage()
	d.h1("Situationsplan — " + drawingTitle)

	if data.Plot == nil || len(data.Plot.Boundary) < 3 {
		d.para("Tegningen har ingen grund (skel) endnu. Tilføj grundens skel og bygningens " +
			"placering i tegnefladen for at generere situationsplanen.")
		return d.output()
	}

	pageW, pageH := d.pdf.GetPageSize()
	area := planArea{x: 15, y: 30, w: pageW - 30, h: pageH - 55}

	// Building footprint transformed onto the plot.
	transformed := transformWalls(data)
	pts := append([]domain.Point{}, data.Plot.Boundary...)
	for _, w := range transformed {
		pts = append(pts, w.From, w.To)
	}
	proj, scaleDenominator := fitProjection(pts, area)

	// Plot boundary.
	d.pdf.SetLineWidth(0.4)
	d.pdf.SetDrawColor(20, 120, 20)
	b := data.Plot.Boundary
	for i := range b {
		x1, y1 := proj.point(b[i])
		x2, y2 := proj.point(b[(i+1)%len(b)])
		d.pdf.Line(x1, y1, x2, y2)
	}

	// Building footprint.
	d.pdf.SetDrawColor(30, 30, 30)
	for _, w := range transformed {
		x1, y1 := proj.point(w.From)
		x2, y2 := proj.point(w.To)
		d.pdf.SetLineWidth(math.Max(w.ThicknessMM*proj.scale, 0.3))
		d.pdf.Line(x1, y1, x2, y2)
	}

	d.pdf.SetAutoPageBreak(false, 0)
	if m.Project.PlotAreaM2 != nil {
		d.pdf.SetFont("Helvetica", "", 9)
		d.pdf.SetXY(15, pageH-27)
		d.pdf.CellFormat(0, 5, d.tr(fmt.Sprintf("Registreret grundareal: %.0f m². Skelpolygonens areal på tegningen: %.0f m².",
			*m.Project.PlotAreaM2, domain.PolygonAreaM2(b))), "", 0, "L", false, 0, "")
	}
	d.pdf.SetFont("Helvetica", "", 9)
	d.pdf.SetXY(15, pageH-22)
	d.pdf.CellFormat(0, 5, d.tr(fmt.Sprintf("Målestok ca. 1:%d (A4 liggende). Grøn: skel. Sort: bygning.", scaleDenominator)), "", 0, "L", false, 0, "")
	return d.output()
}

// --- geometry helpers -------------------------------------------------------

type planArea struct{ x, y, w, h float64 }

// projection maps drawing millimetres onto page millimetres.
type projection struct {
	scale        float64
	minX, minY   float64
	offX, offY   float64
}

func (p projection) point(pt domain.Point) (float64, float64) {
	return p.offX + (pt.X-p.minX)*p.scale, p.offY + (pt.Y-p.minY)*p.scale
}

// fitProjection computes the largest round architectural scale (1:50, 1:100,
// 1:200 …) that fits the geometry into the page area, centred.
func fitProjection(pts []domain.Point, area planArea) (projection, int) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
	}
	w, h := maxX-minX, maxY-minY
	if w == 0 {
		w = 1
	}
	if h == 0 {
		h = 1
	}
	rawScale := math.Min(area.w/w, area.h/h) // page mm per drawing mm

	denominators := []int{20, 50, 100, 200, 500, 1000, 2000}
	den := denominators[len(denominators)-1]
	for _, d := range denominators {
		if 1/float64(d) <= rawScale {
			den = d
			break
		}
	}
	scale := 1 / float64(den)

	return projection{
		scale: scale,
		minX:  minX, minY: minY,
		offX: area.x + (area.w-w*scale)/2,
		offY: area.y + (area.h-h*scale)/2,
	}, den
}

func wallPoints(data *domain.DrawingData) []domain.Point {
	pts := make([]domain.Point, 0, len(data.Walls)*2)
	for _, w := range data.Walls {
		pts = append(pts, w.From, w.To)
	}
	for _, r := range data.Rooms {
		pts = append(pts, r.Polygon...)
	}
	return pts
}

func findWall(data *domain.DrawingData, id string) *domain.Wall {
	for i := range data.Walls {
		if data.Walls[i].ID == id {
			return &data.Walls[i]
		}
	}
	return nil
}

// openingCenter returns the opening's midpoint on the wall and the wall's
// unit direction vector.
func openingCenter(w domain.Wall, o domain.Opening) (cx, cy, ux, uy float64) {
	dx, dy := w.To.X-w.From.X, w.To.Y-w.From.Y
	length := math.Hypot(dx, dy)
	if length == 0 {
		return w.From.X, w.From.Y, 1, 0
	}
	ux, uy = dx/length, dy/length
	mid := o.OffsetMM + o.WidthMM/2
	return w.From.X + ux*mid, w.From.Y + uy*mid, ux, uy
}

func centroid(polygon []domain.Point) (float64, float64) {
	var sx, sy float64
	for _, p := range polygon {
		sx += p.X
		sy += p.Y
	}
	n := float64(len(polygon))
	return sx / n, sy / n
}

// transformWalls applies the plot placement (rotation + offset) to the
// building's walls for the site plan.
func transformWalls(data *domain.DrawingData) []domain.Wall {
	if data.Plot == nil {
		return data.Walls
	}
	rad := data.Plot.RotationDeg * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	transform := func(p domain.Point) domain.Point {
		return domain.Point{
			X: data.Plot.Offset.X + p.X*cos - p.Y*sin,
			Y: data.Plot.Offset.Y + p.X*sin + p.Y*cos,
		}
	}
	out := make([]domain.Wall, len(data.Walls))
	for i, w := range data.Walls {
		out[i] = w
		out[i].From = transform(w.From)
		out[i].To = transform(w.To)
	}
	return out
}
