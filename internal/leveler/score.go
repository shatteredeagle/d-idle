package leveler

import (
	"d2lod-bot/internal/gear"
	"fmt"
	"sort"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/town/hand"
)

// Item scoring. One number per item, comparable across items that fit the same slot. The rules are deliberately
// simple (docs/leveler.md "Gear scoring"); every weight is a named constant so tuning is one place.
//
//	armor     base defence (average of minac..maxac), superior +15%, ethereal x gear.EtherealFactor(), plus
//	          the item's own stat list when identified (resists, life, mana, skills, fcr...), else a flat expectation by
//	          quality for the hidden affixes (magic +8, rare +20, set +25, unique +30, crafted +18)
//	weapons   average damage x the class weight (prefs.go; casters 0.4, two-handed damage for two-handers),
//	          plus the class type bonus (casters: staves/wands/orbs +8) and the stat list as above
//	jewelry   no base value: stat list only, else the quality expectation
//
// A point of defence is worth 1; the other weights convert to defence-equivalents.
const (
	wSuperior   = 1.15
	wCasterBase = 8.0
	wWeaponDmg  = 0.4
	wAllSkills  = 40.0
	wClassSkill = 30.0
	wTabSkill   = 25.0
	wRes        = 1.2 // per useful point of one resistance (gear.ResistValue)
	wLife       = 0.6
	wMana       = 0.3
	wFCR        = 1.5  // residual per point, the breakpoints carry the value
	wFCRBreak   = 25.0 // per faster-cast breakpoint crossed (U: tuning guess)
	wRuneUnread = 40.0 // finished runeword whose name and stats could not be read (U)
	wFRW        = 0.4
	wStat       = 0.4 // str/dex, 1.0 vit, 0.5 ene below
	wMF         = 0.15
	wDefence    = 1.0
)

// Hidden-affix expectation for items whose stat list is not identified.
var unknownBonus = map[int]float64{d2.QualityMagic: 8, d2.QualityRare: 20, d2.QualitySet: 25, d2.QualityUnique: 30, d2.QualityCrafted: 18}

// Stat ids used by the scoring (standard ItemStatCost ids; 0-3, 7-9 VERIFIED LIVE as present in player lists, the rest UNVERIFIED on items).
const (
	stStr, stEne, stDex, stVit = 0, 1, 2, 3
	stMaxLife, stMaxMana       = 7, 9
	stDefence                  = 31
	stFireRes, stLightRes      = 39, 41
	stColdRes, stPoisRes       = 43, 45
	stMF                       = 80
	stFRW                      = 96
	stFCR                      = 105
	stAllSkills                = 127
	stClassSkills              = 83
	stTabSkills                = 188
)

// Body locations of the equipment slots (d2.Body*), with the character panel slot names of inventory.txt.
var slotName = hand.SlotNames

// Cand is one item considered for a slot.
type Cand struct {
	Unit      d2.Unit
	Name      string
	Score     float64
	Slots     []int // body locations it can go to
	TwoHanded bool
	// AvoidTwoHanded: the class fights with a shield, ChooseGear plans no two-hander while a shield is worn or available.
	AvoidTwoHanded bool
	// Ranged marks a bow, crossbow or quiver. Town owns those (gear.StartPurchases); a worn one is never swapped away
	// here, so no shield goes over a quiver and no melee weapon over a bow (GEARWIRE, U live).
	Ranged bool
	Usable bool
	Why    string // why not usable
	// OffClass: a weapon type the class's preference table gives no bonus (a Spear for a caster). It may still be
	// equipped when found, but the shop never spends gold on it.
	OffClass bool
	Price    int // vendor items: base cost x quality factor (rough), 0 for owned items
}

// slotsFor maps a base item to the body locations it can be worn at; nil for non-equipment.
func slotsFor(db *data.DB, b *data.ItemBase) []int {
	switch {
	case b.Kind == data.KindMisc:
		switch {
		case db.IsA(b.Type, "ring"):
			return []int{d2.BodyRRing, d2.BodyLRing}
		case db.IsA(b.Type, "amul"):
			return []int{d2.BodyNeck}
		}
		return nil
	case db.IsA(b.Type, "tors"):
		return []int{d2.BodyTorso}
	case db.IsA(b.Type, "helm"), db.IsA(b.Type, "circ"):
		return []int{d2.BodyHead}
	case db.IsA(b.Type, "glov"):
		return []int{d2.BodyGloves}
	case db.IsA(b.Type, "boot"):
		return []int{d2.BodyFeet}
	case db.IsA(b.Type, "belt"):
		return []int{d2.BodyBelt}
	case db.IsA(b.Type, "shld"):
		return []int{d2.BodyLHand}
	case b.Kind == data.KindWeapon && db.IsA(b.Type, "weap"):
		if db.IsA(b.Type, "miss") { // bows, crossbows, javelins: a Sorceress has no use for them
			return nil
		}
		return []int{d2.BodyRHand}
	}
	return nil
}

// classOf returns the three-letter class code restriction of an item type or any ancestor, "" when unrestricted.
func classOf(db *data.DB, typ string, seen map[string]bool) string {
	if typ == "" || seen[typ] {
		return ""
	}
	seen[typ] = true
	t := db.ItemType(typ)
	if t == nil {
		return ""
	}
	if t.Class != "" {
		return t.Class
	}
	for _, e := range t.Equiv {
		if c := classOf(db, e, seen); c != "" {
			return c
		}
	}
	return ""
}

// Candidate rates one item unit for player p. st is the item's stat list (nil when unreadable or unidentified).
func Candidate(db *data.DB, p *d2.Player, u d2.Unit, st d2.Stats) Cand {
	c := Cand{Unit: u}
	if u.Item == nil || db == nil || u.Class < 0 || u.Class >= len(db.Items) {
		c.Why = "unknown item"
		return c
	}
	b := db.Items[u.Class]
	c.Name = b.Name
	if b.Display != "" {
		c.Name = b.Display
	}
	c.Ranged = gear.Equipped{Weapon: b.Code}.Launcher() || gear.IsA(b.Type, "bowq") || gear.IsA(b.Type, "xboq")
	c.Slots = slotsFor(db, b)
	if c.Slots == nil {
		c.Why = "not equipment for this class"
		return c
	}
	if cl := db.ClassByName(p.Class.String()); cl != nil {
		if rc := classOf(db, b.Type, map[string]bool{}); rc != "" && rc != cl.Code {
			c.Why = "restricted to class " + rc
			return c
		}
	}
	if b.Kind == data.KindWeapon && b.Stackable {
		// Javelins and throwing knives are ammo: Town buys them for the classes that throw (gear.StartPurchases). A
		// vendor stack costs base cost x quantity, so the estimate was 5 for a 240 gold Javelin stack (soak WP2, V log).
		c.Why = "throwing weapon (town owns ammo)"
		return c
	}
	c.TwoHanded = b.TwoHanded && b.Kind == data.KindWeapon
	pref := PrefsFor(p.Class.String())
	c.AvoidTwoHanded = c.TwoHanded && pref.WantsShield
	it := u.Item
	switch {
	case b.Kind == data.KindWeapon && it.Quality == d2.QualityLow:
		c.Why = "low quality"
	case it.Quality >= d2.QualityMagic && !worn(it) && p.Level < it.ILvl:
		// A magic, rare, set or unique item's required level is raised by its affixes and the game gives no way to read it
		// from memory. The item level is an upper bound, so the item counts as unusable below it (VERIFIED LIVE: a magic Cap
		// and a magic Wand, ilvl 7, were refused at level 2 although their bases need level 0 and 1; docs/leveler.md).
		c.Why = fmt.Sprintf("magic item level %d above character level", it.ILvl)
	case p.Level < b.LevelReq:
		c.Why = fmt.Sprintf("needs level %d", b.LevelReq)
	case p.Strength < b.ReqStr:
		c.Why = fmt.Sprintf("needs %d strength", b.ReqStr)
	case p.Dexterity < b.ReqDex:
		c.Why = fmt.Sprintf("needs %d dexterity", b.ReqDex)
	case it.Flags&d2.FlagIdentified != 0 && st != nil && st.Get(stStr) < 0 && p.Strength+int(st.Get(stStr)) < 0:
		c.Why = "negative stats"
	}
	c.Usable = c.Why == ""
	if b.Kind == data.KindWeapon && len(pref.TypeBonus) > 0 {
		c.OffClass = true
		for typ := range pref.TypeBonus {
			if db.IsA(b.Type, typ) {
				c.OffClass = false
			}
		}
	}
	c.Score = scoreOf(db, pref, b, it, st, ctxFor(p, it, st))
	c.Price = estimatePrice(b, it)
	return c
}

func worn(it *d2.Item) bool { return it.Where == d2.ContEquipment }

func scoreOf(db *data.DB, pref Prefs, b *data.ItemBase, it *d2.Item, st d2.Stats, ctx scoreCtx) float64 {
	var s float64
	switch b.Kind {
	case data.KindArmor:
		s = float64(b.MinAC+b.MaxAC) / 2
		if it.Quality == d2.QualitySuperior {
			s *= wSuperior
		}
		if it.Ethereal() {
			s *= gear.EtherealFactor()
		}
	case data.KindWeapon:
		avg := float64(b.MinDam+b.MaxDam) / 2
		if b.TwoHanded {
			avg = float64(b.TwoMinDam+b.TwoMaxDam) / 2
		}
		s = pref.DmgWeight * avg
		if b.TwoHanded && pref.TwoHandMult > 0 {
			s *= pref.TwoHandMult
		}
		for typ, bonus := range pref.TypeBonus {
			if db.IsA(b.Type, typ) {
				s += bonus
				break
			}
		}
	}
	if it.Identified() && st != nil {
		if d, ok := st[uint32(stDefence)]; ok && b.Kind == data.KindArmor {
			s = float64(d) * wDefence // the listed defence already includes enhancement
		}
		s += statValueCtx(st, ctx)
	} else {
		s += unknownBonus[it.Quality]
	}
	if it.Flags&d2.FlagRuneword != 0 && !(it.Identified() && st != nil) {
		// Identified runewords score by their stat list; the flat score covers the unreadable case.
		if n := runewordName(it); n != "" {
			s += gear.RunewordScore(n)
		} else {
			s += wRuneUnread
		}
	}
	return s
}

// runewordName reads a finished runeword's name from the item. UNVERIFIED that it is readable from memory (docs/items.md
// section 5); a hook for whoever finds the source.
var runewordName = func(*d2.Item) string { return "" }

// scoreCtx is what the wearer already has, so a resist or cast-rate point is worth what it adds on top of the rest.
type scoreCtx struct {
	res    map[int]int // resist stat id -> raw total of all worn gear
	fcr    int
	diff   int
	class  string
	hasCtx bool
}

// ctxFor builds the context from the player. The player's stats include worn gear, so a worn item's own share is
// taken out: its value is then what it adds, comparable with a pack item that would replace it.
func ctxFor(p *d2.Player, it *d2.Item, st d2.Stats) scoreCtx {
	if p == nil {
		return scoreCtx{}
	}
	c := scoreCtx{
		res: map[int]int{stFireRes: p.Resist.Fire, stColdRes: p.Resist.Cold, stLightRes: p.Resist.Lightning, stPoisRes: p.Resist.Poison},
		fcr: int(p.Stats.Get(stFCR)), diff: int(p.Difficulty), class: p.Class.String(), hasCtx: true,
	}
	if worn(it) && st != nil {
		for id := range c.res {
			c.res[id] -= int(st.Get(id))
		}
		c.fcr -= int(st.Get(stFCR))
	}
	return c
}

// statValue converts an item stat list into defence-equivalents with no worn gear (see statValueCtx).
func statValue(st d2.Stats) float64 { return statValueCtx(st, scoreCtx{}) }

// statValueCtx converts an item stat list into defence-equivalents. Class and tab skill stats carry the skill's class or
// tab in their sub index; every class and tab counts the same here (the bot is a single-class bot, a foreign
// class's skills never show on gear the Sorceress can use). Resists and faster cast rate depend on ctx.
func statValueCtx(st d2.Stats, ctx scoreCtx) float64 {
	var v float64
	for key, val := range st {
		id := int(key & 0xffff)
		f := float64(val)
		switch id {
		case stAllSkills:
			v += f * wAllSkills
		case stClassSkills:
			v += f * wClassSkill
		case stTabSkills:
			v += f * wTabSkill
		case stFireRes, stLightRes, stColdRes, stPoisRes:
			v += gear.ResistValue(ctx.res[id], int(val), ctx.diff, wRes)
		case stMaxLife:
			v += f / 256 * wLife
		case stMaxMana:
			v += f / 256 * wMana
		case stFCR:
			if ctx.hasCtx && gear.FCRBreakpoints(ctx.class) != nil {
				v += gear.FCRValue(ctx.class, ctx.fcr, int(val), wFCRBreak, wFCR)
			} else {
				v += f * wFCR
			}
		case stFRW:
			v += f * wFRW
		case stStr, stDex:
			v += f * wStat
		case stVit:
			v += f * 1.0
		case stEne:
			v += f * 0.5
		case stMF:
			v += f * wMF
		}
	}
	return v
}

// estimatePrice is a rough vendor price: base cost times a quality factor (superior x1.5, magic x2.5, rare x5).
// A stackable item is priced per unit, so it is taken at a full stack (an upper bound).
func estimatePrice(b *data.ItemBase, it *d2.Item) int {
	f := 1.0
	switch it.Quality {
	case d2.QualitySuperior:
		f = 1.5
	case d2.QualityMagic:
		f = 2.5
	case d2.QualityRare, d2.QualityCrafted:
		f = 5
	case d2.QualitySet, d2.QualityUnique:
		f = 8
	}
	if b.Stackable && b.MaxStack > 1 {
		f *= float64(b.MaxStack)
	}
	return int(float64(b.Cost) * f)
}

// Swap is one equip decision: put Item into body location Slot.
type Swap struct {
	Item Cand
	Slot int
	Gain float64
	Old  *Cand // what the slot holds now, nil when empty
}

func (s Swap) String() string {
	old := "empty"
	if s.Old != nil {
		old = fmt.Sprintf("%s %.1f", s.Old.Name, s.Old.Score)
	}
	return fmt.Sprintf("%s %.1f -> %s (was %s)", s.Item.Name, s.Item.Score, slotName[s.Slot], old)
}

// Margin: a replacement must beat the worn item by this factor plus this flat amount, so ties and noise never
// cause equip churn.
const (
	marginFactor = 1.10
	marginFlat   = 1.0
)

// ChooseGear picks the equip swaps. eq holds what is worn by body location, pack the candidates (pack items and,
// for shopping, vendor stock). It never plans a two-handed weapon over a worn shield, and never a shield over a
// two-handed weapon, so no swap needs a second unequip. Swaps come out in slot order, one per slot (rings: up to two).
func ChooseGear(eq map[int]Cand, pack []Cand) []Swap {
	var out []Swap
	better := func(n Cand, old *Cand) (float64, bool) {
		if !n.Usable {
			return 0, false
		}
		cur := 0.0
		if old != nil {
			cur = old.Score
		}
		return n.Score - cur, n.Score >= cur*marginFactor+marginFlat
	}
	bySlot := map[int][]Cand{}
	for _, c := range pack {
		if !c.Usable {
			continue
		}
		for _, s := range c.Slots {
			if s == d2.BodyRRing || s == d2.BodyLRing {
				s = d2.BodyRRing // rings are handled together below
			}
			bySlot[s] = append(bySlot[s], c)
			break
		}
	}
	for _, list := range bySlot {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Score > list[j].Score })
	}
	for _, slot := range []int{d2.BodyHead, d2.BodyNeck, d2.BodyTorso, d2.BodyGloves, d2.BodyFeet, d2.BodyBelt} {
		if len(bySlot[slot]) == 0 {
			continue
		}
		var old *Cand
		if c, ok := eq[slot]; ok {
			old = &c
		}
		if g, ok := better(bySlot[slot][0], old); ok {
			out = append(out, Swap{Item: bySlot[slot][0], Slot: slot, Gain: g, Old: old})
		}
	}
	// weapon and shield
	wOld, hasW := eq[d2.BodyRHand]
	sOld, hasS := eq[d2.BodyLHand]
	var curW, curS *Cand
	if hasW {
		curW = &wOld
	}
	if hasS {
		curS = &sOld
	}
	if hasW && wOld.Ranged || hasS && sOld.Ranged {
		bySlot[d2.BodyRHand], bySlot[d2.BodyLHand] = nil, nil
	}
	var bw *Cand
	for i := range bySlot[d2.BodyRHand] {
		c := bySlot[d2.BodyRHand][i]
		if c.TwoHanded && (hasS || (c.AvoidTwoHanded && len(bySlot[d2.BodyLHand]) > 0)) { // would need the shield off first, or a shield build
			continue
		}
		bw = &c
		break
	}
	if bw != nil {
		if g, ok := better(*bw, curW); ok {
			out = append(out, Swap{Item: *bw, Slot: d2.BodyRHand, Gain: g, Old: curW})
			curW = bw
		}
	}
	twoH := curW != nil && curW.TwoHanded
	if !twoH && len(bySlot[d2.BodyLHand]) > 0 {
		if g, ok := better(bySlot[d2.BodyLHand][0], curS); ok {
			out = append(out, Swap{Item: bySlot[d2.BodyLHand][0], Slot: d2.BodyLHand, Gain: g, Old: curS})
		}
	}
	// rings: the best two of worn and pack rings
	if len(bySlot[d2.BodyRRing]) > 0 {
		worn := []*Cand{}
		for _, s := range []int{d2.BodyRRing, d2.BodyLRing} {
			if c, ok := eq[s]; ok {
				cc := c
				worn = append(worn, &cc)
			} else {
				worn = append(worn, nil)
			}
		}
		slots := []int{d2.BodyRRing, d2.BodyLRing}
		for _, cand := range bySlot[d2.BodyRRing] {
			// replace an empty slot, else the weaker worn ring
			idx := -1
			for i, w := range worn {
				if w == nil {
					idx = i
					break
				}
			}
			if idx < 0 {
				idx = 0
				if worn[1].Score < worn[0].Score {
					idx = 1
				}
			}
			if g, ok := better(cand, worn[idx]); ok {
				c := cand
				out = append(out, Swap{Item: cand, Slot: slots[idx], Gain: g, Old: worn[idx]})
				worn[idx] = &c
			}
		}
	}
	return out
}
