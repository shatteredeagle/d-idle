package status

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// API compatibility (docs/ui-design-system.md section 0): the tooltip fields are optional and additive. An item without
// them marshals exactly as before, and one with them matches the golden file.
func TestItemTipFieldsAreAdditive(t *testing.T) {
	old, _ := json.Marshal(Item{ID: 1, Code: "cap", Name: "Cap", Quality: QNormal, Identified: true, W: 2, H: 2})
	for _, k := range []string{`"tip"`, `"base"`, `"sell"`} {
		if strings.Contains(string(old), k) {
			t.Errorf("an item without tooltip data must not carry %s: %s", k, old)
		}
	}
	it := Item{ID: 1, Code: "cap", Name: "Cap", Quality: QMagic, Identified: true, Sockets: 2, Level: 12, W: 2, H: 2,
		Tip: []string{"Skull Cap", "Defense: 9"}, Sell: 30,
		Base: &ItemBase{Name: "Skull Cap", MinAC: 2, MaxAC: 5, ReqStr: 0, MaxSockets: 2, Dur: 12}}
	got, _ := json.MarshalIndent(it, "", " ")
	want, err := os.ReadFile("testdata/item_tip.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(got)+"\n" != string(want) {
		t.Errorf("item JSON differs from testdata/item_tip.json:\n%s", got)
	}
}
