package combat

import (
	"context"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Aura state. VERIFIED LIVE (CLASSPAL, docs/classes/smoke-paladin.md finding 6): the player's StatList (UnitAny+0x5C)
// +0x3c points to the newest child list; a child list belongs to the player when +0x34 equals the player list, +0x2c
// links to the previous child, +0x14 is the state number (Might 33, Holy Fire 35, Thorns 36), and its stat vector
// (+0x24 pointer, +0x28 count in the low word, 8-byte entries {sub:u16, id:u16, value:i32}) holds stat 350 = the aura
// skill id. A hotkey switch shows in memory within 20 ms of the key release and costs no mana.
const (
	statAuraSkill   = 350
	slChildren      = 0x3c
	slPrev          = 0x2c
	slOwner         = 0x34
	slVec           = 0x24
	slCount         = 0x28
	uoStatList      = 0x5C
	maxAuraChildren = 12
)

// ActiveAura reads the id of the aura the player has active, 0 for none. ok is false when memory is not readable
// (no memory source, or an aura reader is not installed); callers then fall back to the selected right skill.
func (f *Fighter) ActiveAura(p *d2.Player) (id int, ok bool) {
	if f.auraFn != nil {
		return f.auraFn(p)
	}
	if f.e.Mem == nil || p == nil || p.Addr == 0 {
		return 0, false
	}
	top, err := readU32(f.e, p.Addr+uoStatList)
	if err != nil || top == 0 {
		return 0, false
	}
	child, err := readU32(f.e, top+slChildren)
	if err != nil {
		return 0, false
	}
	for n := 0; child > 0x10000 && n < maxAuraChildren; n++ {
		if owner, err := readU32(f.e, child+slOwner); err == nil && owner == top {
			vec, e1 := readU32(f.e, child+slVec)
			cnt, e2 := readU32(f.e, child+slCount)
			cnt &= 0xffff
			if e1 == nil && e2 == nil && vec > 0x10000 && cnt < 64 {
				for i := uint32(0); i < cnt && i < 32; i++ {
					w0, e3 := readU32(f.e, vec+i*8)
					if e3 != nil {
						break
					}
					if w0>>16 == statAuraSkill {
						v, _ := readU32(f.e, vec+i*8+4)
						return int(int32(v)), true
					}
				}
			}
		}
		if child, err = readU32(f.e, child+slPrev); err != nil {
			return 0, false
		}
	}
	return 0, true
}

// auraActive says whether skill id is the active aura. Without a readable aura state it trusts the selected right
// skill, but only while nothing happened that drops auras (death, area change: auraDirty).
func (f *Fighter) auraActive(p *d2.Player, id int) bool {
	if cur, ok := f.ActiveAura(p); ok {
		return cur == id
	}
	return !f.auraDirty && p.RightSkill == id
}

// wantAura chooses the aura to keep up against target tg (nil: no target): the first ladder entry that is learned,
// reachable (hotkey bound or already selected) and whose element the target is not immune to.
//
// Resting auras (RestLifePct, Prayer) are skipped in a fight and while resting at healthy life; with resting true and
// life below their percent they come first. In a fight they are tried last, so Prayer still beats no aura.
func (f *Fighter) wantAura(p *d2.Player, pr *Profile, tg *Target, resting bool) (SkillSpec, bool) {
	known := KnownSkills(p, f.cat, f.keys)
	has := func(n string) bool { _, ok := known[n]; return ok }
	ladder := pr.AuraLadder(p.Level, has)
	reachable := func(a AuraPref) (SkillSpec, bool) {
		sp, ok := known[a.Name]
		return sp, ok && (sp.Key != "" || p.RightSkill == sp.ID)
	}
	for _, a := range ladder {
		sp, ok := reachable(a)
		if !ok {
			continue
		}
		if a.RestLifePct > 0 {
			if resting && pct(p.Life, p.MaxLife) < a.RestLifePct {
				return sp, true
			}
			continue
		}
		if tg != nil && a.Elem != "" && tg.Immune(elemOf(a.Elem)) {
			continue
		}
		return sp, true
	}
	if !resting {
		for _, a := range ladder {
			if sp, ok := reachable(a); ok && a.RestLifePct > 0 {
				return sp, true
			}
		}
	}
	return SkillSpec{}, false
}

// KeepAura makes sure the profile's aura is active, re-applying it after a death, respawn or area change. force skips
// the check interval. It returns whether an aura was (re)applied.
func (f *Fighter) KeepAura(ctx context.Context, snap *d2.Snapshot, tg *Target, force bool) (bool, error) {
	return f.keepAura(ctx, snap, tg, force, false)
}

// keepAura is KeepAura with the resting flag of wantAura.
func (f *Fighter) keepAura(ctx context.Context, snap *d2.Snapshot, tg *Target, force, resting bool) (bool, error) {
	p := snap.Player
	pr := f.profileFor(p)
	if len(pr.Aura) == 0 && !hasBandAura(pr) {
		return false, nil
	}
	now := f.now()
	if p.Area != f.auraArea {
		f.auraDirty, f.auraArea = true, p.Area
	}
	if !force && !f.auraDirty && now.Sub(f.auraChecked) < ms(f.tune().AuraCheckMs) {
		return false, nil
	}
	f.auraChecked = now
	want, ok := f.wantAura(p, pr, tg, resting)
	if !ok {
		return false, nil
	}
	if f.auraActive(p, want.ID) {
		f.auraDirty = false
		return false, nil
	}
	// A hotkey press changes the aura at once when one is active; with none active the right click casts it.
	for try := 0; try < 3; try++ {
		if err := f.cast(snap, want, input.Point{X: p.X, Y: p.Y}); err != nil {
			return false, err
		}
		if err := f.sleep(ctx, 250*time.Millisecond); err != nil {
			return false, err
		}
		s, err := f.read(ctx)
		if err != nil {
			return false, err
		}
		if cur, ok := f.ActiveAura(s.Player); (ok && cur == want.ID) || (!ok && s.Player.RightSkill == want.ID) {
			f.auraDirty = false
			f.info("aura up", "skill", want.Name, "tries", try+1)
			return true, nil
		}
	}
	f.info("aura did not come up", "skill", want.Name)
	f.auraChecked = now.Add(10 * time.Second) // do not hammer the key when something blocks it
	return false, nil
}

func hasBandAura(p *Profile) bool {
	for _, b := range p.Bands {
		if len(b.Aura) > 0 {
			return true
		}
	}
	return false
}
