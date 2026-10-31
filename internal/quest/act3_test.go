package quest

import "testing"

func TestAct3TablesConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, id := range Act3Order {
		h, ok := Act3Handler(id)
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
				if s.Area < 75 || s.Area > 103 {
					t.Errorf("%s step %d: area %d is not Act 3 (or the Fortress)", id, i, s.Area)
				}
			case KTransmute:
				if len(s.Labels) < 2 || s.Item == "" {
					t.Errorf("%s step %d: transmute needs inputs and an output", id, i)
				}
			}
		}
	}
	for _, h := range Act3 {
		if !seen[h.Quest] {
			t.Errorf("handler %s is not in Act3Order", h.Quest)
		}
	}
}

func TestKhalimsWillNeedsFourInputs(t *testing.T) {
	h, _ := Act3Handler("khalims_will")
	if n := len(h.Steps[0].Labels); n != 4 || h.Steps[0].Item != ItemKhalimWill {
		t.Fatalf("step %+v", h.Steps[0])
	}
}

func TestAct3ProxyStaysDoneOnceWillIsMade(t *testing.T) {
	h := fakeHas{ItemKhalimWill: true}
	for _, q := range []string{"khalims_eye", "khalims_brain", "khalims_heart"} {
		if !Act3Proxy[q](h) {
			t.Errorf("%s not done with qf2 in the pack", q)
		}
	}
}

type fakeHas map[string]bool

func (f fakeHas) Has(code string) bool { return f[code] }
