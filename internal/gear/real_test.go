package gear

import (
	"os"
	"testing"

	"d2lod-bot/internal/data"
)

// TestRealMatches checks the embedded tables against the install when /tmp/d2mpq exists (data.json is regenerated
// with gen.py; this catches a stale file after a data change). Skipped otherwise.
func TestRealMatches(t *testing.T) {
	install := "/tmp/d2mpq"
	if v := os.Getenv("D2MPQ"); v != "" {
		install = v
	}
	if _, err := os.Stat(install + "/d2data.mpq"); err != nil {
		t.Skip("no real install")
	}
	needTables(t)
	db, err := data.Open(install, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for code, b := range load().Bases {
		ib := db.ItemByCode(code)
		if ib == nil || ib.Type != b.Type || ib.Cost != b.Cost || ib.MaxStack != b.MaxStack || ib.ReqDex != b.ReqDex {
			t.Errorf("base %s differs: %+v vs %+v", code, b, ib)
		}
	}
	for _, code := range ArmorCodes() {
		a, _ := ArmorByCode(code)
		ib := db.ItemByCode(code)
		if ib == nil || ib.Type != a.Type || ib.ReqStr != a.ReqStr || ib.LevelReq != a.LevelReq || ib.MinAC != a.MinAC || ib.MaxAC != a.MaxAC {
			t.Errorf("armor %s differs: %+v vs %+v", code, a, ib)
		}
	}
	for code, ty := range load().Types {
		it := db.ItemType(code)
		if it == nil || it.Shoots != ty.Shoots || it.Quiver != ty.Quiver || it.Throwable != ty.Throwable {
			t.Errorf("type %s differs", code)
		}
	}
	for name, s := range load().Skills {
		ds := db.SkillByName(name)
		if ds == nil || ds.ReqLevel != s.ReqLevel || ds.Class != s.Class {
			t.Errorf("skill %s differs", name)
		}
	}
}
