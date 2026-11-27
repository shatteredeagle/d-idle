package inputdata

import (
	"testing"

	"d2lod-bot/internal/input"
)

// TestShippedCaps loads every shipped set and pins the one fact the planner depends on: devp13 has no usable select
// verb (R08), devp14 advertises it.
func TestShippedCaps(t *testing.T) {
	for _, n := range []string{"d2d-sandbox-devp13", "d2d-sandbox-devp14", "d2d-fork-input-items", "d2d-fork-input-verbs2", "d2d-fork-cast-verbs", "d2d-fork-forkcaps", "d2d-lab", "game-exe"} {
		if _, err := input.BackendCapsByName(FS, n); err != nil {
			t.Fatal(n, err)
		}
	}
	c13, _ := input.BackendCapsByName(FS, "d2d-sandbox-devp13")
	c14, _ := input.BackendCapsByName(FS, "d2d-sandbox-devp14")
	if c13.Has(input.CapSelect) || !c14.Has(input.CapSelect) {
		t.Fatalf("devp13 %v devp14 %v", c13.Verbs, c14.Verbs)
	}
	cv, _ := input.BackendCapsByName(FS, "d2d-fork-cast-verbs")
	if !cv.Has(input.CapCastUnit) || !cv.Has(input.CapCastPoint) || cv.Has(input.CapCastHold) {
		t.Fatalf("cast-verbs %v: unit and point, no hold verb exists", cv.Verbs)
	}
	fc, _ := input.BackendCapsByName(FS, "d2d-fork-forkcaps")
	for _, c := range []string{input.CapUseItem, input.CapCastUnit, input.CapCastPoint, input.CapCastHold, input.CapSelect} {
		if !fc.Has(c) {
			t.Fatalf("forkcaps %v: missing %s (the build lists it after its differential)", fc.Verbs, c)
		}
	}
}
