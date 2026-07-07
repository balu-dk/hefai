package calc

import (
	"encoding/json"
	"math"
	"testing"
)

func mustRun(t *testing.T, method string, inputs any) *Outcome {
	t.Helper()
	raw, err := json.Marshal(inputs)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Run(method, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Advisory || out.Notice == "" {
		t.Fatal("every outcome must be advisory with a notice")
	}
	if len(out.Assumptions) == 0 {
		t.Fatal("every outcome must list explicit assumptions")
	}
	for _, a := range out.Assumptions {
		if a.Reference == "" {
			t.Fatalf("assumption without reference: %q", a.Text)
		}
	}
	return out
}

func approx(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %v, want %v (±%v)", what, got, want, tol)
	}
}

// Hand calculation: μ1 = 0.8 below 30°, s = 0.8·0.9 = 0.72 kN/m².
func TestSnowLoadFlatRoof(t *testing.T) {
	out := mustRun(t, "snow_load_dk_v1", SnowLoadInput{RoofAngleDeg: 25})
	approx(t, out.Results["roofSnowLoadKNM2"].(float64), 0.72, 1e-9, "snow load 25°")

	// 45°: μ1 = 0.8·(60-45)/30 = 0.4 → s = 0.36
	out = mustRun(t, "snow_load_dk_v1", SnowLoadInput{RoofAngleDeg: 45})
	approx(t, out.Results["roofSnowLoadKNM2"].(float64), 0.36, 1e-9, "snow load 45°")

	// 60° and above: no snow retained.
	out = mustRun(t, "snow_load_dk_v1", SnowLoadInput{RoofAngleDeg: 70})
	approx(t, out.Results["roofSnowLoadKNM2"].(float64), 0, 1e-9, "snow load 70°")
}

// Hand calculation for terrain II at z = 10 m, vb = 24 m/s:
// kr = 0.19, cr = 0.19·ln(200) ≈ 1.0066, vm ≈ 24.16 m/s,
// Iv = 1/ln(200) ≈ 0.1887, qp = (1+7·0.1887)·0.625·24.16²/1000 ≈ 0.847 kN/m².
func TestWindLoadTerrainII(t *testing.T) {
	out := mustRun(t, "wind_load_dk_v1", WindLoadInput{HeightM: 10, TerrainCategory: "II"})
	approx(t, out.Results["peakVelocityPressureKNM2"].(float64), 0.847, 0.005, "qp terrain II 10m")

	// West coast raises vb to 27 → qp scales by (27/24)² ≈ 1.266.
	coast := mustRun(t, "wind_load_dk_v1", WindLoadInput{HeightM: 10, TerrainCategory: "II", NearWestCoast: true})
	ratio := coast.Results["peakVelocityPressureKNM2"].(float64) / out.Results["peakVelocityPressureKNM2"].(float64)
	approx(t, ratio, math.Pow(27.0/24.0, 2), 0.01, "coast/inland ratio")

	// Below zmin the pressure must clamp to zmin.
	low := mustRun(t, "wind_load_dk_v1", WindLoadInput{HeightM: 1, TerrainCategory: "II"})
	atMin := mustRun(t, "wind_load_dk_v1", WindLoadInput{HeightM: 2, TerrainCategory: "II"})
	approx(t, low.Results["peakVelocityPressureKNM2"].(float64),
		atMin.Results["peakVelocityPressureKNM2"].(float64), 1e-9, "zmin clamp")
}

// Hand calculation, C24 45×195, L=4.0 m, s=0.6 m, g=0.5, q=2.0 kN/m²:
// wd = (1.2·0.5 + 1.5·2.0)·0.6 = 2.16 kN/m, Md = 4.32 kNm
// W = 0.045·0.195²/6 = 2.8519e-4 m³ → σ = 15.15 MPa
// fm,d = 0.8·24/1.3 = 14.77 MPa → bending utilization ≈ 1.026 (fails)
// u = 5·1.5·4⁴/(384·11e9·2.781e-5) ≈ 16.3 mm; limit 13.3 mm → ≈ 1.22
func TestTimberBeamHandCalculation(t *testing.T) {
	out := mustRun(t, "timber_beam_udl_v1", TimberBeamInput{
		SpanM: 4.0, SpacingM: 0.6, PermanentLoadKNM2: 0.5, VariableLoadKNM2: 2.0,
		TimberClass: "C24", WidthMM: 45, HeightMM: 195,
	})
	check := out.Results["check"].(beamCheck)
	approx(t, out.Results["designLineLoadKNM"].(float64), 2.16, 1e-9, "design line load")
	approx(t, out.Results["designMomentKNM"].(float64), 4.32, 1e-9, "design moment")
	approx(t, check.BendingUtilization, 1.026, 0.005, "bending utilization")
	approx(t, check.DeflectionMM, 16.3, 0.15, "deflection")
	approx(t, check.DeflectionUtilization, 1.22, 0.01, "deflection utilization")
	if check.OK {
		t.Error("45×195 must fail this case (bending > 1)")
	}
}

func TestTimberBeamSuggestsDimension(t *testing.T) {
	out := mustRun(t, "timber_beam_udl_v1", TimberBeamInput{
		SpanM: 4.0, SpacingM: 0.6, PermanentLoadKNM2: 0.5, VariableLoadKNM2: 2.0,
		TimberClass: "C24", WidthMM: 45, HeightMM: 195, SuggestDimension: true,
	})
	suggested, ok := out.Results["suggestedSection"].(map[string]any)
	if !ok || suggested == nil {
		t.Fatal("expected a suggested section")
	}
	// 45×220: W = 3.63e-4 → σ = 11.90, bending 0.81 OK;
	// I = 3.993e-5 → u ≈ 11.4 mm ≤ 13.3 OK — first passing standard size.
	approx(t, suggested["heightMm"].(float64), 220, 1e-9, "suggested height")
	if !suggested["check"].(beamCheck).OK {
		t.Error("suggested section must pass all checks")
	}
}

func TestTimberBeamRefusesOutOfScope(t *testing.T) {
	_, err := Run("timber_beam_udl_v1", json.RawMessage(`{"spanM":9,"spacingM":0.6,"permanentLoadKNM2":0.5,"variableLoadKNM2":2,"timberClass":"C24","widthMm":45,"heightMm":195}`))
	if err == nil {
		t.Error("9 m span must be refused as out of scope")
	}
	_, err = Run("ukendt_metode", json.RawMessage(`{}`))
	if err == nil {
		t.Error("unknown method must be refused")
	}
}
