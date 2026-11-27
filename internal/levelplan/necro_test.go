package levelplan

import (
	"testing"

	"d2lod-bot/internal/plandata"
)

// Card necroepic: the Necromancer has a plan next to the Sorceress and Paladin ones, and every plan names a build table
// the plan data ships (two classes through the same check).
func TestNecromancerPlanNamesAShippedBuild(t *testing.T) {
	p, err := Load("normal")
	if err != nil {
		t.Fatal(err)
	}
	for _, class := range []string{"necromancer", "sorceress"} {
		cp, err := p.For(class)
		if err != nil {
			t.Fatal(err)
		}
		if len(cp.Farms()) < 5 {
			t.Errorf("%s: %d farm slices", class, len(cp.Farms()))
		}
		if _, err := plandata.FS.ReadFile("build-" + cp.Build + ".json"); err != nil {
			t.Errorf("%s: build %q is not shipped: %v", class, cp.Build, err)
		}
	}
}
