package pact

import (
	"strings"
	"testing"
	"time"
)

func TestViewLockedSilhouettesAndChoose(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newRig(t, class, "")
		v := r.c.View(nil)
		if v.HeatCap != 12 || v.MaxCap != 20 || len(v.Families) == 0 {
			t.Fatalf("%s: %+v", class, v)
		}
		locked := 0
		for _, f := range v.Families {
			for _, p := range f.Pacts {
				if p.Locked {
					locked++
					if p.Name != "???" || p.Learn != "" || p.Unlock == "" {
						t.Fatalf("%s: not a silhouette: %+v", class, p)
					}
					for _, rk := range p.Ranks {
						if rk.Effect != "" || rk.Status != "locked" {
							t.Fatalf("%s: effect leaked: %+v", class, rk)
						}
					}
				}
			}
		}
		if locked == 0 {
			t.Fatalf("%s: nothing locked before the first Andariel kill", class)
		}
		if err := r.c.Choose([]Pick{{"blood_ravens_tithe", 1}}); err == nil || !strings.Contains(err.Error(), "locked") {
			t.Fatalf("%s: chose a locked Pact: %v", class, err)
		}
		r.kill("Andariel", "normal")
		if err := r.c.Choose([]Pick{{"blood_ravens_tithe", 1}}); err != nil {
			t.Fatalf("%s: %v", class, err)
		}
		v = r.c.View(nil)
		for _, f := range v.Families {
			for _, p := range f.Pacts {
				if p.Locked && p.ID == "blood_ravens_tithe" {
					t.Fatalf("%s: still locked", class)
				}
			}
		}
		if v.Heat == 0 || v.Current.TallyPct == 0 {
			t.Fatalf("%s: heat/reward missing %+v", class, v.Current)
		}
	}
}

func TestViewProposalHeatAndReasons(t *testing.T) {
	r := newRig(t, "paladin", "")
	r.kill("Andariel", "normal")
	r.c.Choose([]Pick{{"blood_ravens_tithe", 1}})
	// a set over the cap or incompatible is shown with its error, not hidden
	all := r.c.cfg.Cat.Pacts
	var big []Pick
	for _, p := range all {
		if p.AllowsClass("paladin") {
			big = append(big, Pick{p.ID, p.Ranks[len(p.Ranks)-1].Rank})
		}
	}
	v := r.c.View(big)
	if v.Proposed == nil || v.Proposed.Error == "" {
		t.Fatalf("a proposal of everything at top rank was not refused: %+v", v.Proposed)
	}
	v = r.c.View([]Pick{{"andariels_shadow", 1}})
	if v.Proposed.Error != "" || len(v.Levers) == 0 {
		t.Fatalf("proposal %+v levers %v", v.Proposed, v.Levers)
	}
	if v.Levers[0].Source == "" || v.Levers[0].Proposed == "" {
		t.Fatalf("lever %+v", v.Levers[0])
	}
	if _, err := ParsePicks("a:1,b"); err == nil {
		t.Fatal("bad pick list accepted")
	}
	if ps, _ := ParsePicks("a:1, b:3"); len(ps) != 2 || ps[1].Rank != 3 {
		t.Fatalf("%v", ps)
	}
}

func TestSpansTrackActivation(t *testing.T) {
	r := newRig(t, "sorceress", "")
	rel := 0.0
	r.c.cfg.Rel = func() float64 { return rel }
	r.c.Select([]Pick{{"blood_ravens_tithe", 1}})
	r.present = true
	rel = 100
	r.c.Evaluate(field)
	rel = 400
	r.c.Evaluate(Where{Level: 6, Age: time.Hour, InTown: true})
	sp := r.c.Spans()
	if len(sp) != 1 || sp[0].Start != 100 || sp[0].End != 400 {
		t.Fatalf("%+v", sp)
	}
	rel = 500
	r.c.Evaluate(field)
	if sp = r.c.Spans(); len(sp) != 2 || sp[1].End != -1 {
		t.Fatalf("%+v", sp)
	}
}
