package invdesk

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// Compatibility of /api/inventory (docs/ui-design-system.md section 0, docs/inventory-actions.md): schema 3, 4 and 5 only add (the view_schema3 and view_schema4 goldens are kept as history; schema 4 added the merc block, schema 5 the identify block, see identify_test.go).
// The golden files are the view of a fixed scenario with a queued, a done and a cancelled request; UPDATE=1 rewrites
// them, and an existing one must never be edited by hand to make a test pass (add a field, do not rename one).
var v2Keys = []string{"schema", "inTown", "class", "gold", "stashGold", "equipment", "backpack", "stash", "belt", "pins", "queue", "recent"}
var v2ItemKeys = []string{"id", "code"}
var v2RequestKeys = []string{"id", "slot", "name", "state", "at"}

func TestViewSchema5Golden(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, _, _ := trashRig(t, class, field)
		now := time.Unix(1_800_000_000, 0)
		d.c.Now = func() time.Time { return now }
		good := map[d2.Class]uint32{d2.Paladin: 1, d2.Sorceress: 2}[class]
		if _, err := d.RequestEquip(good, "rhand"); err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestTrash(4, true); code(err) != "never" {
			t.Fatal(err)
		}
		d.RunPending(context.Background())
		if _, err := d.RequestEquip(good, "rhand"); err != nil {
			t.Fatal(err)
		}
		if err := d.Cancel(good, KindEquip); err != nil {
			t.Fatal(err)
		}
		if _, err := d.RequestEquip(good, "rhand"); err != nil {
			t.Fatal(err)
		}
		b, err := json.MarshalIndent(asSchema5(d.View()), "", " ")
		if err != nil {
			t.Fatal(err)
		}
		b = append(b, '\n')
		path := filepath.Join("testdata", "view_schema5_"+class.String()+".json")
		if os.Getenv("UPDATE") != "" {
			if err := os.WriteFile(path, b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != string(b) {
			t.Errorf("%v: view differs from %s\n got: %s\nwant: %s", class, path, b, want)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		for _, k := range v2Keys {
			if _, ok := m[k]; !ok {
				t.Errorf("%v: schema 2 key %q is gone", class, k)
			}
		}
		pack := m["backpack"].([]any)[0].(map[string]any)
		for _, k := range v2ItemKeys {
			if _, ok := pack[k]; !ok {
				t.Errorf("%v: schema 2 item key %q is gone", class, k)
			}
		}
		for _, r := range append(m["queue"].([]any), m["recent"].([]any)...) {
			for _, k := range v2RequestKeys {
				if _, ok := r.(map[string]any)[k]; !ok && !(k == "slot" && r.(map[string]any)["kind"] == KindTrash) {
					t.Errorf("%v: schema 2 request key %q is gone: %v", class, k, r)
				}
			}
		}
	}
}

// A state file written before schema 3 (no queue, no recent, no trashed list) loads.
func TestOldStateFileLoads(t *testing.T) {
	p := filepath.Join(t.TempDir(), StateFile)
	old := `{"marks":[{"id":1,"print":"10/4/5","mark":"sell"}],"pins":[{"slot":"rhand","id":9,"name":"x","since":1}]}`
	if err := os.WriteFile(p, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := New(Config{Path: p})
	if err != nil || !d.Pinned(d2.BodyRHand) || d.Pending() != 0 {
		t.Fatalf("old file: %v", err)
	}
}

// asSchema5 pins the old goldens: the schema 5 files stay as they were (the new fields are all omitempty), only the
// version number moved on to 6.
func asSchema5(v View) View { v = strip7(v); v.Schema = 5; return v }

// asSchema6 pins the schema 6 goldens the same way: schema 7 only added optional fields (docs/gearsets.md).
func asSchema6(v View) View { v = strip7(v); v.Schema = 6; return v }

// strip7 clears what schema 7 added to the view, so the older goldens keep proving that nothing else moved.
func strip7(v View) View {
	v.Sets = nil
	for _, l := range [][]ItemView{v.Backpack, v.Stash, v.Belt, v.Cube} {
		for i := range l {
			l[i].NoWearSlot, l[i].NoUnequip = nil, ""
		}
	}
	for _, it := range v.Equipment {
		it.NoWearSlot, it.NoUnequip = nil, ""
	}
	return v
}
