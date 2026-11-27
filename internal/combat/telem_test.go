package combat

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

// The aim summary counts the fight's aimed casts per row, in two builds; at knob 0 nothing is logged.
func TestAimSummary(t *testing.T) {
	stray, hs := strayAndPack()
	for _, c := range []struct {
		class        d2.Class
		build, skill string
	}{{d2.Sorceress, "blizzard", "Blizzard"}, {d2.Druid, "fire", "Eruption"}} {
		for _, mode := range []int{0, 1, 2} {
			h, snap := aimHarness(t, c.class, c.build, mode)
			buf := logTo(h)
			st := newFightState()
			st.tel.begin(h.f)
			h.f.aimFor(snap, SkillSpec{Name: c.skill, Range: 30}, stray, hs, true)
			h.f.telemSummaries(st, snap)
			out := buf.String()
			switch mode {
			case 0:
				if out != "" {
					t.Errorf("%s knob 0 logged %q", c.skill, out)
				}
			case 1:
				if !strings.Contains(out, "combat aim summary") || !strings.Contains(out, "shadow_diff=1") || !strings.Contains(out, "hits=0") {
					t.Errorf("%s shadow: %q", c.skill, out)
				}
			case 2:
				if !strings.Contains(out, "hits=1") || !strings.Contains(out, "by_row="+c.skill+":1/0/0") {
					t.Errorf("%s mode 2: %q", c.skill, out)
				}
			}
		}
	}
}

// Stance and minion summaries from the tick samples: casts in place, pack distance, army distance and a lost minion.
func TestStanceMinionSummary(t *testing.T) {
	for _, b := range []struct {
		class d2.Class
		build string
	}{{d2.Paladin, "hammerdin"}, {d2.Necromancer, "poisonnova"}} {
		h, w := packWorld(t, b.class, b.build, 2, 4)
		buf := logTo(h)
		w.snap.Monsters = append(w.snap.Monsters, minionAt(200, clsGolem, 104, 100), minionAt(201, clsSkel, 96, 100))
		st := newFightState()
		st.tel.begin(h.f)
		hs := Hostiles(w.snap, h.f.hostileCtx(w.snap.Player))
		st.tel.sample(h.f, w.snap, hs) // first tick counts as held (no previous cell)
		st.tel.cast()
		st.tel.sample(h.f, w.snap, hs) // same cell
		st.tel.cast()
		w.snap.Monsters = w.snap.Monsters[:len(w.snap.Monsters)-1] // skeleton 201 is gone
		st.tel.sample(h.f, w.snap, hs)
		h.f.telemSummaries(st, w.snap)
		out := buf.String()
		for _, want := range []string{"combat stance summary", "casts=2", "in_place=2", "pack_dist_max=12", "combat minion summary", "alive=1", "lost=1", "seen=2", "max_dist=4"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: missing %q in %q", b.build, want, out)
			}
		}
	}
	h, w := packWorld(t, d2.Paladin, "hammerdin", 0, 4)
	buf := logTo(h)
	st := newFightState()
	st.tel.begin(h.f)
	st.tel.sample(h.f, w.snap, nil)
	h.f.telemSummaries(st, w.snap)
	if buf.Len() != 0 {
		t.Errorf("knob 0 logged %q", buf.String())
	}
}
