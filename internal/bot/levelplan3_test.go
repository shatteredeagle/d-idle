package bot

import "testing"

// TestLevelPlanV3PickPaladin: the active Paladin plan drives Pick (docs/levelplan3.md); the merc and kit gates choose
// between its slices; a class without an active plan (Sorceress, review-only) or an unknown class keeps the
// progression rows and never gets the Paladin's.
func TestLevelPlanV3PickPaladin(t *testing.T) {
	p, err := LoadPlan("")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.ClassSteps["normal"]["paladin"]) == 0 {
		t.Fatal("no v3 steps for the Paladin")
	}
	pal := PickOptions{Class: "paladin", Melee: true, FullKit: true}
	pick := func(level int, o PickOptions) int {
		st, ok := p.Pick("normal", level, o)
		if !ok {
			t.Fatalf("clvl %d: nothing picked", level)
		}
		return st.Area
	}
	if a := pick(3, pal); a != 3 {
		t.Errorf("Paladin clvl 3 with a kit: area %d, want Cold Plains", a)
	}
	thin := pal
	thin.FullKit = false
	if a := pick(3, thin); a != 2 {
		t.Errorf("Paladin clvl 3 without a kit: area %d, want Blood Moor (kit_clvl 4)", a)
	}
	if a := pick(10, pal); a == 7 {
		t.Error("Paladin clvl 10 without a merc went to Tamoe (entry 9 + 2)")
	}
	withMerc := pal
	withMerc.Merc = true
	if a := pick(10, withMerc); a != 7 {
		t.Errorf("Paladin clvl 10 with a merc: area %d, want Tamoe Highland", a)
	}
	for _, class := range []string{"sorceress", "warlock", ""} {
		o := PickOptions{Class: class, FullKit: true}
		got, _ := p.Pick("normal", 12, o)
		o.Class = ""
		want, _ := p.Pick("normal", 12, o)
		if got.Area != want.Area || got.Purpose != want.Purpose {
			t.Errorf("class %q: %+v, want the progression row %+v", class, got, want)
		}
	}
}
