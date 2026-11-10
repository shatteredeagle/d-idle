package data

import (
	"os"
	"testing"
)

// TestRealInvFile runs against a real 1.14d install, read-only (D2ART_INSTALL or /tmp/d2mpq); it skips without
// one. It checks that every icon name the resolver can return exists in the archives and that the unique and
// set override rows name the base item they sit on (the evidence for the Row semantics, U for the live id).
func TestRealInvFile(t *testing.T) {
	inst := os.Getenv("D2ART_INSTALL")
	if inst == "" {
		inst = realInstall
	}
	if _, err := os.Stat(inst); err != nil {
		t.Skip("no real install")
	}
	db, err := Open(inst, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	as, err := openArchives(inst)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAll(as)
	has := func(n string) bool { _, e := readFirst(as, `data\global\items\`+n+`.dc6`); return e == nil }

	var withIcon, missing int
	for class, b := range db.Items {
		for _, q := range []InvQuality{InvPlain, InvUnique, InvSet} {
			n, ok := db.InvFile(class, q, -1)
			if !ok {
				continue
			}
			withIcon++
			if !has(n) {
				missing++
				t.Errorf("class %d (%s) %v: %s.dc6 missing", class, b.Code, q, n)
			}
		}
	}
	var overrides int
	for _, u := range db.Uniques {
		if u.InvFile == "" {
			continue
		}
		overrides++
		if !has(lc(u.InvFile)) {
			t.Errorf("unique row %d %s: %s missing", u.Row, u.Index, u.InvFile)
		}
		if u.Code != "" && db.ItemByCode(u.Code) == nil { // a few placeholder rows have no code
			t.Errorf("unique row %d %s: base code %q unknown", u.Row, u.Index, u.Code)
		}
	}
	t.Logf("%d classes resolve to an icon, %d missing, %d unique overrides of %d uniques, %d set items", withIcon, missing, overrides, len(db.Uniques), len(db.SetItems))
	if withIcon == 0 {
		t.Error("no icon names read: the invfile columns did not load")
	}
	for _, u := range db.Uniques {
		if u.Index == "Harlequin Crest" || u.Index == "The Stone of Jordan" || u.Index == "The Gnasher" {
			t.Logf("row %d = %s (base %s)", u.Row, u.Index, u.Code)
		}
	}
}

// TestRealPanelGeometry logs the equipment slot boxes of every player inventory.txt row relative to the panel
// origin (the numbers docs/paperdoll-art-build.md and web/components.css use for the art layout) and checks
// that all classes share them.
func TestRealPanelGeometry(t *testing.T) {
	inst := os.Getenv("D2ART_INSTALL")
	if inst == "" {
		inst = realInstall
	}
	if _, err := os.Stat(inst); err != nil {
		t.Skip("no real install")
	}
	db, err := Open(inst, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var ref *Inventory
	for _, c := range []string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"} {
		var v *Inventory
		for _, x := range db.Inventories {
			if x.Class == c {
				v = x
			}
		}
		if v == nil {
			t.Fatalf("no inventory row %s", c)
		}
		if ref == nil {
			ref = v
			t.Logf("panel %+v grid %+v cell %dx%d (%dx%d cells)", v.Panel, v.GridRect, v.CellW, v.CellH, v.Cols, v.Rows)
			for _, s := range slotNames {
				r := v.Slots[s]
				t.Logf("%-6s x %3d y %3d w %2d h %2d (panel-relative)", s, r.Left-v.Panel.Left, r.Top-v.Panel.Top, r.Right-r.Left, r.Bottom-r.Top)
			}
			t.Logf("grid   x %3d y %3d", v.GridRect.Left-v.Panel.Left, v.GridRect.Top-v.Panel.Top)
		}
		for s, r := range ref.Slots {
			o := v.Slots[s]
			if o.Left-v.Panel.Left != r.Left-ref.Panel.Left || o.Top-v.Panel.Top != r.Top-ref.Panel.Top || o.Right-o.Left != r.Right-r.Left || o.Bottom-o.Top != r.Bottom-r.Top {
				t.Errorf("%s slot %s differs from %s", c, s, ref.Class)
			}
		}
	}
}
