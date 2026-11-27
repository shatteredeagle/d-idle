package gear

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"d2lod-bot/internal/plandata"
)

// The gear build list must name exactly the plandata tables that carry main_attack, and there must be no second
// copy of the rows under internal/gear (the old builds/ directory drifted from the plan tables).
func TestBuildNamesMatchPlandata(t *testing.T) {
	if _, err := os.Stat("builds"); err == nil {
		t.Fatal("internal/gear/builds exists again: the plan tables live only in internal/plandata")
	}
	ents, err := plandata.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for _, e := range ents {
		if !strings.HasPrefix(e.Name(), "build-") || e.Name() == plandata.CatalogFile {
			continue
		}
		b, _ := plandata.FS.ReadFile(e.Name())
		var f struct {
			Main []json.RawMessage `json:"main_attack"`
		}
		if json.Unmarshal(b, &f) == nil && len(f.Main) > 0 {
			want[strings.TrimSuffix(strings.TrimPrefix(e.Name(), "build-"), ".json")] = true
		}
	}
	for _, n := range Builds() {
		if !want[n] {
			t.Errorf("gear lists %q but plandata has no build-%s.json with main_attack", n, n)
		}
		delete(want, n)
	}
	// Guide and experiment tables that carry main_attack but are deliberately not gear-scored.
	for _, n := range []string{"druid_fire_guide", "necromancer_summoner_ce1"} {
		delete(want, n)
	}
	for n := range want {
		t.Errorf("plandata build-%s.json has main_attack but gear.Builds() does not list it", n)
	}
}

// Two builds: BuildJSON returns the plandata bytes themselves, so an edit there is seen by gear at once.
func TestBuildJSONIsPlandata(t *testing.T) {
	for _, n := range []string{"paladin_hammerdin_zeal_endgame90", "sorceress_blizzard_endgame90"} {
		got, err := BuildJSON(n)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := plandata.FS.ReadFile("build-" + n + ".json")
		if string(got) != string(want) {
			t.Errorf("%s: gear copy differs from plandata", n)
		}
	}
}
