package loot

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Item stat ids (stat list, sub 0). U: ids from the public stat table; armorclass includes the base defense for armor
// whose stat list has it, otherwise the base is read from armor.txt.
const (
	statArmorClass = 31
	statAllSkills  = 127
	statPoisonRes  = 45
)

// WearScore ranks two wearable items for the same slot in the early game (docs/sme/early-gear-policy.md): defense plus
// 10 per +all skills plus the point sum of resists plus max life. The weights are a judgement (U): at clvl 1-15 the
// only question is "is it better than what I wear", and an item with +skills or resists or life is, defense aside.
func WearScore(b *data.ItemBase, st d2.Stats) int {
	if b == nil {
		return 0
	}
	def := (b.MinAC + b.MaxAC) / 2
	if st.Has(statArmorClass) {
		def = max(def, int(st.Get(statArmorClass)))
	}
	res := 0
	for _, id := range []int{d2.StatFireRes, d2.StatColdRes, d2.StatLightRes, statPoisonRes} {
		res += int(st.Get(id))
	}
	return def + 10*int(st.Get(statAllSkills)) + res + st.Fixed(d2.StatMaxLife)
}

// wearSlots maps the body locations of the early-gear slots to the itemtypes bodyloc code.
var wearSlots = []struct {
	loc  string
	body int
}{
	{"tors", d2.BodyTorso}, {"head", d2.BodyHead}, {"feet", d2.BodyFeet}, {"glov", d2.BodyGloves},
	{"belt", d2.BodyBelt}, {"larm", d2.BodyLHand},
}

// WearSlot is the armor body slot base b fills (a shield for larm), ok false for anything else.
func WearSlot(db *data.DB, b *data.ItemBase) (body int, ok bool) {
	if b == nil || db == nil || b.Kind != data.KindArmor {
		return 0, false
	}
	ty := db.ItemType(b.Type)
	if ty == nil {
		return 0, false
	}
	for _, w := range wearSlots {
		if ty.BodyLocs[0] == w.loc || ty.BodyLocs[1] == w.loc {
			return w.body, true
		}
	}
	return 0, false
}
