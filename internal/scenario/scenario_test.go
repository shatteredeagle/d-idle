package scenario

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ip(n int) *int { return &n }

func baseScenario() Scenario {
	s := Scenario{ID: "t.one", Title: "t", Kind: KindBehaviour, Tier: TierFast,
		Budget: Budget{VirtualMin: 10}, Matrix: Matrix{Classes: []string{"Paladin", "Sorceress"}, Seeds: []int64{1, 2}},
		Pre:    Pre{HeroProfile: HeroBanked},
		Assert: []Assert{{Kind: AItemInPack, Code: "bks"}}}
	s.applyDefaults()
	return s
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*Scenario){
		"bad id":        func(s *Scenario) { s.ID = "Bad Id" },
		"no budget":     func(s *Scenario) { s.Budget.VirtualMin = 0 },
		"unknown cls":   func(s *Scenario) { s.Matrix.Classes = []string{"Wizard"} },
		"no asserts":    func(s *Scenario) { s.Assert = nil },
		"field no area": func(s *Scenario) { s.Pre.Start = StartField },
		"step no quest": func(s *Scenario) { s.Kind = KindStep },
		"pending why":   func(s *Scenario) { s.Status = "pending" },
		"bad assert":    func(s *Scenario) { s.Assert = []Assert{{Kind: AQuestBits}} },
	}
	for name, mut := range cases {
		s := baseScenario()
		mut(&s)
		if err := s.Validate(); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if err := baseScenario().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFileRefusesUnknownField(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.json")
	os.WriteFile(p, []byte(`{"version":"didle.scenario/1","scenarios":[{"id":"a","typo":1}]}`), 0o644)
	if _, err := LoadFile(p); err == nil {
		t.Fatal("unknown field accepted")
	}
	os.WriteFile(p, []byte(`{"version":"didle.scenario/9","scenarios":[]}`), 0o644)
	if _, err := LoadFile(p); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("version: %v", err)
	}
}

func TestEvaluateAndVerdict(t *testing.T) {
	st := State{Source: "fixture", Has: []string{"items"}, Items: []StateItem{{Code: "bks", Where: "pack"}}, Quests: map[int]uint16{4: 0x2011}, WritesAfterStart: 0}
	as := []Assert{{Kind: AItemInPack, Code: "bks"}, {Kind: AItemAbsent, Code: "bkd"}, {Kind: AQuestBits, Record: 4, Set: 1, Clear: 2}, {Kind: ANoWrites}}
	if status, why := Verdict(Evaluate(as, st)); status != Green {
		t.Fatalf("%s %s", status, why)
	}
	st.Quests[4] = 0x2012
	if status, _ := Verdict(Evaluate(as, st)); status != Red {
		t.Fatalf("0x2012 must fail clear 0x2: %s", status)
	}
	// an unobserved fact is BLOCKED, never green; a failure wins over an unobserved fact
	if status, _ := Verdict(Evaluate([]Assert{{Kind: AAreaIs, Area: 5}}, State{})); status != Blocked {
		t.Fatalf("area unobserved: %s", status)
	}
	rs := Evaluate([]Assert{{Kind: AAreaIs, Area: 5}, {Kind: AItemInPack, Code: "zzz"}}, State{Has: []string{"items"}})
	if status, _ := Verdict(rs); status != Red {
		t.Fatalf("red must beat blocked: %v", rs)
	}
	// no items group read: unobserved even for item_absent
	if r := Evaluate([]Assert{{Kind: AItemAbsent, Code: "x"}}, State{}); !r[0].Unobserved {
		t.Fatal("items not read must be unobserved")
	}
	if r := Evaluate([]Assert{{Kind: ANoWrites}}, State{WritesAfterStart: -1}); !r[0].Unobserved {
		t.Fatal("unknown write count must be unobserved")
	}
	if r := Evaluate([]Assert{{Kind: AAreaChangesLE, Max: ip(3)}}, State{AreaChanges: ip(5)}); r[0].OK {
		t.Fatal("5 > 3")
	}
	if w := Evaluate([]Assert{{Kind: AWealthBetween, Min: 700}}, State{Gold: ip(300), Stash: ip(450)}); !w[0].OK {
		t.Errorf("wealth 750 >= 700: %+v", w[0])
	}
	if w := Evaluate([]Assert{{Kind: AWealthBetween, Min: 700}}, State{Gold: ip(300), Stash: ip(100)}); w[0].OK {
		t.Errorf("wealth 400 >= 700 passed")
	}
	gold := Evaluate([]Assert{{Kind: AGoldBetween, Min: 1, Max: ip(10)}}, State{Gold: ip(11)})
	if gold[0].OK {
		t.Fatal("gold 11 > 10")
	}
}

func TestBuildSetupDeterministicAndBuildAgnostic(t *testing.T) {
	s := baseScenario()
	s.Pre = Pre{Level: 17, Gold: 500, Waypoints: []int{1, 3}, Items: []Item{{Code: "bks"}}, Board: &QuestRef{Quest: "search_for_cain", Chunk: "scroll"}}
	for _, class := range []string{"Paladin", "Sorceress"} { // two classes: the setup must not hard-wire one
		a, err := BuildSetup(s, class, 11)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := BuildSetup(s, class, 11)
		if len(a.Verbs) != len(b.Verbs) {
			t.Fatal("not deterministic")
		}
		for i := range a.Verbs {
			if a.Verbs[i] != b.Verbs[i] {
				t.Fatalf("verb %d differs", i)
			}
			if a.Verbs[i].Reason == "" || !strings.Contains(a.Verbs[i].Reason, class) {
				t.Fatalf("ledger reason missing the class: %+v", a.Verbs[i])
			}
		}
		if a.Tags[0] != "CHEATED" || a.Tuning["quests.board"] != 1 {
			t.Fatalf("tags %v tuning %v", a.Tags, a.Tuning)
		}
	}
	if _, err := BuildSetup(s, "Wizard", 1); err == nil {
		t.Fatal("unknown class must be an error, not a silent fallback")
	}
	s.Pre = Pre{Level: 8, Gold: 0, Stash: 2500}
	su, _ := BuildSetup(s, "Paladin", 1)
	if su.Tags[0] != "SETUP-only" {
		t.Fatalf("tags %v", su.Tags)
	}
	s.Pre = Pre{Start: StartField, Area: 5, Items: []Item{{Code: "bks", Where: "stash"}}}
	su, _ = BuildSetup(s, "Paladin", 1)
	if !su.Field || len(su.Unsupported) == 0 {
		t.Fatalf("field %v unsupported %v", su.Field, su.Unsupported)
	}
}

func TestHeroNameValidForTheBoardStore(t *testing.T) {
	n := HeroName("cain.scroll.sandbox", "Paladin", 1234567)
	if len(n) < 2 || len(n) > 24 || n[0] < 'A' || n[0] > 'z' {
		t.Fatalf("hero name %q", n)
	}
}

func TestSelectExpandTiersAndChanged(t *testing.T) {
	a, b, c := baseScenario(), baseScenario(), baseScenario()
	a.ID, b.ID, c.ID = "a.x", "b.y", "c.z"
	b.Tier, c.Tier = TierFull, TierNightly
	a.Touches = []string{"internal/town/"}
	c.Status, c.PendingReason = "pending", "skeleton"
	all := []Scenario{a, b, c}
	if got := Select(all, Selector{Tier: TierFast}); len(got) != 1 || got[0].ID != "a.x" {
		t.Fatalf("fast: %v", got)
	}
	if got := Select(all, Selector{Tier: TierFull}); len(got) != 2 {
		t.Fatalf("full includes fast: %v", got)
	}
	if got := Select(all, Selector{Tier: TierFast, Changed: []string{"internal/combat/x.go"}}); len(got) != 0 {
		t.Fatalf("untouched scenario must not run: %v", got)
	}
	if got := Select(all, Selector{Tier: TierFast, Changed: []string{"internal/town/stash.go"}}); len(got) != 1 {
		t.Fatalf("touched: %v", got)
	}
	runs, err := Expand(Select(all, Selector{Tier: TierNightly}), Selector{Tier: TierNightly})
	if err != nil || len(runs) != 8 { // a and b: 2 classes x 2 seeds; the pending c gives none
		t.Fatalf("%d runs %v", len(runs), err)
	}
	runs, _ = Expand([]Scenario{a}, Selector{Tier: TierFast})
	if len(runs) != 1 {
		t.Fatalf("fast is one cell: %v", runs)
	}
	sd := int64(2)
	runs, _ = Expand([]Scenario{a}, Selector{Tier: TierFast, Class: "Sorceress", Seed: &sd})
	if len(runs) != 1 || runs[0].Class != "Sorceress" || runs[0].Seed != 2 {
		t.Fatalf("dev loop: %v", runs)
	}
	if _, err := Expand([]Scenario{a}, Selector{Class: "Wizard"}); err == nil {
		t.Fatal("unknown class")
	}
	if got := Select(all, Selector{IDs: []string{"b."}}); len(got) != 1 || got[0].ID != "b.y" {
		t.Fatalf("prefix: %v", got)
	}
	if Estimate(all, runs, 99) < 1 {
		t.Fatal("estimate")
	}
}

func TestCoverageFindsUncoveredAndUnknown(t *testing.T) {
	bs := []Behaviour{{ID: "hook.a", Group: "loop-hook", Where: "x"}, {ID: "hook.b", Group: "loop-hook", Where: "x"}, {ID: "quest.q.c1", Group: "quest", Where: "x"}, {ID: "hook.c", Group: "loop-hook", Where: "x"}}
	s1 := baseScenario()
	s1.Covers = []string{"hook.a"}
	s2 := baseScenario()
	s2.ID, s2.Status, s2.PendingReason, s2.Covers = "t.two", "pending", "r", []string{"hook.c"}
	s3 := baseScenario()
	s3.ID, s3.Quest, s3.Covers = "t.three", &QuestRef{Quest: "q", Chunk: "c1"}, []string{"hook.nope"}
	rows, problems := Coverage(bs, []Scenario{s1, s2, s3})
	if len(problems) != 1 || !strings.Contains(problems[0], "hook.nope") {
		t.Fatalf("problems %v", problems)
	}
	var buf bytes.Buffer
	CoverageReport(&buf, rows)
	out := buf.String()
	for _, want := range []string{"covered 2, pending-only 1, UNCOVERED 1", "hook.b"} {
		if !strings.Contains(out, want) {
			t.Fatalf("report lacks %q:\n%s", want, out)
		}
	}
}

func TestLoadBehavioursRejectsDuplicates(t *testing.T) {
	p := filepath.Join(t.TempDir(), "behaviours.json")
	os.WriteFile(p, []byte(`{"version":"didle.behaviours/1","behaviours":[{"id":"a","group":"g","title":"t","where":"w"},{"id":"a","group":"g","title":"t","where":"w"}]}`), 0o644)
	if _, err := LoadBehaviours(p); err == nil {
		t.Fatal("duplicate accepted")
	}
}

func TestDeterminism(t *testing.T) {
	st := State{Source: "fixture", Items: []StateItem{{Code: "b", Where: "pack"}, {Code: "a", Where: "pack"}}}
	st2 := st
	st2.Items = []StateItem{{Code: "a", Where: "pack"}, {Code: "b", Where: "pack"}}
	if DigestState(st) != DigestState(st2) {
		t.Fatal("item order must not change the digest")
	}
	r := Result{Run: Run{ID: "x", Class: "Paladin", Seed: 1}, Status: Green, Digest: "d1"}
	ok, vd, dd, miss := CheckDeterminism([]Result{r}, []Result{r})
	if !ok || len(vd)+len(dd)+len(miss) != 0 {
		t.Fatal("identical runs")
	}
	r2 := r
	r2.Status = Red
	if ok, vd, _, _ := CheckDeterminism([]Result{r}, []Result{r2}); ok || len(vd) != 1 {
		t.Fatal("a verdict change is a failure")
	}
	r3 := r
	r3.Digest = "d2"
	if ok, _, dd, _ := CheckDeterminism([]Result{r}, []Result{r3}); !ok || len(dd) != 1 {
		t.Fatal("a digest-only change is a warning")
	}
	if ok, _, _, miss := CheckDeterminism([]Result{r}, nil); ok || len(miss) != 1 {
		t.Fatal("a missing run is a failure")
	}
}

func TestGateAndReport(t *testing.T) {
	g := Result{Run: Run{ID: "a", Class: "Paladin", Seed: 1}, Status: Green}
	p := Result{Run: Run{ID: "p"}, Status: Pending, Reason: "skeleton"}
	if !Gate([]Result{g, p}) {
		t.Fatal("pending must not fail the gate")
	}
	for _, bad := range []string{Red, Blocked} {
		if Gate([]Result{g, {Run: Run{ID: "b"}, Status: bad}}) {
			t.Fatalf("%s must fail the gate", bad)
		}
	}
	if Gate(nil) {
		t.Fatal("an empty run set is not green")
	}
	var buf bytes.Buffer
	Report(&buf, []Result{g, p})
	if !strings.Contains(buf.String(), "GREEN 1  RED 0  BLOCKED 0  PENDING 1") {
		t.Fatal(buf.String())
	}
}

// The shipped data must load, validate and build a setup for every class of every scenario.
func TestShippedScenarios(t *testing.T) {
	scs, err := LoadDir("../../tools/provingground/scenarios")
	if err != nil {
		t.Fatal(err)
	}
	if len(scs) < 10 {
		t.Fatalf("only %d scenarios", len(scs))
	}
	bs, err := LoadBehaviours("../../tools/provingground/scenarios/behaviours.json")
	if err != nil {
		t.Fatal(err)
	}
	cs, err := ChunkBehaviours("../../data/questboard")
	if err != nil {
		t.Fatal(err)
	}
	if _, problems := Coverage(append(bs, cs...), scs); len(problems) > 0 {
		t.Fatal(problems)
	}
	for _, s := range scs {
		for _, c := range s.Matrix.Classes {
			if _, err := BuildSetup(s, c, s.Matrix.Seeds[0]); err != nil {
				t.Errorf("%s %s: %v", s.ID, c, err)
			}
		}
	}
}

func TestStatBetweenAssertion(t *testing.T) {
	max0 := 0
	st := State{WritesAfterStart: 0, Stats: map[string]int{"statpts": 25, "skillpts": 0, "vit": 40}}
	as := []Assert{
		{Kind: AStatBetween, Code: "statpts", Min: 25, Max: func() *int { v := 25; return &v }()},
		{Kind: AStatBetween, Code: "skillpts", Max: &max0},
		{Kind: AStatBetween, Code: "vit", Min: 41},
		{Kind: AStatBetween, Code: "ene", Min: 1},
	}
	for _, a := range as {
		if err := a.Validate(); err != nil {
			t.Fatalf("%+v: %v", a, err)
		}
	}
	got := Evaluate(as, st)
	if !got[0].OK || !got[1].OK || got[2].OK || got[2].Unobserved || !got[3].Unobserved {
		t.Errorf("%+v", got)
	}
	if (Assert{Kind: AStatBetween, Code: "luck", Min: 1}).Validate() == nil || (Assert{Kind: AStatBetween, Code: "vit"}).Validate() == nil {
		t.Error("a bad code or no bound must be refused")
	}
}

func TestSkillAssertions(t *testing.T) {
	st := State{RightSkill: ip(98)}
	if r := Evaluate([]Assert{{Kind: ARightSkillIs, Skill: 98}}, st); !r[0].OK {
		t.Fatal(r)
	}
	if r := Evaluate([]Assert{{Kind: ARightSkillIs, Skill: 102}}, st); r[0].OK || r[0].Unobserved {
		t.Fatal(r)
	}
	if r := Evaluate([]Assert{{Kind: ALeftSkillIs, Skill: 36}}, st); !r[0].Unobserved {
		t.Fatal("an unread hand is unobserved, not red")
	}
	if (Assert{Kind: ARightSkillIs}).Validate() == nil {
		t.Fatal("right_skill_is needs skill")
	}
}

// barb1: the d2d class index follows the character screen (Barbarian 0, Amazon 3), the same table as data/smoke/rules.json;
// the old table had them swapped and a Barbarian scenario ran an Amazon.
func TestClassIndexMatchesSmokeRules(t *testing.T) {
	for name, want := range map[string]int{"Barbarian": 0, "Necromancer": 1, "Paladin": 2, "Amazon": 3, "Sorceress": 4, "Druid": 5, "Assassin": 6} {
		if got, err := ClassIndex(name); err != nil || got != want {
			t.Errorf("%s: %d %v, want %d", name, got, err, want)
		}
	}
	if _, err := ClassIndex("Warlock"); err == nil {
		t.Error("an unknown class must be an error")
	}
}

// druid1: the Druid index is 5 and every class matches data/smoke/rules.json (the file the smoke driver reads), so a swap like
// the Barbarian and Amazon one of barb1 fails here for any class.
func TestClassIndexMatchesSmokeRulesFile(t *testing.T) {
	raw, err := os.ReadFile("../../data/smoke/rules.json")
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		Classes map[string]struct {
			Index int `json:"d2d_index"`
		} `json:"classes"`
	}
	if err := json.Unmarshal(raw, &rules); err != nil {
		t.Fatal(err)
	}
	if len(rules.Classes) != 7 {
		t.Fatalf("rules.json lists %d classes", len(rules.Classes))
	}
	for name, c := range rules.Classes {
		title := strings.ToUpper(name[:1]) + name[1:]
		if got, err := ClassIndex(title); err != nil || got != c.Index {
			t.Errorf("%s: scenario index %d %v, smoke rules %d", title, got, err, c.Index)
		}
	}
	if got, _ := ClassIndex("Druid"); got != 5 {
		t.Errorf("Druid index %d, want 5", got)
	}
}

// pre.hero_profile: there is no default (a missing one is refused), a typo or a played hero with raw point writes is refused.
func TestHeroProfileValidate(t *testing.T) {
	s := baseScenario()
	s.Pre.HeroProfile = ""
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "pre.hero_profile is missing") {
		t.Fatalf("missing profile: %v", err)
	}
	if s.HeroProfile() != "" {
		t.Fatalf("a default crept back: %q", s.HeroProfile())
	}
	s.Pre.HeroProfile = "bank"
	if s.Validate() == nil {
		t.Error("unknown profile accepted")
	}
	s.Pre.HeroProfile = HeroPlayed
	s.Pre.Stats = []StatSet{{Stat: 4, Value: 10}}
	if s.Validate() == nil {
		t.Error("played with a raw statpts write accepted")
	}
	s.Pre.HeroProfile = HeroBanked
	if err := s.Validate(); err != nil {
		t.Errorf("banked with raw points: %v", err)
	}
}
