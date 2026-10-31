package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
)

func testEnv(snaps ...*d2.Snapshot) (*Env, *bytes.Buffer) {
	var buf bytes.Buffer
	i := 0
	clock := time.Unix(1700000000, 0)
	return &Env{
		Out: &buf,
		Snap: func() (*d2.Snapshot, error) {
			s := snaps[min(i, len(snaps)-1)]
			i++
			return s, nil
		},
		Sleep: func(d time.Duration) { clock = clock.Add(d) },
		Now:   func() time.Time { return clock },
		ItemCode: func(c int) string {
			return map[int]string{1: "r14", 2: "gcg", 3: "r07", 4: "xxx"}[c]
		},
		MonsterName: func(c int) (string, bool) { return map[int]string{10: "zombie1", 20: "akara"}[c], c == 20 },
		MissileName: func(c int) string { return map[int]string{5: "firewall1", 6: "mystery"}[c] },
		CurseIDs:    d2.DefaultCurseIDs,
		Missiles:    func() ([]d2.Unit, bool) { return []d2.Unit{{Class: 5, ID: 9, X: 1, Y: 2}, {Class: 6}}, true },
	}, &buf
}

func lines(t *testing.T, b *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(b.String()), "\n") {
		if l == "" {
			continue
		}
		m := map[string]any{}
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("not JSON: %q: %v", l, err)
		}
		if m["probe"] == nil || m["kind"] == nil || m["t"] == nil {
			t.Fatalf("missing envelope keys: %q", l)
		}
		out = append(out, m)
	}
	return out
}

func kinds(ls []map[string]any, k string) []map[string]any {
	var out []map[string]any
	for _, l := range ls {
		if l["kind"] == k {
			out = append(out, l)
		}
	}
	return out
}

func game(units ...d2.Unit) *d2.Snapshot {
	return &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 1, X: 100, Y: 100, Area: 3}, Monsters: units}
}

func TestArgParsing(t *testing.T) {
	noGame := func(io.Writer) (*Env, func(), error) { return nil, nil, errors.New("needs game") }
	for _, tc := range []struct {
		args []string
		want string // substring of the error; the build error means parsing passed
	}{
		{nil, "usage"},
		{[]string{"nope"}, "unknown sub"},
		{[]string{"monflags", "-sec", "0"}, "out of range"},
		{[]string{"monflags", "-sec", "171"}, "out of range"},
		{[]string{"monflags", "-ms", "5"}, "-ms"},
		{[]string{"monflags", "extra"}, "unexpected"},
		{[]string{"runewords"}, "want 'check'"},
		{[]string{"runewords", "cube", "-recipe", "zzz"}, "unknown recipe"},
		{[]string{"runewords", "cube", "-recipe", "custom"}, "needs -inputs"},
		{[]string{"missiledump", "-ms", "1"}, "-ms"},
		{[]string{"wpdump", "-bogus"}, "wpdump"},
		{[]string{"monflags", "-sec", "5", "-radius", "30"}, "needs game"},
		{[]string{"runewords", "check"}, "needs game"},
		{[]string{"runewords", "cube", "-go"}, "needs game"},
		{[]string{"mercdump"}, "needs game"},
		{[]string{"wpdump", "-sec", "3"}, "needs game"},
	} {
		err := run(tc.args, io.Discard, noGame)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: err %v, want %q", tc.args, err, tc.want)
		}
	}
}

func TestMonflagsShape(t *testing.T) {
	champ := d2.Unit{ID: 7, Class: 10, X: 105, Y: 100, Mode: 1, Monster: &d2.MonsterInfo{Flags: 4, Enchants: [9]uint8{5, 9}, Life: 10, MaxLife: 20}}
	far := d2.Unit{ID: 8, Class: 10, X: 400, Y: 400, Mode: 1, Monster: &d2.MonsterInfo{Flags: 2}}
	npc := d2.Unit{ID: 9, Class: 20, X: 101, Y: 100, Mode: 1, Monster: &d2.MonsterInfo{}}
	merc := d2.Unit{ID: 11, Class: 271, X: 101, Y: 100, Mode: 1}
	dead := d2.Unit{ID: 12, Class: 10, X: 101, Y: 100, Mode: d2.MonModeDead, Monster: &d2.MonsterInfo{Flags: 8}}
	e, buf := testEnv(game(champ, far, npc, merc, dead))
	if err := runMonflags(e, monOpts{sec: 3, ms: 1000, radius: 40}); err != nil {
		t.Fatal(err)
	}
	ls := lines(t, buf)
	mons := kinds(ls, "monster")
	if len(mons) != 1 {
		t.Fatalf("want 1 monster, got %d: %v", len(mons), buf.String())
	}
	m := mons[0]
	if m["flags"] != "0x04" || m["mon_kind"] != "champion" || m["name"] != "zombie1" || m["enchants_trusted"] != true {
		t.Errorf("record: %v", m)
	}
	if ids := m["enchant_ids"].([]any); len(ids) != 2 || ids[0].(float64) != 5 {
		t.Errorf("enchant ids %v", ids)
	}
	sum := kinds(ls, "summary")[0]
	if sum["monsters"].(float64) != 1 || sum["kinds"].(map[string]any)["champion"].(float64) != 1 {
		t.Errorf("summary %v", sum)
	}
}

func TestMonVerdict(t *testing.T) {
	if v := monVerdict(0, nil, nil); !strings.Contains(v, "no hostile") {
		t.Error(v)
	}
	if v := monVerdict(3, map[string]int{"0x00": 3}, map[string]int{"unknown": 3}); !strings.Contains(v, "all flags zero") {
		t.Error(v)
	}
}

func TestMercdumpAndDialog(t *testing.T) {
	merc := d2.Unit{ID: 50, Class: 271, X: 90, Y: 90, Mode: 1, Monster: &d2.MonsterInfo{LifePct: 80, Level: 3}}
	npc := d2.Unit{ID: 60, Class: 147, Type: d2.TypeMonster, X: 95, Y: 95, Mode: 1}
	s := game(merc, npc)
	s.UI[d2.UINPCMenu] = 1
	s.Hovered = d2.Hover{Active: true, Type: d2.TypeMonster, ID: 60}
	e, buf := testEnv(s)
	if err := runMercdump(e, 2); err != nil {
		t.Fatal(err)
	}
	ls := lines(t, buf)
	m := kinds(ls, "merc")
	if len(m) != 2 || m[0]["present"] != true || m[0]["life_pct"].(float64) != 80 {
		t.Fatalf("merc lines: %v", m)
	}
	d := kinds(ls, "dialog")
	if len(d) != 2 {
		t.Fatalf("dialog lines %d", len(d))
	}
	if l := d[0]["lines_expected"].([]any); len(l) != 4 || l[2] != "Gamble" {
		t.Errorf("Gheed menu: %v", l)
	}
}

func TestMissiledump(t *testing.T) {
	s := game()
	s.Player.States[0] = 1 << uint(d2.DefaultCurseIDs.Amplify)
	e, buf := testEnv(s)
	e.StateName = func(id int) string { return "state" }
	if err := runMissiledump(e, 1, 250); err != nil {
		t.Fatal(err)
	}
	ls := lines(t, buf)
	p := kinds(ls, "player")[0]
	if p["curses"].(map[string]any)["amplify"] != true {
		t.Errorf("amplify not seen: %v", p)
	}
	sum := kinds(ls, "summary")[0]
	other := sum["classified_other"].([]any)
	if len(other) != 1 || other[0] != "mystery" {
		t.Errorf("classified_other %v", other)
	}
	if sum["missile_names"].(map[string]any)["firewall1"].(map[string]any)["kind"] != "firewall" {
		t.Errorf("summary %v", sum)
	}
}

func TestMissileTableFailure(t *testing.T) {
	e, buf := testEnv(game())
	e.Missiles = func() ([]d2.Unit, bool) { return nil, false }
	if err := runMissiledump(e, 1, 500); err != nil {
		t.Fatal(err)
	}
	sum := kinds(lines(t, buf), "summary")[0]
	if !strings.Contains(sum["verdict"].(string), "unreadable") {
		t.Errorf("%v", sum)
	}
}

func TestWpdump(t *testing.T) {
	s := game()
	s.Player.Addr = 0x1234
	s.Objects = []d2.Unit{{ID: 3, Class: 237, X: 5100, Y: 5200}, {ID: 4, Class: 1}}
	s.UI[d2.UIWaypoint] = 1
	e, buf := testEnv(s)
	e.Progress = func(a uint32) (*d2.PlayerProgress, error) {
		if a != 0x1234 {
			t.Errorf("addr %x", a)
		}
		return &d2.PlayerProgress{Waypoints: [3]uint64{1<<18 | 1<<30, 0, 0}}, nil
	}
	if err := runWpdump(e, 1); err != nil {
		t.Fatal(err)
	}
	ls := lines(t, buf)
	pr := kinds(ls, "progress")
	if len(pr) != 3 {
		t.Fatalf("progress lines %d", len(pr))
	}
	rows := pr[0]["acts3to5"].([]any)
	if len(rows) != 21 || rows[0].(map[string]any)["active"] != true || rows[12].(map[string]any)["name"] != "Harrogath" || rows[12].(map[string]any)["active"] != true || rows[1].(map[string]any)["active"] != false {
		t.Errorf("rows %v", rows)
	}
	st := kinds(ls, "state")[0]
	if st["panel_open"] != true || len(st["waypoint_objects"].([]any)) != 1 {
		t.Errorf("state %v", st)
	}
}

func TestRunewordsCheckAndCandidates(t *testing.T) {
	inv := &d2.Inventory{Items: []d2.Unit{
		{ID: 1, Class: 3, Item: &d2.Item{Where: d2.ContInventory}}, // r07 Tal
		{ID: 2, Class: 3, Item: &d2.Item{Where: d2.ContStash}},
		{ID: 3, Class: 4, Item: &d2.Item{Where: d2.ContInventory, Flags: d2.FlagSocketed}},
		{ID: 4, Class: 4, Item: &d2.Item{Where: d2.ContEquipment, Flags: d2.FlagRuneword}}, // worn: ignored
	}}
	s := game()
	s.Inventory = inv
	e, buf := testEnv(s)
	e.Runewords = []RuneRow{{Name: "Two Tals", Ladder: true, Runes: []string{"r07", "r07"}}, {Name: "Needs Eth", Runes: []string{"r05"}}}
	if err := runRunewords(context.Background(), e, rwOpts{}); err != nil {
		t.Fatal(err)
	}
	sum := kinds(lines(t, buf), "summary")[0]
	if sum["socketed_bases"].(float64) != 1 || sum["runes"].(map[string]any)["r07"].(float64) != 2 {
		t.Errorf("summary %v", sum)
	}
	c := sum["candidates"].([]any)
	if len(c) != 1 || c[0].(map[string]any)["name"] != "Two Tals" || c[0].(map[string]any)["ladder"] != true {
		t.Errorf("candidates %v", c)
	}
}

func TestRunewordsCube(t *testing.T) {
	mk := func() *d2.Snapshot {
		s := game()
		s.Inventory = &d2.Inventory{Items: []d2.Unit{
			{ID: 1, Class: 1, Item: &d2.Item{Where: d2.ContInventory}}, {ID: 2, Class: 1, Item: &d2.Item{Where: d2.ContInventory}},
			{ID: 3, Class: 1, Item: &d2.Item{Where: d2.ContInventory}}, {ID: 4, Class: 2, Item: &d2.Item{Where: d2.ContInventory}},
		}}
		return s
	}
	o := rwOpts{cube: true, recipe: "hel", inputs: recipes["hel"]}

	e, buf := testEnv(mk())
	if err := runRunewords(context.Background(), e, o); err != nil {
		t.Fatal(err)
	}
	if v := kinds(lines(t, buf), "verdict")[0]; v["result"] != "dry-run" {
		t.Errorf("without -go: %v", v)
	}

	o.run = true
	e, buf = testEnv(mk())
	var got []string
	e.Transmute = func(_ context.Context, in []string) ([]string, error) { got = in; return []string{"r15"}, nil }
	closed := false
	e.CloseAll = func() error { closed = true; return nil }
	if err := runRunewords(context.Background(), e, o); err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || !closed {
		t.Errorf("transmute inputs %v closed=%v", got, closed)
	}
	if v := kinds(lines(t, buf), "verdict")[0]; v["result"] != "ladder-recipe-works-offline" {
		t.Errorf("verdict %v", v)
	}

	e, buf = testEnv(mk())
	e.Transmute = func(context.Context, []string) ([]string, error) {
		return nil, errors.New("town: transmute produced nothing")
	}
	if err := runRunewords(context.Background(), e, o); err == nil {
		t.Error("want the transmute error back")
	}
	if v := kinds(lines(t, buf), "verdict")[0]; v["result"] != "error" {
		t.Errorf("verdict %v", v)
	}

	o.inputs = []string{"r14", "r14", "r14", "r14"} // four Dol: pack has three
	e, buf = testEnv(mk())
	if err := runRunewords(context.Background(), e, o); err == nil || !strings.Contains(err.Error(), "lacks") {
		t.Errorf("missing inputs: %v", err)
	}
}

func TestSnapErrorBecomesRecord(t *testing.T) {
	e, buf := testEnv()
	e.Snap = d2.New(mem.NewFake()).Snapshot // empty memory: every read fails
	if err := runWpdump(e, 1); err == nil {
		t.Fatal("want error")
	}
	if k := kinds(lines(t, buf), "error"); len(k) != 1 {
		t.Errorf("error record: %v", buf.String())
	}
}
