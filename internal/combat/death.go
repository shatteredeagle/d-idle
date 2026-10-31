package combat

import (
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// LifeSample is one point of the life curve kept for death reports.
type LifeSample struct {
	Life, Max int
	Pos       input.Point
}

// DeathInfo is the cause data of one death, for the goals Death event and the bot's death log.
type DeathInfo struct {
	Area         int
	Pos          input.Point // where the body lies
	MonsterID    uint32      // nearest hostile at the last live snapshot, 0 if none
	MonsterClass int
	MonsterName  string
	MonsterLevel int  // monstats level for the difficulty; the area level is the caller's
	Boss         bool // named boss or superunique only; a champion is Champion, never Boss
	Champion     bool
	Affixes      Affix
	Distance     float64 // killer distance at the last live snapshot
	LastLife     int
	LastMax      int
	Curve        []LifeSample // newest last, filled by Fighter.Died from its own samples
	Difficulty   d2.Difficulty
	CharLevel    int
	Mode         int // player mode that was read: ModeDying (0) at the first dead sample, ModeDead (17) a second later
	Gold         int // gold carried at the last living snapshot: it lies as a gold pile at Pos afterwards, not lost (VERIFIED LIVE)
}

// Died is the one authoritative death transition: prev was alive in game, cur reads dead. It is pure; the killer is
// the nearest living hostile in prev (the last snapshot where it was still listed alive), ties broken by threat.
// ok is false when no death happened between the snapshots.
func Died(prev, cur *d2.Snapshot, cat Catalog) (DeathInfo, bool) {
	if cur == nil || cur.State != d2.StateInGame || cur.Player == nil || !isDead(cur.Player) {
		return DeathInfo{}, false
	}
	if prev != nil && prev.State == d2.StateInGame && prev.Player != nil && isDead(prev.Player) {
		return DeathInfo{}, false // already known
	}
	cp := cur.Player
	di := DeathInfo{Area: cp.Area, Pos: input.Point{X: cp.X, Y: cp.Y}, Difficulty: cp.Difficulty, CharLevel: cp.Level, Mode: cp.Mode}
	src := cur
	if prev != nil && prev.State == d2.StateInGame && prev.Player != nil {
		src = prev
		di.LastLife, di.LastMax = prev.Player.Life, prev.Player.MaxLife
		di.Gold = prev.Player.Gold
	}
	// hostiles as of the last live view, measured from where we died
	probe := *src.Player
	probe.X, probe.Y = cp.X, cp.Y
	view := *src
	view.Player = &probe
	view.Player.Area = cp.Area
	if cp.InTown() { // Hostiles refuses towns, and nothing hostile lives there
		return di, true
	}
	hs := Hostiles(&view, HostileCtx{Cat: cat, Diff: cp.Difficulty, ClassLvl: cp.Level})
	if len(hs) == 0 {
		// the killer may already be listed dead in cur; take the nearest unit that is not ours
		return di, true
	}
	best := hs[0]
	for _, h := range hs {
		if h.Dist > 8 {
			break
		}
		if h.Boss || h.Elite && !best.Elite {
			best = h
		}
	}
	di.MonsterID, di.MonsterClass, di.MonsterName = best.Unit.ID, best.Unit.Class, best.Name
	di.Boss, di.Champion, di.Affixes, di.Distance = best.Boss, best.Champion, best.Affixes, best.Dist
	if best.Mon != nil {
		di.MonsterLevel = best.Mon.Level[cp.Difficulty]
	}
	return di, true
}

// Died on the Fighter adds the life curve the fight loop recorded.
func (f *Fighter) Died(prev, cur *d2.Snapshot) (DeathInfo, bool) {
	di, ok := Died(prev, cur, f.cat)
	if ok {
		di.Curve = append([]LifeSample(nil), f.curve...)
	}
	return di, ok
}

const curveLen = 40

func (f *Fighter) sampleLife(p *d2.Player) {
	f.curve = append(f.curve, LifeSample{Life: p.Life, Max: p.MaxLife, Pos: input.Point{X: p.X, Y: p.Y}})
	if len(f.curve) > curveLen {
		f.curve = f.curve[len(f.curve)-curveLen:]
	}
}

// DeathWeights are lessons from earlier deaths, filled by the bot: a weight of 1 is "somewhat dangerous", 3 or more
// is "do not fight this without need". Keys are lower-case monstats ids (e.g. "fallenshaman1") and affix names as
// printed by Affix.String ("cursed", "mana-burn", ...).
type DeathWeights struct {
	Monster map[string]float64
	Affix   map[string]float64
}

// avoidWeight is the weight at which a target is no longer fought (unless it is the Kill target).
const avoidWeight = 3.0

func (w DeathWeights) of(monKey string, a Affix) float64 {
	var s float64
	if w.Monster != nil {
		s += w.Monster[strings.ToLower(monKey)]
	}
	if w.Affix != nil {
		for _, n := range strings.Split(a.String(), ",") {
			s += w.Affix[n]
		}
	}
	return s
}

// SetDeathWeights installs the death-history weights used by targeting.
func (f *Fighter) SetDeathWeights(w DeathWeights) { f.weights = w }
