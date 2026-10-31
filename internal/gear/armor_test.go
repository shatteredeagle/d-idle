package gear

import (
	"encoding/json"
	"testing"
)

// V: values read from armor.txt on the install (Patch_D2 > d2exp > d2data) with d2tables.
func TestArmorRows(t *testing.T) {
	needTables(t)
	cases := []struct {
		code      string
		name      string
		str, lvl  int
		sockets   int
		tier      int
		minac, ac int
	}{
		{"xtp", "Mage Plate", 55, 25, 3, 1, 225, 261},
		{"uui", "Dusk Shroud", 77, 49, 4, 2, 361, 467},
		{"utp", "Archon Plate", 103, 63, 4, 2, 410, 524},
		{"uar", "Sacred Armor", 232, 66, 4, 2, 487, 600},
		{"pab", "Sacred Targe", 86, 47, 4, 2, 126, 158},
		{"uit", "Monarch", 156, 54, 4, 2, 133, 148},
	}
	for _, c := range cases {
		a, ok := ArmorByCode(c.code)
		if !ok || a.Name != c.name || a.ReqStr != c.str || a.LevelReq != c.lvl || a.MaxSockets != c.sockets || a.Tier != c.tier || a.MinAC != c.minac || a.MaxAC != c.ac {
			t.Errorf("%s: %+v", c.code, a)
		}
	}
	if n := len(ArmorCodes()); n < 150 {
		t.Errorf("only %d armor rows", n)
	}
	for _, c := range ArmorCodes() {
		a, _ := ArmorByCode(c)
		if a.Type == "" || a.MaxAC < a.MinAC || a.Durability == 0 && !a.NoDurability && a.Type != "circ" && a.MaxAC > 0 && a.Durability <= 0 {
			t.Errorf("%s malformed: %+v", c, a)
		}
	}
}

func TestCanEquip(t *testing.T) {
	needTables(t)
	cases := []struct {
		code          string
		str, dex, lvl int
		ok            bool
	}{
		{"xtp", 54, 0, 30, false},
		{"xtp", 55, 0, 30, true},
		{"xtp", 55, 0, 24, false}, // level 25
		{"uar", 231, 99, 90, false},
		{"pab", 86, 0, 47, true},
		{"9ar", 57, 56, 30, false}, // claws need Dex as well (V weapons.txt)
		{"9ar", 57, 57, 30, true},
		{"nosuch", 999, 999, 99, false},
	}
	for _, c := range cases {
		if ok, why := CanEquip(c.code, c.str, c.dex, c.lvl); ok != c.ok {
			t.Errorf("%s str %d dex %d lvl %d: got %v %q", c.code, c.str, c.dex, c.lvl, ok, why)
		}
	}
}

// Every milestone item is a known base and the endgame build rows give the character enough Str and Dex by its
// deadline (rows hold per-level point gains, the class start comes from charstats).
func TestMilestoneStats(t *testing.T) {
	needTables(t)
	for _, name := range Builds() {
		ms := Milestones(name)
		if ms == nil {
			continue
		}
		raw, err := BuildJSON(name)
		if err != nil {
			t.Fatal(err)
		}
		var b struct {
			Class string `json:"class"`
			Rows  []struct {
				Clvl int `json:"clvl"`
				Str  int `json:"strength"`
				Dex  int `json:"dexterity"`
			} `json:"rows"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			t.Fatal(err)
		}
		cl, ok := ClassStart(b.Class)
		if !ok {
			t.Fatalf("%s: no class %s", name, b.Class)
		}
		for _, m := range ms {
			for _, c := range m.Items {
				if _, _, _, ok := Requirement(c); !ok {
					t.Errorf("%s: unknown milestone item %s", name, c)
				}
			}
			needS, needD := StatNeeds(name, m.Clvl)
			s, d := cl.Str, cl.Dex
			for _, r := range b.Rows {
				if r.Clvl <= m.Clvl {
					s, d = s+r.Str, d+r.Dex
				}
			}
			if s < needS || d < needD {
				t.Errorf("%s clvl %d: has str %d dex %d, %v needs %d/%d", name, m.Clvl, s, d, m.Items, needS, needD)
			}
		}
	}
}
