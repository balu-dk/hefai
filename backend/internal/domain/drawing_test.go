package domain

import (
	"math"
	"testing"
)

func TestPolygonAreaM2(t *testing.T) {
	// 4000 x 5000 mm rectangle = 20 m²
	rect := []Point{{0, 0}, {4000, 0}, {4000, 5000}, {0, 5000}}
	if got := PolygonAreaM2(rect); math.Abs(got-20) > 1e-9 {
		t.Errorf("rectangle area = %v, want 20", got)
	}
	// Winding order must not matter.
	reversed := []Point{{0, 5000}, {4000, 5000}, {4000, 0}, {0, 0}}
	if got := PolygonAreaM2(reversed); math.Abs(got-20) > 1e-9 {
		t.Errorf("reversed area = %v, want 20", got)
	}
	// Triangle 3000 x 4000 / 2 = 6 m²
	tri := []Point{{0, 0}, {3000, 0}, {0, 4000}}
	if got := PolygonAreaM2(tri); math.Abs(got-6) > 1e-9 {
		t.Errorf("triangle area = %v, want 6", got)
	}
	if got := PolygonAreaM2([]Point{{0, 0}, {1, 1}}); got != 0 {
		t.Errorf("degenerate polygon area = %v, want 0", got)
	}
}

func TestDrawingDataValidate(t *testing.T) {
	valid := DrawingData{
		Walls: []Wall{
			{ID: "w1", From: Point{0, 0}, To: Point{4000, 0}, ThicknessMM: 300},
		},
		Rooms: []RoomShape{
			{Name: "Stue", Polygon: []Point{{0, 0}, {4000, 0}, {4000, 3000}}},
		},
		Openings: []Opening{
			{WallID: "w1", Type: OpeningDoor, OffsetMM: 1000, WidthMM: 900, HeightMM: 2100},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid drawing rejected: %v", err)
	}

	cases := []struct {
		name string
		mut  func(*DrawingData)
	}{
		{"zero-length wall", func(d *DrawingData) { d.Walls[0].To = d.Walls[0].From }},
		{"duplicate wall id", func(d *DrawingData) {
			d.Walls = append(d.Walls, Wall{ID: "w1", From: Point{0, 0}, To: Point{1, 1}, ThicknessMM: 100})
		}},
		{"opening on unknown wall", func(d *DrawingData) { d.Openings[0].WallID = "nope" }},
		{"unnamed room", func(d *DrawingData) { d.Rooms[0].Name = "" }},
		{"room under 3 points", func(d *DrawingData) { d.Rooms[0].Polygon = d.Rooms[0].Polygon[:2] }},
		{"negative opening width", func(d *DrawingData) { d.Openings[0].WidthMM = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bad := DrawingData{
				Walls:    append([]Wall{}, valid.Walls...),
				Rooms:    []RoomShape{{Name: valid.Rooms[0].Name, Polygon: append([]Point{}, valid.Rooms[0].Polygon...)}},
				Openings: append([]Opening{}, valid.Openings...),
			}
			tc.mut(&bad)
			if err := bad.Validate(); err == nil {
				t.Error("invalid drawing accepted")
			}
		})
	}
}
