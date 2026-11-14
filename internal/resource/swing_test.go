package resource

import (
	"math"
	"testing"

	"d2lod-bot/internal/d2"
)

func TestSwingSeconds(t *testing.T) {
	for c, want := range map[d2.Class]float64{d2.Sorceress: 0.64, d2.Paladin: 0.6} {
		got, err := SwingSeconds(c)
		if err != nil || math.Abs(got-want) > 1e-9 {
			t.Errorf("class %v: %v %v, want %v", c, got, err, want)
		}
	}
	if _, err := SwingSeconds(d2.Necromancer); err == nil {
		t.Error("unknown class must be an error")
	}
	if BloodMoorSorc().MeleeSec != 0.64 || BloodMoorPaladin().MeleeSec != 0.6 {
		t.Error("scenarios do not use the swing data")
	}
}
