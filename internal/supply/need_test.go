package supply

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProfilesValidAndUnknownKeyRejected(t *testing.T) {
	if _, err := loadProfiles(); err != nil {
		t.Fatal(err)
	}
	var f struct {
		Profiles map[string]map[string]row `json:"profiles"`
	}
	dec := jsonDecoder(`{"profiles":{"x":{"healing":{"min":1,"belt":3}}}}`)
	if err := dec.Decode(&f); err == nil || !strings.Contains(err.Error(), "belt") {
		t.Fatalf("unknown condition key must be an error naming it, got %v", err)
	}
}

// SME review 2026-11-27, confidence high: 8 healing + 3 mana (11 boxes) cannot fit an 8-box belt.
func TestAndarielKitFitsBelt(t *testing.T) {
	want := map[int][2]int{12: {8, 3}, 8: {6, 2}}
	for name, in := range builds {
		for slots, w := range want {
			in.BeltSlots = slots
			n, err := NeedFor("boss", "andariel", in)
			if err != nil {
				t.Fatal(err)
			}
			if h := n.Lines[Healing].Min; h != w[0] {
				t.Errorf("%s belt %d: healing %d, want %d", name, slots, h, w[0])
			}
			m := n.Lines[Mana].Min
			if name == "sorc" && m != w[1] || name == "pal" && m != 0 {
				t.Errorf("%s belt %d: mana %d", name, slots, m)
			}
			if name == "sorc" && n.Lines[Healing].Min+m > slots {
				t.Errorf("%s belt %d: kit %d does not fit", name, slots, n.Lines[Healing].Min+m)
			}
		}
	}
}

func jsonDecoder(s string) *json.Decoder {
	d := json.NewDecoder(strings.NewReader(s))
	d.DisallowUnknownFields()
	return d
}

// SME review 2026-11-27: blood_raven mana 4 (high), duriel, catacombs, tristram, countess (medium), andariel tier floor (medium).
func TestSMEProfileRows(t *testing.T) {
	type want struct{ heal, portal, mana, gold, tier int }
	cases := map[string]want{
		"blood_raven": {4, 2, 4, 250, 0},
		"duriel":      {8, 2, 4, 4500, 0},
		"catacombs":   {6, 2, 2, 0, 0},
		"tristram":    {4, 2, 0, 0, 0},
		"countess":    {6, 2, 2, 0, 0},
		"andariel":    {8, 1, 3, 3000, 2},
	}
	for prof, w := range cases {
		for name, in := range builds {
			in.BeltSlots = 12
			n, err := NeedFor("x", prof, in)
			if err != nil {
				t.Fatalf("%s: %v", prof, err)
			}
			mana := w.mana
			if name == "pal" || prof == "tristram" {
				mana = 0 // mana line applies only to a build that uses mana potions
			}
			got := want{n.Lines[Healing].Min, n.Lines[Portal].Min, n.Lines[Mana].Min, n.Lines[GoldCarry].Min, n.Lines[Healing].TierFloor}
			if got != (want{w.heal, w.portal, mana, w.gold, w.tier}) {
				t.Errorf("%s/%s: got %+v", prof, name, got)
			}
		}
	}
}
