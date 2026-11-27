package combat

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Fight-end telemetry (card combattelem): one Info summary line each for the aim driver, the pack stance and the
// minions, so a drill can assert what the Debug lines and the cumulative counters hid. Counters are kept per fight and
// sampled once a tick (no per-tick log). Lines appear only with combat.deliveryUpkeep above 0 (the knob of the
// drivers they describe), so a default fight logs exactly what it did before.
//
// Tags: pack distance is hero to the centroid of the hostiles within stanceThreatRange (the stance's own measure, V
// code). "In place" is a cast on the cell the hero held on the previous tick (U: a cast right after a step counts as
// moved). The first tick of a fight counts as held. by_row is skill:aimed/refused/shadow. A lost minion is an id seen alive that is no longer among the living friendlies (U: a unit that leaves the
// snapshot's range looks the same as a dead one).
type fightTel struct {
	aim0   AimCounts
	rows0  map[string]aimRowCount
	steps  int // stance steps taken this fight
	casts  int
	inPlc  int
	packN  int
	packSm float64
	packMx float64

	lastPack float64 // pack distance of the latest tick, -1 none
	pos      input.Point
	havePos  bool
	moved    bool

	alive    map[uint32]bool // minions seen alive and not yet lost
	seen     int
	lost     int
	peak     int
	nowAlive int
	distMax  float64
	distSum  float64
	distN    int
}

func (t *fightTel) begin(f *Fighter) {
	t.aim0 = f.AimCounts()
	t.rows0 = map[string]aimRowCount{}
	for k, v := range f.aim.rows {
		t.rows0[k] = *v
	}
	t.lastPack = -1
}

// sample reads the tick's snapshot: hero movement, pack distance and the minions' distance and losses.
func (t *fightTel) sample(f *Fighter, snap *d2.Snapshot, hs []Target) {
	p := snap.Player
	if p == nil || f.upkeepMode() == 0 {
		return
	}
	me := input.Point{X: p.X, Y: p.Y}
	t.moved = t.havePos && me != t.pos
	t.pos, t.havePos = me, true
	t.lastPack = -1
	var cx, cy float64
	n := 0
	for _, h := range hs {
		if pdist(me, input.Point{X: h.Unit.X, Y: h.Unit.Y}) <= stanceThreatRange {
			cx, cy, n = cx+float64(h.Unit.X), cy+float64(h.Unit.Y), n+1
		}
	}
	if n > 0 {
		t.lastPack = pdist(me, input.Point{X: int(math.Round(cx / float64(n))), Y: int(math.Round(cy / float64(n)))})
	}
	if t.alive == nil {
		t.alive = map[uint32]bool{}
	}
	cur := map[uint32]bool{}
	for _, u := range friendlyUnits(snap, f.cat) {
		cur[u.ID] = true
		if !t.alive[u.ID] {
			t.alive[u.ID] = true
			t.seen++
		}
		d := pdist(me, input.Point{X: u.X, Y: u.Y})
		t.distMax = max(t.distMax, d)
		t.distSum += d
		t.distN++
	}
	for id := range t.alive {
		if !cur[id] {
			delete(t.alive, id)
			t.lost++
		}
	}
	t.nowAlive = len(cur)
	t.peak = max(t.peak, t.nowAlive)
}

// cast notes an attack cast against the latest tick's pack distance and whether the hero held its cell.
func (t *fightTel) cast() {
	t.casts++
	if t.havePos && !t.moved {
		t.inPlc++
	}
	if t.lastPack >= 0 {
		t.packN++
		t.packSm += t.lastPack
		t.packMx = max(t.packMx, t.lastPack)
	}
}

// telemSummaries logs the fight-end lines (Info, once each) for the drivers the profile has.
func (f *Fighter) telemSummaries(st *fightState, snap *d2.Snapshot) {
	if f.upkeepMode() == 0 || snap == nil || snap.Player == nil {
		return
	}
	pr := f.profileFor(snap.Player)
	t := &st.tel
	if len(pr.Aims) > 0 {
		c := f.AimCounts()
		var rows []string
		for name, v := range f.aim.rows {
			d := aimRowCount{v.aimed - t.rows0[name].aimed, v.refused - t.rows0[name].refused, v.shadow - t.rows0[name].shadow}
			if d != (aimRowCount{}) {
				rows = append(rows, fmt.Sprintf("%s:%d/%d/%d", strings.ReplaceAll(name, " ", "_"), d.aimed, d.refused, d.shadow))
			}
		}
		sort.Strings(rows)
		by := "-"
		if len(rows) > 0 {
			by = strings.Join(rows, ",")
		}
		f.info("aim summary", "hits", c.Aimed-t.aim0.Aimed, "refused", c.Refused-t.aim0.Refused,
			"shadow_diff", c.ShadowMoved+c.ShadowRefused-t.aim0.ShadowMoved-t.aim0.ShadowRefused, "by_row", by+"")
	}
	if pr.Stance != nil {
		avg := 0.0
		if t.packN > 0 {
			avg = math.Round(t.packSm/float64(t.packN)*10) / 10
		}
		f.info("stance summary", "kind", pr.Stance.Kind, "casts", t.casts, "in_place", t.inPlc, "pack_dist_avg", avg,
			"pack_dist_max", math.Round(t.packMx*10)/10, "steps", t.steps)
	}
	if t.seen > 0 {
		avg := 0.0
		if t.distN > 0 {
			avg = math.Round(t.distSum/float64(t.distN)*10) / 10
		}
		f.info("minion summary", "alive", t.nowAlive, "peak", t.peak, "seen", t.seen, "lost", t.lost,
			"max_dist", math.Round(t.distMax*10)/10, "avg_dist", avg)
	}
}
