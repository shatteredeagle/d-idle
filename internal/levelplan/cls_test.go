package levelplan

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/plandata"
)

// Card cls1: the Druid and the Barbarian have judged plans (no "copied" tag), each names a shipped build table of its
// own class, and the Act 2 archetype agrees with the class trait. One caster and one melee class through the same check.
func TestDruidAndBarbarianPlansAreTheirOwn(t *testing.T) {
	p, err := Load("normal")
	if err != nil {
		t.Fatal(err)
	}
	a2, err := plandata.LoadAct2Plan()
	if err != nil {
		t.Fatal(err)
	}
	for class, c := range map[string]d2.Class{"druid": d2.Druid, "barbarian": d2.Barbarian} {
		cp, err := p.For(class)
		if err != nil {
			t.Fatal(err)
		}
		ac, err := a2.For(class)
		if err != nil {
			t.Fatal(err)
		}
		if len(cp.Farms()) < 5 || len(ac.Slices) < 5 {
			t.Errorf("%s: %d farm slices, %d act 2 slices", class, len(cp.Farms()), len(ac.Slices))
		}
		for what, tag := range map[string]string{"levelplan": cp.Tag, "act2plan": ac.Tag} {
			if strings.Contains(tag, "copied") || !strings.Contains(tag, "docs/sme/class-"+class+".md") {
				t.Errorf("%s %s: the tag must name the class's SME file, not a copy: %q", class, what, tag)
			}
		}
		for what, build := range map[string]string{"levelplan": cp.Build, "act2plan": ac.Build} {
			if !strings.HasPrefix(build, class+"_") {
				t.Errorf("%s %s: build %q is not a %s table", class, what, build, class)
			}
			if _, err := plandata.FS.ReadFile("build-" + build + ".json"); err != nil {
				t.Errorf("%s %s: build %q is not shipped: %v", class, what, build, err)
			}
		}
		tr, err := d2.TraitsOf(c)
		if err != nil {
			t.Fatal(err)
		}
		want := "melee"
		if tr.Caster {
			want = "caster"
		}
		if ac.Archetype != want {
			t.Errorf("%s: act2plan archetype %q, class trait says %s", class, ac.Archetype, want)
		}
		for _, d := range []string{"normal", "nightmare", "hell"} {
			if a, ok, err := d2.MercAuraFor(c, d); err != nil || !ok || a == "" {
				t.Errorf("%s: no merc aura row for %s (%v)", class, d, err)
			}
		}
	}
}
