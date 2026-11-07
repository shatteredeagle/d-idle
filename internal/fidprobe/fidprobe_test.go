package fidprobe

import (
	"context"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

var update = flag.Bool("update", false, "rewrite golden files")

func snap(class d2.Class) *d2.Snapshot {
	p := &d2.Player{Class: class, Name: "X", Act: 0, Area: 1, X: 100, Y: 200, Level: 6, Exp: 3000, Gold: 500,
		Strength: 20, Dexterity: 25, Vitality: 30, Energy: 40, Life: 1000, MaxLife: 1000, Mana: 600, MaxMana: 600,
		Stamina: 200, MaxStamina: 200, Resist: d2.Resists{Fire: 5}, LeftSkill: 36, RightSkill: 40,
		Skills: []d2.Skill{{ID: 36, Level: 1, Bonus: 1}, {ID: 40, Level: 4}},
		Stats:  d2.Stats{0: 20, 2: 25, 6: 256000, 7: 256000, 8: 153600, 9: 153600, 10: 51200, 11: 51200}}
	inv := &d2.Inventory{Items: []d2.Unit{
		{Class: 5, Item: &d2.Item{Where: d2.ContInventory, GX: 1, GY: 2, Quality: 2, ILvl: 3, Flags: d2.FlagIdentified}},
		{Class: 9, Item: &d2.Item{Where: d2.ContEquipment, Body: 4, Quality: 3}},
	}}
	return &d2.Snapshot{State: d2.StateInGame, Player: p, Inventory: inv,
		Monsters: []d2.Unit{{Class: 150, X: 10, Y: 11}, {Class: 148, X: 20, Y: 21}},
		Objects:  []d2.Unit{{Class: 119, X: 5, Y: 6}}}
}

func prog() *d2.PlayerProgress {
	pp := &d2.PlayerProgress{}
	pp.Waypoints[0] = 1
	pp.Quest[0][3] = 0x11
	return pp
}

func sheet(t *testing.T, name string, s *d2.Snapshot) *Sheet {
	t.Helper()
	sh, err := FromInput(Input{Source: name, Snap: s, Progress: prog()})
	if err != nil {
		t.Fatal(err)
	}
	return sh
}

func rules(t *testing.T) *Rules {
	t.Helper()
	r, err := DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func classOf(r *Report, key string) string {
	for _, c := range r.Cells {
		if c.Key == key {
			return c.Class
		}
	}
	return "absent"
}

func TestIdenticalIsAllEqual(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		a, b := sheet(t, "A", snap(cl)), sheet(t, "B", snap(cl))
		r := Diff(a, b, nil, rules(t))
		if !r.Pass() || r.Counts[Unreadable] != 0 || r.Counts[KnownDev] != 0 {
			t.Fatalf("%v: %v", cl, r.Counts)
		}
	}
}

func TestSeededDifferences(t *testing.T) {
	sa, sb := snap(d2.Sorceress), snap(d2.Sorceress)
	sb.Player.Level++                           // int off by one
	sb.Player.MaxLife = 1003                    // 0.3 percent: inside
	sb.Player.MaxMana = 604                     // 0.66 percent: outside
	sb.Inventory.Items = sb.Inventory.Items[:1] // set missing one member
	sb.Player.Skills[0].Bonus = 0               // registered known deviation
	b := sheet(t, "B", sb)
	b.Unreadable("derived.res_cold", "no resist read") // unreadable
	r := Diff(sheet(t, "A", sa), b, nil, rules(t))
	want := map[string]string{
		"base.level": Mismatch, "derived.max_life": Equal, "derived.max_mana": Mismatch,
		"items.equipment": Mismatch, "items.inventory": Equal, "skills.bonus": KnownDev,
		"derived.res_cold": Unreadable, "base.gold": Equal,
	}
	for k, w := range want {
		if g := classOf(r, k); g != w {
			t.Errorf("%s: got %s want %s", k, g, w)
		}
	}
	if r.Pass() {
		t.Fatal("must fail with mismatches")
	}
	for _, c := range r.Cells {
		if c.Key == "skills.bonus" && (c.Row != "SK14" || c.Reason == "") {
			t.Errorf("known cell lacks row/reason: %+v", c)
		}
		if c.Key == "items.equipment" && !strings.Contains(c.Detail, "only A") {
			t.Errorf("set detail: %q", c.Detail)
		}
	}
}

func TestAbsentFieldIsUnreadableNotEqual(t *testing.T) {
	a, b := sheet(t, "A", snap(d2.Sorceress)), sheet(t, "B", snap(d2.Sorceress))
	delete(b.Fields, "base.gold")
	r := Diff(a, b, nil, rules(t))
	if classOf(r, "base.gold") != Unreadable || !r.Pass() {
		t.Fatal(r.Counts)
	}
}

func TestProgressMissingAndUnknownMap(t *testing.T) {
	in := Input{Source: "B", Snap: snap(d2.Sorceress), ProgressWhy: "stock d2d", Unreadable: map[string]string{"item.class": "no tables", "stats": "n/a"}}
	b, err := FromInput(in)
	if err != nil {
		t.Fatal(err)
	}
	r := Diff(sheet(t, "A", snap(d2.Sorceress)), b, nil, rules(t))
	for _, k := range []string{"waypoints.d0", "quests.d1", "items.inventory", "stats.vector"} {
		if classOf(r, k) != Unreadable {
			t.Errorf("%s not unreadable", k)
		}
	}
}

func TestGroupFilterAndEmptyGroup(t *testing.T) {
	a, b := sheet(t, "A", snap(d2.Sorceress)), sheet(t, "B", snap(d2.Sorceress))
	b.Fields["base.level"] = Field{Kind: KindInt, I: 99}
	r := Diff(a, b, []string{"items", "merc"}, rules(t))
	if r.Counts[Mismatch] != 0 || len(r.EmptyGroups) != 1 || r.EmptyGroups[0] != "merc" || r.Counts[Unreadable] != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestNotInGame(t *testing.T) {
	if _, err := FromInput(Input{Source: "A", Snap: &d2.Snapshot{State: d2.StateLoading}}); err == nil {
		t.Fatal("want error")
	}
}

type fake struct {
	name string
	in   []Input
	n    int
}

func (f *fake) Name() string { return f.name }
func (f *fake) Read(context.Context) (Input, error) {
	i := f.in[min(f.n, len(f.in)-1)]
	f.n++
	return i, nil
}

func TestRunTwoBuildsAndStability(t *testing.T) {
	scs, _ := Scenarios()
	if len(scs) != 15 {
		t.Fatalf("catalogue has %d", len(scs))
	}
	for _, c := range []struct {
		ref string
		cl  d2.Class
	}{{"1", d2.Sorceress}, {"fresh-paladin-1", d2.Paladin}} {
		sc, err := FindScenario(c.ref)
		if err != nil {
			t.Fatal(err)
		}
		in := Input{Snap: snap(c.cl), Progress: prog()}
		rep, err := Run(context.Background(), sc, &fake{name: "gameexe", in: []Input{in}}, &fake{name: "d2d", in: []Input{in}}, 3, time.Millisecond, rules(t))
		if err != nil || !rep.Pass() {
			t.Fatalf("%s: %v %v", c.ref, err, rep)
		}
	}
	sc, _ := FindScenario("1")
	if _, err := Run(context.Background(), sc, &fake{name: "a", in: []Input{{Snap: snap(d2.Paladin), Progress: prog()}}},
		&fake{name: "b", in: []Input{{Snap: snap(d2.Paladin), Progress: prog()}}}, 1, 0, rules(t)); err == nil {
		t.Fatal("wrong class must be refused")
	}
	moving := snap(d2.Sorceress)
	moved := snap(d2.Sorceress)
	moved.Player.X++
	_, err := ReadStable(context.Background(), &fake{name: "a", in: []Input{{Snap: moving}, {Snap: moved}}}, 2, 0, rules(t))
	if err == nil {
		t.Fatal("unstable read must fail")
	}
}

func TestGolden(t *testing.T) {
	sa, sb := snap(d2.Sorceress), snap(d2.Sorceress)
	sb.Player.Level++
	sb.Player.MaxMana = 604
	sb.Inventory.Items = sb.Inventory.Items[:1]
	sb.Player.Skills[0].Bonus = 0
	b := sheet(t, "d2d", sb)
	b.Unreadable("derived.res_cold", "no resist read")
	r := Diff(sheet(t, "gameexe", sa), b, nil, rules(t))
	r.Scenario = "fresh-sorceress-1"
	got := r.Text(false)
	const golden = "testdata/report.golden"
	if *update {
		os.WriteFile(golden, []byte(got), 0o644)
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("report differs from %s:\n%s", golden, got)
	}
	if j, err := r.JSON(); err != nil || len(j) == 0 {
		t.Fatal(err)
	}
}

func TestParseInputEnvelope(t *testing.T) {
	doc := `{"report":{"Unknown":{"item.class":"x"}},"snapshot":{"State":2,"Player":{"Class":1},"Inventory":{}},"progress":null}`
	in, err := ParseInput([]byte(doc))
	if err != nil || in.Snap.Player.Class != d2.Sorceress || in.Unreadable["item.class"] == "" || in.ProgressWhy == "" {
		t.Fatalf("%+v %v", in, err)
	}
}
