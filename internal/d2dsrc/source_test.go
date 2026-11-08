package d2dsrc

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2map"
)

func quiet(string, ...any) {}

func lateReplies(t *testing.T) map[string]string {
	r := sampleReplies()
	r[headSections] = readFile(t, "dump_head_late.json") + "\nok\n"
	r["dump items"] = readFile(t, "dump_items_late.json") + "\nok\n"
	r["dump quests"] = readFile(t, "dump_quests.json") + "\nok\n"
	return r
}

func TestSnapshotLate(t *testing.T) {
	_, c := startScript(t, lateReplies(t))
	src := New(c, Options{Tables: fakeTables{}, Logf: quiet})
	res, err := src.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s, rep := res.Snap, res.Report
	if s.State != d2.StateInGame || s.Player == nil || s.Inventory == nil {
		t.Fatalf("snapshot %+v", s)
	}
	p := s.Player
	type chk struct {
		name      string
		got, want any
	}
	for _, c := range []chk{
		{"name", p.Name, "Soaky"}, {"class", p.Class, d2.Sorceress}, {"id", p.ID, uint32(1)},
		{"x subtile", p.X, 201}, {"y subtile", p.Y, 110}, // floor(5*40.30)=201, floor(5*22.10)=110
		{"area", p.Area, 3}, {"act", p.Act, 0}, {"mode WL", p.Mode, 2},
		{"level", p.Level, 24}, {"exp", p.Exp, uint32(1500000)}, {"gold", p.Gold, 12000}, {"stash", p.StashGold, 300},
		{"life", p.Life, 180}, {"maxlife", p.MaxLife, 200}, {"stamina", p.Stamina, 120},
		{"str=base+bonus", p.Strength, 23}, {"ene", p.Energy, 70}, {"vit", p.Vitality, 35},
		{"statpts", p.StatPoints, 5}, {"lskill", p.LeftSkill, 36}, {"rskill", p.RightSkill, 54},
		{"difficulty", p.Difficulty, d2.Nightmare}, {"fire", p.Resist.Fire, 20}, {"cold", p.Resist.Cold, -10},
		{"stat fire res", p.Stats.Get(d2.StatFireRes), int32(20)}, {"life fixed", p.Stats.Fixed(d2.StatLife), 180},
	} {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if p.Stats.Has(d2.StatMagicFind) || p.Stats.Has(d2.StatMagicRes) {
		t.Error("Stats holds a key d2d never gave")
	}
	// skills: base level 1 at index 0 -> Sorceress base id 36 (also the left skill), 54 is the right skill
	if len(p.Skills) != 2 || p.Skills[0] != (d2.Skill{ID: 36, Level: 1}) {
		t.Errorf("skills %+v", p.Skills)
	}
	if !p.InTown() == false && p.Area == 3 {
		t.Error("cold plains is not a town")
	}
	// UI
	if !s.UI.Open(d2.UINPCMenu) || s.UI.Open(d2.UIStash) || !s.UI.Open(d2.UIMiniPanel) {
		t.Errorf("ui %v", s.UI)
	}
	// inventory containers
	inv := s.Inventory
	if inv.ItemCount != 5 { // the socketed rune is not a unit of the list
		t.Errorf("item count %d", inv.ItemCount)
	}
	if w := inv.Equipped(d2.BodyRHand); w == nil || w.Class != 100 || w.Item.Quality != d2.QualityUnique || w.Item.FileIndex != 4 || !w.Item.Identified() {
		t.Errorf("wand %+v", w)
	}
	if b := inv.Container(d2.ContBelt); len(b) != 1 || b[0].Class != 587 || b[0].Mode != d2.ItemModeBelt {
		t.Errorf("belt %+v", b)
	}
	if tsc := inv.Container(d2.ContInventory); len(tsc) != 1 || tsc[0].Item.GX != 9 || tsc[0].Item.GY != 2 {
		t.Errorf("inventory %+v", tsc)
	}
	if st := inv.Container(d2.ContStash); len(st) != 1 || !st[0].Item.Ethereal() || !st[0].Item.Socketed() || st[0].Item.Identified() {
		t.Errorf("stash %+v", st)
	}
	if inv.Cursor == nil || inv.Cursor.Class != 300 {
		t.Errorf("cursor %+v", inv.Cursor)
	}
	if ex := res.Extras.Items[11]; ex.Qty != 20 || ex.Code != "tsc" {
		t.Errorf("extras %+v", ex)
	}
	if ex := res.Extras.Items[12]; ex.Sockets != 2 || ex.Dur != 12 {
		t.Errorf("extras %+v", ex)
	}
	if !res.Extras.Clock.Valid || res.Extras.Clock.Tick != 90000 || !res.Extras.Clock.Lockstep {
		t.Errorf("clock %+v", res.Extras.Clock)
	}

	// monsters: 2 hostile rows + Kashya (npc via tables) + merc
	var fallen, shaman, kashya, merc *d2.Unit
	for i := range s.Monsters {
		u := &s.Monsters[i]
		switch {
		case u.ID == 2:
			fallen = u
		case u.ID == 5:
			shaman = u
		case u.ID == NPCIDBase:
			kashya = u
		case u.ID == MercID:
			merc = u
		}
	}
	if fallen == nil || fallen.Class != 2 || fallen.X != 60 || fallen.Y != 143 || fallen.Mode != d2.MonModeStand || fallen.Monster.LifePct != 100 {
		t.Errorf("fallen %+v %+v", fallen, fallen.Monster)
	}
	if shaman == nil || !shaman.Monster.IsChampion() || shaman.Mode != 4 || shaman.Monster.Enchants[0] != 7 || shaman.Monster.Enchants[1] != 26 || shaman.Monster.LifePct != 77 {
		t.Errorf("shaman %+v", shaman)
	}
	if kashya == nil || kashya.Class != 150 || kashya.Mode != d2.MonModeStand {
		t.Errorf("kashya %+v", kashya)
	}
	if merc == nil || merc.Class != 271 || merc.Monster.MaxLife != 80 || merc.Monster.LifePct != 78 || !d2.IsMercClass(merc.Class) {
		t.Errorf("merc %+v", merc)
	}
	// Cain has no monstats row in fakeTables -> Object with class -1
	if len(s.Objects) < 1 || s.Objects[0].Class != -1 || res.Extras.Names[s.Objects[0].ID] != "Deckard Cain" {
		t.Errorf("objects %+v", s.Objects)
	}
	// portal on level 3? Sample portals are on levels 2 and 1: none here
	if len(s.Objects) != 1 {
		t.Errorf("unexpected objects %+v", s.Objects)
	}
	// ground gold
	if len(s.Ground) != 1 || s.Ground[0].X != 204 || s.Ground[0].Y != 57 || s.Ground[0].Item.Where != d2.ContGround || s.Ground[0].Class != 523 {
		t.Errorf("ground %+v", s.Ground)
	}
	if len(res.Extras.Gold) != 1 {
		t.Errorf("gold %v", res.Extras.Gold)
	}
	if len(res.Extras.Menu) != 3 {
		t.Errorf("menu %+v (npc menu flag is set in this fixture)", res.Extras.Menu)
	}

	// report: unknown fields are explicit
	for _, f := range []string{"player.states", "player.resist.magic", "player.skills.bonus", "monster.states", "monster.resists",
		"missiles", "mouse", "hovered", "viewport", "ui.quest", "ground.quality", "merc.name"} {
		if _, ok := rep.Unknown[f]; !ok {
			t.Errorf("%s should be reported unknown", f)
		}
	}
	for _, f := range []string{"player.level", "player.pos", "inventory", "monsters", "state"} {
		if !rep.Has(f) {
			t.Errorf("%s should be present", f)
		}
	}
	if _, ok := rep.Derived["player.str"]; !ok {
		t.Error("str is derived (base+bonus) and must say so")
	}
	// counters and the one-time log
	cnt := src.UnknownCounts()
	if cnt["player.states"] != 1 {
		t.Errorf("counter %d", cnt["player.states"])
	}
	if _, err := src.Read(context.Background()); err != nil {
		t.Fatal(err)
	}
	if src.UnknownCounts()["player.states"] != 2 {
		t.Error("counter should grow per snapshot")
	}
}

func TestLogOnce(t *testing.T) {
	_, c := startScript(t, lateReplies(t))
	var logs []string
	src := New(c, Options{Logf: func(f string, a ...any) { logs = append(logs, f) }})
	for i := 0; i < 3; i++ {
		if _, err := src.Read(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	n := 0
	for _, l := range logs {
		if strings.Contains(l, "unknown fields") {
			n++
		}
	}
	// with no Tables item.class and monster.class are reported too: each field logs exactly once
	seen := map[string]bool{}
	for f := range src.UnknownCounts() {
		seen[f] = true
	}
	if n != 1 || len(seen) < 2 {
		t.Errorf("%d summary lines for %d distinct unknown fields, want 1 line", n, len(seen))
	}
}

func TestNoTablesIsUnknownNotZero(t *testing.T) {
	_, c := startScript(t, lateReplies(t))
	res, err := New(c, Options{Logf: quiet}).Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range res.Snap.Ground {
		if u.Class != -1 {
			t.Errorf("ground class %d without tables", u.Class)
		}
	}
	for _, u := range res.Snap.Inventory.Items {
		if u.Class != -1 {
			t.Errorf("item class %d without tables", u.Class)
		}
	}
	if _, ok := res.Report.Unknown["item.class"]; !ok {
		t.Error("item.class not reported")
	}
	// without tables the menu flag decides: Kashya (menu=1) is a Monsters unit, Cain (menu=0) an object
	if res.Snap.Objects[0].Class != -1 {
		t.Error("object class")
	}
}

func TestFreshNecroInTown(t *testing.T) {
	r := sampleReplies()
	r[headSections] = readFile(t, "dump_head_necro.json") + "\nok\n"
	r["dump items"] = `{"v":1,"items":[]}` + "\nok\n"
	_, c := startScript(t, r)
	res, err := New(c, Options{Tables: fakeTables{}, Logf: quiet}).Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p := res.Snap.Player
	if !p.InTown() || p.Class != d2.Necromancer || p.X != 153 || p.Y != 68 || p.Stamina != 60 || p.SkillPoints != 1 {
		t.Errorf("player %+v", p)
	}
	if len(p.Skills) != 0 || res.Snap.Inventory.ItemCount != 0 {
		t.Errorf("skills %+v items %d", p.Skills, res.Snap.Inventory.ItemCount)
	}
	// both sample portals: level 1 portal is in town
	if len(res.Snap.Objects) != 2 { // Cain (object) + the town-side portal
		t.Fatalf("objects %+v", res.Snap.Objects)
	}
}

func TestStockD2DDegrades(t *testing.T) {
	r := sampleReplies() // no dump verb at all
	_, c := startScript(t, r)
	res, err := New(c, Options{Tables: fakeTables{}, Logf: quiet}).Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p := res.Snap.Player
	if p == nil || p.Name != "Spike" || p.Class != d2.Necromancer || p.Area != 1 || p.X != 153 || p.Life != 45 {
		t.Fatalf("player %+v", p)
	}
	if res.Report.Source["player"] != "verbs" {
		t.Errorf("source %v", res.Report.Source)
	}
	for _, f := range []string{"player.states", "player.str", "player.skills", "ui", "merc", "item.ilvl", "item.flags"} {
		if _, ok := res.Report.Unknown[f]; !ok {
			t.Errorf("%s should be unknown on stock d2d", f)
		}
	}
	w := res.Snap.Inventory.Equipped(d2.BodyRHand)
	if w == nil || w.Class != 100 {
		t.Errorf("wand from the items verb: %+v", w)
	}
	if _, err := New(c, Options{Logf: quiet}).Progress(context.Background()); err != ErrNoProgress {
		t.Errorf("progress err %v", err)
	}
}

func TestStates(t *testing.T) {
	for screen, want := range map[string]d2.State{"charselect": d2.StateMenu, "title": d2.StateMenu, "loading": d2.StateLoading} {
		r := sampleReplies()
		r["state"] = "screen=" + screen + " class=- name=-\nok\n"
		_, c := startScript(t, r)
		snap, err := New(c, Options{Logf: quiet}).Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if snap.State != want || snap.Player != nil {
			t.Errorf("%s: %v %+v", screen, snap.State, snap.Player)
		}
	}
	r := lateReplies(t)
	r[headSections] = "err no scene\n"
	_, c := startScript(t, r)
	snap, err := New(c, Options{Logf: quiet}).Snapshot()
	if err != nil || snap.State != d2.StateLoading || snap.Player != nil {
		t.Errorf("no scene: %v %+v %v", snap, snap.Player, err)
	}
}

func TestProgress(t *testing.T) {
	_, c := startScript(t, lateReplies(t))
	pr, err := New(c, Options{Logf: quiet}).Progress(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if pr.Quest[0][0] != 1 || pr.Quest[0][47] != 8193 || pr.Waypoints[0] != 3 {
		t.Errorf("progress %+v", pr)
	}
}

func TestOriginOption(t *testing.T) {
	r := lateReplies(t)
	r["debug level"] = "ok 3 40.30 22.10 80 80 100 200\n"
	_, c := startScript(t, r)
	res, err := New(c, Options{Logf: quiet, AddOrigin: true}).Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Snap.Player.X != 201+500 || res.Snap.Player.Y != 110+1000 || res.Extras.Level == nil {
		t.Errorf("origin: %d,%d", res.Snap.Player.X, res.Snap.Player.Y)
	}
}

func TestModeTables(t *testing.T) {
	if monModes["DT"] != d2.MonModeDeath || monModes["NU"] != d2.MonModeStand || monModes["DD"] != d2.MonModeDead {
		t.Error("monster mode table disagrees with d2 constants")
	}
	if len(monModes) != 16 || len(plrModes) != 20 {
		t.Errorf("table sizes %d %d", len(monModes), len(plrModes))
	}
	r := sampleReplies()
	r["monsters"] = "fallen1\t12.10\t28.70\t752\t436\t2/2\tZZ\t-\t-\tlvl1\tFallen\t#2\t-\nok\n"
	r["dump player,stats,skills,ui,merc,time"] = readFile(t, "dump_head_necro.json") + "\nok\n"
	r["dump items"] = `{"v":1,"items":[]}` + "\nok\n"
	_, c := startScript(t, r)
	res, err := New(c, Options{Logf: quiet}).Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range res.Snap.Monsters {
		if u.ID == 2 && u.Mode != -1 {
			t.Errorf("unknown mode code must be -1, got %d", u.Mode)
		}
	}
	if _, ok := res.Report.Unknown["monster.mode.ZZ"]; !ok {
		t.Error("unknown mode code not reported")
	}
}

func TestActOf(t *testing.T) {
	for area, want := range map[int]int{1: 0, 39: 0, 40: 1, 74: 1, 75: 2, 102: 2, 103: 3, 108: 3, 109: 4, 132: 4} {
		if got := ActOf(area); got != want {
			t.Errorf("area %d act %d want %d", area, got, want)
		}
	}
}

// TestPrefetchSameSnapshotOneWrite: the pipelined read builds the same snapshot as one round trip per verb, and
// every verb of the batch reaches d2d exactly once (a consumed prefetch is never resent).
func TestPrefetchSameSnapshotOneWrite(t *testing.T) {
	read := func(noPre bool) (*d2.Snapshot, []string) {
		s, c := startScript(t, lateReplies(t))
		src := New(c, Options{Tables: fakeTables{}, Logf: quiet, NoPrefetch: noPre})
		snap, err := src.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		return snap, append([]string(nil), s.seen...)
	}
	a, seenA := read(false)
	b, seenB := read(true)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("prefetch changed the snapshot:\n%+v\n%+v", a, b)
	}
	count := func(seen []string) map[string]int {
		m := map[string]int{}
		for _, l := range seen {
			m[l]++
		}
		return m
	}
	if !reflect.DeepEqual(count(seenA), count(seenB)) {
		t.Fatalf("verbs sent differ: %v vs %v", count(seenA), count(seenB))
	}
	for _, l := range readBatch {
		if count(seenA)[l] > 1 {
			t.Errorf("%q sent %d times", l, count(seenA)[l])
		}
	}
}

func TestInventoryFlag(t *testing.T) {
	head := func(t *testing.T, ui string) (*d2dctl.Client, *Source) {
		r := lateReplies(t)
		r[headSections] = strings.Replace(r[headSections], `"held":"jav"}`, `"held":"jav"`+ui+`}`, 1)
		r["key i"], r["key Escape"] = "ok\n", "ok\n"
		_, c := startScript(t, r)
		return c, New(c, Options{Tables: fakeTables{}, Logf: quiet})
	}
	open := func(src *Source) bool {
		s, err := src.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		return s.UI.Open(d2.UIInventory)
	}
	t.Run("rects with inv_grid", func(t *testing.T) {
		_, src := head(t, `,"rects":{"inv_grid":[417,225,260,235]}`)
		if !open(src) {
			t.Fatal("inv_grid rect must mean the inventory is open")
		}
	})
	t.Run("rects without inv_grid beat the hint", func(t *testing.T) {
		c, src := head(t, `,"rects":{}`)
		_ = c.Key(context.Background(), "i")
		if open(src) {
			t.Fatal("an empty rects object means closed, whatever the key hint says")
		}
	})
	t.Run("lab2 without rects: key hint", func(t *testing.T) {
		c, src := head(t, ``)
		if open(src) {
			t.Fatal("closed before any key")
		}
		_ = c.Key(context.Background(), "i")
		if !open(src) {
			t.Fatal("open after key i")
		}
		_ = c.Key(context.Background(), "Escape")
		if open(src) {
			t.Fatal("closed after Escape")
		}
	})
}

// W1 (fixwp): the "Waypoint" npcs row of every act becomes an object nav's nearestWaypointUnit accepts, with and
// without Tables; other named objects keep class -1.
func TestWaypointRowHasWaypointClass(t *testing.T) {
	for _, tb := range []Tables{nil, fakeTables{}} {
		r := lateReplies(t)
		r["npcs"] = "Waypoint\t300\t200\t0\t4\t30.0\t10.0\nWaypoint\t300\t200\t0\t5\t31.0\t10.0\nDeckard Cain\t300\t200\t0\t3\t28.0\t10.0\nok\n"
		_, c := startScript(t, r)
		res, err := New(c, Options{Tables: tb, Logf: quiet}).Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		wps, other := 0, 0
		for _, u := range res.Snap.Objects {
			if d2map.IsWaypointObject(u.Class) {
				wps++
				if n := res.Extras.Names[u.ID]; !strings.Contains(n, "Waypoint") {
					t.Errorf("waypoint unit named %q", n)
				}
			} else if u.Class == -1 {
				other++
			}
		}
		if wps != 2 || other != 1 {
			t.Errorf("tables=%v: %d waypoint objects, %d class -1 (want 2 and 1): %+v", tb != nil, wps, other, res.Snap.Objects)
		}
	}
}

// fixobj (drill H.obj): placed objects near the player are Objects with the objects.txt id of the preset row and the id
// NPCIDBase + preset index (the index space of "cmd interact"); hidden rows, rows "npcs" already lists and far rows
// are not listed; the preset list is read once per level while the player stands still.
func TestFieldObjectsListed(t *testing.T) {
	r := lateReplies(t)
	r["npcs"] = "Waypoint\t300\t200\t0\t4\t30.0\t10.0\nok\n"
	r["debug presets"] = "4\t119\t30.0\t10.0\t0\tWaypoint\n" + // listed by npcs: skipped
		"7\t580\t33.0\t12.0\t0\tChest\n" +
		"8\t17\t34.0\t12.0\t1\tHidden Chest\n" +
		"9\t26\t35.0\t12.0\t0\tCain's Gibbet\n" +
		"10\t580\t300.0\t300.0\t0\tFar Chest\nok\n"
	sc, c := startScript(t, r)
	src := New(c, Options{Logf: quiet})
	for i := 0; i < 2; i++ {
		res, err := src.Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		got := map[uint32]int{}
		for _, u := range res.Snap.Objects {
			got[u.ID] = u.Class
		}
		if got[NPCIDBase+7] != 580 || got[NPCIDBase+9] != 26 {
			t.Fatalf("chest and cage not listed with their classes: %+v", res.Snap.Objects)
		}
		if _, ok := got[NPCIDBase+8]; ok {
			t.Error("hidden preset listed")
		}
		if _, ok := got[NPCIDBase+10]; ok {
			t.Error("preset beyond the radius listed")
		}
		if got[NPCIDBase+4] != WaypointClass {
			t.Errorf("the npcs waypoint row was replaced: %+v", res.Snap.Objects)
		}
		if res.Extras.Names[NPCIDBase+7] != "Chest" {
			t.Errorf("chest name %q", res.Extras.Names[NPCIDBase+7])
		}
	}
	n := 0
	for _, l := range sc.seen {
		if l == "debug presets" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("debug presets read %d times for two snapshots on one spot (want 1)", n)
	}
}

func TestBuffStates(t *testing.T) {
	r := lateReplies(t)
	r["debug states"] = "absorb=0 maxlife=100 maxmana=50 def=7 buffs=40:1,68:2,999:1,\nok\n"
	_, c := startScript(t, r)
	src := New(c, Options{Tables: fakeTables{}, Logf: quiet, BuffStates: true})
	res, err := src.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	st := res.Snap.Player.States
	if !st.Has(d2.StateFrozenArmor) || !st.Has(14) || st.Has(d2.StateShiverArmor) || st.Has(d2.StatePoison) {
		t.Errorf("states %v", st)
	}
	// without the option the bit array stays empty and the verb is not asked
	_, c = startScript(t, lateReplies(t))
	res, err = New(c, Options{Tables: fakeTables{}, Logf: quiet}).Read(context.Background())
	if err != nil || res.Snap.Player.States != (d2.StateSet{}) {
		t.Errorf("option off: %v %v", res.Snap.Player.States, err)
	}
}

func TestBuffStatesRefused(t *testing.T) {
	r := lateReplies(t)
	r["debug states"] = "err unknown debug verb\n"
	_, c := startScript(t, r)
	src := New(c, Options{Tables: fakeTables{}, Logf: quiet, BuffStates: true})
	for i := 0; i < 2; i++ {
		res, err := src.Read(context.Background())
		if err != nil || res.Snap.Player.States != (d2.StateSet{}) {
			t.Fatalf("read %d: %v %v", i, res.Snap.Player.States, err)
		}
	}
	if !src.noBuff.Load() {
		t.Error("a refused verb should switch the read off")
	}
}

// The fraction of a fixed-point stat survives when stats.raw carries it (TASK-207): Paladin max mana 19.5 is
// 4992 in 256ths while the player pair holds the whole 19.
func TestPlayerStatsKeepFraction(t *testing.T) {
	cases := []struct {
		name        string
		mana, mxMan int // player pair, whole points
		raw         []int
		wantMax     int32
		wantMana    int32
	}{
		{"paladin 19.5", 19, 19, []int{0, 15, 0, 0, 0, 0, 0, 0, 4992, 4992, 0, 0}, 4992, 4992},
		{"sorceress 100.25", 100, 100, []int{0, 0, 0, 0, 0, 0, 0, 0, 25600, 25664, 0, 0}, 25664, 25600},
		{"no raw shifts whole points", 25, 25, nil, 25 << 8, 25 << 8},
		{"raw disagreeing with the pair is ignored", 25, 25, []int{0, 0, 0, 0, 0, 0, 0, 0, 9999, 9999, 0, 0}, 25 << 8, 25 << 8},
	}
	for _, c := range cases {
		b := &builder{rp: newRep(&tracker{counts: map[string]uint64{}, logf: quiet})}
		pl := &d2.Player{Mana: c.mana, MaxMana: c.mxMan}
		b.rp.have("player.mana")
		st := obj{"base": map[string]any{}}
		if c.raw != nil {
			a := make([]any, len(c.raw))
			for i, v := range c.raw {
				a[i] = float64(v)
			}
			st["raw"] = a
		}
		b.playerStats(pl, st, obj{})
		if got := pl.Stats[uint32(d2.StatMaxMana)]; got != c.wantMax {
			t.Errorf("%s: max mana %d, want %d", c.name, got, c.wantMax)
		}
		if got := pl.Stats[uint32(d2.StatMana)]; got != c.wantMana {
			t.Errorf("%s: mana %d, want %d", c.name, got, c.wantMana)
		}
		if pl.MaxMana != c.mxMan {
			t.Errorf("%s: whole-point MaxMana changed to %d", c.name, pl.MaxMana)
		}
	}
}
