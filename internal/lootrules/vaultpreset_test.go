package lootrules

import "testing"

// The Vault preset is the Starter with the ground stash and sell rules turned into `vault`; kit, gold, gems and the
// poor-mode line keep their own rules. Two classes, record-everything off and on.
func TestVaultPresetGround(t *testing.T) {
	for _, cls := range []string{"Sorceress", "Paladin"} {
		for _, all := range []bool{false, true} {
			rs := VaultPreset(all)
			b, err := rs.Marshal()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Parse(b); err != nil {
				t.Fatalf("the preset does not parse: %v", err)
			}
			c, err := Compile(rs, cls, "")
			if err != nil {
				t.Fatal(err)
			}
			cx := &Ctx{Class: cls, CLvl: 20, Gold: 5000, Needs: func(k string, _ *Item) bool { return k == "potion" }}
			poor := &Ctx{Class: cls, CLvl: 20, Gold: 100, Needs: cx.Needs}
			for _, tc := range []struct {
				label, want string
				ctx         *Ctx
				it          Item
			}{
				{"rune", Vault, cx, Item{Types: []string{"rune"}, Quality: "normal"}},
				{"unique", Vault, cx, Item{Types: []string{"swor"}, Quality: "unique", Hidden: HIdentified}},
				{"magic sword", Vault, cx, Item{Types: []string{"swor", "weap"}, Quality: "magic", Hidden: HIdentified | HStats}},
				{"needed potion stays a pickup", Pickup, cx, Item{Types: []string{"hpot"}, Quality: "normal"}},
				{"white club, rich", map[bool]string{false: Ignore, true: Vault}[all], cx, Item{Code: "clb", Types: []string{"club", "weap"}, Quality: "normal", Value: 5}},
				{"white club, poor: today's sell line", Pickup, poor, Item{Code: "clb", Types: []string{"club", "weap"}, Quality: "normal", Value: 25}},
			} {
				tc.it.Place = PlaceGround
				d := c.Decide(&tc.it, tc.ctx)
				if d.Action != tc.want {
					t.Errorf("%s all=%v %s: %+v", cls, all, tc.label, d)
				}
			}
		}
	}
}

// The Starter itself never says vault, and a pack-scope vault rule is refused.
func TestStarterHasNoVaultAndPackVaultIsRefused(t *testing.T) {
	for _, r := range Starter().Rules {
		if r.Action == Vault {
			t.Errorf("Starter rule %s is a vault rule", r.ID)
		}
	}
	rs := VaultPreset(false)
	rs.Rules = append(rs.Rules, Rule{ID: "bad", Scope: []string{PlacePack}, Action: Vault, Reason: "x"})
	b, _ := rs.Marshal()
	if _, err := Parse(b); err == nil {
		t.Error("a vault rule in the pack scope must be refused")
	}
}

// Rings, amulets and magic or rare charms reach the vault while unidentified (they are always unidentified on the
// ground); the Starter picked them up to identify them and none was ever recorded (docs/probes/vault-rings-2026-11-24.md).
func TestVaultPresetUnidentifiedJewelry(t *testing.T) {
	for _, cls := range []string{"Sorceress", "Paladin"} {
		for _, all := range []bool{false, true} {
			c, err := Compile(VaultPreset(all), cls, "")
			if err != nil {
				t.Fatal(err)
			}
			cx := &Ctx{Class: cls, CLvl: 20, Gold: 5000, Needs: func(string, *Item) bool { return false }}
			for _, tc := range []struct {
				label string
				it    Item
			}{
				{"magic ring", Item{Code: "rin", Types: []string{"ring"}, Quality: "magic"}},
				{"rare amulet", Item{Code: "amu", Types: []string{"amul"}, Quality: "rare"}},
				{"magic small charm", Item{Code: "cm1", Types: []string{"scha"}, Quality: "magic"}},
				{"magic grand charm", Item{Code: "cm3", Types: []string{"lcha"}, Quality: "magic"}},
				{"identified good ring", Item{Code: "rin", Types: []string{"ring"}, Quality: "magic", Hidden: HIdentified | HStats}},
			} {
				tc.it.Place = PlaceGround
				if d := c.Decide(&tc.it, cx); d.Action != Vault {
					t.Errorf("%s all=%v %s: %+v", cls, all, tc.label, d)
				}
			}
		}
	}
}

// Recording a large or grand charm needs no pack cells, so the charmroom need of the Starter does not hold it back.
func TestVaultPresetBigCharmNeedsNoRoom(t *testing.T) {
	for _, cls := range []string{"Sorceress", "Paladin"} {
		c, err := Compile(VaultPreset(false), cls, "")
		if err != nil {
			t.Fatal(err)
		}
		cx := &Ctx{Class: cls, CLvl: 20, Gold: 5000, Needs: func(string, *Item) bool { return false }}
		for _, tc := range []struct {
			label string
			it    Item
		}{
			{"identified grand charm with life", Item{Code: "cm3", Types: []string{"lcha"}, Quality: "magic", Identified: true, Stats: map[string]int{"life": 30}}},
			{"identified large charm with fire res", Item{Code: "cm2", Types: []string{"mcha"}, Quality: "magic", Identified: true, Stats: map[string]int{"fireres": 12}}},
			{"unique charm", Item{Code: "cm1", Types: []string{"scha"}, Quality: "unique", Identified: true}},
		} {
			tc.it.Place = PlaceGround
			if d := c.Decide(&tc.it, cx); d.Action != Vault {
				t.Errorf("%s %s: %+v", cls, tc.label, d)
			}
		}
		// the Starter itself still holds a big charm back while the pack has no room
		plain, _ := Compile(Starter(), cls, "")
		it := Item{Place: PlaceGround, Code: "cm3", Types: []string{"lcha"}, Quality: "magic", Identified: true, Stats: map[string]int{"life": 30}}
		if d := plain.Decide(&it, cx); d.Action == Stash {
			t.Errorf("%s: the Starter must still need charm room: %+v", cls, d)
		}
	}
}
