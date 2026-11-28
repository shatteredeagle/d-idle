package questboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestBoardLegacySwitchIdle pins the seam of the later legacy removal: before anything is accepted the old board and the run
// machine answer byte for byte alike (offers, state, next decision, and the plan file on disk), for two builds. The paths the
// machine changes on purpose (accept, park, Retry) are covered by exec_sm_test.go and are not compared here.
func TestBoardLegacySwitchIdle(t *testing.T) {
	for _, bd := range builds {
		for _, clvl := range []int{5, 12} {
			var got [2]string
			for i, sm := range []bool{false, true} {
				r := &rig{t: t, b: mustBoard(t), dir: t.TempDir(), hero: "Testy", sm: sm, old: !sm}
				r.f = NewFake(r.b, bd.class, clvl)
				r.restart()
				if r.x.boardLegacy() == sm {
					t.Fatalf("%s: boardLegacy() = %v with SM %v", bd.class, r.x.boardLegacy(), sm)
				}
				offers, err := r.x.Offers()
				if err != nil {
					t.Fatal(err)
				}
				st, err := r.x.State()
				if err != nil {
					t.Fatal(err)
				}
				nx := r.next()
				files, _ := filepath.Glob(filepath.Join(r.dir, "*"))
				var disk []string
				for _, f := range files {
					b, _ := os.ReadFile(f)
					disk = append(disk, filepath.Base(f)+":"+string(b))
				}
				out, err := json.Marshal([]any{offers, st, nx, disk})
				if err != nil {
					t.Fatal(err)
				}
				got[i] = string(out)
			}
			if got[0] != got[1] {
				t.Errorf("%s clvl %d: knob 0 and 1 differ on an idle board:\n0: %s\n1: %s", bd.class, clvl, got[0], got[1])
			}
		}
	}
}
