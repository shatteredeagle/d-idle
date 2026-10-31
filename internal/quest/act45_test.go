package quest

import "testing"

func checkOrder(t *testing.T, order []string, get func(string) (Handler, bool), all []Handler, lo, hi int) {
	t.Helper()
	seen := map[string]bool{}
	for _, id := range order {
		h, ok := get(id)
		if !ok {
			t.Fatalf("order names %q without a handler", id)
		}
		seen[id] = true
		if len(h.Steps) == 0 {
			t.Errorf("%s has no steps", id)
		}
		for i, s := range h.Steps {
			switch s.Kind {
			case KGoNPC, KTalk, KTurnIn:
				if _, ok := FindMenu(s.NPC, LevelMain); !ok {
					t.Errorf("%s step %d: npc %d has no dialog table", id, i, s.NPC)
				}
			case KGoArea, KWaitArea:
				if s.Area < lo || s.Area > hi {
					t.Errorf("%s step %d: area %d outside %d..%d", id, i, s.Area, lo, hi)
				}
			}
		}
	}
	for _, h := range all {
		if !seen[h.Quest] {
			t.Errorf("handler %s is not in the order", h.Quest)
		}
	}
}

func TestAct4TablesConsistent(t *testing.T) {
	checkOrder(t, Act4Order, Act4Handler, Act4, AreaFortress, AreaHarrogath)
	if len(Seals) != 5 {
		t.Fatalf("seals %v", Seals)
	}
}

func TestAct5TablesConsistent(t *testing.T) {
	checkOrder(t, Act5Order, Act5Handler, Act5, AreaHarrogath, AreaWorldstoneChamber)
}

func TestCriticalPathAct45(t *testing.T) {
	for _, q := range []string{"terrors_end", "act4_done"} {
		if h, _ := Act4Handler(q); !h.Req {
			t.Errorf("%s must be Req", q)
		}
	}
	for _, q := range []string{"prison_of_ice", "rite_of_passage", "eve_of_destruction"} {
		if h, _ := Act5Handler(q); !h.Req {
			t.Errorf("%s must be Req", q)
		}
	}
}

func TestStepKindsDistinct(t *testing.T) {
	ks := map[Kind]string{KSmashOrb: "smash", KOpenSeals: "seals", KPlaceOnForge: "forge", KWaves: "waves"}
	if len(ks) != 4 {
		t.Fatal("kind values collide")
	}
}
