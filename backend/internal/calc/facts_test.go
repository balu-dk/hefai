package calc

import (
	"math"
	"testing"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// Hus 10×8 m (centerlinjer), vægtykkelse 300 mm, placeret med offset
// (5000, 6000) uden rotation i en 30×40 m grund med skel i (0,0)-(30000,40000).
func testDrawing() *domain.DrawingData {
	return &domain.DrawingData{
		WallHeightMM: 2500,
		RoofAngleDeg: 30,
		Walls: []domain.Wall{
			{ID: "n", From: domain.Point{X: 0, Y: 0}, To: domain.Point{X: 10000, Y: 0}, ThicknessMM: 300},
			{ID: "e", From: domain.Point{X: 10000, Y: 0}, To: domain.Point{X: 10000, Y: 8000}, ThicknessMM: 300},
			{ID: "s", From: domain.Point{X: 10000, Y: 8000}, To: domain.Point{X: 0, Y: 8000}, ThicknessMM: 300},
			{ID: "w", From: domain.Point{X: 0, Y: 8000}, To: domain.Point{X: 0, Y: 0}, ThicknessMM: 300},
		},
		Rooms: []domain.RoomShape{{
			Name:    "Alt",
			Polygon: []domain.Point{{X: 150, Y: 150}, {X: 9850, Y: 150}, {X: 9850, Y: 7850}, {X: 150, Y: 7850}},
		}},
		Plot: &domain.Plot{
			Boundary: []domain.Point{{X: 0, Y: 0}, {X: 30000, Y: 0}, {X: 30000, Y: 40000}, {X: 0, Y: 40000}},
			Offset:   domain.Point{X: 5000, Y: 6000},
		},
	}
}

func approxFact(t *testing.T, got *float64, want, tol float64, what string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil", what)
	}
	if math.Abs(*got-want) > tol {
		t.Errorf("%s = %v, want %v (±%v)", what, *got, want, tol)
	}
}

func TestComputeFacts(t *testing.T) {
	f := ComputeFacts(testDrawing(), 1200)

	// Fodaftryk: 10×8 = 80 m² + omkreds 36 m × 0,15 m = 85,4 m².
	approxFact(t, f.FootprintM2, 85.4, 0.01, "footprint")
	// Indvendigt: 9,7 × 7,7 = 74,69 m².
	approxFact(t, f.RoomAreaM2, 74.69, 0.01, "room area")
	// Registreret grundareal foretrækkes.
	approxFact(t, f.PlotAreaM2, 1200, 1e-9, "plot area")
	// Bebyggelsesprocent: 85,4/1200 = 7,117 %.
	approxFact(t, f.BebyggelsesprocentPct, 85.4/1200*100, 0.01, "bebyggelsesprocent")
	// Min. skelafstand: huset ligger (5000,6000)-(15000,14000); nærmeste
	// kant er vest-skellet: 5000 mm = 5 m.
	approxFact(t, f.MinSkelafstandM, 5.0, 1e-9, "skelafstand")
	// Højde: 2,5 + (8/2)·tan30° = 2,5 + 2,309 = 4,809 m.
	approxFact(t, f.BygningshoejdeM, 2.5+4*math.Tan(30*math.Pi/180), 0.001, "højde")
	approxFact(t, f.TaghaeldningDeg, 30, 1e-9, "taghældning")

	if len(f.Assumptions) < 3 {
		t.Error("facts must document their assumptions")
	}
}

func TestComputeFactsFallsBackToDrawnPlot(t *testing.T) {
	f := ComputeFacts(testDrawing(), 0)
	// Tegnet skel: 30×40 m = 1200 m².
	approxFact(t, f.PlotAreaM2, 1200, 1e-9, "drawn plot area")
}

func TestComputeFactsHonestNils(t *testing.T) {
	f := ComputeFacts(&domain.DrawingData{}, 0)
	if f.FootprintM2 != nil || f.MinSkelafstandM != nil || f.BygningshoejdeM != nil {
		t.Error("empty drawing must yield nil facts, never guesses")
	}
}

func TestComputeFactsRotationAffectsSkelafstand(t *testing.T) {
	data := testDrawing()
	data.Plot.RotationDeg = 45
	f := ComputeFacts(data, 1200)
	if f.MinSkelafstandM == nil {
		t.Fatal("skelafstand missing")
	}
	// Rotation om origo flytter huset — afstanden må ikke være den samme
	// som uden rotation (5,0 m), og skal være beregnet, ikke gættet.
	if math.Abs(*f.MinSkelafstandM-5.0) < 1e-9 {
		t.Error("rotation must affect the computed distance")
	}
}
