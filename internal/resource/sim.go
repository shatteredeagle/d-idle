package resource

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Policy is a set of decision functions the simulator can drive; ResourcePolicy is this package, LegacyPolicy
// reproduces the behaviour seen live (cast until empty, wait, drink at 25%).
type Policy struct {
	Name   string
	Pre    func(Input, Tuning) PreDecision
	Attack func(Input, Tuning) AttackDecision
	Potion func(Input, Tuning) PotionDecision
	Abort  func(Input, Tuning) AbortDecision
	Trip   func(Input, Tuning) Trip
}

func ResourcePolicy() Policy {
	return Policy{"resource", PreEngage, AttackChoice, PotionPolicy, AbortFight, TownTripTrigger}
}

// LegacyPolicy: no resting, first affordable spell, wait 700 ms when empty, no melee, no abort, trips only at 0 potions.
func LegacyPolicy() Policy {
	return Policy{
		Name: "legacy",
		Pre:  func(Input, Tuning) PreDecision { return PreDecision{Action: Proceed} },
		Attack: func(in Input, t Tuning) AttackDecision {
			for _, s := range in.Skills {
				if !s.Melee && s.Cost <= in.Mana && s.Ready {
					return AttackDecision{Kind: Cast, Skill: s}
				}
			}
			return AttackDecision{Kind: Wait, WaitMs: t.LowManaWaitMs}
		},
		Potion: func(in Input, t Tuning) PotionDecision {
			if in.LifePct() < t.DrinkLifePct && in.Belt.HP > 0 {
				return PotionDecision{Drink: PotionLife}
			}
			if in.ManaPct() < t.DrinkManaPct && in.Belt.MP > 0 {
				return PotionDecision{Drink: PotionMana}
			}
			return PotionDecision{}
		},
		Abort: func(in Input, t Tuning) AbortDecision {
			if in.LifePct() <= t.ChickenPct && in.Belt.HP == 0 {
				return AbortDecision{Kind: LeaveGame}
			}
			return AbortDecision{}
		},
		Trip: func(in Input, t Tuning) Trip {
			if in.Belt.HP == 0 && in.HasTP && in.Gold >= 30 {
				return Trip{Go: true, Method: ByPortal}
			}
			return Trip{}
		},
	}
}

// Scenario is a deterministic farm replay. Numbers are U (calibrated against the 4m11s Combi Blood Moor run:
// 13.9 kills/min, mana 69 -> 1, docs/combat.md); the harness shows the policy's shape, not exact live yields.
type Scenario struct {
	Name         string
	Seconds      float64
	Class        d2.Class
	Level        int
	MaxLife      int
	MaxMana      int
	StartMana    int
	Gold         int
	HP, MP       int     // starting belt
	ManaFullSecs float64 // 0 = the package model
	TravelSec    float64 // walking between packs
	PackMin      int
	PackMax      int
	MonsterHP    []float64 // cycled
	MonsterDps   float64   // life lost per second per monster in contact
	Exposure     float64   // share of the fight a monster is in contact (ranged kiting)
	MonLevel     int       // monster level of the area (the gold pile size follows it); 0 = 1, the Blood Moor
	SpellCost    int
	SpellDmg     float64
	CastSec      float64
	MeleeDmg     float64 // flat damage per swing that always lands; used only when Melee is nil
	Blow         *Blow   // the blow model (trace B1); nil keeps the flat MeleeDmg
	Foe          Foe     // what the swings hit (defense, level, undead)
	Blind        bool    // the policy is not told Blow / Foe (production before the wiring); the world still rolls them
	Weak         bool    // the pack counts as WeakPack for the policy (every hostile at most clvl+1)
	MeleeSec     float64
	TripSec      float64
	Seed         uint64
	HasMelee     bool // the policy may use the weapon skill
	// MainAttack is the costed attack the policy casts and WeaponSkill the free weapon swing; empty keeps the
	// Sorceress names (Fire Bolt, Staff). WithLadder takes both from a class profile's attack ladder.
	MainAttack, WeaponSkill string
	Belt                    int // belt rows
}

// WithLadder names the main attack and the weapon skill after a profile attack ladder (first rung and the free
// last rung, combat.Profile.AttackNames), so a class other than the Sorceress is not simulated with her skills.
func (s Scenario) WithLadder(names []string) Scenario {
	if len(names) > 0 {
		s.MainAttack, s.WeaponSkill = names[0], names[len(names)-1]
	}
	return s
}

func (s Scenario) mainAttack() string {
	if s.MainAttack != "" {
		return s.MainAttack
	}
	return "Fire Bolt"
}

func (s Scenario) weaponSkill() string {
	if s.WeaponSkill != "" {
		return s.WeaponSkill
	}
	return "Staff"
}

// BloodMoorLife is the cycle of monster life of the stock scenarios: the real Blood Moor Normal population, plain
// monsters (docs/research/monster-stats-trace.md section 4, V run): Fallen 1..4, Quill Rat 1..5, Zombie 7..12 in a
// 10 : 4 : 4 mix, mean 4.17 (means of the row ranges) against the traced 4.2 per unit (the old 12 / 24 / 24 / 45 was 2.5 to 10 times too high).
// LifeCycle builds the same from the loaded rows of any area.
var BloodMoorLife = []float64{2.5, 3, 9.5, 2.5, 3, 2.5, 2.5, 9.5, 3, 2.5, 2.5, 9.5, 3, 2.5, 2.5, 9.5, 2.5, 2.5}

// BloodMoorSorc is the stock scenario: clvl 8 Sorceress, Fire Bolt, 5 minutes of Blood Moor packs.
func BloodMoorSorc() Scenario {
	return Scenario{
		Name: "Blood Moor, clvl 8 Sorceress", Seconds: 300, Class: d2.Sorceress, Level: 8, MaxLife: 60, MaxMana: 45, StartMana: 45,
		Gold: 120, HP: 3, MP: 0, TravelSec: 9, PackMin: 1, PackMax: 4, MonsterHP: BloodMoorLife,
		MonsterDps: 1.6, Exposure: 0.35, SpellCost: 3, SpellDmg: 8, CastSec: 0.45,
		MeleeDmg: 3, MeleeSec: mustSwing(d2.Sorceress), TripSec: 75, Seed: 7, HasMelee: true, Belt: 1,
		// A clvl 8 Sorceress at 25 Dex and 10 Str with a 1-5 staff (StrBonus 100): trace sk-melee-trace.md B1. The
		// to-hit factor 5, the Blood Moor foe (defense 15, level 3) and the staff range are U, nearest defaults.
		Blow: &Blow{AR: AttackRating(0, 25, 5), Level: 8, Weapon: true, Min: 1, Max: 5, Str: 10, DmgPct: StatDmgPct(0, 10, 25, 100, 0, 0), Blunt: true},
		Foe:  Foe{Defense: 15, Level: 3},
	}
}

// BloodMoorPaladin is the stock melee scenario: clvl 12 Paladin on the Zeal / Smite / Attack ladder (Zeal is the
// costed main attack, the plain weapon swing the free rung), 5 minutes of Blood Moor packs. Life, mana, Zeal's
// 2 mana, the 0.7 s swing and its damage are U (nearest defaults, not measurements); the monsters are
// BloodMoorSorc's. The names are the ladder of profiles.json at clvl 12 (a combat test compares them).
func BloodMoorPaladin() Scenario {
	return Scenario{
		Name: "Blood Moor, clvl 12 Paladin", Seconds: 300, Class: d2.Paladin, Level: 12, MaxLife: 190, MaxMana: 40, StartMana: 40,
		Gold: 120, HP: 3, MP: 0, TravelSec: 9, PackMin: 1, PackMax: 4, MonsterHP: BloodMoorLife,
		MonsterDps: 1.6, Exposure: 1, SpellCost: 2, SpellDmg: 12, CastSec: 0.7,
		MeleeDmg: 9, MeleeSec: mustSwing(d2.Paladin), TripSec: 75, Seed: 7, HasMelee: true, Belt: 1,
		MainAttack: "Zeal", WeaponSkill: "Attack",
	}
}

// Report is what a replay measured.
type Report struct {
	Policy                  string
	Kills                   int
	Casts, MeleeHits, Waits int
	RestSec, WaitSec        float64
	StarvedSec              float64 // hostiles around, mana below the cheapest cast, nothing but waiting possible
	MinMana, EndMana        int
	LifePots, ManaPots      int
	Trips, Chickens, Deaths int
	Retreats                int
	EndGold                 int
	Seconds                 float64
	KillsPerMin             float64
}

type rng uint64

func (r *rng) next() uint64 {
	*r = *r*6364136223846793005 + 1442695040888963407
	return uint64(*r >> 33)
}

type world struct {
	sc         Scenario
	pol        Policy
	t          Tuning
	now        float64
	life, mana float64
	potLeft    float64 // mana potion points still to arrive
	potRate    float64 // points per second of the running potion
	gold       int
	goldFrac   float64 // the expected gold of kills not yet whole coins
	belt       Belt
	hasTP      bool
	lastTrip   int
	rep        Report
	r          rng
	regen      float64
	mHP        []float64 // alive monsters in the current pack, front first
	mIdx       int
}

func (w *world) input(hostiles, seen int) Input {
	sk := []Skill{{Name: w.sc.mainAttack(), Cost: w.sc.SpellCost, Ready: true}}
	if w.sc.HasMelee {
		sk = append(sk, Skill{Name: w.sc.weaponSkill(), Melee: true, Ready: true})
	}
	w.belt.Rows = w.sc.Belt
	in := Input{
		Life: int(w.life + 0.5), MaxLife: w.sc.MaxLife, Mana: int(w.mana), MaxMana: w.sc.MaxMana, Belt: w.belt, HasTP: w.hasTP,
		NearTown: true, Gold: w.gold, LastTripGold: w.lastTrip, Level: w.sc.Level, Class: w.sc.Class, Area: 2,
		Hostiles: hostiles, PackSeen: seen, Skills: sk,
		WeakPack: w.sc.Weak,
	}
	if !w.sc.Blind {
		in.Blow, in.Foe = w.sc.Blow, &w.sc.Foe
	}
	return in
}

// advance moves time by dt with natural regeneration and, when hostiles are in contact, damage taken.
func (w *world) advance(dt float64, contact int) {
	w.now += dt
	w.mana += w.regen * dt
	if w.potLeft > 0 {
		g := min(w.potLeft, w.potRate*dt)
		w.potLeft -= g
		w.mana += g
	}
	if w.mana > float64(w.sc.MaxMana) {
		w.mana = float64(w.sc.MaxMana)
	}
	w.life -= float64(contact) * w.sc.MonsterDps * w.sc.Exposure * dt
	if w.mana < float64(w.rep.MinMana) {
		w.rep.MinMana = int(w.mana)
	}
	if w.life <= 0 {
		w.rep.Deaths++
		w.life, w.mana = float64(w.sc.MaxLife), float64(w.sc.MaxMana)
		w.mHP = nil
		w.potLeft = 0
		w.now += w.sc.TripSec
	}
}

func (w *world) drink(p Potion) {
	switch p {
	case PotionLife:
		w.belt.HP--
		w.rep.LifePots++
		w.life += 40
	case PotionMana:
		w.belt.MP--
		w.rep.ManaPots++
		w.addManaPot(40)
	case PotionRejuv:
		w.belt.RV--
		w.life += float64(w.sc.MaxLife) * 0.6
		w.mana += float64(w.sc.MaxMana) * 0.6
	}
	if w.life > float64(w.sc.MaxLife) {
		w.life = float64(w.sc.MaxLife)
	}
	if w.mana > float64(w.sc.MaxMana) {
		w.mana = float64(w.sc.MaxMana)
	}
	w.advance(0.1, len(w.mHP))
}

// addManaPot starts a mana potion: the points arrive over ManaPotSecs, not at once (V read, sk-mana-trace B3). A
// second potion inside the first joins what is left of it (the join rule is U).
func (w *world) addManaPot(points float64) {
	w.potLeft += points
	w.potRate = w.potLeft / ManaPotSecs
}

func (w *world) townTrip() {
	w.rep.Trips++
	w.life, w.mana = float64(w.sc.MaxLife), float64(w.sc.MaxMana)
	b := PlanBuys(w.input(0, 0), w.t, DefaultPrices())
	w.gold -= b.Cost
	w.belt.HP += b.HP
	w.belt.MP += b.MP
	w.belt.RV += b.RV
	w.hasTP = true
	w.lastTrip = w.gold
	w.mHP = nil
	w.advance(w.sc.TripSec, 0)
	w.mana = float64(w.sc.MaxMana)
	w.potLeft = 0
}

// Replay runs the scenario under a policy. Deterministic for a given Seed.
func Replay(sc Scenario, pol Policy, t Tuning) Report {
	regen := ManaPerSec(sc.MaxMana, 0)
	if sc.ManaFullSecs > 0 {
		regen = float64(sc.MaxMana) / sc.ManaFullSecs
	}
	w := &world{sc: sc, pol: pol, t: t, life: float64(sc.MaxLife), mana: float64(sc.StartMana), gold: sc.Gold, hasTP: true, lastTrip: -1,
		r: rng(sc.Seed), regen: regen}
	w.belt = Belt{HP: sc.HP, MP: sc.MP, HPTier: 1, MPTier: 1}
	w.rep = Report{Policy: pol.Name, MinMana: sc.StartMana}
	kind := 0
	for w.now < sc.Seconds {
		// walk to the next pack
		w.advance(sc.TravelSec, 0)
		if tr := pol.Trip(w.input(0, 0), t); tr.Go {
			w.townTrip()
			continue
		}
		n := sc.PackMin + int(w.r.next()%uint64(sc.PackMax-sc.PackMin+1))
		w.mHP = w.mHP[:0]
		for i := 0; i < n; i++ {
			w.mHP = append(w.mHP, sc.MonsterHP[kind%len(sc.MonsterHP)])
			kind++
		}
		w.beforePack(n)
		w.fight()
	}
	w.rep.EndMana, w.rep.EndGold, w.rep.Seconds = int(w.mana), w.gold, w.now
	w.rep.KillsPerMin = float64(w.rep.Kills) / (w.now / 60)
	return w.rep
}

func (w *world) beforePack(n int) {
	waited := 0.0
	for step := 0; step < 200 && w.now < w.sc.Seconds; step++ {
		d := w.pol.Pre(w.input(0, n), w.t)
		switch d.Action {
		case Proceed, Avoid:
			return
		case DrinkMana:
			w.drink(PotionMana)
			return
		case DrinkLife:
			w.drink(PotionLife)
		case TownHeal:
			w.townTrip()
			return
		case RestHere, RetreatRest:
			if waited >= float64(d.WaitSec) {
				return
			}
			dt := 1.0
			if d.Action == RetreatRest && waited == 0 {
				dt = 5 // walking out of sight
			}
			w.advance(dt, 0)
			waited += dt
			w.rep.RestSec += dt
		}
	}
}

func (w *world) fight() {
	idle := 0.0
	for len(w.mHP) > 0 && w.now < w.sc.Seconds {
		in := w.input(len(w.mHP), 0)
		if a := w.pol.Abort(in, w.t); a.Kind != Continue {
			switch a.Kind {
			case RetreatOut:
				w.rep.Retreats++
				w.advance(10, 0)
				continue
			case LeaveGame:
				w.rep.Chickens++
			}
			w.townTrip()
			return
		}
		if p := w.pol.Potion(in, w.t); p.Drink != PotionNone {
			w.drink(p.Drink)
			continue
		}
		at := w.pol.Attack(in, w.t)
		contact := len(w.mHP)
		switch at.Kind {
		case Cast:
			w.mana -= float64(at.Skill.Cost)
			w.rep.Casts++
			idle = 0
			w.advance(w.sc.CastSec, contact)
			w.hit(w.sc.SpellDmg)
		case Melee:
			w.rep.MeleeHits++
			idle = 0
			w.advance(w.sc.MeleeSec, contact)
			w.hit(w.meleeDmg())
		case Retreat:
			w.rep.Retreats++
			w.advance(10, 0)
		default: // Wait, Free
			dt := float64(at.WaitMs) / 1000
			if dt <= 0 {
				dt = 0.7
			}
			w.rep.Waits++
			w.rep.WaitSec += dt
			if w.mana < float64(w.sc.SpellCost) {
				w.rep.StarvedSec += dt
			}
			idle += dt
			w.advance(dt, contact)
		}
		if idle > 120 { // legacy stuck guard: the pack walked away
			w.mHP = nil
		}
	}
}

// meleeDmg is one swing's damage: the flat figure, or a to-hit and damage roll from the blow model.
func (w *world) meleeDmg() float64 {
	if w.sc.Blow == nil {
		return w.sc.MeleeDmg
	}
	return w.sc.Blow.Swing(w.sc.Foe, func(n int) int { return int(w.r.next() % uint64(n)) })
}

func (w *world) hit(dmg float64) {
	if len(w.mHP) == 0 {
		return
	}
	w.mHP[0] -= dmg
	if w.mHP[0] <= 0 {
		w.mHP = w.mHP[1:]
		w.rep.Kills++
		w.goldFrac += data.GoldPerKill(data.KillPlain, w.sc.MonLevel)
		w.gold += int(w.goldFrac)
		w.goldFrac -= float64(int(w.goldFrac))
	}
}

// LifeCycle builds a MonsterHP cycle from loaded monster rows: weights[i] monsters of mons[i] at monster level mlvl
// (mean life, at least 1), interleaved so a pack mixes kinds. Class and build independent.
func LifeCycle(db *data.DB, d data.Difficulty, mlvl int, mons []*data.Monster, weights []int) []float64 {
	var out []float64
	left := append([]int(nil), weights...)
	for more := true; more; {
		more = false
		for i, m := range mons {
			if left[i] > 0 {
				left[i]--
				out = append(out, max(1, db.MeanHP(m, d, mlvl)))
				more = more || left[i] > 0
			}
		}
	}
	return out
}
