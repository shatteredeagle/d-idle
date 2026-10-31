package combat

import (
	"context"
	"errors"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/resource"
)

// Nightmare/Hell glue of the DIFFP2 rules (docs/difficulty.md). Every path here is gated on Difficulty != Normal, so the
// Normal fight loop behaves exactly as before.

// errHardActed tells the fight loop that hardTick did something and the snapshot is stale.
var errHardActed = errors.New("combat: hard tick acted")

// packDanger is the pack danger with the curse bump; from Nightmare on also raised one step when an elemental affix meets
// a negative effective resist of ours (ResistDanger; U: whether Player.Resist already holds the difficulty penalty).
func (f *Fighter) packDanger(p *d2.Player, hs []Target) int {
	d := CurseDanger(p, PackDanger(hs, p.Difficulty, p.Level))
	if p.Difficulty != d2.Normal {
		d = ResistDanger(d, hs, PlayerResists{p.Resist.Fire, p.Resist.Cold, p.Resist.Lightning, p.Resist.Poison}, p.Difficulty)
	}
	return d
}

// TODO(live check, review finding 10): every resist use in this file goes through PlayerResists.Effective, which
// subtracts ResistPenalty. If the stat read (d2/player.go) already contains the difficulty penalty it is applied twice.
// One d2doctor read of a Nightmare character with known gear resists settles it; the only place to change is
// Effective in rules.go (swap its penalty for 0), not these callers. U until then.
//
// hellCtx is the potion policy context: poison state, antidotes held, poison resist, a Mana Burn unit in view.
func (f *Fighter) hellCtx(snap *d2.Snapshot, hs []Target) resource.HellCtx {
	p := snap.Player
	pr := PlayerResists{p.Resist.Fire, p.Resist.Cold, p.Resist.Lightning, p.Resist.Poison}
	h := resource.HellCtx{Poisoned: p.Poisoned(), Antidotes: f.Antidotes(snap), PoisonRes: pr.Effective(p.Difficulty).Poison}
	for _, t := range hs {
		if t.Affixes&AffixManaBurn != 0 && t.Dist <= float64(f.tune().FightRadius) {
			h.ManaBurn = true
			break
		}
	}
	return h
}

// manaBurnReach is how close a Mana Burn monster must be (subtiles) before a mana drop is attributed to it: the
// affix drains on its melee hit (U: the exact reach; defaultMeleeRange is the table's melee default).
const manaBurnReach = defaultMeleeRange

// manaChickenCooldown is how long after a mana-burn chicken another one is suppressed (a re-entry can land next to the
// same pack; the goal's reentry limit would otherwise pause it). R: not measured.
const manaChickenCooldown = 10 * time.Minute

// manaBurnClose reports whether the mana-burn watch applies: a caster (ManaNeed > 0, the same gate as the potion
// policy) with a Mana Burn monster in melee reach.
func (f *Fighter) manaBurnClose(p *d2.Player, hs []Target) bool {
	if p.MaxMana <= 0 || resource.ManaNeed(p.Class) <= 0 {
		return false
	}
	for _, t := range hs {
		if t.Affixes&AffixManaBurn != 0 && t.Dist <= manaBurnReach {
			return true
		}
	}
	return false
}

// antidoteDuration is how long a drunk antidote's poison-resist boost lasts. S: classic D2 antidote is 30 s; the
// shipped potion tables (misc.txt) hold no duration and it is not measured on this install (U live). While it runs the
// poison resist read from the character already includes the boost or the poison is on its way out, so drinking
// again only drains the belt.
const antidoteDuration = 30 * time.Second

// antidoteReady is false while the last antidote drunk is still in effect.
func (f *Fighter) antidoteReady() bool {
	at, ok := f.lastDrink[PotionAntidote]
	return !ok || f.now().Sub(at) >= antidoteDuration
}

// hardTick runs after the survival step (potions and chicken always get their tick first): an antidote for a poisoned low-resist character in Hell, and the mana burn
// watch (flee when the pool drains, leave the game when it stays empty). It returns errHardActed after acting.
func (f *Fighter) hardTick(ctx context.Context, snap *d2.Snapshot, hs []Target, st *fightState) error {
	p := snap.Player
	if p.Difficulty == d2.Normal {
		return nil
	}
	hc := f.hellCtx(snap, hs)
	t := f.tune()
	if resource.AntidoteNeeded(p.Difficulty, hc) && pct(p.Life, p.MaxLife) > t.ChickenHellPct+10 {
		if f.antidoteReady() {
			if err := f.Antidote(ctx); err == nil {
				return errHardActed
			} else if ctx.Err() != nil {
				return err
			}
		}
	}
	if !f.manaBurnClose(p, hs) {
		st.mb.Reset()
		st.spent = 0
		return nil
	}
	act := st.mb.Observe(f.now().Sub(st.t0).Seconds(), p.Mana, p.MaxMana, st.spent)
	st.spent = 0
	switch act {
	case resource.ManaChicken:
		if at := f.manaChickenAt; !at.IsZero() && f.now().Sub(at) < manaChickenCooldown {
			// the last mana-burn chicken was just now: re-entering next to the same pack must not loop (finding 3)
			f.info("mana burn: pool empty again, not leaving twice within the cooldown")
			st.mb.Reset()
			return nil
		}
		f.manaChickenAt = f.now()
		f.info("mana burn: pool empty twice, leaving the game")
		if err := f.Act(ctx, Chicken); err != nil {
			return err
		}
		return ErrChicken
	case resource.ManaFlee:
		f.info("mana burn: pool draining, backing off", "mana", p.Mana, "max", p.MaxMana)
		var pts []input.Point
		for _, h := range hs {
			if h.Affixes&AffixManaBurn != 0 {
				pts = append(pts, input.Point{X: h.Unit.X, Y: h.Unit.Y})
			}
		}
		if f.mv != nil && f.mayEvade() && len(pts) > 0 && st.kites < 2*t.KiteMaxSteps {
			me := input.Point{X: p.X, Y: p.Y}
			if pt, ok := awayPoint(me, pts, hostileMissiles(snap), t.KiteRetreat, f.valid); ok {
				st.kites++
				if err := f.evade(ctx, me, pt); err != nil {
					return err
				}
				return errHardActed
			}
		}
	}
	return nil
}
