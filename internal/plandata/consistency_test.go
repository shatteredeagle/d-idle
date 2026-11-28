package plandata_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	capdata "d2lod-bot/data/capabilities"
	"d2lod-bot/internal/plandata"
)

// The data-consistency check of fable review 8 (docs/design/fable-review-2026-11-28/08-builds-as-data.md 3.1): the
// build tables, the combat profiles and the capability table must agree. Every finding is a stable string; the ones
// listed in data/build-cap-ratchet.json are today's known debt and are only reported, a new one fails, and a listed
// one that no longer occurs fails too (remove it from the file: the ratchet only goes down).
// `D2_RATCHET_WRITE=1 go test ./internal/plandata -run TestBuildTablesConsistent` rewrites the file.
//
// The per-level skill cap (points in a skill <= clvl - reqLevel + 1) is U on Game.exe (docs/points.md): its findings
// carry the prefix "cap" and live in their own list so the tables can be fixed down once the question is answered.

const ratchetPath = "../../data/build-cap-ratchet.json"

type capRow struct {
	Class     string   `json:"class"`
	ReqLevel  *int     `json:"reqLevel"`
	ReqSkills []string `json:"reqSkills"`
}

type buildTable struct {
	Build  string `json:"build"`
	Class  string `json:"class"`
	Combat string `json:"combat"`
	Respec *struct {
		Enabled *bool `json:"enabled"`
		AtClvl  int   `json:"at_clvl"`
		Pool    []struct {
			Skill  string `json:"skill"`
			Points int    `json:"points"`
		} `json:"respend_pool"`
	} `json:"respec"`
	Hotkeys []struct {
		Skill string `json:"skill"`
	} `json:"hotkeys"`
	MainAttack []struct {
		Skill string `json:"skill"`
	} `json:"main_attack"`
	Rows []struct {
		Clvl   int `json:"clvl"`
		Skills []struct {
			Skill  string `json:"skill"`
			Points int    `json:"points"`
		} `json:"skills"`
	} `json:"rows"`
}

type ratchet struct {
	Note  string   `json:"note"`
	Cap   []string `json:"cap"`
	Known []string `json:"known"`
}

func TestBuildTablesConsistent(t *testing.T) {
	var caps struct {
		Skills  map[string]capRow          `json:"skills"`
		Generic map[string]json.RawMessage `json:"generic"`
	}
	if err := json.Unmarshal(capdata.JSON, &caps); err != nil {
		t.Fatal(err)
	}
	var capHits, hits []string
	seen := map[string]bool{}
	add := func(list *[]string, format string, a ...any) {
		s := fmt.Sprintf(format, a...)
		if !seen[s] {
			seen[s] = true
			*list = append(*list, s)
		}
	}
	// known reports whether name is a skill of class; generic attacks belong to every class.
	known := func(class, name string) bool {
		if _, ok := caps.Generic[name]; ok {
			return true
		}
		r, ok := caps.Skills[name]
		return ok && r.Class == class
	}

	ents, err := plandata.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	tables := 0
	for _, e := range ents {
		if !strings.HasPrefix(e.Name(), "build-") || e.Name() == plandata.CatalogFile {
			continue
		}
		raw, _ := plandata.FS.ReadFile(e.Name())
		var b buildTable
		if err := json.Unmarshal(raw, &b); err != nil {
			t.Errorf("%s: %v", e.Name(), err)
			continue
		}
		tables++
		name := strings.TrimSuffix(strings.TrimPrefix(e.Name(), "build-"), ".json")
		class := strings.ToLower(b.Class)
		if class == "" {
			add(&hits, "table %s: no class key", name)
			continue
		}
		for _, h := range b.Hotkeys {
			if !known(class, h.Skill) {
				add(&hits, "table %s: hotkey skill %q is not a %s skill in capabilities.json", name, h.Skill, class)
			}
		}
		for _, m := range b.MainAttack {
			if !known(class, m.Skill) {
				add(&hits, "table %s: main_attack skill %q is not a %s skill in capabilities.json", name, m.Skill, class)
			}
		}
		pts := map[string]int{}
		for _, r := range b.Rows {
			if rs := b.Respec; rs != nil && rs.AtClvl == r.Clvl && (rs.Enabled == nil || *rs.Enabled) {
				pts = map[string]int{}
				for _, s := range rs.Pool {
					pts[s.Skill] = s.Points
				}
			}
			for _, s := range r.Skills {
				if s.Points > 0 {
					pts[s.Skill] += s.Points
				}
			}
			for _, s := range r.Skills {
				if s.Points <= 0 {
					continue
				}
				c, ok := caps.Skills[s.Skill]
				if !ok || c.Class != class {
					add(&hits, "table %s: row skill %q is not a %s skill in capabilities.json", name, s.Skill, class)
					continue
				}
				if c.ReqLevel == nil {
					add(&hits, "capabilities: %s skill %q has no reqLevel (rows that buy it are unchecked)", class, s.Skill)
					continue
				}
				if r.Clvl < *c.ReqLevel {
					add(&hits, "table %s: clvl %d buys %q before its required level %d", name, r.Clvl, s.Skill, *c.ReqLevel)
				}
				for _, pre := range c.ReqSkills {
					if pts[pre] == 0 {
						add(&hits, "table %s: clvl %d buys %q without its prerequisite %q", name, r.Clvl, s.Skill, pre)
					}
				}
				if limit := min(20, r.Clvl-*c.ReqLevel+1); pts[s.Skill] > limit && r.Clvl >= *c.ReqLevel {
					add(&capHits, "cap %s clvl %d %q", name, r.Clvl, s.Skill)
				}
			}
		}
	}
	if tables < 20 {
		t.Fatalf("only %d build tables seen", tables)
	}

	var profiles struct {
		Profiles []struct {
			Class  string            `json:"class"`
			Build  string            `json:"build"`
			Layout map[string]string `json:"layout"`
			Buffs  [][]string        `json:"buffs"`
			Bands  []struct {
				Attack []string `json:"attack"`
			} `json:"bands"`
		} `json:"profiles"`
	}
	raw, err := os.ReadFile(filepath.Join("..", "combat", "classdata", "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &profiles); err != nil {
		t.Fatal(err)
	}
	for _, p := range profiles.Profiles {
		id := strings.TrimSuffix(p.Class+"/"+p.Build, "/")
		var names []string
		for n := range p.Layout {
			names = append(names, n)
		}
		for _, g := range p.Buffs {
			names = append(names, g...)
		}
		for _, b := range p.Bands {
			names = append(names, b.Attack...)
		}
		sort.Strings(names)
		for _, n := range names {
			if !known(p.Class, n) {
				add(&hits, "profile %s: skill %q is not a %s skill in capabilities.json", id, n, p.Class)
			}
		}
	}
	sort.Strings(hits)
	sort.Strings(capHits)

	if os.Getenv("D2_RATCHET_WRITE") != "" {
		out, _ := json.MarshalIndent(ratchet{
			Note:  "Known findings of TestBuildTablesConsistent (internal/plandata). cap: rows above the per-level skill cap, enforced only if Game.exe enforces it (U, docs/points.md). known: every other finding. A new finding fails the test; fix findings and delete their lines, never add one without a reason in the commit. Rewrite: D2_RATCHET_WRITE=1 go test ./internal/plandata -run TestBuildTablesConsistent",
			Cap:   capHits,
			Known: hits,
		}, "", " ")
		if err := os.WriteFile(ratchetPath, append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var rt ratchet
	raw, err = os.ReadFile(ratchetPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &rt); err != nil {
		t.Fatal(err)
	}
	compare := func(what string, got, allowed []string) {
		ok := map[string]bool{}
		for _, a := range allowed {
			ok[a] = true
		}
		for _, g := range got {
			if !ok[g] {
				t.Errorf("new %s finding: %s", what, g)
			}
			delete(ok, g)
		}
		for a := range ok {
			t.Errorf("%s finding is fixed, remove it from %s: %s", what, ratchetPath, a)
		}
	}
	compare("cap", capHits, rt.Cap)
	compare("consistency", hits, rt.Known)
	t.Logf("reported, not failing: %d rows above the per-level skill cap (U on Game.exe), %d other known findings", len(capHits), len(hits))
}
