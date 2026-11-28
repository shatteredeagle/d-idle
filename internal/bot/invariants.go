package bot

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// The invariant monitor (docs/invariants.md, knob bot.invariants, review fable-logic-review-2026-11-25 P1). A table of
// named checks runs once per tick on a plain facts struct. A hit logs ONE line `invariant <name> ...` (rate limited per
// name) and counts for the minute line. It only watches: nothing here changes what the hero does.

// invRow is one invariant and its thresholds. A row is data: a new class or build needs no new row, the facts carry the
// per-hero numbers (drink line, chicken line, kit minimum).
type invRow struct {
	Name   string
	Secs   int // the condition must hold this long (0: instant)
	Count  int // count threshold, meaning per row
	Pct    int // percent threshold, meaning per row
	GapSec int // minimum seconds between two lines of this name
	Why    string
	Hard   bool // a hard breach: the flip watcher (tools/flip) reverts a knob on any hit since the flip
}

const invDefaultGapSec = 60

var invRows = []invRow{
	{Name: "potion-not-drunk", Hard: true, Secs: 8, Why: "in the field under the drink line with a healing potion in the belt and none drunk for Secs"},
	{Name: "portal-not-cast", Secs: 6, Why: "in the field at or under the chicken line with hostiles near and a town portal scroll, still in the same area after Secs"},
	{Name: "guard-stall", Secs: 60, Count: 2, GapSec: 300, Why: "no log line for Secs while Count or more hostiles are near (stall.go logs the goroutine stack once per five minutes)"},
	{Name: "no-earn", Secs: 300, Why: "in a clearing, traveling or resupplying state for Secs with no change of xp, kills or gold (area changes do not count: Questy 2026-11-26 walked camp and Blood Moor for 28 minutes with nothing to kill), not paused or parked"},
	{Name: "fight-stall", Secs: 120, Count: 60, Pct: 5, GapSec: 300, Why: "a Fighting action older than Secs with no kill, xp or target life change for Count s: the guard breaks it and keeps the target out for Pct minutes (stallguard.go)"},
	{Name: "stuck-in-pack", Secs: 8, Count: 5, Pct: 60, Why: "in the field with Count or more hostiles near and life under Pct for Secs (a crowd fight that neither kills nor leaves)"},
	{Name: "field-entry-hurt", Pct: 50, Why: "left town with life under Pct of max"},
	{Name: "field-entry-no-kit", Why: "left town while the kit (portal, healing, mana minimum) is incomplete"},
	{Name: "resupply-in-field", Secs: 10, Why: "state resupplying outside town for Secs (a town chore running in a field)"},
	{Name: "refusal-as-failure", GapSec: 30, Why: "a refusal (not ready, gated) went through the failure counter"},
	{Name: "refusal-storm", Hard: true, Count: 10, Why: "Count refusals since the last kill, xp or area change"},
	{Name: "life-zero-alive", Hard: true, Secs: 3, Why: "alive (not dying) with life 0 for Secs"},
	{Name: "xp-frozen", Hard: true, Secs: 1800, Why: "in the field, not paused, no xp change for Secs"},
	{Name: "mode-no-exit", Hard: true, Secs: 1200, Why: "the same state and step for Secs with none of the mode's progress signals moving (invModes), not paused or parked"},
	{Name: "corpse-unresolved", Hard: true, Secs: 900, Why: "a corpse waits in town for Secs, neither retrieved nor abandoned"},
	{Name: "town-thrash", Count: 17, Secs: 3600, Why: "Count or more field entries within Secs"},
	{Name: "arbiter-starved", GapSec: 600, Why: "an arbiter intent was outranked for 600 s or more (docs/arbiter.md); only with the arbiter knobs on"},
	{Name: "aura-off", Secs: 10, Why: "in the field with an aura skill learned and the state line reading no active aura (aura=0) for Secs; only where the backend reports the aura (d2d)"},
	{Name: "summon-missing", Secs: 10, Why: "in the field with hostiles near, a summon skill learned and its mana in hand, and the state line reading no own minion (pets=0) for Secs; only where the backend reports the count (d2d); soft: the count rule is unverified live"},
	{Name: "no-combat-profile", Hard: true, GapSec: 300, Why: "the fighter has no combat profile for the hero's class and build (Fighter.ProfileErr): it refuses to fight instead of using another class's profile"},
	{Name: "economy-stall", Secs: 7200, Count: 100, Why: "gold in purse and stash under Count for Secs while kills continue"},
}

// invMode is the progress rule of one state for mode-no-exit: the facts whose change counts as progress, and how long the mode may
// go without any. A state that is not listed uses invModeDefault. Levelling or farming in a state with rising xp never fires; a
// state whose job has no progress signal of its own (a town chore run) gets a hard cap instead (Questy 2026-11-26: 19 false hits
// in 37 minutes of healthy levelling, xp +6300).
type invMode struct {
	Signals []string // any of "xp", "kills", "gold", "area"; none: elapsed time alone counts
	Secs    int      // 0: the row's Secs
}

var invModeDefault = invMode{Signals: []string{"xp", "kills", "gold", "area"}}

var invModes = map[string]invMode{
	string(StateResupplying): {Secs: 600}, // a town chore run has no progress signal; ten minutes is the cap
}

func invModeOf(state string) invMode {
	if m, ok := invModes[state]; ok {
		return m
	}
	return invModeDefault
}

// progress is the signature of the signals of the mode in f.
func (m invMode) progress(f invFacts) string {
	var b strings.Builder
	for _, sg := range m.Signals {
		switch sg {
		case "xp":
			fmt.Fprintf(&b, "x%d", f.xp)
		case "kills":
			fmt.Fprintf(&b, "k%d", f.kills)
		case "gold":
			fmt.Fprintf(&b, "g%d", f.gold)
		case "area":
			fmt.Fprintf(&b, "a%d", f.area)
		}
	}
	return b.String()
}

func invRowOf(name string) invRow {
	for _, r := range invRows {
		if r.Name == name {
			return r
		}
	}
	panic("invariant row missing: " + name) // a programming error caught by the table test
}

// invFacts is everything one tick needs; built from the snapshot and loop state by (*Runner).invTick.
type invFacts struct {
	now                    time.Time
	inTown, dead           bool
	paused, parked         bool
	area                   int
	life, maxLife, lifePct int
	xp                     uint32
	kills, gold            int
	drank                  int64
	hostiles               int
	beltHP, tp             int
	state, step            string
	drinkPct, chickenPct   int
	leaving                bool // this tick is the first one outside town
	kitKnown               bool
	kitMissing             []string
	corpsePending          bool
	corpseSince            time.Time
	auraWatch              bool // an aura skill is learned and the backend reports the active aura
	aura                   int  // the active aura id (0 none), meaningful with auraWatch
	summonWatch            bool // a summon skill is learned and affordable and the backend reports the minion count
	minions                int  // the own minion count, meaningful with summonWatch
}

type invHit struct{ Name, Detail string }

type invMon struct {
	mu          sync.Mutex
	lastLog     map[string]time.Time
	total       map[string]int       // hits since the process started
	first, last map[string]time.Time // first and latest hit per name
	started     time.Time            // first tick: the start of every counter's window
	minute      map[string]int       // hits since the last minute line
	pending     []invHit             // hits noted outside the tick (refusal counted as a failure)
	incidents   incidentGate         // rate limit of the pre-state files a hit writes (incident.go)

	seen, prevTown        bool
	lowSince              time.Time
	lowDrank              int64
	tpSince               time.Time
	tpArea                int
	packSince             time.Time
	resupSince, zeroSince time.Time
	auraOffSince          time.Time
	summonSince           time.Time
	xpAt                  time.Time
	lastXP                uint32
	modeKey               string
	modeSince             time.Time
	modeProg              string
	earnKey               string
	earnSince             time.Time
	trips                 []time.Time
	poorSince             time.Time
	poorKills             int
	refusals, refusalBase int
	refStall              int            // refusals since the last kill, xp or area change; not cleared by a storm hit (failsafe rung 4, quests.refusalPark)
	refWhy                map[string]int // refusal reason -> count since the last progress (refusal-storm top list)
	progKey               string
}

// invSince starts or keeps a stopwatch while cond holds and returns how long it has run (0 when cond is false).
func invSince(t *time.Time, cond bool, now time.Time) time.Duration {
	if !cond {
		*t = time.Time{}
		return 0
	}
	if t.IsZero() {
		*t = now
	}
	return now.Sub(*t)
}

func (r invRow) dur() time.Duration { return time.Duration(r.Secs) * time.Second }

// noteRefusal counts a refused step (the quest step skipped, an area gate) for the refusal-storm row.
// why names the refusal (step and reason); the storm line lists the three that repeated most.
func (m *invMon) noteRefusal(why string) {
	m.mu.Lock()
	m.refusals++
	m.refStall++
	if m.refWhy == nil {
		m.refWhy = map[string]int{}
	}
	if len(m.refWhy) < 64 || m.refWhy[why] > 0 {
		m.refWhy[why]++
	}
	m.mu.Unlock()
}

// topRefusals formats the three most repeated refusal reasons since the last progress, and clears the tally. Caller holds mu.
func (m *invMon) topRefusals() string {
	type kv struct {
		k string
		n int
	}
	var all []kv
	for k, n := range m.refWhy {
		all = append(all, kv{k, n})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n || all[i].n == all[j].n && all[i].k < all[j].k })
	var out []string
	for i, x := range all {
		if i == 3 {
			break
		}
		out = append(out, fmt.Sprintf("%q x%d", x.k, x.n))
	}
	m.refWhy = nil
	return strings.Join(out, "; ")
}

// noteRefusalFailure records that a refusal reached the failure counter; the hit is emitted by the next tick.
func (m *invMon) noteRefusalFailure(what string, err error) {
	m.mu.Lock()
	m.pending = append(m.pending, invHit{"refusal-as-failure", fmt.Sprintf("what=%q err=%q", what, err)})
	m.mu.Unlock()
}

// noteStall records a stall of the guard's look (soft row guard-stall); the incident file and the minute count follow from the hit.
func (m *invMon) noteStall(what string) {
	m.mu.Lock()
	m.pending = append(m.pending, invHit{"guard-stall", what})
	m.mu.Unlock()
}

// noteFightStall records a broken Fighting action (soft row fight-stall).
func (m *invMon) noteFightStall(what string) {
	m.mu.Lock()
	m.pending = append(m.pending, invHit{"fight-stall", what})
	m.mu.Unlock()
}

// noteNoProfile records that the fighter has no combat profile (hard row no-combat-profile).
func (m *invMon) noteNoProfile(what string) {
	m.mu.Lock()
	m.pending = append(m.pending, invHit{"no-combat-profile", what})
	m.mu.Unlock()
}

// noteStarved records that the arbiter left one source's proposal unserved for too long (soft row arbiter-starved).
func (m *invMon) noteStarved(what string) {
	m.mu.Lock()
	m.pending = append(m.pending, invHit{"arbiter-starved", what})
	m.mu.Unlock()
}

// tick checks every row against f and returns the hits that pass the rate limit.
func (m *invMon) tick(f invFacts) []invHit {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started.IsZero() {
		m.started = f.now
	}
	var raw []invHit
	hit := func(name, format string, a ...any) { raw = append(raw, invHit{name, fmt.Sprintf(format, a...)}) }
	raw = append(raw, m.pending...)
	m.pending = nil
	if f.dead {
		// a dead hero breaks no field invariant; keep the stopwatches clean
		m.lowSince, m.tpSince, m.resupSince, m.zeroSince, m.packSince, m.auraOffSince, m.summonSince = time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}
		m.seen, m.prevTown = false, false
		return m.limit(raw, f.now)
	}
	field := !f.inTown

	r := invRowOf("potion-not-drunk")
	low := field && f.maxLife > 0 && f.lifePct < f.drinkPct && f.beltHP >= 1
	if !low || f.drank != m.lowDrank {
		m.lowSince, m.lowDrank = time.Time{}, f.drank
	}
	if d := invSince(&m.lowSince, low, f.now); d >= r.dur() {
		hit(r.Name, "life=%d%% line=%d%% beltHP=%d drank=%d for=%ds hostiles=%d step=%q", f.lifePct, f.drinkPct, f.beltHP, f.drank, int(d.Seconds()), f.hostiles, f.step)
	}

	r = invRowOf("stuck-in-pack")
	packed := field && f.maxLife > 0 && f.lifePct < r.Pct && f.hostiles >= r.Count
	if d := invSince(&m.packSince, packed, f.now); d >= r.dur() {
		hit(r.Name, "life=%d%% hostiles=%d scrolls=%d beltHP=%d area=%d for=%ds step=%q", f.lifePct, f.hostiles, f.tp, f.beltHP, f.area, int(d.Seconds()), f.step)
	}

	r = invRowOf("portal-not-cast")
	tpLow := field && f.maxLife > 0 && f.lifePct <= f.chickenPct && f.hostiles > 0 && f.tp > 0
	if !tpLow || f.area != m.tpArea {
		m.tpSince = time.Time{}
	}
	m.tpArea = f.area
	if d := invSince(&m.tpSince, tpLow, f.now); d >= r.dur() {
		hit(r.Name, "life=%d%% chicken=%d%% scrolls=%d hostiles=%d area=%d for=%ds", f.lifePct, f.chickenPct, f.tp, f.hostiles, f.area, int(d.Seconds()))
	}

	if m.seen && m.prevTown && field {
		f.leaving = true
	}
	m.seen, m.prevTown = true, f.inTown
	if f.leaving {
		m.trips = append(m.trips, f.now)
		if r = invRowOf("field-entry-hurt"); f.maxLife > 0 && f.lifePct < r.Pct {
			hit(r.Name, "life=%d%% min=%d%% area=%d", f.lifePct, r.Pct, f.area)
		}
		if f.kitKnown && len(f.kitMissing) > 0 {
			hit("field-entry-no-kit", "missing=%s area=%d gold=%d", strings.Join(f.kitMissing, ","), f.area, f.gold)
		}
	}

	r = invRowOf("aura-off")
	if d := invSince(&m.auraOffSince, field && f.auraWatch && f.aura == 0, f.now); d >= r.dur() {
		hit(r.Name, "no active aura for=%ds area=%d step=%q hostiles=%d", int(d.Seconds()), f.area, f.step, f.hostiles)
	}

	r = invRowOf("summon-missing")
	if d := invSince(&m.summonSince, field && f.summonWatch && f.minions == 0 && f.hostiles > 0, f.now); d >= r.dur() {
		hit(r.Name, "no own minion for=%ds area=%d step=%q hostiles=%d", int(d.Seconds()), f.area, f.step, f.hostiles)
	}

	r = invRowOf("resupply-in-field")
	if d := invSince(&m.resupSince, field && f.state == string(StateResupplying), f.now); d >= r.dur() {
		hit(r.Name, "state=%s step=%q area=%d for=%ds hostiles=%d", f.state, f.step, f.area, int(d.Seconds()), f.hostiles)
	}

	r = invRowOf("life-zero-alive")
	if d := invSince(&m.zeroSince, f.maxLife > 0 && f.life <= 0, f.now); d >= r.dur() {
		hit(r.Name, "life=0/%d inTown=%v area=%d for=%ds", f.maxLife, f.inTown, f.area, int(d.Seconds()))
	}

	r = invRowOf("xp-frozen")
	if f.xp != m.lastXP || m.xpAt.IsZero() || !field || f.paused || f.parked {
		m.lastXP, m.xpAt = f.xp, f.now
	}
	if d := f.now.Sub(m.xpAt); d >= r.dur() {
		hit(r.Name, "xp=%d for=%ds area=%d state=%s step=%q", f.xp, int(d.Seconds()), f.area, f.state, f.step)
	}

	r = invRowOf("mode-no-exit")
	key := f.state + "|" + f.step
	mode := invModeOf(f.state)
	prog := mode.progress(f)
	if key != m.modeKey || prog != m.modeProg || f.paused || f.parked || f.state == string(StateIdle) || f.state == string(StatePaused) {
		m.modeKey, m.modeProg, m.modeSince = key, prog, f.now
	}
	limit := r.dur()
	if mode.Secs > 0 {
		limit = time.Duration(mode.Secs) * time.Second
	}
	if d := f.now.Sub(m.modeSince); d >= limit {
		hit(r.Name, "state=%s step=%q for=%ds without progress", f.state, f.step, int(d.Seconds()))
	}

	r = invRowOf("no-earn")
	earn := fmt.Sprintf("%d|%d|%d", f.xp, f.kills, f.gold)
	if earn != m.earnKey || f.paused || f.parked || !watchedState(State(f.state)) {
		m.earnKey, m.earnSince = earn, f.now
	}
	if d := f.now.Sub(m.earnSince); d >= r.dur() {
		hit(r.Name, "xp=%d kills=%d gold=%d for=%ds area=%d state=%s step=%q", f.xp, f.kills, f.gold, int(d.Seconds()), f.area, f.state, f.step)
	}

	r = invRowOf("corpse-unresolved")
	if f.corpsePending && !f.corpseSince.IsZero() && f.now.Sub(f.corpseSince) >= r.dur() {
		hit(r.Name, "waiting=%ds", int(f.now.Sub(f.corpseSince).Seconds()))
	}

	r = invRowOf("town-thrash")
	for len(m.trips) > 0 && f.now.Sub(m.trips[0]) > r.dur() {
		m.trips = m.trips[1:]
	}
	if len(m.trips) >= r.Count {
		hit(r.Name, "entries=%d window=%ds", len(m.trips), r.Secs)
	}

	r = invRowOf("economy-stall")
	if f.gold >= r.Count {
		m.poorSince = time.Time{}
	} else if m.poorSince.IsZero() {
		m.poorSince, m.poorKills = f.now, f.kills
	}
	if !m.poorSince.IsZero() && f.now.Sub(m.poorSince) >= r.dur() && f.kills > m.poorKills {
		hit(r.Name, "gold=%d min=%d for=%ds kills+=%d", f.gold, r.Count, int(f.now.Sub(m.poorSince).Seconds()), f.kills-m.poorKills)
	}

	r = invRowOf("refusal-storm")
	if pk := fmt.Sprintf("%d|%d|%d", f.kills, f.xp, f.area); pk != m.progKey || f.parked || f.paused {
		m.progKey, m.refusalBase, m.refStall = pk, m.refusals, 0
		m.refWhy = nil
	}
	if n := m.refusals - m.refusalBase; n >= r.Count {
		hit(r.Name, "refusals=%d since the last kill, xp or area change top=[%s]", n, m.topRefusals())
		m.refusalBase = m.refusals
	}
	return m.limit(raw, f.now)
}

// limit applies the per-name rate limit and counts the lines that go out. Caller holds mu.
func (m *invMon) limit(raw []invHit, now time.Time) []invHit {
	var out []invHit
	for _, h := range raw {
		gap := time.Duration(invDefaultGapSec) * time.Second
		if g := invRowOf(h.Name).GapSec; g > 0 {
			gap = time.Duration(g) * time.Second
		}
		if m.lastLog == nil {
			m.lastLog, m.total, m.minute = map[string]time.Time{}, map[string]int{}, map[string]int{}
		}
		if t, ok := m.lastLog[h.Name]; ok && now.Sub(t) < gap {
			continue
		}
		m.lastLog[h.Name] = now
		m.total[h.Name]++
		if m.first == nil {
			m.first, m.last = map[string]time.Time{}, map[string]time.Time{}
		}
		if _, ok := m.first[h.Name]; !ok {
			m.first[h.Name] = now
		}
		m.last[h.Name] = now
		m.minute[h.Name]++
		out = append(out, h)
	}
	return out
}

// drainMinute returns the per-name counts since the last minute line as "name=n,..." (empty when none).
func (m *invMon) drainMinute() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.minute) == 0 {
		return ""
	}
	names := make([]string, 0, len(m.minute))
	for n := range m.minute {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = fmt.Sprintf("%s=%d", n, m.minute[n])
	}
	m.minute = map[string]int{}
	return strings.Join(parts, ",")
}

// Totals are the lifetime hit counts per name (for ops status and the lab oracles).
func (m *invMon) Totals() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int, len(m.total))
	for k, v := range m.total {
		out[k] = v
	}
	return out
}

// minuteKV is the "inv" attribute of the minute line; nothing while no invariant fired, so the line is unchanged.
func (m *invMon) minuteKV() []any {
	if s := m.drainMinute(); s != "" {
		return []any{"inv", s}
	}
	return nil
}

// InvariantHits returns the lifetime invariant hit counts by name. A lab run (lockstep, Proving Ground) reads it after
// the run, or sets Deps.OnInvariant, and fails when anything fired.
func (r *Runner) InvariantHits() map[string]int { return r.inv.Totals() }

// invTick builds the facts for this tick and logs the hits. snap is a valid in-game snapshot.
func (r *Runner) invTick(snap *d2.Snapshot, c control) {
	if r.e.Tune.Bot.Invariants == 0 || snap == nil || snap.Player == nil {
		return
	}
	p := snap.Player
	now := r.e.Clock.Now()
	t := r.e.Tune
	f := invFacts{now: now, inTown: p.InTown(), dead: isDead(p), paused: c.paused, parked: r.parkedBoard.Load(),
		area: p.Area, life: p.Life, maxLife: p.MaxLife, lifePct: p.LifePct(), xp: p.Exp, gold: p.Gold + p.StashGold,
		drank: r.drank.Load(), drinkPct: t.Combat.DrinkLifePct, chickenPct: t.Combat.ChickenPct}
	r.mu.Lock()
	f.state, f.step = string(r.st), r.activity
	r.mu.Unlock()
	r.obMu.Lock()
	f.kills = r.ob.kills
	r.obMu.Unlock()
	r.corpseTown.mu.Lock()
	f.corpsePending, f.corpseSince = r.corpseTown.pending, r.corpseTown.since
	r.corpseTown.mu.Unlock()
	f.auraWatch, f.aura = r.auraFacts(p)
	f.summonWatch, f.minions = r.summonFacts(p)
	if !f.inTown && !f.dead {
		f.hostiles = r.hostileNear(snap, t.Combat.FightRadius)
		if f.lifePct < f.drinkPct || f.lifePct <= f.chickenPct {
			k := r.kitOf(snap)
			f.beltHP, f.tp = k.BeltHP, k.TownPortals
		}
		if r.inv.wasInTown() { // the kit is read once per departure, kitRead can log
			if k, known := r.kitRead(snap); known {
				f.kitKnown, f.kitMissing = true, k.missing()
			}
		}
	}
	if pe, ok := r.d.Fighter.(interface{ ProfileErr() error }); ok {
		if err := pe.ProfileErr(); err != nil {
			r.inv.noteNoProfile(err.Error())
		}
	}
	for _, h := range r.inv.tick(f) {
		r.e.Log.Warn("invariant "+h.Name, "detail", h.Detail)
		if r.d.OnInvariant != nil {
			r.d.OnInvariant(h.Name, h.Detail)
		}
		r.noteMonitorIncident(snap, h)
	}
}

func (m *invMon) wasInTown() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.seen && m.prevTown
}

// isRefusal reports an error that is a refusal (a gate, a step that is not ready), not a fault. Until the typed
// outcome lands (docs/outcomes.md) the wording is all there is; it matches the planningFault rule.
func isRefusal(err error) bool {
	return err != nil && (errors.Is(err, errAreaGated) || strings.Contains(err.Error(), "not ready"))
}

// InvCounter is one invariant counter of the read-only status surface (GET /api/explain and `ops status --why`): the
// flip watcher (tools/flip, docs/process/dark-launch.md) reads the Hard ones. Since is when counting began (the first
// tick of this process), so a restart shows as a new Since; First and Last are the first and latest hit.
type InvCounter struct {
	Name  string    `json:"name"`
	Hard  bool      `json:"hard"`
	Count int       `json:"count"`
	Since time.Time `json:"since"`
	First time.Time `json:"first,omitempty"`
	Last  time.Time `json:"last,omitempty"`
}

// Counters lists every row, in table order, with its lifetime count. Empty until the first tick.
func (m *invMon) Counters() []InvCounter {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started.IsZero() {
		return []InvCounter{}
	}
	out := make([]InvCounter, 0, len(invRows))
	for _, r := range invRows {
		out = append(out, InvCounter{Name: r.Name, Hard: r.Hard, Count: m.total[r.Name], Since: m.started, First: m.first[r.Name], Last: m.last[r.Name]})
	}
	return out
}

// auraFacts reports whether the aura-off row applies: the backend gives the active aura (d2d state line) and the hero has
// learned a skill the data marks as an aura. No class check: a class without aura skills never qualifies.
func (r *Runner) auraFacts(p *d2.Player) (watch bool, id int) {
	if !p.AuraKnown || r.e.Data == nil {
		return false, 0
	}
	for _, sk := range p.Skills {
		if d := r.e.Data.SkillByID(sk.ID); sk.Level > 0 && d != nil && d.Aura {
			return true, p.Aura
		}
	}
	return false, p.Aura
}

// summonFacts reports whether the summon-missing row applies: the backend gives the minion count (d2d state line) and the hero
// has learned a summon skill (skills.txt `summon` column, data: no class check) whose mana cost it can pay now.
func (r *Runner) summonFacts(p *d2.Player) (watch bool, minions int) {
	if r.e.Data == nil {
		return false, 0
	}
	return summonWatch(p, r.e.Data.SkillByID)
}

func summonWatch(p *d2.Player, byID func(int) *data.Skill) (bool, int) {
	if !p.PetsKnown {
		return false, 0
	}
	for _, sk := range p.Skills {
		if d := byID(sk.ID); sk.Level > 0 && d != nil && d.Summon != "" && !d.Passive && float64(p.Mana) >= d.ManaCost(sk.Level) {
			return true, p.Pets
		}
	}
	return false, p.Pets
}

// petsKV is the own minion count for the minute line, only where the backend reports it (d2d state line).
func petsKV(p *d2.Player) []any {
	if p.PetsKnown {
		return []any{"minions", p.Pets}
	}
	return nil
}

// refusalOnlyStall reports whether the stall so far is made of refusals: at least min refused steps since the last kill,
// xp or area change (the same progress the refusal-storm row watches).
func (m *invMon) refusalOnlyStall(min int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.refStall >= min
}
