package gear

import "sort"

// Vendor is a town NPC that sells ammo and thrown stacks.
type Vendor struct {
	Act      int
	NPC      int    // monstats class id (internal/town Acts.Repair)
	Name     string // for logs
	Tab      string // vendor tab holding ammo
	Verified bool   // true: V live
}

// AmmoVendors lists who sells arrows, bolts and javelins. V live: Charsi (class 154, Act 1) sells Arrows and Bolts on
// the Misc tab, a full Arrows stack cost 81 gold (docs/classes/smoke-amazon.md finding 3). V data: Arrows and Bolts are
// PermStoreItem in misc.txt, javelins are StorePage "misc" (itemtypes). U: that the other act smiths (the Repair NPCs of
// internal/town/acts.go) stock the Misc tab the same way; no live check yet.
var AmmoVendors = []Vendor{
	{1, 154, "Charsi", "misc", true},
	{2, 178, "Fara", "misc", false},
	{3, 253, "Hratli", "misc", false},
	{4, 257, "Halbu", "misc", false},
	{5, 511, "Larzuk", "misc", false},
}

// VendorForAct returns the ammo vendor of an act.
func VendorForAct(act int) (Vendor, bool) {
	for _, v := range AmmoVendors {
		if v.Act == act {
			return v, true
		}
	}
	return Vendor{}, false
}

// AmmoFor returns the ammo base a launcher base shoots ("hbw" -> "aqv" Arrows, crossbows -> "cqv" Bolts).
func AmmoFor(launcher string) (Base, string, bool) {
	t := launcherShoots(TypeOf(launcher))
	switch {
	case IsA(t, "bowq"):
		b, _ := BaseByCode("aqv")
		return b, "aqv", true
	case IsA(t, "xboq"):
		b, _ := BaseByCode("cqv")
		return b, "cqv", true
	}
	return Base{}, "", false
}

// LowAmmo is the quantity at or below which the bot should restock. U policy, not game data: a quiver is "low" at 20%
// of its stack (Arrows 350 -> 70, Bolts 250 -> 50); V live consumption was about 0.7 arrow per kill (350 -> 339 after 15
// kills, smoke-amazon finding 3), so 70 arrows is roughly 100 kills of margin. A thrown stack is low at a third (Javelin
// 60 -> 20): javelins also lose durability (V: start Javelin durability 2) and recovery from the ground is untested (U).
func LowAmmo(e Equipped) (int, bool) {
	switch {
	case e.Launcher():
		b, _, ok := AmmoFor(e.Weapon)
		if !ok {
			return 0, false
		}
		return b.MaxStack / 5, true
	case e.Thrower():
		b, _ := BaseByCode(e.Weapon)
		if b.MaxStack == 0 {
			return 0, false
		}
		return b.MaxStack / 3, true
	}
	return 0, false
}

// AmmoIsLow reports whether the equipped weapon's ammo is at or below the low line.
func AmmoIsLow(e Equipped) bool {
	low, ok := LowAmmo(e)
	if !ok {
		return false
	}
	if e.Launcher() {
		return e.OffhandQty <= low
	}
	return e.WeaponQty <= low
}

// Purchase is one shopping line.
type Purchase struct {
	Code string
	Name string
	Qty  int // stack size to end up with (ammo: the full stack, vendors sell a stack at a time)
	Why  string
}

// Restock returns what to buy when the ammo is low or the quiver is missing. havePack is the count of the right ammo
// already in the pack (a spare stack); the quiver itself is e. Returns nil when nothing is needed.
func Restock(e Equipped) []Purchase {
	switch {
	case e.Launcher():
		b, code, ok := AmmoFor(e.Weapon)
		if !ok {
			return nil
		}
		wantType := b.Type
		if e.Offhand == "" || !IsA(TypeOf(e.Offhand), wantType) {
			return []Purchase{{code, b.Name, b.MaxStack, "no matching quiver"}}
		}
		if AmmoIsLow(e) {
			return []Purchase{{code, b.Name, b.MaxStack, "ammo low"}}
		}
	case e.Thrower():
		b, _ := BaseByCode(e.Weapon)
		if AmmoIsLow(e) && b.MaxStack > 0 {
			return []Purchase{{e.Weapon, b.Name, b.MaxStack, "thrown stack low"}}
		}
	}
	return nil
}

// StartPurchases is the first shopping list of a class build, in priority order (cheapest first, a bow is useless
// without arrows so both are listed together). Amazon bow path: V charstats gives only a Javelin stack and a Buckler
// (docs/classes/smoke-amazon.md finding 1), so the bot buys a Short Bow (V weapons.txt: dex 15, cost 100) and a full
// Arrows stack at Charsi, then wears the quiver in the left hand (it replaces the Buckler in slot 5, V live).
// Gold at the start is not known here (U); callers skip lines they cannot pay for.
func StartPurchases(class, path string) []Purchase {
	switch ClassCode(class) {
	case "ama":
		arrows, _ := BaseByCode("aqv")
		switch path {
		case "javelin":
			j, _ := BaseByCode("jav")
			return []Purchase{{"jav", j.Name, j.MaxStack, "spare Javelin stack for Poison Javelin"}}
		default:
			return []Purchase{
				{"sbw", "Short Bow", 1, "bow path needs a bow (start gear is a Javelin)"},
				{"aqv", arrows.Name, arrows.MaxStack, "quiver for the bow"},
			}
		}
	}
	return nil
}

// PickLauncher chooses the best bow or crossbow among candidate base codes (a vendor's stock) the character can use
// now. Ranking is base cost: in weapons.txt cost rises with damage inside the bow family (V, Short Bow 100 up to
// Long War Bow 3575), so it is a stand-in for damage that the data package does not expose (bows read dam 0-0, smoke-amazon
// finding 1). Class-restricted bases (Amazon bows) need the class and the character level.
func PickLauncher(candidates []string, class string, str, dex, clvl int) string {
	cc := ClassCode(class)
	var ok []Base
	codes := map[string]string{}
	for _, c := range candidates {
		b, found := BaseByCode(c)
		if !found || !IsA(b.Type, "miss") {
			continue
		}
		if cl := classOf(b.Type); cl != "" && cl != cc {
			continue
		}
		if b.ReqStr > str || b.ReqDex > dex || b.LevelReq > clvl {
			continue
		}
		ok = append(ok, b)
		codes[b.Name] = c
	}
	sort.Slice(ok, func(i, j int) bool { return ok[i].Cost > ok[j].Cost })
	if len(ok) == 0 {
		return ""
	}
	return codes[ok[0].Name]
}
