package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

// knownHotkeyDrift is the ratchet of card hotkeydrift: per plan table, how many of its `hotkeys` rows name a skill the
// table's combat profile binds to another key or not at all (S: counted by this test; V by running it, 2026-11-26). Card classkeys took the Amazon, Barbarian, Druid and hammerdin tables to zero.
// Nothing reads Build.Hotkeys outside the parser (S: grep), and on d2d every learned skill gets a pseudo key
// (d2dcombat Rig.BindSkills), so the drift costs nothing today; it would on the game.exe path, where the profile layout is
// the key map. A table may only lower its number; a new table starts at zero (the profile layout is the one source of truth).
var knownHotkeyDrift = map[string]int{
	"assassin_fireblast_normal30": 2, "assassin_sentry_endgame90": 5,
	"paladin-normal30": 5, "paladin-normal30-v2": 4, "paladin-zealot-lite": 3, "sorceress_blizzard_endgame90": 5,
}

func classByName(name string) (d2.Class, bool) {
	for c := d2.Class(0); c < 8; c++ {
		if strings.EqualFold(c.String(), name) {
			return c, true
		}
	}
	return 0, false
}

func TestBuildHotkeysDoNotDriftFromTheProfileLayout(t *testing.T) {
	files, _ := filepath.Glob("../../internal/plandata/build-*.json")
	if len(files) < 20 {
		t.Fatalf("only %d plan tables found", len(files))
	}
	got := map[string]int{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var b struct {
			Class   string `json:"class"`
			Combat  string `json:"combat"`
			Hotkeys []struct {
				Key   string `json:"key"`
				Skill string `json:"skill"`
			} `json:"hotkeys"`
		}
		if json.Unmarshal(raw, &b) != nil || len(b.Hotkeys) == 0 {
			continue // a catalog or a table without a hotkey plan
		}
		if b.Class == "" {
			b.Class = "sorceress" // the loader's default (leveler.Build.Class)
		}
		c, ok := classByName(b.Class)
		if !ok {
			t.Errorf("%s: unknown class %q", f, b.Class)
			continue
		}
		cb := combat.Build(b.Combat)
		if b.Combat == "" && c == d2.Sorceress {
			cb = combat.BuildLightning // the legacy sorceress tables predate the combat key
		}
		prof, err := combat.LookupProfile(c, cb)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		lay := map[string]string{}
		for k, v := range prof.Layout {
			lay[strings.ToLower(k)] = strings.ToLower(v)
		}
		n := 0
		for _, h := range b.Hotkeys {
			if lay[strings.ToLower(h.Skill)] != strings.ToLower(h.Key) {
				n++
			}
		}
		got[strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "build-"), ".json")] = n
	}
	var names []string
	for n := range got {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		want := knownHotkeyDrift[n]
		switch {
		case got[n] > want:
			t.Errorf("table %s: %d hotkey rows differ from the profile layout, ratchet %d (fix the table or the layout)", n, got[n], want)
		case got[n] < want:
			t.Errorf("table %s: drift fell to %d, lower knownHotkeyDrift from %d", n, got[n], want)
		}
	}
}
