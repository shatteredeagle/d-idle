package resource

import "d2lod-bot/internal/d2"

// Hell potion and belt policy (docs/difficulty.md P2 items 24, 25 and P1 item 9; DIFFP2). The thresholds are R
// (reasoned, tune with the soak): nothing here is measured, the numbers come from the difficulty doc.

const (
	hellChickenFloor = 25 // never chicken below this life % in Hell (item 24)
	hellRejuvPct     = 40 // Hell: rejuvenation from this life % (normal tuning 35)
	hellLifePct      = 60 // Hell: healing potion from this life % (normal tuning 55)
	hellRejuvLevel   = 45 // from this level a Hell character carries two rejuvenation columns

	// ManaBurn* follow item 9/24: more than 20% of the mana pool gone in under 3 s means a mana burn unit is on us.
	ManaBurnLossPct  = 20
	ManaBurnWindow   = 3.0
	ManaBurnChicken  = 15 // mana % the pool must fall under, twice, to leave the game
	ManaBurnChickens = 2

	// MinAntidotes is the stock kept in Hell with a poison resist under AntidoteResist (item 25).
	MinAntidotes   = 2
	AntidoteResist = 25
)

// HellTuning raises the potion thresholds for Hell and leaves Normal/Nightmare untouched. The chicken line is
// floored at 25% (item 24); the drink lines are a little higher because Hell hits harder than the potion gap.
func HellTuning(t Tuning, d d2.Difficulty) Tuning {
	if d != d2.Hell {
		return t
	}
	t.ChickenHell = max(t.ChickenHell, hellChickenFloor)
	t.DrinkRejuvPct = max(t.DrinkRejuvPct, hellRejuvPct)
	t.DrinkLifePct = max(t.DrinkLifePct, hellLifePct)
	return t
}

// PlanForDiff is PlanFor with the Hell belt: from hellRejuvLevel on the two rejuvenation columns of the level 61
// layout are used, so one bad hit can always be answered by a rejuvenation potion. Other difficulties keep PlanFor.
func PlanForDiff(c d2.Class, level int, d d2.Difficulty) BeltPlan {
	p := PlanFor(c, level)
	if d == d2.Hell && level >= hellRejuvLevel && level < 61 {
		p.Cols = PlanFor(c, 61).Cols
	}
	return p
}

// HellCtx is what the potion decision needs besides Input.
type HellCtx struct {
	Poisoned  bool // the poison state is on the character
	Antidotes int  // antidote potions held
	PoisonRes int  // effective poison resist after the difficulty penalty
	ManaBurn  bool // a Mana Burn unit is on the screen or the tracker says so
}

// AntidoteNeeded reports whether to drink an antidote now: poisoned in Hell with an effective poison resist below
// AntidoteResist (item 25).
func AntidoteNeeded(d d2.Difficulty, h HellCtx) bool {
	return d == d2.Hell && h.Poisoned && h.Antidotes > 0 && h.PoisonRes < AntidoteResist
}

// AntidotesToCarry is the stock to buy: MinAntidotes in Hell when poison resist is low, else 0.
func AntidotesToCarry(d d2.Difficulty, poisonRes int) int {
	if d == d2.Hell && poisonRes < AntidoteResist {
		return MinAntidotes
	}
	return 0
}

// PotionPolicyHell is PotionPolicy with the Hell thresholds, plus: against a Mana Burn unit a caster drinks a mana
// potion earlier (below 40% of the pool) because the pool is the target, and a poisoned low-resist character is
// told to drink an antidote first (a PotionNone decision with the reason "antidote"; the caller owns the antidote
// slot, which Potion does not model).
func PotionPolicyHell(in Input, t Tuning, h HellCtx) PotionDecision {
	if in.InTown || in.MaxLife <= 0 || in.Life <= 0 {
		return PotionDecision{Reason: "town or dead"}
	}
	if AntidoteNeeded(in.Diff, h) && in.LifePct() > t.Chicken(in.Diff)+10 {
		return PotionDecision{Reason: "antidote"}
	}
	t = HellTuning(t, in.Diff)
	d := PotionPolicy(in, t)
	if d.Drink == PotionNone && h.ManaBurn && in.Diff != d2.Normal && in.manaNeed() > 0 && in.MaxMana > 0 &&
		in.ManaPct() < 40 && in.Belt.MP > 0 && in.Hostiles > 0 {
		return PotionDecision{Drink: PotionMana, Best: true, Reason: "mana burn on screen, refill the pool"}
	}
	return d
}

// ManaAction is the verdict of the ManaBurnTracker.
type ManaAction int

const (
	ManaOK      ManaAction = iota
	ManaFlee               // more than ManaBurnLossPct of the pool lost inside ManaBurnWindow: back off 8+ subtiles and drink
	ManaChicken            // the pool fell under ManaBurnChicken % twice
)

func (a ManaAction) String() string { return [...]string{"ok", "flee", "chicken"}[a] }

type manaSample struct {
	at   float64
	mana int
}

// ManaBurnTracker watches the mana curve of one fight. Feed it every snapshot; it is pure (time is the caller's
// clock, seconds). Natural mana loss from casting also shows as a drop, so callers should Observe with
// spent set to the mana they themselves spent since the last sample.
type ManaBurnTracker struct {
	samples []manaSample
	lows    int
	low     bool
}

// Observe records a reading and returns the action. spent is mana the bot itself used since the previous call.
func (m *ManaBurnTracker) Observe(now float64, mana, maxMana, spent int) ManaAction {
	if maxMana <= 0 {
		return ManaOK
	}
	// mana the bot spent is taken off the older samples so only foreign loss counts
	if spent > 0 {
		for i := range m.samples {
			m.samples[i].mana -= spent
		}
	}
	m.samples = append(m.samples, manaSample{now, mana})
	cut := 0
	for cut < len(m.samples)-1 && now-m.samples[cut].at > ManaBurnWindow {
		cut++
	}
	m.samples = m.samples[cut:]
	top := 0
	for _, s := range m.samples {
		top = max(top, s.mana)
	}
	foreign := (top-mana)*100/maxMana > ManaBurnLossPct
	pc := mana * 100 / maxMana
	// A low counts only when a foreign drop (not our own casting, which is already taken off the samples) is what
	// carried the pool under the line: a caster that routinely casts down to 10% must not chicken (review finding 3).
	if pc < ManaBurnChicken {
		if !m.low && foreign {
			m.lows++
		}
		m.low = true
	} else if pc >= ManaBurnChicken*2 {
		m.low = false
	}
	if m.lows >= ManaBurnChickens {
		return ManaChicken
	}
	if foreign {
		return ManaFlee
	}
	return ManaOK
}

// Reset clears the tracker for a new fight.
func (m *ManaBurnTracker) Reset() { *m = ManaBurnTracker{} }
