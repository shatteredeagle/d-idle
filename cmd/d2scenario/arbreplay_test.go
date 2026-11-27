package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -session scores a recorded tick log with the rules of the outcome files. The chores ticks without a hit on either
// side are no evidence and do not fail -strict; the survive stretch (two ticks under the drink line) is one episode.
func TestArbiterReplaySession(t *testing.T) {
	var b strings.Builder
	for i, life := range []int{100, 30, 30, 100} {
		fmt.Fprintf(&b, `{"time":"2026-11-27T01:00:00Z","msg":"tick","vtMs":%d,"area":"Cold Plains","chain":"chores","facts":{"in_town":0,"life_pct":%d,"hostile":9,"survive_below":50,"potions":3,"potions_min":4,"trip":1}}`+"\n", (i+1)*1000, life)
	}
	log := filepath.Join(t.TempDir(), "night.jsonl")
	if err := os.WriteFile(log, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	rules := filepath.Join("..", "..", defaultOutcomes, "questy-2026-11-25.json")
	var out strings.Builder
	if err := cmdArbiterReplay([]string{"-strict", "-session", log, rules}, &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if s := out.String(); !strings.Contains(s, "ticks 4, scored 4") || !strings.Contains(s, "survive   cases=2 hard 1 vs 0 soft 0 vs 0 pass=true") {
		t.Fatalf("output:\n%s", s)
	}
}
