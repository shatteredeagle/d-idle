package unlock

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/currency"
)

const fixture = `{"schema":"unlock.ladder/1","entries":[
 {"id":"map","group":"senses","tier":1,"trigger":{"kind":"codex_first","args":{"sub":"area"}},"alt":[{"kind":"level","args":{"min":2}}],
  "reveal":{"title":"Minimap","text":"You can see the map.","teaser":"Meet a new place."}},
 {"id":"dots","group":"senses","tier":2,"trigger":{"kind":"codex_first","args":{"sub":"monster","ids":["fallen","quill_rat"]}},
  "reveal":{"title":"Enemy dots","text":"Red dots.","teaser":"Meet a monster."}},
 {"id":"quest","group":"ui","trigger":{"kind":"quest","args":{"quest":"Den of Evil"}},"reveal":{"title":"Quest log","text":"Q"}},
 {"id":"best","group":"ui","trigger":{"kind":"kills_of_kind","args":{"kind":"Fallen","min":10}},"reveal":{"title":"Bestiary","text":"B"}},
 {"id":"hall","group":"ui","trigger":{"kind":"retire_count","args":{"min":1}},"reveal":{"title":"Hall","text":"H"}},
 {"id":"loot","group":"senses","tier":5,"trigger":{"kind":"tally_spend"},"price":40,"reveal":{"title":"Eye for loot","text":"L","teaser":"Costs tally."}}
]}`

func mustLadder(t *testing.T, s string) *Ladder {
	t.Helper()
	l, err := Parse([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func ids(es []Entry) string {
	var s []string
	for _, e := range es {
		s = append(s, e.ID)
	}
	return strings.Join(s, ",")
}

func TestTriggerKinds(t *testing.T) {
	cases := []struct {
		name string
		ev   Event
		want string
	}{
		{"codex area", Event{Type: EvCodexFirst, Sub: "area", ID: "blood_moor", Name: "Blood Moor"}, "map"},
		{"codex wrong sub", Event{Type: EvCodexFirst, Sub: "waypoint", ID: "wp"}, ""},
		{"codex monster in ids", Event{Type: EvCodexFirst, Sub: "monster", ID: "fallen", Name: "Fallen"}, "dots"},
		{"codex monster not in ids", Event{Type: EvCodexFirst, Sub: "monster", ID: "griswold", Name: "Griswold"}, ""},
		{"level below", Event{Type: EvLevel, N: 1}, ""},
		{"level alt", Event{Type: EvLevel, N: 2}, "map"},
		{"quest", Event{Type: EvQuest, Name: "den of evil"}, "quest"},
		{"kills short", Event{Type: EvKills, Name: "Fallen", N: 9}, ""},
		{"kills other kind", Event{Type: EvKills, Name: "Quill Rat", N: 50}, ""},
		{"kills", Event{Type: EvKills, Name: "fallen", N: 10}, "best"},
		{"retire", Event{Type: EvRetire, N: 1}, "hall"},
	}
	for _, c := range cases {
		e, _ := Open("", "p", mustLadder(t, fixture), nil)
		got, err := e.Observe(c.ev)
		if err != nil || ids(got) != c.want {
			t.Errorf("%s: got %q err %v, want %q", c.name, ids(got), err, c.want)
		}
	}
}

func TestReplayIdempotentAndNeverRelocks(t *testing.T) {
	e, _ := Open("", "p", mustLadder(t, fixture), nil)
	ev := Event{Type: EvLevel, N: 5}
	if got, _ := e.Observe(ev); ids(got) != "map" {
		t.Fatal(ids(got))
	}
	for i := 0; i < 3; i++ {
		if got, _ := e.Observe(ev); len(got) != 0 {
			t.Fatalf("replay reopened %s", ids(got))
		}
	}
	// a lower level later (a new hero) does not close it
	e.Observe(Event{Type: EvLevel, N: 1})
	if !e.Allowed("map") || !e.SenseTier(1) || e.SenseTier(2) {
		t.Fatal("state")
	}
}

func TestPersistenceAcrossReload(t *testing.T) {
	dir := t.TempDir()
	clk := func() time.Time { return time.Date(2026, 11, 8, 1, 2, 3, 0, time.UTC) }
	e, err := Open(dir, "sorc", mustLadder(t, fixture), clk)
	if err != nil {
		t.Fatal(err)
	}
	e.Observe(Event{Type: EvCodexFirst, Sub: "monster", ID: "fallen", Name: "Fallen"})
	e2, err := Open(dir, "sorc", mustLadder(t, fixture), clk)
	if err != nil {
		t.Fatal(err)
	}
	if !e2.Allowed("dots") || !e2.Met("monster", "fallen") || e2.Met("monster", "griswold") {
		t.Fatal("reload lost state")
	}
	if got, _ := e2.Observe(Event{Type: EvCodexFirst, Sub: "monster", ID: "fallen", Name: "Fallen"}); len(got) != 0 {
		t.Fatal("reload re-revealed")
	}
	// profiles are separate
	e3, _ := Open(dir, "pal", mustLadder(t, fixture), clk)
	if e3.Allowed("dots") {
		t.Fatal("profile leak")
	}
	b, _ := os.ReadFile(Path(dir, "sorc"))
	if !strings.Contains(string(b), `"at": "2026-11-08T01:02:03Z"`) {
		t.Fatal(string(b))
	}
}

type fakeSpender struct {
	bal   int64
	keys  map[string]bool
	calls int
}

func (f *fakeSpender) Spend(key string, price int64, note string) error {
	f.calls++
	if f.keys[key] {
		return nil
	}
	if f.bal < price {
		return currency.ErrInsufficient
	}
	f.bal -= price
	f.keys[key] = true
	return nil
}

func TestBuy(t *testing.T) {
	e, _ := Open(t.TempDir(), "p", mustLadder(t, fixture), nil)
	sp := &fakeSpender{bal: 39, keys: map[string]bool{}}
	if p := e.Pending(); len(p) != 1 || p[0].ID != "loot" || p[0].Price != 40 || p[0].Text != "" {
		t.Fatalf("pending %+v", p)
	}
	if _, err := e.Buy("loot", sp); !errors.Is(err, currency.ErrInsufficient) || e.Allowed("loot") || sp.bal != 39 {
		t.Fatalf("short: %v", err)
	}
	sp.bal = 100
	if _, err := e.Buy("loot", sp); err != nil || !e.SenseTier(5) || sp.bal != 60 {
		t.Fatalf("buy: %v bal %d", err, sp.bal)
	}
	if _, err := e.Buy("loot", sp); !errors.Is(err, ErrAlreadyOpen) || sp.bal != 60 {
		t.Fatalf("again: %v", err)
	}
	if len(e.Pending()) != 0 {
		t.Fatal("still pending")
	}
	if _, err := e.Buy("quest", sp); !errors.Is(err, ErrNotBuyable) {
		t.Fatal(err)
	}
	if _, err := e.Buy("nope", sp); !errors.Is(err, ErrUnknownEntry) || !strings.Contains(err.Error(), "nope") {
		t.Fatal(err)
	}
}

func TestBuyAfterCrashDoesNotChargeTwice(t *testing.T) {
	dir := t.TempDir()
	sp := &fakeSpender{bal: 100, keys: map[string]bool{}}
	e, _ := Open(dir, "p", mustLadder(t, fixture), nil)
	sp.keys["unlock/p/loot"] = true // paid, then the process died before the save
	sp.bal = 60
	if _, err := e.Buy("loot", sp); err != nil || sp.bal != 60 {
		t.Fatalf("%v %d", err, sp.bal)
	}
}

func TestUnknownEventAndTrigger(t *testing.T) {
	e, _ := Open("", "p", mustLadder(t, fixture), nil)
	if _, err := e.Observe(Event{Type: "teleport"}); !errors.Is(err, ErrUnknownEvent) || !strings.Contains(err.Error(), "teleport") {
		t.Fatal(err)
	}
	bad := strings.Replace(fixture, `"kind":"quest"`, `"kind":"vibes"`, 1)
	if _, err := Parse([]byte(bad)); !errors.Is(err, ErrUnknownTrigger) || !strings.Contains(err.Error(), "vibes") || !strings.Contains(err.Error(), `"quest"`) {
		t.Fatal(err)
	}
	if _, err := e.Check("ghost"); !errors.Is(err, ErrUnknownEntry) {
		t.Fatal(err)
	}
}

func TestParseRejects(t *testing.T) {
	for name, tc := range map[string]struct{ from, to, want string }{
		"no price":    {`"price":40,`, ``, "price above 0"},
		"free price":  {`"kind":"quest","args"`, `"kind":"quest","price":3,"args"`, "unknown field"},
		"dup tier":    {`"tier":2`, `"tier":1`, "both own senses tier 1"},
		"silent":      {`"text":"Q"`, `"text":""`, "nothing unlocks silently"},
		"bad group":   {`"group":"ui"`, `"group":"hud"`, "unknown group"},
		"min zero":    {`"min":10`, `"min":0`, "min must be"},
		"missing arg": {`"args":{"quest":"Den of Evil"}`, `"args":{}`, `needs arg "quest"`},
		"tier on ui":  {`{"id":"quest","group":"ui",`, `{"id":"quest","group":"ui","tier":3,`, "senses group only"},
		"dup id":      {`"id":"hall"`, `"id":"quest"`, "duplicate"},
	} {
		s := strings.Replace(fixture, tc.from, tc.to, 1)
		if s == fixture {
			t.Fatalf("%s: fixture unchanged", name)
		}
		if _, err := Parse([]byte(s)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want %q", name, err, tc.want)
		}
	}
}

func TestShippedParses(t *testing.T) {
	if _, err := Shipped(); err != nil {
		t.Fatal(err)
	}
}

func TestStatesHideLockedText(t *testing.T) {
	e, _ := Open("", "p", mustLadder(t, fixture), nil)
	for _, s := range e.States("") {
		if s.Text != "" {
			t.Fatalf("locked %s leaks text", s.ID)
		}
	}
	e.Observe(Event{Type: EvQuest, Name: "Den of Evil"})
	var open int
	for _, s := range e.States("ui") {
		if s.Open {
			open++
			if s.Text == "" || s.Teaser != "" {
				t.Fatalf("%+v", s)
			}
		}
	}
	if open != 1 {
		t.Fatal(open)
	}
}

func TestBossKillTrigger(t *testing.T) {
	const row = `{"schema":"unlock.ladder/1","entries":[
 {"id":"any","group":"pacts","trigger":{"kind":"boss_kill","args":{"boss":"Andariel"}},"reveal":{"title":"A","text":"a"}},
 {"id":"normal","group":"pacts","trigger":{"kind":"boss_kill","args":{"boss":"Baal","difficulty":"normal"}},"reveal":{"title":"B","text":"b"}}]}`
	for _, c := range []struct {
		ev   Event
		want string
	}{
		{Event{Type: EvBossKill, Name: "andariel", Sub: "hell"}, "any"},
		{Event{Type: EvBossKill, Name: "Baal", Sub: "nightmare"}, ""},
		{Event{Type: EvBossKill, Name: "Baal", Sub: "Normal"}, "normal"},
		{Event{Type: EvBossKill, Name: "Mephisto", Sub: "normal"}, ""},
		{Event{Type: EvKills, Name: "Andariel", N: 5}, ""},
	} {
		e, _ := Open("", "p", mustLadder(t, row), nil)
		got, err := e.Observe(c.ev)
		if err != nil || ids(got) != c.want {
			t.Errorf("%+v: got %q err %v, want %q", c.ev, ids(got), err, c.want)
		}
	}
	for _, bad := range []string{
		`{"kind":"boss_kill","args":{}}`,
		`{"kind":"boss_kill","args":{"boss":"Baal","difficulty":"easy"}}`,
	} {
		s := `{"schema":"unlock.ladder/1","entries":[{"id":"x","group":"pacts","trigger":` + bad + `,"reveal":{"title":"A","text":"a"}}]}`
		if _, err := Parse([]byte(s)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	e, _ := Open("", "p", mustLadder(t, row), nil)
	if ok, err := e.Adopt("any"); !ok || err != nil || !e.Allowed("any") {
		t.Fatal("adopt")
	}
	if got, _ := e.Observe(Event{Type: EvBossKill, Name: "Andariel", Sub: "normal"}); len(got) != 0 {
		t.Fatal("an adopted entry opened again")
	}
}
