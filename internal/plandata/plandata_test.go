package plandata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchesDocs(t *testing.T) {
	ents, err := FS.ReadDir(".")
	if err != nil || len(ents) == 0 {
		t.Fatalf("nothing embedded: %v", err)
	}
	for _, e := range ents {
		doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "plan", "data", e.Name()))
		if err != nil {
			t.Skip("docs/plan/data not present")
		}
		emb, _ := FS.ReadFile(e.Name())
		if string(doc) != string(emb) {
			t.Errorf("internal/plandata/%s is out of date; copy docs/plan/data/%s", e.Name(), e.Name())
		}
	}
}

func TestHasWhatTheConsumersLoad(t *testing.T) {
	for _, n := range []string{"areas", "runs", "progression-normal", "progression-nightmare", "progression-hell",
		"build-primary-blizzard", "build-fallback-lightning", "build-fallback-lightning-v2", "build-paladin-normal30-v2", "build-paladin-normal30", "quest-points",
		"build-necromancer_leveling_normal30", "build-necromancer_summoner_normal30", "build-amazon_bow_normal30", "build-amazon_javelin_normal30",
		"build-druid_fire_normal30", "build-assassin_fireblast_normal30", "build-barbarian_melee_normal30"} {
		if _, err := FS.ReadFile(n + ".json"); err != nil {
			t.Errorf("%s.json: %v", n, err)
		}
	}
}
