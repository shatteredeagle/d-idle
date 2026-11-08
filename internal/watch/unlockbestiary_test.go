package watch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bestiarydata "d2lod-bot/data/bestiary"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/unlock"
)

type fakeMonsters map[string]*data.Monster

func (f fakeMonsters) MonsterByName(n string) *data.Monster { return f[strings.ToLower(n)] }

// bestiaryTables has every shipped kind; Fallen is the one the tests read. Values are fixture numbers, not game data.
func bestiaryTables(t *testing.T) fakeMonsters {
	t.Helper()
	notes, err := ParseBestiaryNotes(bestiarydata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	m := fakeMonsters{}
	for _, n := range notes {
		m[strings.ToLower(n.Name)] = &data.Monster{Name: n.Name, Type: "demon", Melee: true}
	}
	f := m["fallen"]
	f.Res[data.Normal][data.Fire], f.Res[data.Nightmare][data.Fire], f.Res[data.Hell][data.Fire] = 0, 25, 100
	f.Elem[0] = data.MonElem{Type: "fire", Pct: 20}
	return m
}

func bestiaryRig(t *testing.T, class string) (*funRig, *unlock.Engine) {
	t.Helper()
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open(t.TempDir(), strings.ToLower(class), l, func() time.Time { return time.Unix(1_800_000_000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	return newFunRigWith(t, class, 1, func(_ *funRig, fc *FunConfig) { fc.Unlock = eng; fc.Monsters = bestiaryTables(t) }), eng
}

func bestiaryOf(r *funRig) BestiaryView {
	var v BestiaryView
	r.getJSON(BestiaryRoute, &v)
	return v
}

func kills(name string, n int, seq0 uint64) []goals.Event {
	var evs []goals.Event
	for i := 0; i < n; i++ {
		evs = append(evs, goals.Event{Type: goals.EvKill, Monster: name, Seq: seq0 + uint64(i)})
	}
	return evs
}

func cardOf(v BestiaryView, id string) BestiaryCard {
	for _, c := range v.Cards {
		if c.ID == id {
			return c
		}
	}
	return BestiaryCard{}
}

func TestBestiaryGoldenTwoClasses(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			r, eng := bestiaryRig(t, class)
			if v := bestiaryOf(r); v.Open || len(v.Cards) != 0 || v.Teaser == "" {
				t.Fatalf("locked view: %+v", v)
			}
			r.feed(kills("Fallen", 9, 1)...)
			if eng.Allowed(BestiaryID) {
				t.Fatal("opened at 9 kills")
			}
			r.feed(kills("Fallen", 1, 10)...)
			r.feed(kills("Zombie", 3, 11)...)
			if !eng.Allowed(BestiaryID) || eng.Allowed(BestiaryResistsID) {
				t.Fatal("ten kills must open the cards and not the resists")
			}
			b, _ := json.Marshal(bestiaryOf(r))
			golden(t, "bestiary-card-"+strings.ToLower(class), string(b)+"\n")
			r.feed(kills("Fallen", 40, 14)...)
			if !eng.Allowed(BestiaryResistsID) {
				t.Fatal("fifty kills did not open the resists")
			}
			b, _ = json.Marshal(bestiaryOf(r))
			golden(t, "bestiary-full-"+strings.ToLower(class), string(b)+"\n")
			found := false
			for _, l := range storyLines(r) {
				found = found || strings.Contains(l, "Unlocked: Cain's notes.")
			}
			if !found {
				t.Fatalf("no reveal line: %v", storyLines(r))
			}
		})
	}
}

func TestBestiarySpoilerRule(t *testing.T) {
	r, _ := bestiaryRig(t, "Sorceress")
	r.feed(kills("Fallen", 49, 1)...)
	r.feed(kills("Zombie", 2, 50)...)
	_, raw := r.do("GET", r.view, BestiaryRoute, funToken, "")
	if strings.Contains(string(raw), "resists") || strings.Contains(string(raw), "immune") {
		t.Fatalf("resist details before the 50th kill: %s", raw)
	}
	v := bestiaryOf(r)
	if c := cardOf(v, "fallen"); c.State != "card" || c.Next != 1 || c.Resists != nil || c.Tip == "" {
		t.Fatalf("fallen at 49: %+v", c)
	}
	if c := cardOf(v, "zombie"); c.State != "met" || c.Tip != "" || c.Type != "" || c.Kills != 2 {
		t.Fatalf("zombie at 2: %+v", c)
	}
	// a kind never met is a silhouette: a family, no name, no counts
	if c := cardOf(v, "skeleton"); c.State != "silhouette" || c.Name != "" || c.Family == "" || c.Kills != 0 {
		t.Fatalf("skeleton: %+v", c)
	}
	r.feed(kills("Fallen", 1, 60)...)
	c := cardOf(bestiaryOf(r), "fallen")
	if c.State != "full" || c.Resists == nil || c.Resists.Hell.Fire != 100 || c.Resists.Nightmare.Fire != 25 || len(c.Attack) < 2 {
		t.Fatalf("fallen at 50: %+v", c)
	}
}

func TestBestiaryReplayAndUnlisted(t *testing.T) {
	r, _ := bestiaryRig(t, "Paladin")
	evs := kills("Fallen", 12, 1)
	r.feed(evs...)
	r.feed(evs...) // the same events again
	r.feed(goals.Event{Type: goals.EvKill, Monster: "Corpsefire", Seq: 99})
	v := bestiaryOf(r)
	if c := cardOf(v, "fallen"); c.Kills != 12 {
		t.Fatalf("replay changed the count: %+v", c)
	}
	if v.Met != 1 {
		t.Fatalf("met %d", v.Met)
	}
}

func TestBestiaryReturnSection(t *testing.T) {
	r, _ := bestiaryRig(t, "Sorceress")
	r.feed(kills("Fallen", 10, 1)...)
	r.wall = r.wall.Add(time.Minute)
	var d ReturnData
	r.getJSON("/api/return", &d)
	found := false
	for _, s := range d.Sections {
		if s.ID == BestiarySectionID && len(s.Lines) == 1 && strings.Contains(s.Lines[0], "Fallen") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no bestiary section: %+v", d.Sections)
	}
}

func TestBestiaryErrors(t *testing.T) {
	l, _ := unlock.Parse([]byte(`{"schema":"unlock.ladder/1","entries":[
 {"id":"bestiary","group":"ui","trigger":{"kind":"kills_of_kind","args":{"kind":"*","min":10}},"reveal":{"title":"B","text":"b"}}]}`))
	eng, _ := unlock.Open("", "p", l, nil)
	if _, err := NewBestiary(&Unlocks{Engine: eng}, nil, nil, nil); err == nil || !strings.Contains(err.Error(), `"bestiary_resists"`) {
		t.Fatalf("missing row: %v", err)
	}
	if _, err := NewBestiary(mustUnlocks(t), nil, fakeMonsters{}, nil); err == nil || !strings.Contains(err.Error(), `"Fallen"`) {
		t.Fatalf("kind absent from the table: %v", err)
	}
	if _, err := ParseBestiaryNotes([]byte(`{"schema":"bestiary.notes/1","kinds":[{"name":"X","family":"f","flavour":"a","drops":"d","tip":"t","tipTag":"V"}]}`)); err == nil || !strings.Contains(err.Error(), `"X"`) {
		t.Fatalf("bad tag: %v", err)
	}
}

func TestBestiaryNoTablesAndShippedNotes(t *testing.T) {
	r, _ := shippedRig(t, "Sorceress") // no Monsters: flavour only
	r.feed(kills("Fallen", 50, 1)...)
	v := bestiaryOf(r)
	c := cardOf(v, "fallen")
	if v.Tables || c.State != "full" || c.Resists != nil || c.Type != "" || c.Tip == "" {
		t.Fatalf("%+v %+v", v.Tables, c)
	}
}

func TestBestiaryStoreKeepsCountsAndFilterSkipsUnknownKinds(t *testing.T) {
	notes, skipped, err := FilterBestiaryNotes(nil)
	if err != nil || len(skipped) != 0 || len(notes) == 0 {
		t.Fatalf("no tables: %d notes, skipped %v, %v", len(notes), skipped, err)
	}
	tbl := bestiaryTables(t)
	delete(tbl, "zombie")
	kept, skipped, err := FilterBestiaryNotes(tbl)
	if err != nil || len(skipped) != 1 || skipped[0] != "Zombie" || len(kept) != len(notes)-1 {
		t.Fatalf("filter: %d kept, skipped %v, %v", len(kept), skipped, err)
	}
	path := filepath.Join(t.TempDir(), "sub", "bestiary.json")
	r, _ := bestiaryRig(t, "Sorceress")
	r.fun.Bestiary.SetStore(path)
	r.feed(kills("Fallen", 12, 1)...)
	b2 := &Bestiary{kills: map[string]int{}}
	if err := b2.SetStore(path); err != nil || b2.kills["fallen"] != 12 {
		t.Fatalf("reload: %v %v", err, b2.kills)
	}
	if err := os.WriteFile(path, []byte(`{"schema":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (&Bestiary{kills: map[string]int{}}).SetStore(path); err == nil {
		t.Error("a foreign file was accepted")
	}
}
