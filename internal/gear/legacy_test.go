package gear

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestMatchesLegacyJSON proves the Go extractor equals the old committed data.json / armor.json (made by gen.py and
// gen_armor.py). It needs both the real install (D2_DATA_DIR) and the old files (D2_LEGACY_DIR, from git history).
func TestMatchesLegacyJSON(t *testing.T) {
	data, legacy := os.Getenv("D2_DATA_DIR"), os.Getenv("D2_LEGACY_DIR")
	if data == "" || legacy == "" {
		t.Skip("needs D2_DATA_DIR and D2_LEGACY_DIR")
	}
	cache := t.TempDir()
	if err := ExtractTables(data, cache); err != nil {
		t.Fatal(err)
	}
	var got, want table
	var gotA, wantA struct {
		Armor map[string]Armor `json:"armor"`
	}
	for _, c := range []struct {
		file, old string
		got, want any
	}{{"gear.json", "data.json", &got, &want}, {"armor.json", "armor.json", &gotA, &wantA}} {
		for p, v := range map[string]any{filepath.Join(cache, c.file): c.got, filepath.Join(legacy, c.old): c.want} {
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	norm := func(v *table) { // nil and empty slices are the same value
		for k, c := range v.Classes {
			if len(c.Items) == 0 {
				c.Items = nil
				v.Classes[k] = c
			}
		}
	}
	norm(&got)
	norm(&want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("gear tables differ (types %d/%d bases %d/%d skills %d/%d classes %d/%d)", len(got.Types), len(want.Types),
			len(got.Bases), len(want.Bases), len(got.Skills), len(want.Skills), len(got.Classes), len(want.Classes))
	}
	if !reflect.DeepEqual(gotA, wantA) {
		t.Errorf("armor differs (%d/%d)", len(gotA.Armor), len(wantA.Armor))
	}
	t.Logf("equal: types %d bases %d skills %d classes %d armor %d", len(got.Types), len(got.Bases), len(got.Skills), len(got.Classes), len(gotA.Armor))
}
