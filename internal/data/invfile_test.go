package data

import (
	"os"
	"path/filepath"
	"testing"
)

// Fixture rows are made up: a weapon with a unique and a set icon, an armor with none, a misc rune and gem,
// uniques with and without an override.
func invDB() *DB {
	return &DB{
		Items: []*ItemBase{
			{Code: "ssd", InvFile: "invSsd", UniqueInvFile: "invSsdU", SetInvFile: "invSsdS"},
			{Code: "cap", InvFile: "invCap"},
			{Code: "r01", InvFile: "invrEl"},
			{Code: "gpv", InvFile: "invgsve"},
			{Code: "nofile"},
		},
		Uniques: []*UniqueItem{
			{Row: 0, Code: "ssd", InvFile: "invSsdX"},
			{Row: 1, Code: "ssd"},
			{Row: 2, Code: "cap", InvFile: "invCapX"},
			{Row: 4, Code: "cap", InvFile: "invCapY"}, // row 3 is the "Expansion" separator, not loaded
		},
		SetItems: []*SetItem{{Row: 0, Code: "ssd", InvFile: "invSetOverride"}, {Row: 1, Code: "ssd"}},
	}
}

func TestInvFile(t *testing.T) {
	db := invDB()
	for _, tc := range []struct {
		name  string
		class int
		q     InvQuality
		idx   int
		want  string
		ok    bool
	}{
		{"weapon plain", 0, InvPlain, 0, "invssd", true},
		{"weapon plain ignores index", 0, InvPlain, 2, "invssd", true},
		{"unique with row override", 0, InvUnique, 0, "invssdx", true},
		{"unique without override uses base uniqueinvfile", 0, InvUnique, 1, "invssdu", true},
		{"unique row of another base is ignored", 0, InvUnique, 2, "invssdu", true},
		{"unique, unknown row", 0, InvUnique, 99, "invssdu", true},
		{"armor unique falls back to invfile", 1, InvUnique, 3, "invcap", true},
		{"armor unique override after separator", 1, InvUnique, 4, "invcapy", true},
		{"set with row override", 0, InvSet, 0, "invsetoverride", true},
		{"set base setinvfile", 0, InvSet, 1, "invssds", true},
		{"armor set falls back", 1, InvSet, 0, "invcap", true},
		{"rune lowercased", 2, InvPlain, 0, "invrel", true},
		{"gem", 3, InvPlain, 0, "invgsve", true},
		{"no icon name", 4, InvPlain, 0, "", false},
		{"unknown class", 99, InvPlain, 0, "", false},
		{"negative class", -1, InvPlain, 0, "", false},
	} {
		got, ok := db.InvFile(tc.class, tc.q, tc.idx)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s: %q %v, want %q %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
	var nilDB *DB
	if _, ok := nilDB.InvFile(0, InvPlain, 0); ok {
		t.Error("nil db")
	}
}

// The columns reach the structs through a real extraction, including the schema bump that invalidates old
// caches (an old cache has no invfile column).
func TestInvFileColumns(t *testing.T) {
	inst, cache := t.TempDir(), filepath.Join(t.TempDir(), "cache")
	writeInstall(t, inst, func(patch, exp, base map[string][]byte) {
		exp[excelDir+"weapons.txt"] = []byte("name\ttype\tcode\tnamestr\tinvwidth\tinvheight\tinvfile\tuniqueinvfile\tsetinvfile\n" +
			"Short Sword\tswor\tssd\tssd\t1\t3\tinvssd\tinvssdu\tinvssds\n")
		exp[excelDir+"misc.txt"] = []byte("name\ttype\tcode\tnamestr\tinvwidth\tinvheight\tinvfile\tuniqueinvfile\n" +
			"El Rune\trune\tr01\tr01\t1\t1\tinvrEl\t\n")
		exp[excelDir+"uniqueitems.txt"] = []byte("index\tenabled\tlvl\tlvl req\tcode\tinvfile\nThe Gnasher\t1\t1\t1\tssd\tinvgnash\nExpansion\t\t\t\t\t\nHarlequin\t1\t62\t62\tssd\t\n")
	})
	db, err := Open(inst, cache)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := db.InvFile(0, InvUnique, 0); got != "invgnash" {
		t.Errorf("unique 0 icon %q", got)
	}
	if got, _ := db.InvFile(0, InvUnique, 2); got != "invssdu" { // the separator row counts: Harlequin is row 2
		t.Errorf("unique 2 icon %q", got)
	}
	if got, ok := db.InvFile(3, InvPlain, 0); !ok || got != "invrel" {
		t.Errorf("rune %q %v", got, ok)
	}
	if _, err := os.Stat(filepath.Join(cache, "weapons.tsv")); err != nil {
		t.Fatal(err)
	}
}
