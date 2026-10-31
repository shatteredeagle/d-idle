package merc

import "strings"

// Reasons are the stable words in Decision.Reason and in the log lines (logs.go); harness code may grep them.
const (
	ReasonOK            = "ok"
	ReasonUnknown       = "unknown"
	ReasonNotAtKashya   = "not at kashya"
	ReasonQuestOpen     = "blood raven not done"
	ReasonOwned         = "merc owned"
	ReasonNotOwned      = "no merc owned"
	ReasonAlive         = "alive"
	ReasonFreePending   = "free rogue pending"
	ReasonFreeOnly      = "free rogue only"
	ReasonClass         = "class not planned"
	ReasonAura          = "aura not wanted"
	ReasonNoBetter      = "no better aura"
	ReasonGold          = "gold"
	ReasonKit           = "kit"
	ReasonCap           = "revive cap"
	ReasonStopped       = "stopped by deaths"
	ReasonNotFirstVisit = "not first visit"
	ReasonNotAct2       = "not act 2"
	ReasonKnobOff       = "knob off"
	ReasonNoLevel       = "level unknown"
	ReasonNotNormal     = "not normal"
)

// Ctx is the player side of a decision. Gold is carried plus stash (revive pays carried first, then stash, S);
// Kit is the gold the town portal, potions and antidotes need first.
type Ctx struct {
	Gold, Kit    int
	Difficulty   Difficulty
	Act          int
	PlayerClass  string // "Amazon" "Sorceress" "Necromancer" "Paladin" "Barbarian" "Druid" "Assassin"
	BowOrJavelin bool   // an Amazon built on bow or javelin
	// BloodRavenSkipped: the bot has given up on Blood Raven for good (budget spent, brAttempts).
	BloodRavenSkipped bool
	// FirstLutVisit: this is the first Lut Gholein visit of the character in this difficulty.
	FirstLutVisit bool
	HaveTP        bool
	HavePotions   bool
	// Revives and merc deaths in the last 60 minutes.
	RevivesLastHour, DeathsLastHour int
	Knobs                           Knobs
}

func (c Ctx) spendable() int { return c.Gold - c.Kit }

func no(why string) Decision { return Decision{Reason: why} }
func yes() Decision          { return Decision{Do: true, Reason: ReasonOK} }

// FreeRogueCtx is the situation at Kashya.
type FreeRogueCtx struct {
	AtKashya   bool
	Difficulty Difficulty
	// BloodRavenDone: the quest word shows Blood Raven killed (reward available). Read from internal/questmem.
	BloodRavenDone bool
}

// ShouldClaimFreeRogue: claim the quest-reward Rogue when Blood Raven is done, no merc is employed and we stand at
// Kashya. S: the reward is forfeited when ANY merc is owned, even dead, with no retry (d2decomped). Normal only
// (plan-v2 10.1). An unreadable state still claims: the claim costs nothing when it is already lost.
func ShouldClaimFreeRogue(s MercState, c FreeRogueCtx) Decision {
	switch {
	case c.Difficulty != Normal:
		return no(ReasonNotNormal)
	case !c.AtKashya:
		return no(ReasonNotAtKashya)
	case !c.BloodRavenDone:
		return no(ReasonQuestOpen)
	case s.Known && s.Owned:
		return no(ReasonOwned)
	}
	return yes()
}

// AuraPrefs lists the acceptable auras of a Desert Mercenary best first. Casters (Sorceress, Necromancer, Druid)
// want Defiance then Prayer; melee classes the same; Blessed Aim only for bow and javelin Amazons, and last (it
// raises attack rating, survival first: U judgment). Nightmare has Might (physical classes) and Holy Freeze (casters)
// ahead of those; Hell rows have neither (V hireling.tsv, docs/town.md).
func AuraPrefs(class string, d Difficulty, bowOrJavelin bool) []string {
	caster := class == "Sorceress" || class == "Necromancer" || class == "Druid"
	var p []string
	if d == Nightmare {
		if caster {
			p = append(p, "Holy Freeze")
		} else {
			p = append(p, "Might")
		}
	}
	p = append(p, "Defiance", "Prayer")
	if class == "Amazon" && bowOrJavelin {
		p = append(p, "Blessed Aim")
	}
	return p
}

// AuraRank is the index of aura in AuraPrefs (0 best), -1 when not acceptable or empty.
func AuraRank(prefs []string, aura string) int {
	for i, p := range prefs {
		if strings.EqualFold(p, aura) {
			return i
		}
	}
	return -1
}

func coldArrow(skill string) bool { return strings.Contains(strings.ToLower(skill), "cold") }

// offerOK is the class, aura and gold check shared by hire and replace.
func offerOK(o HireOffer, c Ctx) Decision {
	switch o.Class {
	case ClassRogue:
		if !coldArrow(o.Aura) {
			return no(ReasonAura)
		}
	case ClassDesert:
		if AuraRank(AuraPrefs(c.PlayerClass, c.Difficulty, c.BowOrJavelin), o.Aura) < 0 {
			return no(ReasonAura)
		}
	default:
		return no(ReasonClass)
	}
	if o.Price <= 0 || c.spendable() < o.Price*c.Knobs.HireGoldMult {
		return no(ReasonGold)
	}
	return yes()
}

// ShouldHire is the paid hire when no merc is employed. The Act 1 Rogue only as the fallback once Blood Raven is
// skipped for good (a paid hire forfeits the free one); the Act 2 Desert Mercenary by class aura. Price must stay
// within (gold - kit) / hireGoldMult.
func ShouldHire(o HireOffer, s MercState, c Ctx) Decision {
	if s.Known && s.Owned {
		return no(ReasonOwned)
	}
	if o.Class == ClassRogue {
		if c.Knobs.FreeRogueOnly {
			return no(ReasonFreeOnly)
		}
		if !c.BloodRavenSkipped {
			return no(ReasonFreePending)
		}
	}
	return offerOK(o, c)
}

// ShouldReplace swaps the current merc for offer: only at Act 2, on the first Lut Gholein visit per difficulty,
// with the knob on, gold after the kit at least hireGoldMult x the quote, and a class aura that is acceptable and
// better than the current one (a Rogue, or a Desert Mercenary with a lower-ranked aura; unknown current aura never
// replaces). The old gear is destroyed by the hire, so a living merc with gear says StripFirst.
func ShouldReplace(cur MercState, o HireOffer, c Ctx) Decision {
	switch {
	case !cur.Known:
		return no(ReasonUnknown)
	case !cur.Owned:
		return no(ReasonNotOwned)
	case !c.Knobs.ReplaceAtAct2:
		return no(ReasonKnobOff)
	case c.Act != 2:
		return no(ReasonNotAct2)
	case !c.FirstLutVisit:
		return no(ReasonNotFirstVisit)
	case o.Class != ClassDesert:
		return no(ReasonClass)
	}
	if d := offerOK(o, c); !d.Do {
		return d
	}
	if cur.Class == ClassDesert {
		rank := AuraRank(AuraPrefs(c.PlayerClass, c.Difficulty, c.BowOrJavelin), o.Aura)
		curRank := AuraRank(AuraPrefs(c.PlayerClass, c.Difficulty, c.BowOrJavelin), cur.Aura)
		if curRank < 0 && cur.Aura == "" || (curRank >= 0 && curRank <= rank) {
			return no(ReasonNoBetter)
		}
	}
	d := yes()
	d.StripFirst = cur.Alive && hasGear(cur)
	return d
}

func hasGear(s MercState) bool {
	for _, it := range s.Gear {
		if it != nil {
			return true
		}
	}
	return false
}

// ShouldRevive: the dead merc is revived in town with gold >= reviveGoldMult x price plus the TP and potions kit,
// at most reviveMaxPerHour times an hour; more than stopDeathsPerHour merc deaths in the hour raises the gold bar to
// stopGoldMult x price. price is the quote read from the dialog, or 0 to use the formula (needs the merc level).
func ShouldRevive(s MercState, c Ctx, price int) (Decision, int) {
	switch {
	case !s.Known:
		return no(ReasonUnknown), 0
	case !s.Owned:
		return no(ReasonNotOwned), 0
	case s.Alive:
		return no(ReasonAlive), 0
	}
	if price <= 0 {
		if s.Level < 1 {
			return no(ReasonNoLevel), 0
		}
		price = ReviveCost(s.Level)
	}
	switch {
	case c.RevivesLastHour >= c.Knobs.ReviveMaxPerHour:
		return no(ReasonCap), price
	case !c.HaveTP || !c.HavePotions:
		return no(ReasonKit), price
	}
	mult, why := c.Knobs.ReviveGoldMult, ReasonGold
	if c.DeathsLastHour > c.Knobs.StopDeathsPerHour {
		mult, why = c.Knobs.StopGoldMult, ReasonStopped
	}
	if c.Gold < mult*price {
		return no(why), price
	}
	return yes(), price
}

// MercDied is the field signal: the merc was alive and is now dead while the player is outside town. Leave the area
// (the pack that killed him is still there) and let the town visit decide on the revive.
func MercDied(wasAlive, nowAlive, inTown bool) Decision {
	if wasAlive && !nowAlive && !inTown {
		return Decision{Do: true, Reason: ReasonOK, LeaveArea: true}
	}
	return no(ReasonAlive)
}

// Constants of the soft Andariel rule, the same numbers as internal/task/gate.go (mercLowGold, mercDeadExtra).
const (
	LowGold   = 1000
	DeadExtra = 2
)

// MercAliveGate raises a boss floor by DeadExtra when the merc is known dead and gold is below LowGold. An unknown
// merc state passes the floor unchanged (fail open, like the task hook).
func MercAliveGate(floor int, s MercState, gold int) int {
	if s.Known && s.Owned && !s.Alive && gold < LowGold {
		return floor + DeadExtra
	}
	return floor
}

// AliveHook adapts a Reader to the signature of task.MercAlive: alive is meaningful only when known. A merc that is
// not employed counts as unknown (no merc is not a reason to raise a floor: the bot may be on its way to the Rogue).
func AliveHook(r Reader) func() (alive bool, known bool) {
	return func() (bool, bool) {
		s, err := r.State()
		if err != nil || !s.Known || !s.Owned {
			return false, false
		}
		return s.Alive, true
	}
}
