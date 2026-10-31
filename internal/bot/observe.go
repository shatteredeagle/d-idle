package bot

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/stepgroup"
	"d2lod-bot/internal/task"
)

// moveEpsilon is the number of subtiles that count as movement for the watchdog.
const moveEpsilon = 3

type monTrack struct {
	aliveSeen bool
	counted   bool
}

type itemTrack struct {
	meta  ItemMeta
	known bool
	class int
	item  *d2.Item
}

// observer turns the stream of snapshots into events and progress marks. All fields are guarded by Runner.obMu.
type observer struct {
	init       bool
	state      d2.State
	menuSince  time.Time
	inGameAt   time.Time // when the current game was entered
	area       int
	level      int
	exp        uint32
	savedExp   uint32 // experience the character had when this game was entered = what its .d2s held (U: area changes may save too, then this overstates the risk)
	savedNew   bool   // a game was entered and savedExp is not yet taken
	xpLost     int64  // experience that vanished across game restarts (memory exp dropped to the save's)
	pendingXP  int64
	dead       bool
	portalFrom int // the field area the character last went to town from alive: where a town portal read there leads back (0: unknown)
	lastGood   struct {
		area int
		pos  input.Point
	}
	kt      killClock
	mons    map[uint32]monTrack
	monLife int
	pos     input.Point
	inv     map[uint32]itemTrack
	belt    map[uint32]itemTrack
	invInit bool

	questAt   time.Time
	questSeen map[string]task.Stage // "diff/quest" -> last stage emitted

	lastProgress  time.Time
	rearmAt       time.Time
	real          uint64 // progress that resets the failsafe ladder
	progress      uint64 // monotonic progress counter (health endpoint)
	stuckReason   string
	gold          int
	lastDisarm    time.Time
	lastTrip      time.Time
	taskRep       taskReport
	tripGold      int // gold when the last chore ended (resource.Input.LastTripGold)
	tripGoldSet   bool
	tripTimes     []time.Time            // finished chore trips, for the trip budget
	idleTrips     int                    // consecutive chores that left no more gold (resource.Input.IdleTrips)
	povertyPlan   resource.PovertyAction // last resource.PovertyAdvice verdict
	maxPackDanger int                    // PlayLean: highest pack danger worth engaging while poor
	gateKey       string                 // last logged gear gate state (noteGate)
	riskKey       resource.Risk          // last logged gear risk (noteGate)
	riskSeen      bool
	blocked       map[string][3]int // slot -> str,dex,level when a rebuy found only gear it cannot wear yet
	recovering    bool              // recovery phase running (broke.go), counts as poor
	poor          bool              // poverty mode (resource.PovertyCheck), read by the guard
	routeLine     string            // last route guard verdict (task.RouteLine), for the death autopsy
	sellLeft      int               // sellable items still in the pack after the last poverty sell trip
	packFullUntil time.Time         // a trip left the pack full: the full-pack reason is quiet until then
	lastStat      time.Time
	lastMinute    time.Time
	kills         int // counters for the "minute" summary line (since the bot process started, see counters.go)
	cInit         bool
	cName         string
	cBase, cSaved int64 // per-character kill total from the counters file, and the last value written
	cMark         int   // kills at the time the base was loaded
	recent        []int // last distinct areas entered, newest last (route id of a death)
	pendingDeaths []*deathFollow
	deaths        int
	potions       int
	hp            hpStat // dmgTaken / hpMinPct of the minute line (hpfields.go)
	lastDeath     aliveInfo
	lastDeathAt   time.Time
	deathPending  bool // a death was recorded and the loop has not opened its recovery yet
	targetArea    int  // the area the loop wants to be clearing
	stepLevel     int  // level the reach_level planner last chose for

	lastHealth   time.Time
	gone         *gameGone
	oog          int       // out-of-game actions running (guard keeps off the menu grace)
	expectExit   bool      // the bot itself is restarting the game: an exit now is the plan, not a crash
	errSince     time.Time // first of a run of failed snapshots
	plannedGames int       // newGames entries that were planned save points (not restarts)
	newGames     int       // incremented whenever the menus were left for a game
	recycles     int       // unplanned games the bot started itself (newGameRun, Recycle): counted even when no menu poll sees them (d2d)
	alive        aliveInfo
	curve        []lifeStamp
	trail        []trailStamp // one sample per second for the last minute (autopsy.go)
	activity     string
	goalID       string
	lastSnap     *d2.Snapshot

	lastSafety   SafetyAction
	lastSafetyAt time.Time
}

type lifeStamp struct {
	at  time.Time
	pct int
}

// aliveInfo is what the last living snapshot knew; a death record is built from it.
type aliveInfo struct {
	gold, equip, equipValue int
	stashGold               int
	merc                    *bool // nil: no merc unit listed (unknown)
	killer                  *d2.Unit
	killerName              string
	killerBoss              bool // named boss or superunique, never a champion
	killerChampion          bool
	killerLevel             int // monstats level in the current difficulty, 0 unknown
	killerAffix             combat.Affix
	level                   int
	difficulty              d2.Difficulty
	kit                     Kit
	hostiles                int
	near                    []NearHostile // up to 5, nearest first
}

// isDead uses the verified death facts (combat/deathfacts.go: mode 0 while dying, 17 once fallen, life 0).
func isDead(p *d2.Player) bool { return p != nil && combat.IsDead(p) }

// poll takes a snapshot, derives events and feeds the watchdog. It returns nil while loading or when the read failed.
func (r *Runner) poll() *d2.Snapshot {
	snap, err := r.d.Snapshot()
	now := r.e.Clock.Now()
	r.obMu.Lock()
	defer r.obMu.Unlock()
	o := &r.ob
	r.healthLocked(now)
	if err != nil {
		if o.errSince.IsZero() {
			o.errSince = now
		} else if r.d.Health == nil && now.Sub(o.errSince) > sec(r.cfg.MenuGraceSec)+5*time.Second && o.gone == nil && !o.expectExit {
			o.gone = &gameGone{Event: launch.EventHung}
			r.e.Log.Error("game memory unreadable", "err", err, "for", now.Sub(o.errSince))
		}
		return nil
	}
	o.errSince = time.Time{}
	o.lastSnap = snap
	r.observe(snap, now)
	if snap.State != d2.StateInGame || snap.Player == nil {
		return nil
	}
	if r.d.OnSnapshot != nil {
		r.d.OnSnapshot(snap)
	}
	return snap
}

func (r *Runner) healthLocked(now time.Time) {
	o := &r.ob
	if r.d.Health == nil || now.Sub(o.lastHealth) < time.Second {
		return
	}
	o.lastHealth = now
	switch ev := r.d.Health.Step(now); ev {
	case launch.EventExited, launch.EventCrashed, launch.EventHung:
		if o.gone == nil && !o.expectExit {
			o.gone = &gameGone{Event: ev}
		}
	}
}

// expectGameExit brackets an action that restarts the game on purpose (failsafe rung 4). Turning it off forgets any exit
// seen meanwhile, as the recycle path in betweenGoals does.
// oogRun marks f as an out-of-game action (menu re-entry, login, restart): the guard must not cancel it for being at the
// menus, that is where it works.
func (r *Runner) oogRun(f func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		r.obMu.Lock()
		r.ob.oog++
		r.obMu.Unlock()
		defer func() {
			r.obMu.Lock()
			r.ob.oog--
			r.obMu.Unlock()
			r.Beat()
		}()
		r.Beat()
		return f(ctx)
	}
}

// newGameRun is oogRun for an unplanned new game (farm, difficulty switch, fresh game, failsafe): a success also counts
// as a restart on the minute line, because the d2d backend restarts the process without a menu poll and the observer's
// newGames never moves there. Save points are planned and use oogRun directly.
func (r *Runner) newGameRun(f func(context.Context) error) func(context.Context) error {
	run := r.oogRun(f)
	return func(ctx context.Context) error {
		err := run(ctx)
		if err == nil {
			r.noteRecycle()
		}
		return err
	}
}

func (r *Runner) noteRecycle() {
	r.obMu.Lock()
	r.ob.recycles++
	r.obMu.Unlock()
}

// restartCount is the minute line's restarts: game entries after the first that were not planned save points, as the
// menu observer saw them (Game.exe: also catches crash re-entries), or the games the bot started itself, whichever is
// larger (d2d never shows the menus, so only the second is non-zero there).
func restartCount(newGames, planned, recycles int) int {
	return max(0, newGames-1-planned, recycles)
}

func (r *Runner) expectGameExit(on bool) {
	r.obMu.Lock()
	defer r.obMu.Unlock() // Health.Step is foreign code: a recovered panic must not leave obMu locked
	r.ob.expectExit = on
	if !on {
		r.ob.gone, r.ob.errSince = nil, time.Time{}
		// An exit the guard did not get to poll during the bracket would otherwise latch right after it as an unplanned
		// loss (and spend the restart budget a second time): read and drop it now.
		if r.d.Health != nil {
			r.d.Health.Step(r.e.Clock.Now())
			r.ob.lastHealth = r.e.Clock.Now()
		}
	}
}

// gameEvent returns the latched game-gone condition.
func (r *Runner) gameEvent() *gameGone {
	r.obMu.Lock()
	defer r.obMu.Unlock()
	return r.ob.gone
}

// markProgress tells the watchdog the bot is getting somewhere.
func (r *Runner) markProgress() {
	r.obMu.Lock()
	r.ob.bump(r.e.Clock.Now())
	r.obMu.Unlock()
}

// armWatchdog is kept as the per-action hook: it only records disarm times for the diagnostics.
func (r *Runner) armWatchdog(on bool) {}

// bump records progress: it feeds the monotonic counter and clears the stuck reason. Caller holds obMu.
func (o *observer) bump(now time.Time) {
	o.lastProgress = now
	o.progress++
	o.stuckReason = ""
}

// bumpReal is progress that counts for the failsafe ladder: experience, kills and levels only. Walking, area changes,
// gold, potions, drops and damage dealt keep the watchdog quiet (bump) but a crash loop causes them too, so they must
// not refill the restart budgets.
func (o *observer) bumpReal(now time.Time) {
	o.bump(now)
	o.real++
}

// rearm gives the watchdog a fresh window without counting progress (after a ladder step). Caller holds obMu.
func (o *observer) rearm(now time.Time) { o.rearmAt = now }

func (r *Runner) rearm() {
	r.obMu.Lock()
	r.ob.rearm(r.e.Clock.Now())
	r.obMu.Unlock()
}

// skippedState: states in which doing nothing is the job or already an error; the watchdog does not run in them.
func skippedState(st State, idleGoal bool) bool {
	switch st {
	case StateIdle, StatePaused, StateError:
		return true
	case StateInTown:
		return idleGoal
	}
	return false
}

// stuckCheck is the watchdog: a reason when the bot has made no progress for too long, else "". Two clocks: the
// per-state clock (since the last progress or state entry, limit Config.StateStuckSec[state] or StuckAfterSec) and the
// global clock (since the last progress, limit GlobalStuckSec), so a loop that flaps between states is caught too.
func (r *Runner) stuckCheck(now time.Time) string {
	r.mu.Lock()
	st, since, active, goal := r.st, r.stateSince, r.activeSince, r.goal
	idleGoal := goal != nil && goal.Type == goals.Idle
	paused := r.paused
	r.mu.Unlock()
	if goal == nil || paused || skippedState(st, idleGoal) {
		return ""
	}
	r.obMu.Lock()
	defer r.obMu.Unlock()
	o := &r.ob
	base := o.lastProgress
	for _, t := range []time.Time{o.rearmAt, active} {
		if t.After(base) {
			base = t
		}
	}
	lim := sec(r.cfg.StuckAfterSec)
	if v, ok := r.cfg.StateStuckSec[string(st)]; ok && v > 0 {
		lim = sec(v)
	}
	if st == StateClearing && r.cfg.ClearIdleSec > 0 && sec(r.cfg.ClearIdleSec) < lim {
		lim = sec(r.cfg.ClearIdleSec) // see Config.ClearIdleSec
	}
	sb := base
	if since.After(sb) {
		sb = since
	}
	var why string
	switch {
	case now.Sub(sb) >= lim:
		why = fmt.Sprintf("no progress in state %s for %ds", st, int(now.Sub(sb)/time.Second))
	case now.Sub(base) >= sec(r.cfg.GlobalStuckSec):
		why = fmt.Sprintf("no progress for %ds (state %s)", int(now.Sub(base)/time.Second), st)
	}
	if why != "" {
		o.stuckReason = why
	}
	return why
}

// observe is called with obMu held.
func (r *Runner) observe(s *d2.Snapshot, now time.Time) {
	o := &r.ob
	if s.State == d2.StateMenu {
		if o.menuSince.IsZero() {
			o.menuSince = now
		}
	} else if s.State == d2.StateInGame {
		if !o.menuSince.IsZero() || !o.init {
			o.newGames++
			o.inGameAt = now
			o.mons, o.inv, o.belt, o.invInit = map[uint32]monTrack{}, map[uint32]itemTrack{}, map[uint32]itemTrack{}, false
			o.area, o.dead = 0, false
			o.savedNew = true
		}
		o.menuSince = time.Time{}
	}
	o.state = s.State
	if s.State != d2.StateInGame || s.Player == nil {
		o.kt.Gap()
		return
	}
	p := s.Player
	o.kt.Tick(now, p.Mode, p.Area, p.X, p.Y)
	r.updateCharacter(s)
	first := !o.init
	o.init = true
	if o.savedNew {
		o.savedNew, o.savedExp = false, p.Exp
	}

	if now.Sub(o.lastStat) >= 30*time.Second {
		o.lastStat = now
		r.mu.Lock()
		st := r.st
		r.mu.Unlock()
		r.e.Log.Info("status", "state", st, "area", p.Area, "x", p.X, "y", p.Y, "life", p.Life, "level", p.Level, "hostileNear", r.hostileNear(s, r.e.Tune.Bot.ClearRadius), "stuckFor", now.Sub(o.lastProgress).Round(time.Second))
	}

	if now.Sub(o.lastMinute) >= time.Minute {
		first := o.lastMinute.IsZero()
		o.lastMinute = now
		if !first {
			r.minuteLine(s, p, now)
		}
	}

	// level up
	if o.level > 0 && p.Level > o.level {
		r.emit(goals.Event{Type: goals.EvLevelUp, Level: p.Level, Area: r.areaName(p.Area), AreaID: p.Area})
		o.bumpReal(now)
	}
	o.level = p.Level

	r.pollQuests(p, now)

	// area change
	if p.Area != o.area {
		if o.area != 0 && !o.dead && r.d.Plan.IsTown(p.Area) && !r.d.Plan.IsTown(o.area) {
			o.portalFrom = o.area
		}
		o.area = p.Area
		o.noteRoute(p.Area)
		o.mons = map[uint32]monTrack{}
		o.bump(now)
		r.emit(goals.Event{Type: goals.EvAreaChange, Area: r.areaName(p.Area), AreaID: p.Area})
	}

	// death
	dead := isDead(p)
	if dead && !o.dead {
		o.dead = true
		o.noteHP(p, 1)
		r.recordDeath(p, now)
	}
	if !dead {
		o.dead = false
		r.noteBackInField(p, now)
		o.lastGood.area, o.lastGood.pos = p.Area, input.Point{X: p.X, Y: p.Y}
		r.trackAlive(s, now)
	}

	// movement
	if pos := (input.Point{X: p.X, Y: p.Y}); abs(pos.X-o.pos.X) >= moveEpsilon || abs(pos.Y-o.pos.Y) >= moveEpsilon {
		o.pos = pos
		o.bump(now)
	}

	if !dead && !first {
		r.observeKills(s, now)
		r.observeItems(s, now)
	} else {
		r.baselineItems(s)
		o.exp = p.Exp
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (r *Runner) areaName(id int) string {
	if r.d.Catalog != nil {
		if n := r.d.Catalog.AreaName(id); n != "" {
			return n
		}
	}
	if a, ok := r.d.Plan.Areas[id]; ok && a.Name != "" {
		return a.Name
	}
	return fmt.Sprintf("Area %d", id)
}

func (r *Runner) monsterMeta(class int) (MonsterMeta, bool) {
	if r.d.Catalog == nil {
		return MonsterMeta{Killable: true}, false
	}
	m, ok := r.d.Catalog.Monster(class)
	if !ok {
		return MonsterMeta{Killable: true}, false
	}
	return m, true
}

// friendCat feeds combat.IsFriendly from the bot catalog: only Monster is real, the fight-only lookups are empty.
type friendCat struct{ c Catalog }

// FriendCatalog adapts the bot catalog for combat.IsFriendly (nil gives class table and states only).
func FriendCatalog(c Catalog) combat.Catalog { return friendCat{c} }

func (c friendCat) Monster(class int) *data.Monster {
	if c.c == nil {
		return nil
	}
	m, ok := c.c.Monster(class)
	if !ok {
		return nil
	}
	return &data.Monster{Name: m.Name, AI: m.AI, NPC: m.NPC, Killable: m.Killable}
}
func (friendCat) MonsterName(m *data.Monster) string {
	if m == nil {
		return ""
	}
	return m.Name
}
func (friendCat) Skill(int) *data.Skill          { return nil }
func (friendCat) SkillByName(string) *data.Skill { return nil }
func (friendCat) Item(int) combat.ItemClass      { return combat.ItemClass{} }

// hostile reports whether a monster unit is something to fight: alive, killable, and not friendly by
// combat.IsFriendly (monstats npc/AI, class table, minion state, owner words).
func (r *Runner) hostile(s *d2.Snapshot, u *d2.Unit) bool {
	if !u.Alive() {
		return false
	}
	m, _ := r.monsterMeta(u.Class)
	return m.Killable && !combat.IsFriendly(s, *u, friendCat{r.d.Catalog})
}

// hostileNear counts hostile living monsters within radius subtiles of the player.
func (r *Runner) hostileNear(s *d2.Snapshot, radius int) int {
	if s == nil || s.Player == nil || s.Player.InTown() {
		return 0
	}
	ig, _ := r.d.Fighter.(ignorer)
	n := 0
	for i := range s.Monsters {
		u := &s.Monsters[i]
		if !r.hostile(s, u) || max(abs(u.X-s.Player.X), abs(u.Y-s.Player.Y)) > radius {
			continue
		}
		// A unit the fighter gave up on (unreachable, too dangerous) is not worth a Clear: it would kill nothing.
		if ig != nil && ig.IsIgnored(u.ID, math.Hypot(float64(u.X-s.Player.X), float64(u.Y-s.Player.Y))) {
			continue
		}
		n++
	}
	return n
}

// ignorer is the optional part of a Fighter that remembers targets it gave up on (combat.Fighter.IsIgnored).
type ignorer interface {
	IsIgnored(id uint32, dist float64) bool
}

func (r *Runner) observeKills(s *d2.Snapshot, now time.Time) {
	o := &r.ob
	p := s.Player
	dx := int64(p.Exp) - int64(o.exp)
	prev := o.exp
	o.exp = p.Exp
	if dx > 0 {
		o.pendingXP += dx
		o.bumpReal(now)
	} else if dx < 0 {
		// Experience only falls when the game was reloaded from the .d2s (written on Save and Exit, V arithmetic in
		// docs/soak-analysis.md section 6; U whether area changes save too). The XP counter keeps what was lost.
		o.xpLost += -dx
		r.e.Log.Warn("xp lost", "lost", -dx, "before", prev, "after", p.Exp, "totalLost", o.xpLost, "games", o.newGames)
	}
	if p.Gold != o.gold {
		o.gold = p.Gold
		o.bump(now) // shopping is not progress that may refill the restart budgets
	}
	if p.InTown() {
		o.mons = map[uint32]monTrack{}
		return
	}
	next := make(map[uint32]monTrack, len(s.Monsters))
	var kills []goals.Event
	life := 0
	for i := range s.Monsters {
		u := &s.Monsters[i]
		m, _ := r.monsterMeta(u.Class)
		if !m.Killable || combat.IsFriendly(s, *u, friendCat{r.d.Catalog}) {
			continue
		}
		tr := o.mons[u.ID]
		if u.Alive() {
			tr.aliveSeen = true
			if u.Monster != nil {
				life += u.Monster.LifePct
			}
		} else if tr.aliveSeen && !tr.counted {
			tr.counted = true
			name := m.Name
			boss := m.Boss
			if u.Monster != nil {
				if u.Monster.Name != "" {
					name = u.Monster.Name
				}
				boss = boss || u.Monster.IsUnique()
			}
			if name == "" {
				name = fmt.Sprintf("monster %d", u.Class)
			}
			kills = append(kills, goals.Event{Type: goals.EvKill, Monster: name, Boss: boss})
		}
		next[u.ID] = tr
	}
	o.mons = next
	if life < o.monLife { // monsters losing life is progress while a fight is on
		o.bump(now)
	}
	o.monLife = life
	if len(kills) == 0 {
		return
	}
	o.bumpReal(now)
	o.kills += len(kills)
	r.mu.Lock()
	st := r.st
	r.mu.Unlock()
	for i := range kills {
		sp := o.kt.Kill(now)
		r.e.Log.Info("killtime", "monster", kills[i].Monster, "area", p.Area, "total", secs(sp.Total), "walk", secs(sp.Walk),
			"cast", secs(sp.Cast), "wait", secs(sp.Wait), "gap_since_last_kill", secs(sp.Gap), "state", st)
	}
	if !r.cfg.observes("kill") {
		return
	}
	each := o.pendingXP / int64(len(kills))
	for i := range kills {
		kills[i].XP = each
		r.emit(kills[i])
	}
	o.pendingXP = 0
}

func (r *Runner) itemTrackOf(u *d2.Unit) itemTrack {
	t := itemTrack{class: u.Class, item: u.Item}
	if r.d.Catalog != nil {
		t.meta, t.known = r.d.Catalog.Item(u.Class)
	}
	return t
}

func (r *Runner) scanItems(s *d2.Snapshot) (inv, belt map[uint32]itemTrack) {
	inv, belt = map[uint32]itemTrack{}, map[uint32]itemTrack{}
	if s.Inventory == nil {
		return
	}
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil {
			continue
		}
		switch u.Item.Where {
		case d2.ContInventory:
			inv[u.ID] = r.itemTrackOf(u)
		case d2.ContBelt:
			belt[u.ID] = r.itemTrackOf(u)
		}
	}
	return
}

func (r *Runner) baselineItems(s *d2.Snapshot) {
	r.ob.inv, r.ob.belt = r.scanItems(s)
	r.ob.invInit = true
}

// observeItems derives Drop (new items in the inventory or belt while in the field) and Potion (a potion left the
// belt or inventory while in the field). In town the baseline just follows, so purchases and stash moves count as nothing.
func (r *Runner) observeItems(s *d2.Snapshot, now time.Time) {
	o := &r.ob
	inv, belt := r.scanItems(s)
	defer func() { o.inv, o.belt = inv, belt }()
	if !o.invInit || s.Player.InTown() {
		o.invInit = true
		return
	}
	if r.cfg.observes("drop") {
		for id, t := range inv {
			if _, had := o.inv[id]; !had {
				r.emitDrop(t)
				o.bump(now)
			}
		}
		for id, t := range belt {
			if _, had := o.belt[id]; !had {
				r.emitDrop(t)
				o.bump(now)
			}
		}
	}
	if r.cfg.observes("potion") {
		for _, prev := range []map[uint32]itemTrack{o.belt, o.inv} {
			for id, t := range prev {
				if !t.known || t.meta.Potion == PotionNone {
					continue
				}
				if _, ok := belt[id]; ok {
					continue
				}
				if _, ok := inv[id]; ok {
					continue
				}
				o.potions++
				r.emit(goals.Event{Type: goals.EvPotion, Potion: t.meta.Name})
				o.bump(now)
			}
		}
	}
}

var qualityNames = map[int]string{1: "normal", 2: "normal", 3: "normal", 4: "magic", 5: "set", 6: "rare", 7: "unique", 8: "crafted"}

func (r *Runner) emitDrop(t itemTrack) {
	it := &goals.Item{Name: t.meta.Name, Type: t.meta.GoalType}
	if !t.known {
		it.Name = fmt.Sprintf("item %d", t.class)
	}
	if t.item != nil {
		it.Quality = qualityNames[t.item.Quality]
	}
	r.emit(goals.Event{Type: goals.EvDrop, Item: it})
}

// updateCharacter refreshes the cached character card.
func (r *Runner) updateCharacter(s *d2.Snapshot) {
	p := s.Player
	c := goals.Character{
		Name: p.Name, Class: p.Class.String(), Level: p.Level, Life: p.Life, MaxLife: p.MaxLife, Mana: p.Mana, MaxMana: p.MaxMana,
		Area: r.areaName(p.Area), AreaID: p.Area, Position: goals.Point{X: p.X, Y: p.Y},
		Difficulty: strings.ToLower([...]string{"Normal", "Nightmare", "Hell"}[min(max(int(p.Difficulty), 0), 2)]),
	}
	if r.d.Catalog != nil {
		if base := r.d.Catalog.ExpToReach(p.Level); base > 0 || p.Level == 1 {
			c.XP = int64(p.Exp) - base
			if next := r.d.Catalog.ExpToReach(p.Level + 1); next > base {
				c.XPNext = next - base
			}
		}
	}
	r.mu.Lock()
	r.char = c
	r.act0 = p.Act
	r.mu.Unlock()
}

// qualityValue multiplies a base price into a rough market value. A crude estimate: UNVERIFIED, tune against real prices.
var qualityValue = map[int]int{1: 0, 2: 1, 3: 2, 4: 4, 5: 12, 6: 10, 7: 20, 8: 10}

// trackAlive remembers what a death record will need: life curve, nearest hostile, gold, equipped gear and its value.
func (r *Runner) trackAlive(s *d2.Snapshot, now time.Time) {
	o := &r.ob
	p := s.Player
	o.curve = append(o.curve, lifeStamp{at: now, pct: p.LifePct()})
	for len(o.curve) > 0 && now.Sub(o.curve[0].at) > 5*time.Second {
		o.curve = o.curve[1:]
	}
	r.noteTrail(s, now)
	a := aliveInfo{gold: p.Gold, stashGold: p.StashGold, level: p.Level, difficulty: p.Difficulty, kit: r.kitOf(s), hostiles: r.hostileNear(s, r.e.Tune.Combat.FightRadius)}
	o.noteHP(p, a.hostiles)
	if mc := d2.FindMerc(s); mc.Present {
		alive := mc.Alive
		a.merc = &alive
	}
	if s.Inventory != nil {
		for _, u := range s.Inventory.Container(d2.ContEquipment) {
			a.equip++
			if r.d.Catalog != nil && u.Item != nil {
				if m, ok := r.d.Catalog.Item(u.Class); ok {
					a.equipValue += m.Cost * qualityValue[u.Item.Quality]
				}
			}
		}
	}
	best := 1 << 30
	for i := range s.Monsters {
		u := &s.Monsters[i]
		if !r.hostile(s, u) {
			continue
		}
		d := max(abs(u.X-p.X), abs(u.Y-p.Y))
		if d <= 25 {
			m, _ := r.monsterMeta(u.Class)
			a.near = append(a.near, NearHostile{Class: u.Class, Name: m.Name, Dist: d})
		}
		if d < best && d <= 25 {
			best = d
			a.killer = u
		}
	}
	sort.SliceStable(a.near, func(i, j int) bool { return a.near[i].Dist < a.near[j].Dist })
	if len(a.near) > 5 {
		a.near = a.near[:5]
	}
	if a.killer != nil {
		m, _ := r.monsterMeta(a.killer.Class)
		a.killerName, a.killerBoss = m.Name, m.Boss
		a.killerLevel = m.Level[min(max(int(p.Difficulty), 0), 2)]
		if mi := a.killer.Monster; mi != nil {
			if mi.Name != "" {
				a.killerName = mi.Name
			}
			a.killerBoss = a.killerBoss || mi.IsUnique()
			a.killerChampion = mi.IsChampion() && !mi.IsUnique() && !a.killerBoss
			a.killerAffix = combat.MonsterAffixes(mi)
		}
		if a.killerName == "" {
			a.killerName = fmt.Sprintf("monster %d", a.killer.Class)
		}
		k := *a.killer
		a.killer = &k
	}
	o.alive = a
}

// recordDeath builds the death record from the last living snapshot, logs it, persists it and emits the Death event.
func (r *Runner) recordDeath(p *d2.Player, now time.Time) {
	o := &r.ob
	a := o.alive
	rec := DeathRecord{
		Time: now, Area: o.lastGood.area, Pos: o.lastGood.pos, Level: p.Level, Difficulty: difficultyName(p.Difficulty),
		Killer: a.killerName, KillerBoss: a.killerBoss, KillerChampion: a.killerChampion, CorpseValue: a.equipValue,
	}
	if rec.Area == 0 {
		rec.Area = p.Area
	}
	rec.AreaName = r.areaName(rec.Area)
	if a.killer != nil {
		rec.KillerClass = a.killer.Class
		if a.killer.Monster != nil {
			rec.Enchants = append([]uint8(nil), a.killer.Monster.Enchants[:]...)
		}
	}
	for _, s := range o.curve {
		rec.LifeCurve = append(rec.LifeCurve, LifeSample{MsBefore: int(now.Sub(s.at) / time.Millisecond), Pct: s.pct})
	}
	rec.Autopsy = r.buildAutopsy(p, a, now)
	r.dlog.add(rec)
	r.appendAutopsy(rec)
	r.addFollow(now)
	o.deathPending = true
	o.deaths++
	o.lastDeath = a
	o.lastDeathAt = now
	if err := r.dlog.save(r.cfg.DeathLogPath); err != nil {
		r.e.Log.Warn("death log not saved", "err", err)
	}
	msg := "Died in " + rec.AreaName
	if rec.Killer != "" {
		msg = "Killed by " + rec.Killer + " in " + rec.AreaName
	}
	r.e.Log.Warn("death autopsy: "+rec.Autopsy.Summary, "area", rec.AreaName, "killer", rec.Killer, "killerType", rec.Autopsy.KillerType, "killerLevel", rec.Autopsy.KillerLevel, "hostiles", rec.Autopsy.HostilesNear, "lastSafety", rec.Autopsy.LastSafety, "bug", rec.Autopsy.Bug, "beltHP", rec.Autopsy.BeltHP)
	r.e.Log.Warn("character died", "area", rec.Area, "killer", rec.Killer, "x", rec.Pos.X, "y", rec.Pos.Y, "corpseValue", rec.CorpseValue)
	di := combat.DeathInfo{Area: rec.Area, Difficulty: p.Difficulty, CharLevel: p.Level, MonsterName: a.killerName, MonsterClass: rec.KillerClass,
		Boss: a.killerBoss, Champion: a.killerChampion, MonsterLevel: a.killerLevel, Affixes: a.killerAffix}
	if a.killer != nil {
		di.Distance = float64(max(abs(a.killer.X-p.X), abs(a.killer.Y-p.Y)))
	}
	r.e.Log.Warn(di.Line() + " build=" + r.buildName(p)) // key=value, parsed by cmd/d2soak ParseDeathLines (docs/difficulty.md P2, W10)
	r.emit(goals.Event{Type: goals.EvDeath, Message: msg, Monster: rec.Killer, Area: rec.AreaName, AreaID: rec.Area})
}

// DeathReport is the display view of the death log: records with cause and cost, per-area and per-monster counters,
// the heat map, the danger weights and the areas currently avoided. Safe to call from any goroutine.
func (r *Runner) DeathReport() DeathReport {
	r.mu.Lock()
	paused := r.selfStop
	r.mu.Unlock()
	r.obMu.Lock()
	defer r.obMu.Unlock()
	return r.dlog.report(r.e.Clock.Now(), r.avoid, paused)
}

// LastSnapshot returns the newest in-game snapshot (nil before the first). Treat it as read-only.
func (r *Runner) LastSnapshot() *d2.Snapshot { return r.latest() }

// Catalog returns the game-table view the runner was built with (may be nil).
func (r *Runner) Catalog() Catalog { return r.d.Catalog }

// questPollEvery rate-limits the quest reader (it reads the save or memory).
const questPollEvery = 15 * time.Second

// pollQuests emits EvQuest for each quest of the character's difficulty whose stage rose since the last poll. After a
// restart everything non-empty is emitted once; the manager drops stages it already saw, so this stays idempotent.
// Called with obMu held.
func (r *Runner) pollQuests(p *d2.Player, now time.Time) {
	o := &r.ob
	if r.d.Task == nil || r.d.Task.Quests == nil || now.Sub(o.questAt) < questPollEvery {
		return
	}
	o.questAt = now
	diff := int(p.Difficulty)
	v, ok := r.d.Task.Quests.Read(diff)
	if !ok {
		return
	}
	if o.questSeen == nil {
		o.questSeen = map[string]task.Stage{}
	}
	ids := make([]string, 0, len(v.Quests))
	for id := range v.Quests {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		st := v.Quests[id].Stage
		key := fmt.Sprintf("%d/%s", diff, id)
		if st <= o.questSeen[key] {
			continue
		}
		o.questSeen[key] = st
		done := st >= task.StageDone
		ev := goals.Event{Type: goals.EvQuest, Quest: id, Stage: int(st), Done: done, Diff: diff}
		if done {
			ev.Unlock = questUnlocks[id]
		}
		r.emit(ev)
	}
}

// minuteLine writes the once-a-minute structured summary that cmd/d2soak ParseMinuteLines reads (msg "minute"; keep the
// key names in step with it). Caller holds obMu. restarts counts game entries after the first, rung is the failsafe
// ladder step reached since the last real progress.
func (r *Runner) minuteLine(s *d2.Snapshot, p *d2.Player, now time.Time) {
	o := &r.ob
	r.mu.Lock()
	goal, step, rstate := "", r.activity, string(r.st)
	if r.goal != nil {
		goal = string(r.goal.Type)
	}
	r.mu.Unlock()
	r.hmu.Lock()
	rung := r.ladder
	r.hmu.Unlock()
	_, invPct, invKnown := r.beltAndInv(s)
	if !invKnown {
		invPct = -1
	}
	kit := r.kitOf(s) // belt counts read from the snapshot's belt slots (zero when no inventory was read)
	group, _ := stepgroup.Of(step)
	r.e.Log.Info("minute", "area", p.Area, "clvl", p.Level, "xp", p.Exp, "kills", r.lifetimeKills(p.Name), "killsGame", o.kills, "deaths", o.deaths,
		"potions", kit.BeltHP, "mana", kit.BeltMP, "potionsUsed", int(r.drank.Load()),
		"gold", p.Gold, "restarts", restartCount(o.newGames, o.plannedGames, o.recycles), "xpRisk", o.xpAtRiskLocked(r.d.SavedExp), "xpLost", o.xpLost, "rung", rung, "goal", goal, "step", step,
		"state", group, "rstate", rstate, "invPct", invPct, "dmgTaken", o.hp.dmg, "hpMinPct", o.takeHPMin())
	r.noteGoldNextMinute(p.Gold, now)
}

// xpAtRiskLocked is the experience a hard stop would lose now: memory exp minus the last saved exp (savedFn, if the
// caller can read the .d2s, else the exp at game entry). Never negative.
func (o *observer) xpAtRiskLocked(savedFn func() (uint32, bool)) int64 {
	saved := o.savedExp
	if savedFn != nil {
		if v, ok := savedFn(); ok {
			saved = v
		}
	}
	return max(int64(o.exp)-int64(saved), 0)
}

// logXPAtRisk logs what an unplanned stop is about to cost.
func (r *Runner) logXPAtRisk(why string) {
	r.obMu.Lock()
	risk, exp := r.ob.xpAtRiskLocked(r.d.SavedExp), r.ob.exp
	r.obMu.Unlock()
	r.e.Log.Warn("xp at risk", "why", why, "xp", exp, "atRisk", risk)
}
