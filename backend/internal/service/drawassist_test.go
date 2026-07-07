package service

import (
	"testing"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

func TestTemplateDrawingParsesDimensionsAndRooms(t *testing.T) {
	data, err := templateDrawing("Et hus på 10x8 m med 3 rum og saddeltag")
	if err != nil {
		t.Fatal(err)
	}
	if err := data.Validate(); err != nil {
		t.Fatalf("template drawing must validate: %v", err)
	}
	// 4 ydervægge + 2 skillevægge for 3 rum.
	if len(data.Walls) != 6 {
		t.Errorf("expected 6 walls, got %d", len(data.Walls))
	}
	if len(data.Rooms) != 3 {
		t.Errorf("expected 3 rooms, got %d", len(data.Rooms))
	}
	// 3 vinduer + 1 dør.
	doors, windows := 0, 0
	for _, o := range data.Openings {
		if o.Type == domain.OpeningDoor {
			doors++
		} else {
			windows++
		}
	}
	if doors != 1 || windows != 3 {
		t.Errorf("expected 1 door and 3 windows, got %d/%d", doors, windows)
	}
	if data.RoofAngleDeg != 30 {
		t.Errorf("saddeltag should set roof angle, got %v", data.RoofAngleDeg)
	}
	// Længste side lægges langs x: 10 m bredde.
	total := data.TotalRoomAreaM2()
	if total < 60 || total > 80 {
		t.Errorf("indvendigt areal %v m² ligner ikke et 10x8-hus", total)
	}
}

func TestTemplateDrawingDefaultsWithoutNumbers(t *testing.T) {
	data, err := templateDrawing("bare et lille hus")
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Walls) < 4 || len(data.Rooms) < 1 {
		t.Error("default drawing should still produce a valid house")
	}
	if data.RoofAngleDeg != 0 {
		t.Error("no roof mentioned: flat roof expected")
	}
}

func TestParseDrawingDataStripsSiteElements(t *testing.T) {
	raw := "```json\n" + `{
		"walls":[{"id":"w1","from":{"x":0,"y":0},"to":{"x":4000,"y":0},"thicknessMm":350,"isLoadBearing":true}],
		"rooms":[],"openings":[],
		"plot":{"boundary":[{"x":0,"y":0},{"x":1,"y":0},{"x":1,"y":1}],"offset":{"x":0,"y":0},"rotationDeg":0},
		"trees":[{"position":{"x":0,"y":0},"heightMm":5000,"crownDiameterMm":3000}],
		"wallHeightMm":2500}` + "\n```"
	data, err := parseDrawingData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if data.Plot != nil || data.Trees != nil || data.Geo != nil {
		t.Error("AI drawings must not carry plot/trees/geo — those belong to the user")
	}
	if _, err := parseDrawingData(`{"walls":[]}`); err == nil {
		t.Error("empty wall list must be rejected")
	}
}
