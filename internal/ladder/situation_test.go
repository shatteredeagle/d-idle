package ladder_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

// The situation reaches the open thought (hero facts, the area name from Config.AreaNames, the helper's and the point's
// own facts) and never changes the key or the answer. Two classes.
func TestSituationIsDisplayOnly(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			cat, err := decide.Shipped()
			if err != nil {
				t.Fatal(err)
			}
			var keys []string
			for _, withSit := range []bool{true, false} {
				clk := &decide.ManualClock{}
				st, err := decide.New(cat, "t0", clk)
				if err != nil {
					t.Fatal(err)
				}
				l, err := ladder.New(ladder.Config{Cat: cat, Store: st, Clock: clk,
					AreaNames: func(id int) string { return map[int]string{3: "Cold Plains"}[id] }})
				if err != nil {
					t.Fatal(err)
				}
				h := ladder.Hero{Class: class, Gold: 45, Area: 3, Level: 9, AreaClass: "field", Durability: -1,
					Sit: decide.Situation{"life": "28"}}
				if withSit {
					h = h.With("potions", 1, "hostiles", 2)
				} else {
					h.Sit, h.Area, h.Level = nil, 0, 0
				}
				a, err := l.DrinkChoice(h, 28, "hp", "OK", false, "fight")
				if err != nil || a != "fight" {
					t.Fatalf("answer %q %v", a, err)
				}
				open := st.OpenThoughts()
				if len(open) != 1 {
					t.Fatalf("open thoughts: %d", len(open))
				}
				keys = append(keys, open[0].Key)
				ti, _ := st.Thought(open[0].ID)
				if !withSit {
					continue
				}
				for k, want := range map[string]string{"area": "Cold Plains", "herolevel": "9", "gold": "45", "life": "28", "potions": "1", "hostiles": "2"} {
					if ti.Sit[k] != want {
						t.Errorf("fact %s = %q, want %q (%v)", k, ti.Sit[k], want, ti.Sit)
					}
				}
			}
			if keys[0] != keys[1] {
				t.Errorf("the situation changed the key:\n%s\n%s", keys[0], keys[1])
			}
		})
	}
}
