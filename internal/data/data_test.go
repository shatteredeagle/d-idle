package data

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/data/mpq"
	"d2lod-bot/internal/paths"
)

func TestTable(t *testing.T) {
	tb := ParseTable([]byte("\xef\xbb\xbfName\tLevel\tLevel(N)\tFoo\r\nA\t5\t 7.9 \t\r\n\r\nB\t\t\n"))
	if tb.Len() != 2 {
		t.Fatalf("rows %d", tb.Len())
	}
	a, b := tb.Row(0), tb.Row(1)
	if a.Str("name") != "A" || a.Int("LEVEL") != 5 || a.Int("level(n)") != 7 || a.Float("Level(N)") != 7.9 {
		t.Errorf("row a: %q %d %d", a.Str("name"), a.Int("level"), a.Int("level(n)"))
	}
	if b.Int("Level") != 0 || b.Str("Foo") != "" || b.Str("Missing") != "" || b.Int("Missing") != 0 || b.Bool("Missing") {
		t.Error("blank/missing cells should read as zero values")
	}
	if a.Int("Missing", "Level") != 5 {
		t.Error("alias fallback")
	}
	if !tb.Has("foo") || tb.Has("nope") {
		t.Error("Has")
	}
	if got := a.Int("Name"); got != 0 {
		t.Errorf("non-numeric int = %d", got)
	}
}

// makeTBL builds a .tbl image holding the given pairs.
func makeTBL(pairs [][2]string) []byte {
	n := len(pairs)
	hs := n
	hdr := make([]byte, 21)
	binary.LittleEndian.PutUint16(hdr[2:], uint16(n))
	binary.LittleEndian.PutUint32(hdr[4:], uint32(hs))
	hdr[8] = 1
	idx := make([]byte, n*2)
	nodes := make([]byte, hs*17)
	var data []byte
	base := 21 + n*2 + hs*17
	for i, p := range pairs {
		binary.LittleEndian.PutUint16(idx[i*2:], uint16(i))
		ko := base + len(data)
		data = append(append(data, p[0]...), 0)
		vo := base + len(data)
		data = append(append(data, p[1]...), 0)
		e := nodes[i*17:]
		e[0] = 1
		binary.LittleEndian.PutUint16(e[1:], uint16(i))
		binary.LittleEndian.PutUint32(e[7:], uint32(ko))
		binary.LittleEndian.PutUint32(e[11:], uint32(vo))
		binary.LittleEndian.PutUint16(e[15:], uint16(len(p[1])+1))
	}
	out := append(append(append(hdr, idx...), nodes...), data...)
	binary.LittleEndian.PutUint32(out[9:], uint32(base))
	binary.LittleEndian.PutUint32(out[17:], uint32(len(out)))
	return out
}

func TestTBL(t *testing.T) {
	m, err := ParseTBL(makeTBL([][2]string{{"Andariel", "Andariel"}, {"x", "ÿc1Redÿc0 text"}}))
	if err != nil || m["Andariel"] != "Andariel" {
		t.Fatalf("%v %v", m, err)
	}
	if CleanString(m["x"]) != "Red text" {
		t.Errorf("clean: %q", CleanString(m["x"]))
	}
	if CleanString("\xffc2Blue") != "Blue" {
		t.Error("raw 0xff colour code")
	}
	if _, err := ParseTBL([]byte{1, 2, 3}); err == nil {
		t.Error("short tbl")
	}
}

// Hand-written excerpts (our own content, shaped like the real tables).
var fixtures = map[string]string{
	"monstats": "Id\thcIdx\tNameStr\tLevel\tLevel(N)\tLevel(H)\tResFi\tResPo(H)\tboss\tminion1\tExtra\n" +
		"skeleton1\t0\tskeleton1\t2\t37\t68\t\t\t\t\tx\n" +
		"andariel\t1\tandariel\t12\t49\t75\t-50\t100\t1\tskeleton1\ty\n" +
		"Expansion\t\t\t\t\t\t\t\t\t\t\n",
	"levels": "Name\tId\tAct\tMonLvl1\tMonLvl1Ex\tLevelName\tmon1\tVis0\n" +
		"Null\t0\t0\t0\t0\t\t\t0\n" +
		"Act 1 - Town\t1\t0\t1\t1\tRogue Encampment\t\t2\n" +
		"Act 1 - Wilderness 1\t2\t0\t1\t5\tBlood Moor\tzombie1\t1\n" +
		"Expansion\t\t\t\t\t\t\t\n",
	"skills": "skill\tId\tcharclass\tskilldesc\tmana\tlvlmana\tmanashift\tminmana\tEDmgSymPerCalc\n" +
		"Fire Bolt\t36\tsor\tfire bolt\t5\t0\t7\t1\t(skill('Fire Ball'.blvl)+skill('Meteor'.blvl))*par8\n" +
		"Teleport\t54\tsor\tteleport\t24\t-1\t8\t1\t\n",
	"skilldesc":  "skilldesc\tSkillPage\tSkillRow\tSkillColumn\tstr name\nfire bolt\t1\t1\t2\tskillname36\nteleport\t3\t3\t2\tskillname54\n",
	"charstats":  "class\tstr\tdex\tint\tvit\tLifePerLevel\titem1\titem1loc\titem1count\titem2\titem2loc\titem2count\nSorceress\t10\t25\t35\t10\t4\tsst\trarm\t1\t0\t\t0\nExpansion\t\t\t\t\t\t\t\t\t\t\t\n",
	"experience": "Level\tAmazon\tSorceress\tExpRatio\nMaxLvl\t4\t4\t10\n0\t0\t0\t1024\n1\t500\t500\t1024\n2\t1500\t1500\t1024\n3\t3750\t3750\t1024\n",
	"itemtypes":  "ItemType\tCode\tEquiv1\tEquiv2\tBody\nAny\tweap\t\t\t0\nSword\tswor\tmele\t\t1\nMelee\tmele\tweap\t\t0\n",
	"weapons":    "name\ttype\tcode\tnamestr\tinvwidth\tinvheight\tmindam\tmaxdam\nShort Sword\tswor\tssd\tssd\t1\t3\t2\t7\n",
	"armor":      "name\ttype\tcode\tnamestr\tinvwidth\tinvheight\tminac\tmaxac\tbelt\nCap\thelm\tcap\tcap\t2\t2\t3\t5\t0\nSash\tbelt\tlbl\tlbl\t2\t1\t1\t1\t1\n",
	"misc":       "name\ttype\tcode\tnamestr\tinvwidth\tinvheight\tstackable\tmaxstack\tautobelt\nMinor Healing Potion\thpot\thp1\thp1\t1\t1\t0\t0\t1\nArrows\tbowq\taqv\taqv\t1\t3\t1\t350\t0\n",
	"belts":      "name\tnumboxes\tboxwidth\tboxheight\tbox1left\tbox1right\tbox1top\tbox1bottom\tbox2left\tbox2right\tbox2top\tbox2bottom\nbelt\t2\t29\t29\t343\t372\t442\t471\t374\t403\t442\t471\nsash\t1\t29\t29\t343\t372\t442\t471\t\t\t\t\n",
	"inventory": "class\tinvLeft\tinvRight\tinvTop\tinvBottom\tgridX\tgridY\tgridLeft\tgridRight\tgridTop\tgridBottom\tgridBoxWidth\tgridBoxHeight\theadLeft\theadRight\theadTop\theadBottom\theadWidth\theadHeight\n" +
		"Sorceress\t320\t640\t0\t441\t10\t4\t339\t626\t255\t368\t29\t29\t455\t509\t8\t59\t54\t51\n" +
		"Bank Page 1\t0\t321\t220\t442\t6\t8\t74\t244\t273\t386\t29\t29\t-1\t-1\t-1\t-1\t-1\t-1\n",
	"treasureclassex":  "Treasure Class\tgroup\tlevel\tPicks\tNoDrop\tItem1\tProb1\tItem2\tProb2\nAndariel\t\t\t7\t19\tgld,mul=1280\t11\tAct 2 Junk\t15\n",
	"uniqueitems":      "index\tenabled\tlvl\tlvl req\tcode\tprop1\tpar1\tmin1\tmax1\nHarlequin Crest\t1\t62\t62\tuap\tallskills\t\t2\t2\nThe Gnasher\t1\t1\t1\thax\tdmg%\t\t60\t100\n",
	"runes":            "Name\tRune Name\tcomplete\tserver\titype1\titype2\tRune1\tRune2\tT1Code1\tT1Param1\tT1Min1\tT1Max1\nRuneword1\tAncient's Pledge\t1\t\tshld\t\tr08\tr07\tres-all\t\t43\t48\nRuneword130\tSpirit\t1\t1\tswor\t\tr07\tr10\tcast2\t\t25\t35\nRuneword99\tUnfinished\t\t\tswor\t\tr01\tr02\tac\t\t1\t1\n",
	"hireling":         "Hireling\tSubType\tId\tClass\tAct\tDifficulty\tLevel\tSeller\tGold\tHP\tHP/Lvl\tSkill1\tMode1\tChance1\tSkill2\tLevel2\nDesert Mercenary\tComb-Normal\t6\t338\t2\t1\t9\t198\t350\t100\t10\tPrayer\t4\t100\tSmite\t3\nDesert Mercenary\tComb-Normal\t6\t338\t2\t1\t12\t198\t350\t130\t10\tPrayer\t4\t100\t\t\nDesert Mercenary\tComb-Hell\t12\t338\t2\t3\t55\t198\t15000\t600\t30\tPrayer\t4\t100\t\t\nExpansion\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n",
	"itemstatcost":     "Stat\tID\tSigned\tSend Bits\tSaved\tCSvBits\tValShift\tSave Bits\tSave Add\tdescstrpos\nstrength\t0\t\t11\t1\t10\t\t8\t32\tModStr1a\nhitpoints\t6\t\t32\t1\t21\t8\t\t\t\ngold\t14\t\t32\t1\t25\t\t\t\t\n",
	"properties":       "code\tset1\tval1\tfunc1\tstat1\tset2\tval2\tfunc2\tstat2\nres-all\t\t\t1\tfireresist\t\t\t3\tlightresist\nmove2\t\t\t1\tvelocitypercent\t\t\t\t\n",
	"magicprefix":      "Name\tversion\tspawnable\tlevel\tlevelreq\tgroup\tmod1code\tmod1min\tmod1max\titype1\titype2\tetype1\n\t\t\t\t\t\t\t\t\t\t\t\nSturdy\t0\t1\t4\t3\t101\tac%\t20\t30\tarmo\t\t\n",
	"magicsuffix":      "Name\tspawnable\tlevel\tgroup\tmod1code\tmod1param\tmod1min\tmod1max\titype1\titype2\tetype1\nof Health\t1\t7\t1\tlife\t\t1\t1\ttors\thelm\tshld\n",
	"states":           "state\tid\tgroup\tcurse\thide\toverlay1\toverlay2\tstat\nnone\t0\t\t\t\t\t\t\nfreeze\t1\t\t\t1\tov_a\tov_b\t\nslowed\t56\t\t1\t\t\t\tslow\n",
	"missiles":         "Missile\tId\tVel\tRange\tLevRange\tSkill\tPierce\tCollideKill\narrow\t0\t24\t40\t\t\t\t1\nfirebolt\t58\t20\t50\t4\tFire Bolt\t\t1\n",
	"lvlwarp":          "Name\tId\tSelectX\tSelectY\tSelectDX\tSelectDY\tExitWalkX\tExitWalkY\tOffsetX\tOffsetY\tLitVersion\tTiles\tDirection\nAct 1 Wilderness to Cave Cliff L\t0\t-90\t-100\t90\t110\t5\t0\t1\t-1\t1\t2\tb\n",
	"cubemain":         "description\tenabled\tnuminputs\tinput 1\tinput 2\toutput\nPerfect Amethyst\t1\t3\t\"gsv,qty=3\"\t\t\"gsw\"\n",
	"superuniques":     "Superunique\tName\tClass\thcIdx\tMod1\tTC\nBishibosh\tBishibosh\tfallenshaman1\t0\t8\tAct 1 Super A\n",
	"objects":          "Name\tId\tToken\tSizeX\tSizeY\tSelectable0\tSubClass\nWaypoint\t119\tWS\t2\t2\t1\t64\n",
	"monlvl":           "Level\tL-HP\tL-HP(N)\tL-HP(H)\tHP\n0\t1\t1\t1\t1\n1\t10\t200\t300\t7\n",
	"difficultylevels": "Name\tResistPenalty\tDeathExpPenalty\nNormal\t0\t0\nNightmare\t-40\t5\nHell\t-100\t10\n",
}

func writeInstall(t *testing.T, dir string, mutate func(patch, exp, base map[string][]byte)) {
	t.Helper()
	patch := map[string][]byte{}
	exp := map[string][]byte{}
	base := map[string][]byte{}
	for n, s := range fixtures {
		// Spread tables over the archives so priority order matters.
		switch n {
		case "monstats", "skills", "belts":
			base[excelDir+n+".txt"] = []byte(s)
		default:
			exp[excelDir+n+".txt"] = []byte(s)
		}
	}
	// The patch overrides a monstats value and the expansion string table.
	patch[excelDir+"monstats.txt"] = []byte("Id\thcIdx\tNameStr\tLevel\tLevel(N)\tLevel(H)\tResFi\tResPo(H)\tboss\tminion1\n" +
		"skeleton1\t0\tskeleton1\t3\t37\t68\t\t\t\t\n" +
		"andariel\t1\tandariel\t12\t49\t75\t-50\t100\t1\tskeleton1\n" +
		"Expansion\t\t\t\t\t\t\t\t\t\n")
	base[lngDir+"string.tbl"] = makeTBL([][2]string{
		{"skeleton1", "Skeleton"}, {"andariel", "Andariel"}, {"skillname36", "Fire Bolt"}, {"skillname54", "Teleport"},
		{"ssd", "Short Sword"}, {"cap", "Cap"}, {"The Gnasher", "The Gnasher"}, {"Harlequin Crest", "Harlequin Crest"},
		{"Bishibosh", "Bishibosh"}, {"hp1", "Minor Healing Potion"}, {"Ancient's Pledge", "Ancient's Pledge"},
	})
	patch[lngDir+"patchstring.tbl"] = makeTBL([][2]string{{"skeleton1", "ÿc1Skeleton Warrior"}})
	if mutate != nil {
		mutate(patch, exp, base)
	}
	for name, files := range map[string]map[string][]byte{"Patch_D2.mpq": patch, "d2exp.mpq": exp, "d2data.mpq": base} {
		if err := os.WriteFile(filepath.Join(dir, name), mpq.Build(files), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "Game.exe"), []byte("MZ fake"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExtractAndLoad(t *testing.T) {
	inst, cache := t.TempDir(), filepath.Join(t.TempDir(), "cache")
	writeInstall(t, inst, nil)

	if CacheCurrent(inst, cache) {
		t.Fatal("no cache yet")
	}
	db, err := Open(inst, cache)
	if err != nil {
		t.Fatal(err)
	}
	if !CacheCurrent(inst, cache) {
		t.Fatal("cache should be current")
	}
	// Cache files are trimmed: the "Extra" column of monstats is not kept.
	b, _ := os.ReadFile(filepath.Join(cache, "monstats.tsv"))
	if ParseTable(b).Has("Extra") {
		t.Error("unused column kept in cache")
	}

	// Patch beats base for monstats; patchstring beats string.
	m := db.MonsterByClassID(0)
	if m == nil || m.Level[Normal] != 3 {
		t.Fatalf("patch override: %+v", m)
	}
	if m.Name != "Skeleton Warrior" {
		t.Errorf("name %q (colour codes stripped, patchstring first)", m.Name)
	}
	if db.MonsterByName("skeleton1") != m || db.MonsterByName("skeleton warrior") != m {
		t.Error("MonsterByName")
	}
	// The Expansion separator row is not a class; blank resists read 0.
	if len(db.Monsters) != 2 {
		t.Errorf("monsters %d", len(db.Monsters))
	}
	a := db.MonsterByName("Andariel")
	if !a.Boss || a.Minion1 != "skeleton1" || a.Res[Normal][Fire] != -50 || !a.Immune(Poison, Hell) || a.Immune(Poison, Normal) {
		t.Errorf("andariel %+v", a)
	}
	// levels: the Ex column wins, separator rows dropped
	l := db.AreaByID(2)
	if l == nil || l.Name != "Blood Moor" || l.MonLvl[0] != 5 || len(l.Mon) != 1 || l.Vis[0] != 1 || db.AreaByName("blood moor") != l {
		t.Errorf("level %+v", l)
	}
	if len(db.Levels) != 3 {
		t.Errorf("levels %d", len(db.Levels))
	}
	s := db.SkillByID(36)
	if s == nil || s.Name != "Fire Bolt" || s.Page != 1 || s.Column != 2 || s.ManaCost(1) != 2.5 || s.ManaCost(30) != 2.5 {
		t.Errorf("skill %+v", s)
	}
	if syn := s.Synergies(); len(syn) != 2 || syn[0] != "Fire Ball" {
		t.Errorf("syn %v", syn)
	}
	if db.SkillByName("teleport").ManaCost(5) != 24-4 {
		t.Errorf("teleport mana %v", db.SkillByName("teleport").ManaCost(5))
	}
	if db.ExpToLevel(1) != 0 || db.ExpToLevel(2) != 500 || db.ExpToLevel(4) != 3750 || db.ExpToLevel(50) != 3750 {
		t.Errorf("exp %d %d", db.ExpToLevel(2), db.ExpToLevel(4))
	}
	if db.Exp.LevelFor(1499) != 2 || db.Exp.LevelFor(1500) != 3 || db.Exp.MaxLevel != 4 {
		t.Error("LevelFor")
	}
	c := db.ClassByName("sorceress")
	if c == nil || c.Code != "sor" || len(c.Items) != 1 || c.Items[0].Code != "sst" || len(db.Classes) != 1 {
		t.Errorf("class %+v", c)
	}
	if !db.IsA("swor", "weap") || db.IsA("weap", "swor") || db.IsA("nope", "weap") {
		t.Error("IsA")
	}
	if it := db.ItemByCode("ssd"); it == nil || it.Display != "Short Sword" || it.Width != 1 || it.Height != 3 || it.MaxDam != 7 || it.Kind != KindWeapon {
		t.Errorf("ssd %+v", it)
	}
	if it := db.ItemByCode("aqv"); it == nil || !it.Stackable || it.MaxStack != 350 || it.Kind != KindMisc || db.ItemByCode("lbl").BeltIndex != 1 || !db.ItemByCode("hp1").AutoBelt {
		t.Errorf("misc items")
	}
	inv := db.InventoryLayout("Sorceress")
	if inv == nil || inv.Cols != 10 || inv.Rows != 4 || inv.Slots["head"].Width != 54 {
		t.Fatalf("inv %+v", inv)
	}
	if x, y := inv.CellCenter(0, 0); x != 339+14 || y != 255+14 {
		t.Errorf("cell centre %d,%d", x, y)
	}
	if bank := db.InventoryLayout("Bank Page 1"); bank == nil || bank.Cols != 6 || bank.Rows != 8 || len(bank.Slots) != 0 {
		t.Errorf("bank %+v", bank)
	}
	if bl := db.BeltLayout("belt"); bl == nil || len(bl.Boxes) != 2 || bl.Boxes[1].Left != 374 || db.BeltByIndex(1).Name != "sash" || len(db.BeltLayout("sash").Boxes) != 1 {
		t.Errorf("belt %+v", bl)
	}
	if tc := db.TreasureClassByName("andariel"); tc == nil || tc.Picks != 7 || len(tc.Items) != 2 || tc.TotalProb != 19+11+15 {
		t.Errorf("tc %+v", tc)
	}
	if u := db.UniqueByName("Harlequin Crest"); u == nil || u.Code != "uap" || len(u.Props) != 1 || u.Props[0].Min != 2 {
		t.Errorf("unique %+v", u)
	}
	if w := db.RunewordByName("Ancient's Pledge"); w == nil || len(w.Runes) != 2 || w.Runes[1] != "r07" || w.ItemTypes[0] != "shld" || w.Props[0].Max != 48 {
		t.Errorf("runeword %+v", w)
	}
	if len(db.Cube) != 1 || db.Cube[0].Inputs[0] != `"gsv,qty=3"` {
		t.Errorf("cube %+v", db.Cube)
	}
	if su := db.SuperUniqueByName("Bishibosh"); su == nil || su.Class != "fallenshaman1" || db.SuperUniqueByID(0) != su {
		t.Errorf("su %+v", su)
	}
	if o := db.ObjectByID(119); o == nil || o.Name != "Waypoint" || !o.Selectable || o.SubClass != 64 {
		t.Errorf("object %+v", o)
	}
	if db.Difficulties[Hell].ResistPenalty != -100 || db.MonLevelRow(1).HP[Hell] != 300 || db.ScaledHP(a, Hell, 1) != a.HPMax[Hell]*3 {
		t.Errorf("monlvl/difficulty")
	}
	if !IsTown(40) || IsTown(2) || Town(4) != 109 {
		t.Error("town helpers")
	}
	if db.Text("nonexistent") != "nonexistent" {
		t.Error("Text fallback")
	}

	extraTables(t, db)
	if st, err := InstallStamp(inst); err != nil || len(st.PatchSHA256) != 64 {
		t.Errorf("stamp sha: %+v %v", st, err)
	}

	// Stamp: changing an archive size invalidates the cache; re-Open re-extracts.
	if err := os.WriteFile(filepath.Join(inst, "Game.exe"), []byte("MZ different size"), 0o644); err != nil {
		t.Fatal(err)
	}
	if CacheCurrent(inst, cache) {
		t.Error("cache should be stale after Game.exe changed")
	}
	if _, err := Open(inst, cache); err != nil || !CacheCurrent(inst, cache) {
		t.Errorf("re-extract: %v", err)
	}

	// Round trip without a cache gives the same answers.
	db2, err := LoadInstall(inst)
	if err != nil || db2.SkillByID(36).Name != "Fire Bolt" {
		t.Errorf("LoadInstall: %v", err)
	}
}

func TestExtractErrors(t *testing.T) {
	empty := t.TempDir()
	if err := Extract(empty, t.TempDir()); err == nil {
		t.Error("expected error for empty install")
	}
	// A missing required table is an error naming it; a missing optional one is not.
	inst := t.TempDir()
	writeInstall(t, inst, func(patch, exp, base map[string][]byte) {
		delete(exp, excelDir+"experience.txt")
	})
	if err := Extract(inst, t.TempDir()); err == nil {
		t.Error("missing experience.txt should fail")
	}
	inst = t.TempDir()
	writeInstall(t, inst, func(patch, exp, base map[string][]byte) {
		delete(exp, excelDir+"cubemain.txt")
		delete(exp, excelDir+"belts.txt")
		delete(base, excelDir+"belts.txt")
	})
	db, err := Open(inst, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Cube) != 0 || db.BeltLayout("belt") != nil {
		t.Error("optional tables should be empty")
	}
	// Load on a dir without a stamp (interrupted extraction) fails.
	if _, err := Load(t.TempDir()); err == nil {
		t.Error("Load of empty dir should fail")
	}
}

func extraTables(t *testing.T, db *DB) {
	t.Helper()
	hs := db.HirelingsFor(2, Normal)
	if len(hs) != 2 || len(db.Hirelings) != 3 || hs[0].Level != 9 || hs[1].HP != 130 || hs[0].Gold != 350 || hs[0].Seller != 198 {
		t.Errorf("hirelings %d %+v", len(hs), hs)
	}
	if len(hs[0].Skills) != 2 || hs[0].Skills[0].Name != "Prayer" || hs[0].Skills[0].Chance != 100 || hs[0].Skills[1].Level != 3 {
		t.Errorf("hire skills %+v", hs[0].Skills)
	}
	if h := db.HirelingByID(12); h == nil || h.Gold != 15000 || len(db.HirelingsFor(2, Hell)) != 1 || h.Aura(db) != "" {
		t.Errorf("hireling 12 %+v", h)
	}
	hp := db.StatByName("HitPoints")
	if hp == nil || hp.ID != 6 || hp.ValShift != 8 || hp.FromMemory(100<<8) != 100 || hp.ToMemory(3) != 768 || db.StatByID(6) != hp {
		t.Errorf("stat %+v", hp)
	}
	if s := db.StatByID(0); s == nil || s.SaveBits != 8 || s.SaveAdd != 32 || s.CSvBits != 10 || s.DescStrPos != "ModStr1a" || s.FromMemory(7) != 7 {
		t.Errorf("strength %+v", s)
	}
	if p := db.PropertyByCode("res-all"); p == nil || len(p.Slots) != 2 || p.Slots[1].Func != 3 || p.Slots[1].Stat != "lightresist" {
		t.Errorf("property %+v", p)
	}
	if len(db.Prefixes) != 1 || len(db.Suffixes) != 1 {
		t.Fatalf("affixes %d %d (blank row must be skipped)", len(db.Prefixes), len(db.Suffixes))
	}
	if a := db.AffixByName("sturdy", false); a == nil || a.Suffix || a.Level != 4 || a.LevelReq != 3 || a.Group != 101 || a.Mods[0].Code != "ac%" || a.Mods[0].Max != 30 || a.Include[0] != "armo" {
		t.Errorf("prefix %+v", a)
	}
	if a := db.AffixByName("of Health", true); a == nil || !a.Suffix || len(a.Include) != 2 || a.Exclude[0] != "shld" || db.AffixByName("of Health", false) != nil {
		t.Errorf("suffix %+v", a)
	}
	if s := db.StateByName("freeze"); s == nil || s.ID != 1 || !s.Hide || len(s.Overlay) != 2 || db.StateByID(56) == nil || !db.StateByID(56).Curse || db.StateByID(56).Stat != "slow" {
		t.Errorf("states %+v", s)
	}
	if m := db.MissileByName("firebolt"); m == nil || m.ID != 58 || m.Range != 50 || m.RangeAt(1) != 50 || m.RangeAt(3) != 58 || m.RangeAt(0) != 50 || !m.CollideKill || db.MissileByID(0).Vel != 24 {
		t.Errorf("missile %+v", m)
	}
	if w := db.WarpByID(0); w == nil || w.SelectX != -90 || w.SelectDY != 110 || !w.LitVersion || w.Direction != "b" || db.WarpByID(5) != nil {
		t.Errorf("warp %+v", w)
	}
	if sp := db.RunewordByName("Spirit"); sp == nil || !sp.Ladder || !sp.Server {
		t.Errorf("spirit %+v", sp)
	}
	if !db.RunewordAvailableOffline("Ancient's Pledge") || db.RunewordAvailableOffline("Spirit") || db.RunewordAvailableOffline("Unfinished") || db.RunewordAvailableOffline("Nope") {
		t.Error("RunewordAvailableOffline")
	}
}

func TestStampPatchHash(t *testing.T) {
	inst, cache := t.TempDir(), t.TempDir()
	writeInstall(t, inst, nil)
	if _, err := Open(inst, cache); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(inst, "Patch_D2.mpq")
	b, _ := os.ReadFile(p)
	b[len(b)-1] ^= 0xFF // same size, different content
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if CacheCurrent(inst, cache) {
		t.Error("same-size patch change must invalidate the cache")
	}
	// A cache stamped with an older schema is stale.
	old := Stamp{Schema: schemaVersion - 1}
	if old.equal(Stamp{Schema: schemaVersion}) {
		t.Error("schema bump")
	}
}

func TestOpenCachedLayoutAndLegacy(t *testing.T) {
	inst, root, legacy := t.TempDir(), t.TempDir(), filepath.Join(t.TempDir(), "old")
	writeInstall(t, inst, nil)
	l := paths.Layout{Root: root}
	if _, err := Open(inst, legacy); err != nil { // an old-style cache
		t.Fatal(err)
	}
	// Migration: the current legacy cache is copied into <data>/cache/data/v<schema>-<stamp>, not re-extracted.
	sentinel := filepath.Join(legacy, "marker.txt")
	os.WriteFile(sentinel, []byte("copied"), 0o644)
	db, err := OpenCached(inst, l, legacy)
	if err != nil || db.SkillByID(36).Name != "Fire Bolt" {
		t.Fatalf("OpenCached: %v", err)
	}
	dir, _ := CacheDir(inst, l)
	if !strings.HasPrefix(dir, filepath.Join(root, "cache", "data", "v")) || !CacheCurrent(inst, dir) {
		t.Fatalf("dir %s", dir)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "marker.txt")); string(b) != "copied" {
		t.Error("legacy cache was re-extracted instead of copied")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Error("legacy cache removed")
	}
	// A different install gets its own directory; the first one is untouched.
	os.WriteFile(filepath.Join(inst, "Game.exe"), []byte("MZ different size"), 0o644)
	dir2, _ := CacheDir(inst, l)
	if dir2 == dir {
		t.Fatal("stamp change must change the directory")
	}
	if _, err := OpenCached(inst, l); err != nil || !CacheCurrent(inst, dir2) {
		t.Fatalf("second: %v", err)
	}
	if !dirHas(dir, "marker.txt") {
		t.Error("old stamp dir was modified")
	}
}

func dirHas(d, n string) bool { _, err := os.Stat(filepath.Join(d, n)); return err == nil }
