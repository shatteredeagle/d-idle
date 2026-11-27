package combat

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
)

// Upkeep driver, state-observed (card dlv-upkeep-state; combat plan 3.2 and Q7). It decides which buff of the
// rotation's upkeep groups is due from the game tables, never from a skill name:
//   - the state the skill switches on is its Skills.txt aurastate, resolved to a states.txt id by internal/data
//     (data.Skill.StateID). Present on the player = up. The id is trusted only once it has been seen present after a
//     cast ("learned"); until then the recast clock decides, so a wrong id cannot spam casts.
//   - no state: the clock (buffLimit: the skill's real duration minus the margin, or combat.buffRecastSec).
//   - a skill whose own or target state is a curse is a mark, never a self upkeep (it replaces the profile's curse
//     list in the buff pass).
//
// Inside a fight a slot not named vital by the profile casts at most once in UpkeepParams.ShareEvery casts of the
// fight and only while the mana left after it stays above ManaReservePct of the pool. Knob combat.deliveryUpkeep:
// 0 the old pass (buffDue), 1 shadow (the old pass acts, disagreements are logged and counted), 2 the driver acts.
//
// Auras (card auraupkeep) run through the same knob in keepAura (aura.go). Which aura stays the profile's ladder
// (wantAura); the driver only decides whether that aura is up and re-picks it at once when it went down between
// checks, instead of waiting for combat.auraCheckMs. Auras ignore the share and the reserve: a pick costs no mana.

// UpkeepParams is the profile's in-fight upkeep budget. Zero values take the defaults of Q7 (user-approved
// 2026-11-21): one cast in four, a 10 percent mana reserve.
type UpkeepParams struct {
	// Vital names upkeep slots whose loss is dangerous: they ignore the share and the reserve.
	Vital          []string `json:"vital,omitempty"`
	ShareEvery     int      `json:"shareEvery,omitempty"`
	ManaReservePct int      `json:"manaReservePct,omitempty"`
}

const (
	upkeepShareDefault   = 4
	upkeepReserveDefault = 10
	upkeepGrace          = 4 * time.Second // a learned state may lag the cast by this much before it counts as gone
)

// upkeepParams returns the profile's budget with the defaults filled in.
func (p *Profile) upkeepParams() UpkeepParams {
	var u UpkeepParams
	if p != nil && p.Upkeep != nil {
		u = *p.Upkeep
	}
	if u.ShareEvery <= 0 {
		u.ShareEvery = upkeepShareDefault
	}
	if u.ManaReservePct <= 0 {
		u.ManaReservePct = upkeepReserveDefault
	}
	return u
}

func (u UpkeepParams) vital(name string) bool {
	for _, v := range u.Vital {
		if v == name {
			return true
		}
	}
	return false
}

// upkeepState is the driver's memory: learned state ids, the fight's cast counts and the shadow disagreements.
type upkeepState struct {
	learned      map[int]bool
	fightCasts   int // every cast since the fight began
	fightUpkeeps int // upkeep casts since the fight began
	inFight      bool
	shadowDiffs  int
	shadowLast   map[int]string    // per skill: the last disagreement logged, so a lasting one is logged once
	minionLast   map[string]string // the minion driver's last shadow disagreement per call site (minions.go)
	aura         SkillSpec         // the aura keepAura chose last (the profile's choice, never the driver's)
	auraAt       time.Time         // when that aura was last confirmed up after an apply
}

func (u *upkeepState) fightStart() { u.fightCasts, u.fightUpkeeps, u.inFight = 0, 0, true }
func (u *upkeepState) fightEnd()   { u.inFight = false }

// noteCast counts a cast of any kind for the in-fight share.
func (u *upkeepState) noteCast() {
	if u.inFight {
		u.fightCasts++
	}
}

func (f *Fighter) upkeepMode() int { return f.tune().DeliveryUpkeep }

// upkeepStateID is the states.txt id the skill switches on (0: none known, the clock decides).
func (f *Fighter) upkeepStateID(sp SkillSpec) int {
	if f.cat == nil {
		return 0
	}
	if s := f.cat.Skill(sp.ID); s != nil {
		return s.StateID
	}
	return 0
}

// upkeepIsMark says the skill puts a curse (on itself or its target): the mark driver's, not an upkeep.
func (f *Fighter) upkeepIsMark(sp SkillSpec) bool {
	if f.cat == nil {
		return false
	}
	s := f.cat.Skill(sp.ID)
	return s != nil && s.Curse
}

// upkeepDue is the driver's answer to "does this upkeep slot need a cast now".
func (f *Fighter) upkeepDue(p *d2.Player, sp SkillSpec) bool {
	if f.upk.learned == nil {
		f.upk.learned = map[int]bool{}
	}
	at, seen := f.buffAt[sp.ID]
	limit, timed := f.buffLimit(sp)
	if st := f.upkeepStateID(sp); st > 0 {
		if p.States.Has(st) {
			f.upk.learned[st] = true
			return timed && seen && f.now().Sub(at) >= limit
		}
		if f.upk.learned[st] {
			return !seen || f.now().Sub(at) >= upkeepGrace
		}
	}
	return !seen || f.now().Sub(at) >= limit
}

// upkeepBudget says whether an in-fight upkeep cast fits the share and the mana reserve. mana is the pool left
// before this cast. Outside a fight (the need-buff edges, the pass after a clear) everything fits.
func (f *Fighter) upkeepBudget(p *d2.Player, prof *Profile, sp SkillSpec, mana int) bool {
	if !f.upk.inFight {
		return true
	}
	u := prof.upkeepParams()
	if u.vital(sp.Name) {
		return true
	}
	if f.upk.fightUpkeeps > 0 && f.upk.fightUpkeeps*u.ShareEvery > f.upk.fightCasts {
		return false
	}
	return mana-sp.Cost >= p.MaxMana*u.ManaReservePct/100
}

// upkeepShadow logs one disagreement of mode 1 (what the driver would do against what the old pass does).
func (f *Fighter) upkeepShadow(sp SkillSpec, what string, old, driver bool) {
	key := fmt.Sprint(what, old, driver)
	if f.upk.shadowLast == nil {
		f.upk.shadowLast = map[int]string{}
	}
	if f.upk.shadowLast[sp.ID] == key {
		return
	}
	f.upk.shadowLast[sp.ID] = key
	f.upk.shadowDiffs++
	f.info("upkeep shadow", "skill", sp.Name, "what", what, "old", old, "driver", driver)
}

// UpkeepShadowDiffs is how many distinct disagreements mode 1 logged (a lasting one counts once); it is for telemetry and tests.
func (f *Fighter) UpkeepShadowDiffs() int { return f.upk.shadowDiffs }

// upkeepAfterFight is the pass after a clear (mode 2): the driver with no share limit.
func (f *Fighter) upkeepAfterFight(ctx context.Context) {
	if f.upkeepMode() != 2 {
		return
	}
	snap, err := f.read(ctx)
	if err != nil || snap == nil || snap.Player == nil {
		return
	}
	if _, err := f.buffPass(ctx, snap); err != nil {
		f.info("upkeep after fight failed", "err", err)
	}
}

// upkeepAuraUp is the driver's view of "aura sp is up", from what the game shows:
//   - the right hand must hold it: V live on d2d (docs/probes/paladin-aura-live-2026-11-26.md: a non-aura right pick
//     turns the aura off), S on game.exe (an aura runs only from the right slot).
//   - the active aura when the backend reports it (d2d state aura=, game.exe stat 350): exact.
//   - else its Skills.txt aurastate, trusted once learned like a buff's (absent past the grace after an apply = down).
//     U: on d2d `debug states` stays empty with an aura on, so there the active-aura read is the only signal.
//   - else nothing contradicts the right hand: up unless a death or area change marked it dirty.
func (f *Fighter) upkeepAuraUp(p *d2.Player, sp SkillSpec) bool {
	if p.RightSkill != sp.ID {
		return false
	}
	if cur, ok := f.ActiveAura(p); ok {
		return cur == sp.ID
	}
	if st := f.upkeepStateID(sp); st > 0 {
		if f.upk.learned == nil {
			f.upk.learned = map[int]bool{}
		}
		if p.States.Has(st) {
			f.upk.learned[st] = true
			return true
		}
		if f.upk.learned[st] {
			return f.now().Sub(f.upk.auraAt) < upkeepGrace
		}
	}
	return !f.auraDirty
}
