package survival

import (
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/store"
)

// LedgerVersion is the schema version of the ledger document (store.File envelope). The ledger lives in its own file
// next to deaths.json, so a bot without this package never reads it (design 10: additive, versioned).
const LedgerVersion = 1

// ZoneKey is one (area, difficulty) pair.
type ZoneKey struct {
	Area int
	Diff string
}

func (k ZoneKey) String() string { return fmt.Sprintf("%d/%s", k.Area, k.Diff) }

// Event kinds a caller reports.
const (
	EvFight     = "fight"  // Kills, Seconds, Taken, PredTaken, PredKillSec, Potions, XP, GoldIn, GoldOut
	EvIntake    = "intake" // LifeLost over Seconds of a stay (observed intake, not the fight-only Taken); adds no fight time
	EvXP        = "xp"     // XP and gold earned over a stay (area summary); adds no fight time; with Seconds it also feeds the realized yield
	EvChicken   = "chicken"
	EvDeath     = "death"
	EvMercDeath = "merc_death"
)

// Event is one observation. Taken is life lost in the fight; PredTaken and PredKillSec are what the model predicted for
// the same kills (the caller knows them; the ledger only forms the ratios).
type Event struct {
	Kind                                          string
	Kills, Seconds, Taken, PredTaken, PredKillSec float64
	Starved                                       float64 // EvFight: watchdog lines where no attack was affordable (mana)
	Potions, XP, GoldIn, GoldOut                  float64
	LifeLost                                      float64 // EvIntake
}

// Ev is one hard event kept for the step-down rule and the revive rule.
type Ev struct {
	T    time.Time `json:"t"`
	Kind string    `json:"kind"`
}

// ZoneStats is the ledger row of one zone. The decayed sums halve every HalfLifeMinutes of field time in the zone.
type ZoneStats struct {
	Kills      float64 `json:"kills"`
	Seconds    float64 `json:"seconds"`
	Taken      float64 `json:"taken"`
	PredTaken  float64 `json:"predTaken"`
	Potions    float64 `json:"potions"`
	Chickens   float64 `json:"chickens"`
	MercDeaths float64 `json:"mercDeaths"`
	Deaths     float64 `json:"deaths"`
	XP         float64 `json:"xp"`
	GoldIn     float64 `json:"goldIn"`
	GoldOut    float64 `json:"goldOut"`
	// IntakeLife and IntakeSec are the observed life lost over the observed stay seconds (decayed together, additive in
	// the file: an old ledger reads 0 and a newer one loads in an older build). See IntakePerMin.
	IntakeLife float64 `json:"intakeLife,omitempty"`
	IntakeSec  float64 `json:"intakeSec,omitempty"`
	// YieldXP and YieldSec are the XP earned over the stay seconds that earned it (decayed together, additive in the file):
	// the realized xp/min the Economy blends with the model's prior (card survfix2). Starved counts the watchdog lines
	// that found every attack too dear for the mana pool (decayed with the fights).
	YieldXP  float64 `json:"yieldXP,omitempty"`
	YieldSec float64 `json:"yieldSec,omitempty"`
	Starved  float64 `json:"starved,omitempty"`

	Hard      float64   `json:"hard"` // product of the hard-event multipliers, relaxing with time away (>= 1; 0 = 1)
	HardAt    time.Time `json:"hardAt"`
	PowerAt   float64   `json:"powerAt"` // the hero power the row was learned at
	Recent    []Ev      `json:"recent,omitempty"`
	LeftAt    time.Time `json:"leftAt"`
	LeftPower float64   `json:"leftPower"`
	NoRevive  time.Time `json:"noReviveUntil"`
}

// Ledger is the persisted document: per zone rows and the pooled offence correction.
type Ledger struct {
	Zones    map[string]*ZoneStats `json:"zones"`
	HeroSec  float64               `json:"heroSec"`  // observed seconds of fighting, pooled
	HeroPred float64               `json:"heroPred"` // predicted seconds for the same kills
	HeroKill float64               `json:"heroKills"`
	P        *Params               `json:"-"`
}

// NewLedger is an empty ledger under p.
func NewLedger(p *Params) *Ledger { return &Ledger{Zones: map[string]*ZoneStats{}, P: p} }

// LedgerFile is the store file of the ledger at path.
func LedgerFile(path string) *store.File[Ledger] {
	return &store.File[Ledger]{Path: path, Version: LedgerVersion}
}

// LoadLedger reads the ledger; a missing file is an empty ledger. A file from a newer build is an error and is not
// touched (store.ErrFutureVersion).
func LoadLedger(path string, p *Params) (*Ledger, store.Info, error) {
	l, info, err := LedgerFile(path).Load()
	if err != nil {
		return NewLedger(p), info, err
	}
	if l.Zones == nil {
		l.Zones = map[string]*ZoneStats{}
	}
	l.P = p
	return &l, info, nil
}

// Save writes the ledger.
func (l *Ledger) Save(path string) error { return LedgerFile(path).Save(*l) }

func (l *Ledger) row(k ZoneKey) *ZoneStats {
	if s := l.Zones[k.String()]; s != nil {
		return s
	}
	s := &ZoneStats{}
	l.Zones[k.String()] = s
	return s
}

func (l *Ledger) peek(k ZoneKey) *ZoneStats { return l.Zones[k.String()] }

func decay(seconds, halfLifeMin float64) float64 { return math.Pow(0.5, seconds/(halfLifeMin*60)) }

func (s *ZoneStats) hard() float64 {
	if s.Hard < 1 {
		return 1
	}
	return s.Hard
}

// hardNow is the hard-event multiplier relaxed toward 1 by the time since the last hard event.
func (l *Ledger) hardNow(s *ZoneStats, now time.Time) float64 {
	h := s.hard()
	if h <= 1 || s.HardAt.IsZero() {
		return 1
	}
	return 1 + (h-1)*decay(now.Sub(s.HardAt).Seconds(), l.P.HardHalfLifeMinutes)
}

func (l *Ledger) clamp(v float64) float64 { return math.Min(l.P.KMax, math.Max(l.P.KMin, v)) }

// Observe adds one event to the zone's row. power is the hero power estimate now (Refresh handles a changed one).
func (l *Ledger) Observe(k ZoneKey, now time.Time, e Event, power float64) {
	s := l.row(k)
	l.refresh(s, power)
	s.LeftAt = time.Time{}
	switch e.Kind {
	case EvFight:
		f := decay(e.Seconds, l.P.HalfLifeMinutes)
		s.Kills, s.Seconds, s.Taken, s.PredTaken = s.Kills*f, s.Seconds*f, s.Taken*f, s.PredTaken*f
		s.Potions, s.XP, s.GoldIn, s.GoldOut = s.Potions*f, s.XP*f, s.GoldIn*f, s.GoldOut*f
		s.Chickens, s.MercDeaths, s.Deaths = s.Chickens*f, s.MercDeaths*f, s.Deaths*f
		s.Starved = s.Starved*f + e.Starved
		s.Kills += e.Kills
		s.Seconds += e.Seconds
		// A fight the model made no prediction for still counts as kills and time (exposure, yield), but its damage
		// stays out of the taken/predicted ratio and its seconds out of the offence pool: both would read as a model error.
		predicted := e.PredTaken > 0 || e.PredKillSec > 0
		if predicted {
			s.Taken += e.Taken
			s.PredTaken += l.floored(e)
		}
		s.Potions += e.Potions
		s.XP += e.XP
		s.GoldIn += e.GoldIn
		s.GoldOut += e.GoldOut
		if predicted {
			h := decay(e.Seconds, l.P.HeroHalfLifeMinutes)
			l.HeroSec, l.HeroPred, l.HeroKill = l.HeroSec*h+e.Seconds, l.HeroPred*h+e.PredKillSec, l.HeroKill*h+e.Kills
		}
	case EvIntake:
		f := decay(e.Seconds, l.P.HalfLifeMinutes)
		s.IntakeLife, s.IntakeSec = s.IntakeLife*f+e.LifeLost, s.IntakeSec*f+e.Seconds
	case EvXP:
		s.XP += e.XP
		s.GoldIn += e.GoldIn
		if e.Seconds > 0 {
			f := decay(e.Seconds, l.P.HalfLifeMinutes)
			s.YieldXP, s.YieldSec = s.YieldXP*f+e.XP, s.YieldSec*f+e.Seconds
		}
	case EvChicken, EvDeath, EvMercDeath:
		mul := map[string]float64{EvChicken: l.P.ChickenMul, EvDeath: l.P.DeathMul, EvMercDeath: l.P.MercDeathMul}[e.Kind]
		s.Hard = l.hardNow(s, now) * mul // the earlier events have relaxed by now; the new one lands in full
		s.HardAt = now
		switch e.Kind {
		case EvChicken:
			s.Chickens++
		case EvDeath:
			s.Deaths++
		case EvMercDeath:
			s.MercDeaths++
		}
		s.Recent = append(s.Recent, Ev{T: now, Kind: e.Kind})
		if len(s.Recent) > 16 {
			s.Recent = s.Recent[len(s.Recent)-16:]
		}
		if e.Kind == EvMercDeath && l.count(s, now, EvMercDeath) >= 2 {
			s.NoRevive = now.Add(time.Duration(l.P.NoReviveMinutes * float64(time.Minute)))
		}
	}
}

// floored is the predicted damage of a fight with the per-kill floor applied: a prediction near zero says nothing about
// the ratio taken/predicted, and on a trivial zone the ratio of two tiny numbers hit the clamp (card survcal, replay). A
// fight with no prediction at all (0) stays 0.
func (l *Ledger) floored(e Event) float64 {
	if e.PredTaken <= 0 {
		return 0
	}
	return math.Max(e.PredTaken, e.Kills*l.P.PredTakenFloor)
}

// refresh drops what was learned about a different hero: when the power moved by more than PowerChange since the row
// was learned, the average relaxes (sums reset) and the hard multiplier goes back to 1 (design 5.4).
func (l *Ledger) refresh(s *ZoneStats, power float64) {
	if power <= 0 {
		return
	}
	if s.PowerAt > 0 && math.Abs(power/s.PowerAt-1) > l.P.PowerChange {
		s.Kills, s.Seconds, s.Taken, s.PredTaken, s.Potions = 0, 0, 0, 0, 0
		s.Hard, s.HardAt = 1, time.Time{}
	}
	if s.PowerAt == 0 || math.Abs(power/s.PowerAt-1) > l.P.PowerChange {
		s.PowerAt = power
	}
}

// Refresh applies the power-change rule to a zone's row (the controller calls it for every candidate).
func (l *Ledger) Refresh(k ZoneKey, power float64) {
	if s := l.peek(k); s != nil {
		l.refresh(s, power)
	}
}

// count is the number of events of the kinds in the window ending at now ("" counts chickens and merc deaths).
func (l *Ledger) count(s *ZoneStats, now time.Time, kinds ...string) int {
	n := 0
	for _, e := range s.Recent {
		if now.Sub(e.T) > time.Duration(l.P.WindowMinutes*float64(time.Minute)) {
			continue
		}
		for _, k := range kinds {
			if e.Kind == k {
				n++
			}
		}
	}
	return n
}

// costCount is the number the `either` row compares: chickens in the window plus merc deaths in the longer merc window
// (a merc that dies every 30 to 60 minutes in one zone never put two deaths inside 30).
func (l *Ledger) costCount(s *ZoneStats, now time.Time) int {
	n := l.count(s, now, EvChicken)
	w := time.Duration(math.Max(l.P.MercWindowMinutes, l.P.WindowMinutes) * float64(time.Minute))
	for _, e := range s.Recent {
		if e.Kind == EvMercDeath && now.Sub(e.T) <= w {
			n++
		}
	}
	return n
}

// KHero is the offence correction pooled over all zones: observed kill seconds over predicted, shrunk toward 1 with few
// kills. It is 1 with no data.
func (l *Ledger) KHero() float64 { return l.KHeroFor(false) }

// KHeroFor is KHero for a hero whose offence was or was not read. With the weapon unread the model's kill speed is a
// named constant, the ratio only measures how wrong that constant is, and it is capped at KHeroUnreadMax so a missing
// term cannot turn every zone into "too hard" (the plan tables decide instead).
func (l *Ledger) KHeroFor(offenceUnread bool) float64 {
	k := l.khero()
	if offenceUnread {
		k = math.Min(k, l.P.KHeroUnreadMax)
	}
	return k
}

func (l *Ledger) khero() float64 {
	if l.HeroPred <= 0 || l.HeroKill <= 0 {
		return 1
	}
	w := l.HeroKill / (l.HeroKill + l.P.ShrinkKills)
	return l.clamp(1 + w*(l.HeroSec/l.HeroPred-1))
}

// KZone multiplies the zone's predicted damage taken (design 5.4): the observed ratio net of KHero (which already
// carries the offence error), shrunk with few kills, times the hard-event multiplier relaxed by time away.
func (l *Ledger) KZone(k ZoneKey, now time.Time) float64 { return l.KZoneFor(k, now, false) }

// KZoneFor is KZone net of the KHeroFor of the same hero.
func (l *Ledger) KZoneFor(k ZoneKey, now time.Time, offenceUnread bool) float64 {
	s := l.peek(k)
	if s == nil {
		return 1
	}
	soft := 1.0
	if s.PredTaken > 0 && s.Kills > 0 {
		w := s.Kills / (s.Kills + l.P.ShrinkKills)
		soft = 1 + w*((s.Taken/s.PredTaken)/l.KHeroFor(offenceUnread)-1)
	}
	return l.clamp(soft * l.hardNow(s, now))
}

// ForcedOut reports whether the observed cost of the zone in the window crosses the dial's step-down row, and why.
func (l *Ledger) ForcedOut(k ZoneKey, dial string, now time.Time) (bool, string) {
	s := l.peek(k)
	if s == nil {
		return false, ""
	}
	row, err := l.P.stepDown(dial)
	if err != nil {
		return false, ""
	}
	if row.Deaths > 0 && l.count(s, now, EvDeath) >= row.Deaths {
		return true, "died"
	}
	if row.Either > 0 && l.costCount(s, now) >= row.Either {
		return true, "hard_events"
	}
	return false, ""
}

// MarkLeft records that the hero left k for cost at this power: the zone stays closed until the power moves by more than
// PowerChange. Any later observation in the zone (the hero is back in it) lifts the lock (design 5.4).
func (l *Ledger) MarkLeft(k ZoneKey, now time.Time, power float64) {
	s := l.row(k)
	s.LeftAt, s.LeftPower = now, power
}

// Locked reports whether a left-for-cost lock on k still holds at this power.
func (l *Ledger) Locked(k ZoneKey, power float64) bool { return l.LockedAt(k, power, time.Time{}) }

// LockedAt is Locked with the time rule: the lock also lapses LockHoldMinutes after the hero left (zero now = no time rule).
func (l *Ledger) LockedAt(k ZoneKey, power float64, now time.Time) bool {
	s := l.peek(k)
	if s == nil || s.LeftAt.IsZero() {
		return false
	}
	if h := l.P.LockHoldMinutes; h > 0 && !now.IsZero() && now.Sub(s.LeftAt) > time.Duration(h*float64(time.Minute)) {
		return false
	}
	if s.LeftPower > 0 && power > 0 && math.Abs(power/s.LeftPower-1) > l.P.PowerChange {
		return false
	}
	return true
}

// NoReviveNow reports whether the zone is closed to reviving (two merc deaths in the window, not yet expired).
func (l *Ledger) NoReviveNow(k ZoneKey, now time.Time) bool {
	s := l.peek(k)
	return s != nil && now.Before(s.NoRevive)
}

// MercMinutes is the expected field minutes until the merc dies again in k: the observed rate, else the prior.
func (l *Ledger) MercMinutes(k ZoneKey) float64 {
	if s := l.peek(k); s != nil && s.MercDeaths >= 0.5 && s.Seconds > 0 {
		return math.Max(1, s.Seconds/60/s.MercDeaths)
	}
	return l.P.MercPriorMinutes
}

// IntakePerMin is the observed life lost per minute in a zone over its decayed stays, with the stay seconds it rests on
// (0, 0 when the zone has no intake data). Observed only: the model's Taken stays the separate fight-only ratio.
func (l *Ledger) IntakePerMin(k ZoneKey) (perMin, seconds float64) {
	s := l.peek(k)
	if s == nil || s.IntakeSec <= 0 {
		return 0, 0
	}
	return s.IntakeLife / (s.IntakeSec / 60), s.IntakeSec
}

// RealizedXPPerMin is the observed XP per minute of a zone over its decayed stays, with the stay seconds it rests on
// (0, 0 without data).
func (l *Ledger) RealizedXPPerMin(k ZoneKey) (perMin, seconds float64) {
	s := l.peek(k)
	if s == nil || s.YieldSec <= 0 {
		return 0, 0
	}
	return s.YieldXP / (s.YieldSec / 60), s.YieldSec
}

// BlendYield moves the zone's xp/min from the model's prior to the realized one as minutes accumulate: weight
// m/(m+shrink) capped at YieldMaxWeight, and none below YieldMinMinutes (the trust floor: one bad minute never poisons the
// value). arch picks the archetype's shrink (data). The second result is the realized value used, 0 when the prior stands.
func (l *Ledger) BlendYield(k ZoneKey, prior float64, arch string) (blended, realized float64) {
	obs, sec := l.RealizedXPPerMin(k)
	m := sec / 60
	if m < l.P.yieldMin(arch) || m <= 0 {
		return prior, 0
	}
	w := math.Min(l.P.YieldMaxWeight, m/(m+l.P.yieldShrink(arch)))
	return prior*(1-w) + obs*w, obs
}

// Starving reports whether the hero repeatedly could not cast in the zone: the decayed count of mana-starved watchdog
// lines reached the archetype's limit.
func (l *Ledger) Starving(k ZoneKey, arch string) bool {
	s := l.peek(k)
	return s != nil && s.Starved >= l.P.starveLimit(arch)
}
