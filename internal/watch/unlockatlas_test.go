package watch

import (
	atlasdata "d2lod-bot/data/atlas"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/unlock"
)

// atlasRig is the fun rig over the shipped ladder with the zone proposals on.
func atlasRig(t *testing.T, class string, level int) (*funRig, *unlock.Engine) {
	t.Helper()
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open(t.TempDir(), strings.ToLower(class), l, func() time.Time { return time.Unix(1_800_000_000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "proposals.json")
	r := newFunRigWith(t, class, level, func(r *funRig, fc *FunConfig) {
		fc.Unlock = eng
		tbl, err := decide.ShippedProposeTable()
		if err != nil {
			t.Fatal(err)
		}
		pr, err := decide.NewProposer(tbl, tallyRanges(currency.DefaultTallyRows()), r.clock, path)
		if err != nil {
			t.Fatal(err)
		}
		fc.Proposals = &ProposalConfig{Proposer: pr, Clock: r.clock, Facts: func() ProposalFacts { return ProposalFacts{Kit: "OK", Potions: 4, Ok: true} }}
	})
	r.hub.SetHero(HeroInfo{Class: class, Level: level, Area: "Blood Moor", AreaID: 2})
	return r, eng
}

func (r *funRig) raiseProposal() {
	r.do("GET", r.view, "/watch?live=1", funToken, "")
	r.fun.Settle()
	for i := 0; i < 20; i++ {
		r.clock.Advance(30 * time.Minute)
		r.fun.Settle()
	}
}

func atlasOf(r *funRig) AtlasView {
	var v AtlasView
	r.getJSON(AtlasRoute, &v)
	return v
}

func TestAtlasGoldenTwoClasses(t *testing.T) {
	for _, tc := range []struct {
		class string
		level int
	}{{"Sorceress", 7}, {"Paladin", 6}} {
		t.Run(tc.class, func(t *testing.T) {
			r, eng := atlasRig(t, tc.class, tc.level)
			if v := atlasOf(r); v.Open || len(v.Slots) != 0 || v.Teaser == "" {
				t.Fatalf("locked view: %+v", v)
			}
			r.raiseProposal()
			r.fun.OnWaypoint("Stony Field", "wp1") // the first waypoint taken, not the proposed one
			r.fun.Settle()
			if !eng.Allowed(AtlasID) {
				t.Fatal("the first waypoint did not open the atlas")
			}
			v := atlasOf(r)
			if v.Lit != 2 || v.Next != "Cold Plains" || v.Total != 9 {
				t.Fatalf("view: %+v", v)
			}
			b, _ := json.Marshal(v)
			golden(t, "atlas-dark-"+strings.ToLower(tc.class), string(b)+"\n")
			// a silhouette names nothing, not even the proposed zone
			for _, s := range v.Slots {
				if s.State == "dark" && (s.Name != "" || s.ID != "" || s.AreaID != 0) {
					t.Fatalf("silhouette leaks: %+v", s)
				}
			}
			r.fun.OnWaypoint("Cold Plains", "wp2")
			r.fun.Settle()
			b, _ = json.Marshal(atlasOf(r))
			golden(t, "atlas-lit-"+strings.ToLower(tc.class), string(b)+"\n")
			found := false
			for _, l := range storyLines(r) {
				found = found || strings.Contains(l, "Unlocked: Waypoint atlas.")
			}
			if !found {
				t.Fatalf("no reveal line: %v", storyLines(r))
			}
		})
	}
}

func TestAtlasReplayAndGates(t *testing.T) {
	r, _ := atlasRig(t, "Sorceress", 1)
	r.fun.OnWaypoint("Cold Plains", "a")
	r.fun.OnWaypoint("Cold Plains", "a") // the same waypoint again
	r.fun.Settle()
	v := atlasOf(r)
	if v.Lit != 2 || v.Next != "" {
		t.Fatalf("%+v", v)
	}
	n := 0
	for _, l := range storyLines(r) {
		if strings.Contains(l, "Waypoint atlas") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("reveal lines: %d", n)
	}
	if resp, _ := r.do("GET", r.view, AtlasRoute, "", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("viewer without token: %d", resp.StatusCode)
	}
	if resp, _ := r.do("POST", r.view, AtlasRoute, funToken, "{}"); resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST: %d", resp.StatusCode)
	}
}

func TestAtlasTableAndErrors(t *testing.T) {
	tb, err := ParseAtlasTable(atlasdata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Rogue Encampment", "Cold Plains", "Stony Field", "Dark Wood", "Black Marsh", "Outer Cloister", "Jail Level 1", "Inner Cloister", "Catacombs Level 2"}
	for i, n := range tb.Nodes {
		if n.Name != want[i] {
			t.Fatalf("node %d is %q, want %q", i, n.Name, want[i])
		}
	}
	// every Codex id of the table exists in the Act 1 firsts
	firsts, err := decide.ShippedFirsts()
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, f := range firsts.Firsts {
		have[f.ID] = f.Kind == "waypoint"
	}
	for _, n := range tb.Nodes {
		if n.CodexID != "" && !have[n.CodexID] {
			t.Errorf("node %q: Codex id %q is not a waypoint first", n.ID, n.CodexID)
		}
	}
	if _, err := ParseAtlasTable([]byte(`{"schema":"atlas.waypoints/1","nodes":[{"id":"a","name":"A","areaId":1},{"id":"a","name":"B","areaId":2}]}`)); err == nil || !strings.Contains(err.Error(), `"a"`) {
		t.Fatalf("duplicate node: %v", err)
	}
	l, _ := unlock.Parse([]byte(`{"schema":"unlock.ladder/1","entries":[
 {"id":"chronicle","group":"ui","trigger":{"kind":"kills_of_kind","args":{"kind":"*","min":1}},"reveal":{"title":"C","text":"c"}}]}`))
	eng, _ := unlock.Open("", "p", l, nil)
	if _, err := NewAtlas(&Unlocks{Engine: eng}, nil, nil, nil); err == nil || !strings.Contains(err.Error(), `"atlas"`) {
		t.Fatalf("missing row: %v", err)
	}
}
