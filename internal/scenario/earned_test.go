package scenario

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const milestonesFile = "../../tools/provingground/earned/milestones.json"

func mustMilestones(t *testing.T) MilestoneFile {
	t.Helper()
	ms, err := LoadMilestones(milestonesFile)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

func zero() *int { n := 0; return &n }

func earnedRec(class string) LedgerRecord {
	return LedgerRecord{Run: "j1/necro.act1-earned/" + class + "/11", Scenario: "necro.act1-earned", Class: class, Seed: 11, Verdict: Green,
		Setup: SetupEarned, SetupTags: []string{SetupEarned}, BotCommit: "abc123", D2DSHA256: "d2dhash", WritesN: zero(), Tier: TierT4,
		Host: "builder", Started: "2026-11-27T10:00:00Z"}
}

func goodManifest(t *testing.T) SaveManifest {
	t.Helper()
	files := []SaveFile{{Name: "Hero.d2s", SHA256: strings.Repeat("a", 64)}}
	m, err := NewManifest(earnedRec("Paladin"), mustMilestones(t), "clvl-12", "Hero", 12, files, "", time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMilestonesFile(t *testing.T) {
	ms := mustMilestones(t)
	for _, id := range []string{"den-done", "clvl-12", "cain-freed", "before-andariel"} {
		if _, ok := ms.Find(id); !ok {
			t.Errorf("milestone %s missing", id)
		}
	}
}

// A CHEATED, written, dirty, red or too-low hero never founds an earned chain. Dropping any one check makes its case fail.
func TestManifestRefusesNonEarnedHeroesMustFail(t *testing.T) {
	ms := mustMilestones(t)
	files := []SaveFile{{Name: "Hero.d2s", SHA256: strings.Repeat("a", 64)}}
	now := time.Now()
	cases := map[string]func(*LedgerRecord){
		"cheated":    func(r *LedgerRecord) { r.Setup, r.SetupTags = SetupCheated, []string{SetupCheated} },
		"impossible": func(r *LedgerRecord) { r.Setup = SetupImpossible },
		"setup":      func(r *LedgerRecord) { r.Setup = SetupSetup },
		"writes":     func(r *LedgerRecord) { n := 3; r.WritesN = &n },
		"no writes":  func(r *LedgerRecord) { r.WritesN = nil },
		"dirty":      func(r *LedgerRecord) { r.BotDirty = true },
		"red":        func(r *LedgerRecord) { r.Verdict = Red },
		"tag":        func(r *LedgerRecord) { r.SetupTags = []string{SetupEarned, SetupCheated} },
	}
	for name, mut := range cases {
		r := earnedRec("Paladin")
		mut(&r)
		if _, err := NewManifest(r, ms, "clvl-12", "Hero", 12, files, "", now); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := NewManifest(earnedRec("Paladin"), ms, "clvl-12", "Hero", 11, files, "", now); err == nil {
		t.Error("level below the milestone accepted")
	}
	if _, err := NewManifest(earnedRec("Paladin"), ms, "nowhere", "Hero", 12, files, "", now); err == nil {
		t.Error("unknown milestone accepted")
	}
	if _, err := NewManifest(earnedRec("Paladin"), ms, "clvl-12", "Hero", 12, []SaveFile{{Name: "../x", SHA256: "00"}}, "", now); err == nil {
		t.Error("path in a file name accepted")
	}
	if _, err := NewManifest(earnedRec("Warlock"), ms, "clvl-12", "Hero", 12, files, "", now); err == nil {
		t.Error("unknown class accepted")
	}
}

func TestPortability(t *testing.T) {
	m := goodManifest(t)
	if got := m.Portability("d2dhash"); got != PortSame {
		t.Fatalf("same binary: %s", got)
	}
	if got := m.Portability("newer"); got != PortUnverified {
		t.Fatalf("other binary, never loaded: %s", got)
	}
	m.LoadedOn = []SaveLoad{{D2DSHA256: "newer", Result: "ok", Run: "j9/x/Paladin/1"}}
	if got := m.Portability("newer"); got != PortVerified {
		t.Fatalf("loaded ok: %s", got)
	}
	m.LoadedOn = append(m.LoadedOn, SaveLoad{D2DSHA256: "newer", Result: "changed", Run: "j10/x/Paladin/1"})
	if got := m.Portability("newer"); got != PortRefused {
		t.Fatalf("a later change must win: %s", got)
	}
}

func TestResolveSave(t *testing.T) {
	ms := mustMilestones(t)
	dir := t.TempDir()
	if _, err := ResolveSave(dir, "clvl-12", "Paladin", ms); err == nil || !strings.Contains(err.Error(), "clvl-12/Paladin") {
		t.Fatalf("missing save: %v", err)
	}
	m := goodManifest(t)
	path := ManifestPath(dir, "clvl-12", "Paladin")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveSave(dir, "clvl-12", "Paladin", ms)
	if err != nil || got.Run != m.Run {
		t.Fatalf("resolve: %v %+v", err, got)
	}
	if _, err := ResolveSave(dir, "clvl-12", "Sorceress", ms); err == nil {
		t.Fatal("another class resolved")
	}
	m.Setup = SetupCheated // a hand-edited manifest is refused on read
	b, _ = json.Marshal(m)
	_ = os.WriteFile(path, b, 0o644)
	if _, err := ResolveSave(dir, "clvl-12", "Paladin", ms); err == nil {
		t.Fatal("a CHEATED manifest resolved")
	}
}

func fromSaveScenario() Scenario {
	return Scenario{ID: "earned.step", Title: "t", Kind: KindBehaviour, Tier: TierFull, Build: "lab",
		Pre:    Pre{HeroProfile: HeroBanked, Start: StartTown, FromSave: "clvl-12"},
		Budget: Budget{VirtualMin: 10}, Matrix: Matrix{Classes: []string{"Paladin", "Sorceress"}, Seeds: []int64{1}},
		Assert: []Assert{{Kind: "no_writes"}}}
}

func TestFromSaveScenarioIsEarnedForTwoClasses(t *testing.T) {
	s := fromSaveScenario()
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := s.CheckProvenance(); err != nil {
		t.Fatal(err)
	}
	for _, c := range s.Matrix.Classes {
		su, err := BuildSetup(s, c, 1)
		if err != nil {
			t.Fatal(err)
		}
		if SetupTag(su.Tags) != SetupEarned || su.FromSave != "clvl-12" || len(su.Verbs) != 0 {
			t.Fatalf("%s: tags %v from %q verbs %d", c, su.Tags, su.FromSave, len(su.Verbs))
		}
		// no verb loads a save yet: the run is BLOCKED naming it, never faked
		if len(su.Unsupported) != 1 || !strings.Contains(su.Unsupported[0], "from_save clvl-12") {
			t.Fatalf("%s: unsupported %v", c, su.Unsupported)
		}
	}
}

// Every setup write turns the earned hero into a changed one; the validation names it.
func TestFromSaveRefusesSetupWrites(t *testing.T) {
	for name, mut := range map[string]func(*Scenario){
		"gold":    func(s *Scenario) { s.Pre.Gold = 100 },
		"level":   func(s *Scenario) { s.Pre.Level = 20 },
		"quests":  func(s *Scenario) { s.Pre.Quests = []QWord{{Record: 4, Word: 1}} },
		"pack":    func(s *Scenario) { s.Pre.Pack = []Item{{Code: "hp1"}} },
		"worn":    func(s *Scenario) { s.Pre.Worn = []WornItem{{Code: "cap", Slot: "head"}} },
		"field":   func(s *Scenario) { s.Pre.Start, s.Pre.Area = StartField, 2 },
		"cheated": func(s *Scenario) { s.Provenance = "CHEATED: x" },
		"id":      func(s *Scenario) { s.Pre.FromSave = "Bad Id" },
	} {
		s := fromSaveScenario()
		mut(&s)
		if err := s.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// The tag rules: a fresh hero with no setup write is EARNED; a level verb is SETUP-only; a warp is CHEATED and must say so.
func TestSetupTagsAndProvenanceRule(t *testing.T) {
	base := func() Scenario {
		s := fromSaveScenario()
		s.Pre.FromSave = ""
		return s
	}
	tagOf := func(s Scenario) string {
		su, err := BuildSetup(s, "Paladin", 1)
		if err != nil {
			t.Fatal(err)
		}
		return SetupTag(su.Tags)
	}
	fresh := base()
	if tagOf(fresh) != SetupEarned {
		t.Fatalf("fresh hero: %s", tagOf(fresh))
	}
	lvl := base()
	lvl.Pre.Level = 5
	if tagOf(lvl) != SetupSetup {
		t.Fatalf("level verbs: %s", tagOf(lvl))
	}
	warp := base()
	warp.Pre = Pre{Start: StartField, Area: 2, Warp: &Cell{X: 1, Y: 1}}
	if tagOf(warp) != SetupCheated {
		t.Fatalf("warp: %s", tagOf(warp))
	}
	if err := warp.CheckProvenance(); err == nil {
		t.Fatal("a CHEATED scenario without the word in provenance passed")
	}
	warp.Provenance = "CHEATED position: warped, mechanics only."
	if err := warp.CheckProvenance(); err != nil {
		t.Fatal(err)
	}
}

func TestLedgerEarnedRules(t *testing.T) {
	ok := earnedRec("Paladin")
	ok.FromSave, ok.SaveRun, ok.SavePort = "clvl-12", "j0/x/Paladin/1", PortSame
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*LedgerRecord){
		"writes":   func(r *LedgerRecord) { n := 1; r.WritesN = &n },
		"no run":   func(r *LedgerRecord) { r.SaveRun = "" },
		"unverif.": func(r *LedgerRecord) { r.SavePort = PortUnverified },
		"setup":    func(r *LedgerRecord) { r.Setup = SetupSetup },
	} {
		r := ok
		mut(&r)
		if err := r.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestReleaseEarned(t *testing.T) {
	e := earnedRec("Paladin")
	e.BotCommit = "rel1"
	cheated := e
	cheated.Setup, cheated.Run = SetupCheated, "j2/cain.chain/Paladin/11"
	red := e
	red.Verdict, red.Run = Red, "j3/x/Paladin/1"
	t2 := e
	t2.Tier, t2.Run = TierT2, "j4/y/Paladin/1"
	if _, err := ReleaseEarned([]LedgerRecord{cheated, red, t2}, "rel1"); err == nil {
		t.Fatal("a release without an EARNED green act run passed")
	}
	if _, err := ReleaseEarned([]LedgerRecord{e}, "rel2"); err == nil {
		t.Fatal("an EARNED run of another commit counted")
	}
	got, err := ReleaseEarned([]LedgerRecord{cheated, red, t2, e}, "rel1")
	if err != nil || len(got) != 1 || got[0].Run != e.Run {
		t.Fatalf("%v %v", got, err)
	}
	for _, r := range ReleaseView([]LedgerRecord{cheated, e}, "rel1") {
		if r.Setup == SetupCheated {
			t.Fatal("a CHEATED run is in the release view")
		}
	}
}
