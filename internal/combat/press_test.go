package combat

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

func TestValidatePresses(t *testing.T) {
	skills := map[string]skillMeta{"Zeal": {Hold: true}, "Fire Bolt": {}, "Arctic Blast": {Hold: true}}
	good := func(r PressRule) []Profile { return []Profile{{Class: "x", Presses: []PressRule{r}}} }
	ok := PressRule{Name: "Arctic Blast", SliceMs: 500, Commit: true, CommitLifePct: 60, Source: "U test"}
	if err := validatePresses(good(ok), skills); err != nil {
		t.Fatal(err)
	}
	for _, mod := range []func(*PressRule){
		func(r *PressRule) { r.Name = "Nope" },
		func(r *PressRule) { r.Name = "Fire Bolt" },
		func(r *PressRule) { r.SliceMs = 50 },
		func(r *PressRule) { r.SliceMs = 5000 },
		func(r *PressRule) { r.CommitLifePct = 0 },
		func(r *PressRule) { r.Commit, r.CommitLifePct = false, 60 },
		func(r *PressRule) { r.Source = "" },
	} {
		r := ok
		mod(&r)
		if validatePresses(good(r), skills) == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
	if validatePresses([]Profile{{Class: "x", Presses: []PressRule{ok, ok}}}, skills) == nil {
		t.Fatal("accepted a row twice")
	}
}

// withPresses swaps the press rows of a loaded profile for the length of a test.
func withPresses(t *testing.T, p *Profile, rows []PressRule) {
	old := p.Presses
	p.Presses = rows
	t.Cleanup(func() { p.Presses = old })
}

// The slice and the commit gate follow the knob: 0 and 1 keep the old behaviour (1 counts), 2 acts. Two builds: a
// Paladin on Zeal (default profile) and a Druid on Arctic Blast with a commit row.
func TestPressSliceAndCommitGate(t *testing.T) {
	for _, b := range []struct {
		class d2.Class
		skill string
		hold  time.Duration
	}{{d2.Paladin, "Zeal", 600 * time.Millisecond}, {d2.Druid, "Arctic Blast", 600 * time.Millisecond}} {
		for _, mode := range []int{0, 1, 2} {
			h, w := newClassWorld(t, b.class, nil, nil, nil)
			h.f.e.Tune.Combat.DeliveryUpkeep = mode
			p := w.snap.Player
			pr := h.f.profileFor(p)
			withPresses(t, pr, []PressRule{{Name: b.skill, SliceMs: 350, Commit: true, CommitLifePct: 60, Source: "U test"}})
			sp := SkillSpec{Name: b.skill}
			want := b.hold
			if mode == 2 {
				want = 350 * time.Millisecond
			}
			if got := h.f.pressSlice(p, sp); got != want {
				t.Fatalf("%s mode %d: slice %v, want %v", b.skill, mode, got, want)
			}
			p.MaxLife, p.Life = 400, 100
			if got, wantOK := h.f.pressMayStart(p, sp), mode != 2; got != wantOK {
				t.Fatalf("%s mode %d: low life start %v", b.skill, mode, got)
			}
			p.Life = 400
			if !h.f.pressMayStart(p, sp) {
				t.Fatalf("%s mode %d: refused at full life", b.skill, mode)
			}
			if !h.f.pressMayStart(p, SkillSpec{Name: "Other"}) {
				t.Fatalf("%s mode %d: refused a skill with no row", b.skill, mode)
			}
			if mode == 1 && (h.f.PressShadowDiffs() != 2 || h.f.PressRefusals() != 0) {
				t.Fatalf("%s shadow diffs %d refusals %d", b.skill, h.f.PressShadowDiffs(), h.f.PressRefusals())
			}
			if mode == 2 && h.f.PressRefusals() != 1 {
				t.Fatalf("%s refusals %d", b.skill, h.f.PressRefusals())
			}
		}
	}
}
