package agency

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/danger"
	"d2lod-bot/internal/power"
)

// agreementRow is one row of testdata/agreement.json: the planner against the hand-written combat profile for the
// allocation a shipped plan table implies at a character level. `why` documents every mismatch.
type agreementRow struct {
	Table   string `json:"table"`
	Class   string `json:"class"`
	Clvl    int    `json:"clvl"`
	Profile string `json:"profile"`
	Plan    string `json:"plan"`
	Best    string `json:"best"` // the highest estimate before the prior margin
	Agree   bool   `json:"agree"`
	Why     string `json:"why,omitempty"`
}

var agreementTables = []string{
	"sorceress_normal30", "fallback-lightning-v2", "sorceress_blizzard_endgame90",
	"paladin-zealot-lite", "paladin-normal30-v2", "paladin_hammerdin_endgame90",
	"barbarian_melee_normal30", "necromancer_summoner_normal30", "necromancer_summoner_guide", "paladin_hammerdin_guide", "barbarian_frenzy_guide", "druid_fire_guide", // paper: no engine support needed, the table marks the kinds
}

func computeAgreement(t *testing.T) []agreementRow {
	env := testEnv(t)
	var rows []agreementRow
	for _, tb := range agreementTables {
		for _, clvl := range []int{6, 12, 18, 24, 30} {
			a, comb := allocFromTable(t, tb, clvl)
			env.Combat = combat.Build(comb)
			p := mustDerive(t, a, env)
			r := agreementRow{Table: tb, Class: a.Class, Clvl: clvl, Profile: botPick(t, a.Class, comb, clvl, a.Skills), Plan: attackName(p), Best: p.Best, Agree: false}
			r.Agree = r.Plan == r.Profile
			if !r.Agree {
				pd, ok := dpsOf(p, r.Profile)
				switch {
				case !ok:
					r.Why = fmt.Sprintf("the profile's %s is not a planner candidate (no points or an unsupported tier)", r.Profile)
				case p.Attack == nil:
					r.Why = "the planner finds no usable attack"
				default:
					r.Why = fmt.Sprintf("estimate %s %.0f dps against the profile's %s %.0f dps: more than the %.1fx margin that overrules the band order; the estimate ignores Sacrifice's life cost, counts Zeal as one strike, and gives a summoner's plain swing the damage the pets do", p.Plan(), p.Attack.DPS, r.Profile, pd, env.Caps.Planner.PriorMargin)
				}
			}
			rows = append(rows, r)
		}
	}
	return rows
}

// Plan names the chosen attack for messages.
func (p Plan) Plan() string { return attackName(p) }

func TestPlannerAgreementFixture(t *testing.T) {
	got := computeAgreement(t)
	const path = "testdata/agreement.json"
	if os.Getenv("UPDATE") != "" {
		b, _ := json.MarshalIndent(got, "", " ")
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (UPDATE=1 go test ./internal/agency -run AgreementFixture writes it)", err)
	}
	var want []agreementRow
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(got) {
		t.Fatalf("fixture has %d rows, the run %d", len(want), len(got))
	}
	agree, total, palsor, palsorAgree := 0, 0, 0, 0
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("row %d changed:\n  fixture %+v\n  now     %+v", i, want[i], got[i])
		}
		total++
		if got[i].Agree {
			agree++
		} else if got[i].Why == "" {
			t.Errorf("row %d: a mismatch with no documented reason", i)
		}
		if got[i].Class == "paladin" || got[i].Class == "sorceress" {
			palsor++
			if got[i].Agree {
				palsorAgree++
			}
		}
	}
	t.Logf("planner agrees with the combat profile in %d of %d checkpoints; Paladin and Sorceress %d of %d", agree, total, palsorAgree, palsor)
	if palsorAgree*100 < palsor*80 {
		t.Errorf("Paladin and Sorceress agreement fell to %d of %d (floor 80 percent)", palsorAgree, palsor)
	}
}

// An unsupported skill is never chosen into any slot and is always flagged, whatever the points in it, for every class
// row of the table (the must-fail of ADR 0020: "the hero never silently plays a skill it cannot").
func TestUnsupportedNeverChosenAndAlwaysFlagged(t *testing.T) {
	env := testEnv(t)
	n := 0
	for name, c := range env.Caps.Skills {
		if c.Tier != Unsupported {
			continue
		}
		n++
		a := alloc(c.Class, 30, map[string]int{name: 20})
		p := mustDerive(t, a, env)
		for _, ch := range append(append(append(append([]*Choice{p.Main, p.Attack, p.Secondary, p.Aura}, ptrs(p.Buffs)...), ptrs(p.Summons)...), ptrs(p.Movement)...), ptrs(p.Candidates)...) {
			if ch != nil && ch.Skill == name {
				t.Errorf("%s (%s) is unsupported but the planner chose it as %s", name, c.Class, ch.Role)
			}
		}
		if c.Role != RolePassive && !p.Has(WUnsupportedPts) {
			t.Errorf("%s: 20 points in an unsupported skill raised no %s warning (warnings: %s)", name, WUnsupportedPts, warnCodes(p))
		}
		if c.Role != RolePassive && p.Unplayed.Skills[name] != 20 {
			t.Errorf("%s: unplayed %v", name, p.Unplayed.Skills)
		}
	}
	if n < 10 {
		t.Fatalf("only %d unsupported rows checked", n)
	}
}

func ptrs(c []Choice) []*Choice {
	out := make([]*Choice, len(c))
	for i := range c {
		out[i] = &c[i]
	}
	return out
}

// Paper test: a Barbarian and a Necromancer skill set, no engine support needed for the question "what does the planner
// say": the table marks the kinds the engine cannot play and the plan says so instead of playing them.
func TestPaperClasses(t *testing.T) {
	env := testEnv(t)
	t.Run("barbarian whirlwind", func(t *testing.T) {
		p := mustDerive(t, alloc("barbarian", 30, map[string]int{"Bash": 1, "Double Swing": 1, "Whirlwind": 20, "Leap Attack": 1}), env)
		if p.Attack == nil || p.Attack.Skill == "Whirlwind" || p.Unplayed.Skills["Whirlwind"] != 20 || !p.Has(WUnsupportedPts) {
			t.Fatalf("attack %v unplayed %v warnings %s", p.Attack, p.Unplayed.Skills, warnCodes(p))
		}
	})
	t.Run("necromancer corpse explosion and bone spear", func(t *testing.T) {
		p := mustDerive(t, alloc("necromancer", 30, map[string]int{"Teeth": 5, "Bone Spear": 10, "Corpse Explosion": 10, "Amplify Damage": 5}), env)
		cap := env.Caps.Skills["Corpse Explosion"]
		if cap.Tier == Supported {
			t.Fatalf("Corpse Explosion needs a corpse driver the engine lacks: tier %s", cap.Tier)
		}
		if cap.Tier == Unsupported && p.Unplayed.Skills["Corpse Explosion"] != 10 {
			t.Errorf("unplayed %v", p.Unplayed.Skills)
		}
	})
	t.Run("necromancer summoner has no personal attack to name", func(t *testing.T) {
		p := mustDerive(t, alloc("necromancer", 30, map[string]int{"Raise Skeleton": 20, "Skeleton Mastery": 20, "Clay Golem": 10}), env)
		if p.Archetype != "summoner" || p.Main == nil || p.Main.Role != RoleSummon {
			t.Fatalf("archetype %q main %+v", p.Archetype, p.Main)
		}
	})
}

// Capability table: every row of the two covered classes carries a reason when it is not supported, a requirement, a mana
// source and a delivery; the rows count is the class's 30 skills.
func TestCapabilityTableCoversPaladinAndSorceress(t *testing.T) {
	c, err := LoadCaps()
	if err != nil {
		t.Fatal(err)
	}
	for _, class := range []string{"paladin", "sorceress"} {
		n := 0
		for name, s := range c.Skills {
			if s.Class != class {
				continue
			}
			n++
			if s.ReqLevel < 1 || s.Mana.Source == "" || s.Delivery == "" {
				t.Errorf("%s: incomplete row %+v", name, s)
			}
			if s.Role == RolePassive && s.Mana.Source != "none" {
				t.Errorf("%s: a passive has no mana cost, source %q", name, s.Mana.Source)
			}
			if s.Tier != Supported && s.Note == "" {
				t.Errorf("%s: %s without a reason", name, s.Tier)
			}
		}
		if n != 30 {
			t.Errorf("%s: %d skills in the table, the class has 30", class, n)
		}
	}
}

func TestShadowLineAndLimiter(t *testing.T) {
	env := testEnv(t)
	a, comb := allocFromTable(t, "paladin-zealot-lite", 24)
	pl := &Planner{Env: env}
	v, p, err := pl.Evaluate(Input{Hero: "Testy", Alloc: a, Combat: comb, Weapon: Weapon{Class: "1HS", Avg: 20}})
	if err != nil {
		t.Fatal(err)
	}
	line := ShadowLine(v, p)
	// the attack agrees, the aura does not (planner Might, profile Concentration): the flag says so (auraflap)
	for _, want := range []string{"planner shadow:", "clvl=24", "class=paladin", "plan=Blessed Hammer/", "profile=Blessed Hammer", "agree=false", "aura=Might auraProfile=Concentration", "via=none"} {
		if !strings.Contains(line, want) {
			t.Errorf("line lacks %q: %s", want, line)
		}
	}
	if !strings.Contains(v.Line, "your build: Blessed Hammer") || !strings.Contains(v.Line, "viability: not judged") {
		t.Errorf("view line %q", v.Line)
	}
	if !v.ShadowOnly || v.Mode != "shadow" {
		t.Errorf("a P1 view is shadow only: %+v", v)
	}
	l := NewLimiter(time.Minute)
	t0 := time.Unix(1000, 0)
	if !l.Allow(t0) || l.Allow(t0.Add(59*time.Second)) || !l.Allow(t0.Add(60*time.Second)) || l.Allow(t0.Add(61*time.Second)) {
		t.Error("limiter must allow one line per minute")
	}
}

// madeUpTable is two Normal Act 1 zones; the numbers only have to separate a weak hero from a strong one.
func madeUpTable() *danger.Table {
	z := func(area int, name string, mlvl int, life, dmg float64) *danger.Zone {
		mon := func(reach string) danger.Monster {
			return danger.Monster{Key: reach, Share: 0.5, Life: int(life), Def: 30, ToHit: 60 + 5*mlvl, Dmg: dmg, Reach: reach}
		}
		return &danger.Zone{Area: area, Name: name, Act: 1, Diff: "normal", MLvl: mlvl, Monsters: []danger.Monster{mon("melee"), mon("ranged")},
			Encounters: danger.Encounters{
				Typical: danger.Encounter{N: 4, Melee: 3, Ranged: 1},
				Bad:     danger.Encounter{N: 8, Melee: 6, Ranged: 2},
				Elite:   danger.Encounter{N: 5, Melee: 4, Ranged: 1, Leader: "unique"}}}
	}
	return &danger.Table{Zones: []*danger.Zone{z(2, "Easy Field", 2, 20, 3), z(3, "Hard Field", 12, 120, 20)}}
}

func TestViabilityFollowsTheSurvivalModel(t *testing.T) {
	env := testEnv(t)
	m, err := power.NewModel(danger.DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	tab := madeUpTable()
	a, comb := allocFromTable(t, "paladin_hammerdin_endgame90", 18)
	pl := &Planner{Env: env, Model: m, Tab: tab}
	hero := func(life, def int) power.Hero {
		h := power.Hero{Class: "Paladin", Level: 18, MaxLife: life, MaxMana: 60, Defence: def, BlockPct: 30, AR: 200, DmgMin: 20, DmgMax: 30, Potions: 6, Resist: [4]int{10, 10, 10, 10}}
		for _, term := range []string{power.TermLife, power.TermDefence, power.TermBlock, power.TermResist, power.TermOffence, power.TermPotions} {
			h.SetQuality(term, power.Measured)
		}
		return h
	}
	in := func(h power.Hero) Input {
		return Input{Hero: "Testy", Alloc: a, Combat: comb, Weapon: Weapon{Class: "1HS", Avg: 25}, Power: h, Diff: "normal", Act: 1}
	}
	weak, _, err := pl.Evaluate(in(hero(40, 5)))
	if err != nil {
		t.Fatal(err)
	}
	strong, _, err := pl.Evaluate(in(hero(900, 400)))
	if err != nil {
		t.Fatal(err)
	}
	if weak.Viability == nil || strong.Viability == nil {
		t.Fatalf("no viability: %q / %q", weak.Note, strong.Note)
	}
	if wordRank[strong.Viability.Word] > wordRank[weak.Viability.Word] {
		t.Errorf("more life and defence made the act worse: weak %s strong %s", weak.Viability.Word, strong.Viability.Word)
	}
	if weak.Viability.Word != "too hard" || weak.Viability.Hardest != "Hard Field" {
		t.Errorf("a 40 life hero should find the act too hard: %+v", weak.Viability)
	}
	if strong.Viability.Zones != 2 || strong.Viability.Safe+strong.Viability.Risky+strong.Viability.TooHard != 2 {
		t.Errorf("zones counted %+v", strong.Viability)
	}
	t.Logf("weak: %s\nstrong: %s", weak.Line, strong.Line)
	if _, err := Judge(m, tab, hero(100, 10), mustDerive(t, a, env), "normal", 4); err == nil || !strings.Contains(err.Error(), "act 4") {
		t.Errorf("an act with no zone must be an error naming it: %v", err)
	}
}
