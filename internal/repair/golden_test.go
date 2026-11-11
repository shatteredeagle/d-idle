package repair

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Golden replies of GET /api/repair (schema 2) for every state the page shows, for both classes. The watch package's
// DOM goldens (internal/watch/testdata/gear_repairtrip_dom.js) read these files, so the page is tested against what the
// controller really sends. UPDATE=1 rewrites them.
func TestViewGoldens(t *testing.T) {
	type scenario struct {
		name string
		run  func(g *rig)
	}
	lowAll := func(g *rig, class string) { g.r.c.Gear = gearFor(class, worn) }
	whole := func(g *rig, class string) { g.r.c.Gear = gearFor(class, full) }
	scenarios := []scenario{
		{"ready", func(g *rig) { g.gold = 300 }},
		{"farm", func(g *rig) { g.gold = 100 }},
		{"disabled-undamaged", func(g *rig) { whole(g, g.class) }},
		{"disabled-pact", func(g *rig) { g.mode = "never" }},
		{"queued", func(g *rig) { g.gold = 300; g.r.Request() }},
		{"waiting", func(g *rig) {
			g.gold = 300
			g.r.Request()
			g.r.SetPhase(PhaseWaiting, "finishing the pack first, 12 s of 60")
		}},
		{"going", func(g *rig) { g.gold = 300; g.r.Request(); g.r.BeginTrip(g.now) }},
		{"repairing", func(g *rig) { g.gold = 300; g.r.Request(); g.r.BeginTrip(g.now); g.r.SetPhase(PhaseRepairing, "") }},
		{"returning", func(g *rig) {
			g.gold = 300
			g.r.Request()
			g.r.BeginTrip(g.now)
			g.r.SetPhase(PhaseRepairing, "")
			g.r.Repaired(g.now)
		}},
		{"farming", func(g *rig) { g.gold = 140; g.r.Request(); g.r.FarmProgress(140, 300, "Stony Field") }},
		{"done", func(g *rig) {
			g.gold = 300
			g.r.Request()
			g.r.BeginTrip(g.now)
			g.r.SetPhase(PhaseRepairing, "")
			g.r.Repaired(g.now)
			whole(g, g.class)
			g.r.EndTrip(g.now, true, false, "")
			lowAll(g, g.class)
		}},
		{"cancelled", func(g *rig) { g.gold = 300; g.r.Request(); g.r.Cancel() }},
		{"cancelled-auto", func(g *rig) {
			g.gold = 100
			g.r.Request()
			g.r.FarmReached()
			g.r.BeginTrip(g.now)
			g.r.EndTrip(g.now, false, true, "need 300, have 120")
		}},
		{"failed", func(g *rig) {
			g.gold = 300
			g.r.Request()
			g.r.BeginTrip(g.now)
			g.r.EndTrip(g.now, false, false, "repair not confirmed")
			g.now = g.now.Add(21 * time.Minute)
			g.r.BeginTrip(g.now)
			g.r.EndTrip(g.now, false, false, "repair not confirmed")
		}},
	}
	dir := filepath.Join("testdata")
	for _, class := range []string{"sorceress", "paladin"} {
		for _, sc := range scenarios {
			g := newRig(t, class, "")
			g.class = class
			sc.run(g)
			g.r.c.Gear = gearFor(class, map[string]int{"rhand": 40, "torso": 70, "head": 80, "lhand": 90}) // fixed readings for the view
			if sc.name == "disabled-undamaged" {
				g.r.c.Gear = gearFor(class, full)
			}
			v := g.r.View()
			// sort the gear (map order) so the file is stable
			sortCond(v.Gear)
			b, err := json.MarshalIndent(v, "", " ")
			if err != nil {
				t.Fatal(err)
			}
			b = append(b, '\n')
			file := filepath.Join(dir, "view-"+sc.name+"-"+class+".json")
			if os.Getenv("UPDATE") != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, b, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("%s: %v (UPDATE=1 writes the goldens)", file, err)
			}
			if string(want) != string(b) {
				t.Errorf("%s differs:\n%s", file, b)
			}
		}
	}
}

func sortCond(c []Cond) {
	for i := 1; i < len(c); i++ {
		for j := i; j > 0 && c[j].Slot < c[j-1].Slot; j-- {
			c[j], c[j-1] = c[j-1], c[j]
		}
	}
}
