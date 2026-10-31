package gear

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A synthetic game: every row below is made up (names, numbers), only the column names follow the real tables. It lets
// CI and public forks test the extractor without any game file.
func fixtureRead(name string) ([]byte, error) {
	tsv := func(rows ...string) ([]byte, error) { return []byte(strings.Join(rows, "\r\n") + "\r\n"), nil }
	switch name {
	case "itemtypes":
		return tsv("ItemType\tCode\tEquiv1\tEquiv2\tShoots\tQuiver\tThrowable\tClass\tBodyLoc1\tBodyLoc2",
			"Thing\tthng\t\t\t\t\t0\t\ttors\t",
			"Shield\tshld\tthng\t\t\t\t0\t\tlarm\trarm",
			"Gizmo\tgzmo\tshld\t\t\t\t0\tzzz\tlarm\t",
			"Sling\tslng\tthng\t\tfoo\tbar\t0\t\trarm\tlarm",
			"Pebble\tpebl\tthng\t\t\t\t1\t\t\t",
			"Foo\tfoo\t\t\t\t\t0\t\t\t",
			"\t\t\t\t\t\t\t\t\t")
	case "weapons":
		return tsv("name\ttype\tcode\t2handed\treqstr\treqdex\tlevelreq\tstackable\tminstack\tmaxstack\tcost\tPermStoreItem",
			"Test Sling\tslng\ts01\t0\t10\t20\t3\t0\t0\t0\t55\t0",
			"Big Gizmo\tgzmo\ts02\t1\t99\t\t7\t0\t0\t0\t900\t1",
			"No Code\tslng\t\t0\t1\t1\t1\t0\t0\t0\t1\t0")
	case "misc":
		return tsv("name\ttype\tcode\tstackable\tminstack\tmaxstack\tcost\tPermStoreItem",
			"Fake Ammo\tbowq\tm01\t1\t10\t90\t2\t1",
			"Other Thing\tthng\tm02\t0\t0\t0\t5\t0")
	case "armor":
		// duplicated header (cost): armor rows take the first column, the base list the last (as the old gen.py); Gizmo shields are picked up as bases
		return tsv("name\ttype\tcode\tnormcode\tubercode\tultracode\tminac\tmaxac\treqstr\tlevelreq\tlevel\tspeed\tblock\tdurability\tnodurability\tgemsockets\tStrBonus\tDexBonus\tinvwidth\tinvheight\tcost\tcost",
			"Made Plate\tthng\ta01\ta01\ta02\ta03\t10\t20\t30\t4\t5\t6\t0\t40\t0\t3\t100\t0\t2\t3\t11\t99",
			"Better Plate\tthng\ta02\ta01\ta02\ta03\t11\t21\t31\t14\t15\t6\t0\t40\t1\t3\t100\t0\t2\t3\t12\t98",
			"Ultra Plate\tthng\ta03\ta01\ta02\ta03\t12\t22\t32\t24\t25\t6\t0\t40\t0\t4\t100\t0\t2\t3\t13\t97",
			"Wall\tgzmo\tg01\tg01\t\t\t5\t6\t7\t8\t9\t0\t55\t12\t0\t1\t\t\t2\t2\t30\t31")
	case "skills":
		return tsv("skill\tcharclass\tId\treqlevel\treqskill1\treqskill2\treqskill3\titypea1\titypea2\titypea3\tetypea1\tetypea2\tleftskill\tpassive",
			"Fakebolt\tzzz\t7\t1\t\t\t\t\t\t\t\t\t1\t0",
			"Fakenova\tzzz\t8\t6\tFakebolt\t\t\tslng\tgzmo\t\tfoo\t\t0\t1",
			"Throw\t\t9\t1\t\t\t\t\t\t\t\t\t0\t0")
	case "charstats":
		return tsv("class\tstr\tdex\tvit\tint\tStartSkill\titem1\titem1loc\titem1count\titem2\titem2loc\titem2count",
			"Testmage\t10\t11\t12\t13\tFakebolt\ts01\trarm\t\tm01\tbelt\t4",
			"Expansion\t\t\t\t\t\t\t\t\t\t\t",
			"Emptyman\t1\t2\t3\t4\t\t0\t\t\t\t\t")
	}
	return nil, fmt.Errorf("no table %s", name)
}

func TestGenerateFixture(t *testing.T) {
	tb, ar, err := generate(fixtureRead)
	if err != nil {
		t.Fatal(err)
	}
	if got := tb.Types["gzmo"]; !reflect.DeepEqual(got, typeRow{Equiv: []string{"shld"}, Class: "zzz", Body: []string{"larm"}}) {
		t.Errorf("gzmo %+v", got)
	}
	if len(tb.Types) != 6 {
		t.Errorf("types %d, blank code row must be skipped", len(tb.Types))
	}
	if !tb.Types["pebl"].Throwable || tb.Types["slng"].Shoots != "foo" {
		t.Error("type flags")
	}
	want := map[string]Base{
		"s01": {Name: "Test Sling", Type: "slng", ReqStr: 10, ReqDex: 20, LevelReq: 3, Cost: 55},
		"s02": {Name: "Big Gizmo", Type: "gzmo", TwoHanded: true, ReqStr: 99, LevelReq: 7, Cost: 900, PermStore: true},
		"m01": {Name: "Fake Ammo", Type: "bowq", Stackable: true, MinStack: 10, MaxStack: 90, Cost: 2, PermStore: true},
		"g01": {Name: "Wall", Type: "gzmo", ReqStr: 7, LevelReq: 8, Cost: 31},
	}
	if !reflect.DeepEqual(tb.Bases, want) {
		t.Errorf("bases %+v", tb.Bases)
	}
	// m02 is a misc item that is not ammo, a01..a03 are armor that is not a shield: neither is a base
	if len(ar) != 4 || ar["a02"].Tier != 1 || ar["a01"].Tier != 0 || ar["a03"].Tier != 2 || !ar["a02"].NoDurability || ar["a01"].InvHeight != 3 {
		t.Errorf("armor %+v", ar)
	}
	if sk := tb.Skills["Fakenova"]; sk.ReqLevel != 6 || !sk.Passive || sk.Left || sk.ID != 8 ||
		!reflect.DeepEqual(sk.Req, []string{"Fakebolt"}) || !reflect.DeepEqual(sk.ItypeA, []string{"slng", "gzmo"}) || !reflect.DeepEqual(sk.EtypeA, []string{"foo"}) {
		t.Errorf("skill %+v", sk)
	}
	if _, ok := tb.Skills["Throw"]; ok || len(tb.Skills) != 2 {
		t.Error("skills without a class are not tables rows")
	}
	c, ok := tb.Classes["tes"]
	if !ok || len(tb.Classes) != 2 || c.Str != 10 || c.Ene != 13 || len(c.Items) != 2 || c.Items[0].Count != 1 || c.Items[1].Count != 4 || c.Items[1].Loc != "belt" {
		t.Errorf("classes %+v", tb.Classes)
	}
}

func TestWriteAndLoadFixture(t *testing.T) {
	dir := t.TempDir()
	if err := writeTables(fixtureRead, TablesStamp{Schema: tablesSchema}, dir); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"gear.json", "armor.json", "stamp.json"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Error(err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(dir, "armor.json"))
	var a struct {
		Armor map[string]Armor `json:"armor"`
	}
	if err := json.Unmarshal(b, &a); err != nil || a.Armor["g01"].Block != 55 {
		t.Errorf("armor.json %v %+v", err, a.Armor["g01"])
	}
	if TablesCurrent(t.TempDir(), dir) {
		t.Error("an empty data dir must not match the stamp")
	}
}

func TestMissingTablesMessage(t *testing.T) {
	t.Setenv("D2_TABLES_DIR", t.TempDir())
	if err := loadTables("", t.TempDir()); err != ErrNoTables {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(ErrNoTables.Error(), "run `go run ./cmd/d2tables") {
		t.Error("message must say how to fix it")
	}
}
