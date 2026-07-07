package service

import (
	"math"
	"testing"
)

// 150 m window at 56°N: dLat = 75/111320 ≈ 0.000674°,
// dLon = 75/(111320·cos 56°) ≈ 0.001205°.
func TestDegreeWindow(t *testing.T) {
	minLon, minLat, maxLon, maxLat := degreeWindow(56.0, 10.0, 150)

	if math.Abs((maxLat-minLat)*111320-150) > 0.01 {
		t.Errorf("lat window = %v m, want 150", (maxLat-minLat)*111320)
	}
	widthM := (maxLon - minLon) * 111320 * math.Cos(56*math.Pi/180)
	if math.Abs(widthM-150) > 0.01 {
		t.Errorf("lon window = %v m, want 150", widthM)
	}
	if minLat >= maxLat || minLon >= maxLon {
		t.Error("degenerate bbox")
	}
	// Centered on the anchor.
	if math.Abs((minLat+maxLat)/2-56.0) > 1e-12 || math.Abs((minLon+maxLon)/2-10.0) > 1e-12 {
		t.Error("bbox not centered on anchor")
	}
}
