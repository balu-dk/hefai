package calc

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// WindLoadInput describes the site for the peak velocity pressure.
type WindLoadInput struct {
	HeightM         float64 `json:"heightM"`         // reference height (typically ridge height)
	TerrainCategory string  `json:"terrainCategory"` // "0", "I", "II", "III", "IV"
	// NearWestCoast: within ~25 km of the North Sea/west coast, where the
	// Danish NA raises the basic wind velocity.
	NearWestCoast bool `json:"nearWestCoast"`
}

type terrainParams struct {
	z0   float64 // roughness length [m]
	zMin float64 // minimum height [m]
}

var terrains = map[string]terrainParams{
	"0":   {z0: 0.003, zMin: 1},
	"I":   {z0: 0.01, zMin: 1},
	"II":  {z0: 0.05, zMin: 2},
	"III": {z0: 0.3, zMin: 5},
	"IV":  {z0: 1.0, zMin: 10},
}

const airDensity = 1.25 // kg/m³

// runWindLoad computes the peak velocity pressure qp(z) per EN 1991-1-4
// (flat terrain, orography factor c0 = 1.0).
func runWindLoad(raw json.RawMessage) (*Outcome, error) {
	var in WindLoadInput
	if err := decodeInputs(raw, &in); err != nil {
		return nil, err
	}
	if in.HeightM <= 0 || in.HeightM > 25 {
		return nil, domain.Validation("højden skal være mellem 0 og 25 m — højere bygninger er uden for overslagets gyldighed, kontakt statiker")
	}
	terrain, ok := terrains[in.TerrainCategory]
	if !ok {
		return nil, domain.Validation(`terrænkategori skal være "0", "I", "II", "III" eller "IV"`)
	}

	// Basic wind velocity per the Danish NA.
	vb := 24.0
	coastNote := "indland"
	if in.NearWestCoast {
		vb = 27.0
		coastNote = "mindre end ca. 25 km fra Vesterhavet"
	}

	z := math.Max(in.HeightM, terrain.zMin)
	kr := 0.19 * math.Pow(terrain.z0/0.05, 0.07)
	cr := kr * math.Log(z/terrain.z0)
	vm := cr * vb              // mean wind velocity, c0 = 1.0
	iv := 1 / math.Log(z/terrain.z0) // turbulence intensity, kI = 1.0
	qp := (1 + 7*iv) * 0.5 * airDensity * vm * vm / 1000 // kN/m²

	inputsJSON, _ := json.Marshal(in)
	return &Outcome{
		Method:            "wind_load_dk_v1",
		MethodVersion:     "1",
		StandardReference: "DS/EN 1991-1-4 + DK NA",
		Inputs:            inputsJSON,
		Assumptions: []Assumption{
			{Text: fmt.Sprintf("Basisvindhastighed vb = %.0f m/s (%s).", vb, coastNote),
				Reference: "DS/EN 1991-1-4 DK NA"},
			{Text: fmt.Sprintf("Terrænkategori %s (z0 = %.3g m, zmin = %.0f m) ensartet i alle retninger — SKAL bekræftes for den konkrete grund.", in.TerrainCategory, terrain.z0, terrain.zMin),
				Reference: "DS/EN 1991-1-4, tabel 4.1"},
			{Text: "Fladt terræn uden orografieffekter (c0 = 1,0), turbulensfaktor kI = 1,0, luftdensitet 1,25 kg/m³.",
				Reference: "DS/EN 1991-1-4, 4.3"},
			{Text: "qp(z) er hastighedstrykket. Fladelaster kræver formfaktorer (cpe/cpi) for den konkrete geometri — statikerens opgave.",
				Reference: "DS/EN 1991-1-4, kap. 7"},
		},
		Results: map[string]any{
			"basicWindVelocityMS":       vb,
			"meanWindVelocityMS":        round(vm, 2),
			"turbulenceIntensity":       round(iv, 4),
			"peakVelocityPressureKNM2":  round(qp, 3),
			"formula":                   "qp = (1 + 7·Iv) · ½·ρ·vm², vm = kr·ln(z/z0)·vb",
		},
		Advisory: true,
		Notice:   advisoryNotice,
	}, nil
}
