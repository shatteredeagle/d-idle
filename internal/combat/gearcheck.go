package combat

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear"
)

const statQuantity = 70 // V live: stack count of a quiver or thrown stack (docs/classes/smoke-amazon.md finding 3)

// EquippedGear reads the main-hand and left-hand base codes and quantities (the input of internal/gear). ok is false
// when the item table or the inventory is unavailable, so callers never refuse an attack on missing data.
func (f *Fighter) EquippedGear(snap *d2.Snapshot) (gear.Equipped, bool) {
	var e gear.Equipped
	if snap == nil || snap.Inventory == nil || f.e == nil || f.e.Data == nil {
		return e, false
	}
	one := func(body int) (string, int) {
		u := snap.Inventory.Equipped(body)
		if u == nil || u.Class < 0 || u.Class >= len(f.e.Data.Items) || f.e.Data.Items[u.Class] == nil {
			return "", 0
		}
		code := f.e.Data.Items[u.Class].Code
		if _, ok := gear.BaseByCode(code); !ok {
			return code, 0
		}
		var st d2.Stats
		var err error
		switch {
		case f.statsFn != nil:
			st, err = f.statsFn(u.Addr)
		case f.e.Read != nil:
			st, err = f.e.Read.UnitStats(u.Addr)
		default:
			return code, 0
		}
		if err != nil || !st.Has(statQuantity) {
			return code, 0
		}
		return code, int(st.Get(statQuantity))
	}
	e.Weapon, e.WeaponQty = one(d2.BodyRHand)
	e.Offhand, e.OffhandQty = one(d2.BodyLHand)
	return e, true
}

// noteGear refreshes the gear view used by attackSkills and logs a weapon problem once per distinct reason.
func (f *Fighter) noteGear(snap *d2.Snapshot) {
	e, ok := f.EquippedGear(snap)
	if !ok {
		f.eq = nil
		return
	}
	f.eq = &e
	why := gear.WeaponProblem(snap.Player.Class.String(), e)
	f.weaponProblem = why
	if why != "" && why != f.weaponLogged {
		f.info("weapon problem, not attacking", "why", why, "weapon", e.Weapon, "offhand", e.Offhand, "ammo", e.OffhandQty)
	}
	f.weaponLogged = why
}

// WeaponProblem is the reason the last noted gear cannot attack ("" when fine or unknown). The bot turns it into a
// town trip (restock or equip).
func (f *Fighter) WeaponProblem() string { return f.weaponProblem }

// AmmoLow reports that the last noted quiver or thrown stack is at or below gear.LowAmmo.
func (f *Fighter) AmmoLow() bool { return f.eq != nil && gear.AmmoIsLow(*f.eq) }

// dualWield are skills the bot only casts with a melee weapon in each hand (U: skills.txt itypea says only "mele", the
// pair requirement is the bot's rule from docs/classes/barbarian.md, so Frenzy is not wasted on a lone weapon or a shield).
var dualWield = map[string]bool{"Frenzy": true}

func dualWieldOK(skill string, e gear.Equipped) bool {
	if !dualWield[skill] {
		return true
	}
	return e.Offhand != "" && gear.IsA(gear.TypeOf(e.Offhand), "mele")
}

// gearFilter drops skills the equipped weapon cannot use and promotes the build's main attack. With a weapon problem
// or unknown gear: problem returns nil, unknown leaves the list unchanged.
func (f *Fighter) gearFilter(p *d2.Player, in []SkillSpec) []SkillSpec {
	if f.eq == nil {
		return in
	}
	if f.weaponProblem != "" {
		return nil
	}
	out := in[:0:0]
	for _, sp := range in {
		if ok, _ := gear.SkillUsable(sp.Name, *f.eq); ok && dualWieldOK(sp.Name, *f.eq) {
			out = append(out, sp)
		}
	}
	if p.Class == d2.Amazon && f.GearBuild != "" {
		if m, err := gear.MainAttackAt(f.GearBuild, p.Level); err == nil && m.Skill != "" {
			for i, sp := range out {
				if sp.Name == m.Skill && i > 0 {
					out = append([]SkillSpec{sp}, append(out[:i:i], out[i+1:]...)...)
					break
				}
			}
		}
	}
	return out
}
