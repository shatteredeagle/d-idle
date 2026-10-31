package data

import (
	"os"
	"testing"
	"time"
)

const realInstall = "/tmp/d2mpq"

func realDB(t *testing.T) *DB {
	t.Helper()
	if _, err := os.Stat(realInstall + "/d2data.mpq"); err != nil {
		t.Skip("no real install at " + realInstall)
	}
	cache := t.TempDir()
	t0 := time.Now()
	db, err := Open(realInstall, cache)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("open (extract+load) %v", time.Since(t0))
	if !CacheCurrent(realInstall, cache) {
		t.Fatal("cache should be current after Open")
	}
	t0 = time.Now()
	db2, err := Load(cache)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("load from cache %v", time.Since(t0))
	if len(db2.Monsters) != len(db.Monsters) {
		t.Fatal("cache round trip differs")
	}
	return db2
}

func TestRealData(t *testing.T) {
	db := realDB(t)
	t.Logf("monsters %d superuniques %d levels %d skills %d classes %d types %d items %d objects %d inv %d belts %d tc %d uniq %d set %d rw %d cube %d strs %d",
		len(db.Monsters), len(db.SuperUniques), len(db.Levels), len(db.Skills), len(db.Classes), len(db.ItemTypes), len(db.Items), len(db.Objects),
		len(db.Inventories), len(db.Belts), len(db.Treasure), len(db.Uniques), len(db.SetItems), len(db.Runewords), len(db.Cube), len(db.strs))

	if m := db.MonsterByClassID(156); m == nil || m.Key != "andariel" {
		t.Errorf("class 156 = %+v", m)
	}
	if m := db.MonsterByName("Andariel"); m == nil || m.Level[Hell] == 0 {
		t.Errorf("andariel: %+v", m)
	} else {
		t.Logf("Andariel: %+v", *m)
	}
	if a := db.AreaByID(2); a == nil || a.Name != "Blood Moor" || a.Act != 0 {
		t.Errorf("area 2 = %+v", a)
	}
	if a := db.AreaByName("Catacombs Level 4"); a == nil {
		t.Error("no catacombs 4")
	} else {
		t.Logf("%+v", *a)
	}
	if s := db.SkillByID(36); s == nil || s.Name != "Fire Bolt" || s.ManaCost(1) != 2.5 {
		t.Errorf("skill 36 = %+v", s)
	} else {
		t.Logf("fire bolt: syn %v mana(20)=%v page %d row %d col %d", s.Synergies(), s.ManaCost(20), s.Page, s.Row, s.Column)
	}
	if got := db.ExpToLevel(2); got != 500 {
		t.Errorf("exp lvl2 = %d", got)
	}
	if got := db.ExpToLevel(99); got != 3520485254 {
		t.Errorf("exp lvl99 = %d", got)
	}
	if db.Exp.LevelFor(499) != 1 || db.Exp.LevelFor(500) != 2 {
		t.Error("LevelFor")
	}
	if c := db.ClassByName("Sorceress"); c == nil || c.ID != 1 || c.Code != "sor" {
		t.Errorf("sorc %+v", c)
	} else {
		t.Logf("%+v", *c)
	}
	if b := db.ItemByCode("hax"); b == nil || b.Display != "Hand Axe" || b.Width != 1 || b.Height != 3 {
		t.Errorf("hax %+v", b)
	}
	if !db.IsA("swor", "weap") || db.IsA("swor", "armo") {
		t.Error("IsA")
	}
	if r := db.ItemByCode("r01"); r == nil || r.Display == "" {
		t.Errorf("r01 %+v", r)
	} else {
		t.Logf("%+v", *r)
	}
	if v := db.InventoryLayout("Sorceress"); v == nil || v.Cols != 10 || v.Rows != 4 {
		t.Errorf("inv %+v", v)
	}
	if v := db.InventoryLayout("Bank Page 1"); v == nil {
		t.Error("no bank")
	} else {
		t.Logf("bank %+v", *v)
	}
	if v := db.InventoryLayout("Transmogrify Box Page 1"); v == nil {
		t.Error("no cube")
	} else {
		t.Logf("cube %d x %d", v.Cols, v.Rows)
	}
	if b := db.BeltLayout("belt"); b == nil || b.Slots != 16 && b.Slots != 12 {
		t.Errorf("belt %+v", b)
	}
	if tc := db.TreasureClassByName("Andariel"); tc == nil {
		t.Error("no tc andariel")
	} else {
		t.Logf("%+v", *tc)
	}
	if u := db.UniqueByName("Harlequin Crest"); u == nil || u.Code != "uap" {
		t.Errorf("shako %+v", u)
	}
	if w := db.RunewordByName("Spirit"); w == nil || len(w.Runes) != 4 {
		t.Errorf("spirit %+v", w)
	} else {
		t.Logf("%+v", *w)
	}
	if s := db.SuperUniqueByName("Bishibosh"); s == nil {
		t.Error("bishi")
	}
	if o := db.ObjectByID(119); o == nil || o.Name != "Waypoint" {
		t.Errorf("obj119 %+v", o)
	}
	if len(db.Cube) < 50 {
		t.Errorf("cube recipes %d", len(db.Cube))
	}
	// Immunities sanity: Hell Andariel poison-immune? just log.
	m := db.MonsterByName("Mephisto")
	t.Logf("Mephisto res %v lvl %v boss %v", m.Res, m.Level, m.Boss)
}

func TestRealMore(t *testing.T) {
	db := realDB(t)
	m := db.MonsterByName("Andariel")
	t.Logf("andariel hp N/NM/H at lvl: %d %d %d", db.ScaledHP(m, Normal, m.Level[0]), db.ScaledHP(m, Nightmare, m.Level[1]), db.ScaledHP(m, Hell, m.Level[2]))
	t.Logf("difficulties %+v", db.Difficulties)
	t.Logf("monlvl 85 %+v", db.MonLevelRow(85))
	t.Logf("sorc inv %+v", db.InventoryLayout("Sorceress"))
	t.Logf("sorc inv2 %+v", db.InventoryLayout("Sorceress2"))
	t.Logf("belt %+v", db.BeltLayout("belt"))
	t.Logf("tp tome %+v", db.ItemByCode("tbk"))
	t.Logf("hp pots %+v", db.ItemByCode("hp5"))
	t.Logf("area names sample: %q %q", db.AreaByID(40).Name, db.AreaByID(109).Name)
	t.Logf("tsc %q", db.Text("Scroll of Town Portal"))
	for _, k := range []string{"Gnasher", "The Gnasher", "Harlequin Crest"} {
		s, ok := db.Str(k)
		t.Logf("str %q -> %q %v", k, s, ok)
	}
}

func TestRealExtra(t *testing.T) {
	db := realDB(t)
	t.Logf("hire %d stat %d prop %d pre %d suf %d state %d missile %d warp %d", len(db.Hirelings), len(db.StatCosts), len(db.Properties),
		len(db.Prefixes), len(db.Suffixes), len(db.States), len(db.Missiles), len(db.Warps))
	for _, d := range []Difficulty{Normal, Nightmare, Hell} {
		for act := 1; act <= 5; act++ {
			hs := db.HirelingsFor(act, d)
			if len(hs) == 0 {
				continue
			}
			h := hs[0]
			t.Logf("act%d d%d: %d rows first %s/%s id %d lvl %d gold %d aura %q seller %d", act, d, len(hs), h.Name, h.SubType, h.ID, h.Level, h.Gold, h.Aura(db), h.Seller)
		}
	}
	for _, n := range []string{"strength", "maxhp", "maxmana", "gold", "goldbank", "experience", "level", "hitpoints"} {
		if s := db.StatByName(n); s != nil {
			t.Logf("stat %s: %+v", n, *s)
		}
	}
	if !db.RunewordAvailableOffline("Enigma") || db.RunewordAvailableOffline("Spirit") || db.RunewordAvailableOffline("Nope") {
		t.Errorf("runeword offline: Enigma %v Spirit %v", db.RunewordAvailableOffline("Enigma"), db.RunewordAvailableOffline("Spirit"))
	}
	if m := db.MissileByName("firebolt"); m == nil {
		t.Error("no firebolt missile")
	} else {
		t.Logf("firebolt %+v", *m)
	}
	t.Logf("freeze %+v", db.StateByName("freeze"))
	t.Logf("res-all %+v", db.PropertyByCode("res-all"))
	t.Logf("Sturdy %+v warp0 %+v", db.AffixByName("Sturdy", false), db.WarpByID(0))
}
