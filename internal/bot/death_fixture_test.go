package bot

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

type deathSample struct {
	T          float64
	Mode, Life int
	Max        int
	Phase      string
}

func loadDeath(t *testing.T, name string) []deathSample {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "combat", "testdata", "death", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []deathSample
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var s deathSample
		if err := json.Unmarshal(sc.Bytes(), &s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

// The real recordings (i2, Combi): the bot's death test must flip at the killing blow (mode 0, life 0), exactly once per
// death, and an Esc must be allowed only from mode 17 on; the rest of the dead interval is never "alive".
func TestDeathFixturesDetectionAndEscWindow(t *testing.T) {
	for _, name := range []string{"death1.jsonl", "death2.jsonl"} {
		samples := loadDeath(t, name)
		var prev *d2.Snapshot
		transitions, deadSeen, escAllowedWhileDying := 0, false, false
		for _, s := range samples {
			p := &d2.Player{Mode: s.Mode, Life: s.Life, MaxLife: s.Max}
			cur := &d2.Snapshot{State: d2.StateInGame, Player: p}
			if diedTransition(prev, cur) {
				transitions++
			}
			prev = cur
			if isDead(p) {
				deadSeen = true
				if s.Mode == combat.ModeDying && combat.CanRespawn(p) {
					escAllowedWhileDying = true
				}
			} else if deadSeen && s.Mode != combat.ModeTownNeutral {
				// after the respawn the character lives again in town (mode 5)
				t.Fatalf("%s t=%.1f: alive in mode %d after death", name, s.T, s.Mode)
			}
		}
		if transitions != 1 || !deadSeen || escAllowedWhileDying {
			t.Fatalf("%s: transitions %d deadSeen %v escWhileDying %v", name, transitions, deadSeen, escAllowedWhileDying)
		}
	}
}

// Mode 0 alone (life already 0) is dead but not yet respawnable.
func TestDyingIsDeadButEscWaits(t *testing.T) {
	p := &d2.Player{Mode: combat.ModeDying, Life: 0, MaxLife: 40}
	if !isDead(p) || combat.CanRespawn(p) {
		t.Fatal("dying: dead, not respawnable")
	}
	p.Mode = combat.ModeDead
	if !combat.CanRespawn(p) {
		t.Fatal("mode 17 respawnable")
	}
}
