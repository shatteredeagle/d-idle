package bot

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/task"
)

// The campaign goal: internal/task decides WHICH step, this file lets it read the world (Env) and use the bot's
// capabilities (Actions). One taskStep runs one executor tick inside the usual r.do guard, so the watchdog, the death
// policy and the failsafe ladder apply unchanged (docs/task.md section 3).

// Dialog talks to a town NPC: open the dialog, pick the named option, close it. pick "" only opens and closes.
type Dialog interface {
	Talk(ctx context.Context, npc, pick string) error
}

// Objects clicks a named world object (quest tree, cairn stones, cage).
type Objects interface {
	Interact(ctx context.Context, object string) error
}

// QuestView is what the quest reader knows about one difficulty: task quest ids with their stage and evidence, and the
// area ids whose waypoint is active. The reader is fed from the save (internal/saves) and later from memory.
type QuestView struct {
	Quests map[string]task.QuestState
	WPs    map[int]bool
}

// QuestReader returns the view for a difficulty (0 normal, 1 nightmare, 2 hell); ok is false when nothing is known.
type QuestReader interface {
	Read(difficulty int) (QuestView, bool)
}

// TaskDeps are the optional capabilities of campaign goals. A nil Deps.Task refuses campaign goals at Start.
type TaskDeps struct {
	Dialog Dialog
	// Objects may be nil: steps that need it (Search for Cain) then fail with a visible needs_attention.
	Objects Objects
	// Waypoint activates the waypoint of the area the character stands in (nav.ActivateWaypoint); nil skips waypoints.
	Waypoint func(ctx context.Context) error
	// Quests may be nil: the task layer then decides from proxies only and reports Verified false.
	Quests QuestReader
	// Boss runs the scripted fight for the named boss (internal/boss through the d2bot adapter): travel to the arena if
	// needed, find, fight in phases, loot. Errors are already mapped (ErrPortalOpened, ErrLeftGame, task kinds). Nil: the
	// plain Explore + Fighter.Kill + PickUp fallback.
	Boss func(ctx context.Context, name string) error
	// Cube runs one Horadric Cube transmute over the named inputs (town.Transmute); nil: the step fails with a prereq.
	Cube func(ctx context.Context, inputs []string) error
	// HireMerc hires the act's mercenary with the aura (town.Hire, the row picked by aura); nil: a prereq fault.
	HireMerc func(ctx context.Context, act int, aura string) error
	// ClaimMerc takes the free Rogue from Kashya after Blood Raven (task.MercClaimer). nil = no backend: the claim step logs
	// "merc: claim step skipped (no backend)" and finishes; wire it when the merc package lands (agent/mercpolicy).
	ClaimMerc func(ctx context.Context) error
	// Farm runs the farm loop of a runs goal (boss.FarmLoop over the boss runner); nil refuses runs goals at Start.
	Farm func(ctx context.Context, call FarmCall) (boss.FarmReport, error)
	// QuestsLive: the reader sees in-session changes (a future memory reader). False for the save reader, whose file only
	// changes on Save and Exit.
	QuestsLive bool
}

// ---- state ---------------------------------------------------------------------------------------------

// taskState is the campaign bookkeeping of the Run goroutine (loop.task) plus the report the manager reads.
type taskState struct {
	ex       *task.Executor
	gen      uint64
	err      error // sentinel error an action hit (portal opened, left the game ...) for r.do to react to
	wp       map[int]bool
	killed   map[string]bool
	scene    int            // number of new games seen when killed/wp were last reset
	pseudo   map[string]int // pseudo items the actions set for this game (staff placed, book read, orb smashed)
	done     bool
	lastRep  time.Time
	blocks   int             // consecutive planning blocks (no eligible area, quest not ready), see planningBlock
	refused  map[string]bool // refusals already logged (noteStepRefused)
	andyLine string          // last Andariel gate line logged (noteAndariel)
}

type taskReport struct {
	prog goals.Progress
	have bool
	gen  uint64
}

// Progress implements goals.ProgressReporter: the last campaign report of the running goal.
func (r *Runner) Progress() (goals.Progress, bool) {
	r.mu.Lock()
	gen := r.gen
	r.mu.Unlock()
	r.obMu.Lock()
	defer r.obMu.Unlock()
	rep := r.ob.taskRep
	return rep.prog, rep.have && rep.gen == gen
}

func (r *Runner) setTaskReport(gen uint64, p goals.Progress) {
	r.obMu.Lock()
	r.ob.taskRep = taskReport{prog: p, have: true, gen: gen}
	r.obMu.Unlock()
}

// noteKill remembers a kill by monster name for the current game (proxy evidence for boss quests).
func (r *Runner) noteKill(name string) {
	if name == "" {
		return
	}
	r.killMu.Lock()
	if r.killedNames == nil {
		r.killedNames = map[string]bool{}
	}
	r.killedNames[normName(name)] = true
	r.killMu.Unlock()
}

func (r *Runner) killedBoss(name string) bool {
	r.killMu.Lock()
	defer r.killMu.Unlock()
	n := normName(strings.TrimPrefix(name, "The "))
	for k := range r.killedNames {
		if k == n || k == "the"+n {
			return true
		}
	}
	return false
}

// ---- the step ------------------------------------------------------------------------------------------

func (r *Runner) campaign(c control) *task.Campaign { return c.spec.camp }

// phaseState picks the runner state shown while a step of the given phase runs.
func phaseState(phase string) (State, string) {
	switch phase {
	case "travel":
		return StateTraveling, "Quest: travelling"
	case "fight", "level":
		return StateClearing, "Quest: fighting"
	case "turn-in", "chore":
		return StateResupplying, "Quest: in town"
	}
	return StateTraveling, "Quest step"
}

// taskStep runs one executor tick of the campaign goal.
func (r *Runner) taskStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	l := &r.loop
	camp := c.spec.camp
	if l.task.ex == nil || l.task.gen != c.gen {
		ex := task.NewExecutor(taskEnv{r}, taskActs{r, c})
		ex.Now = r.e.Clock.Now // strike expiry on the env clock: virtual under d2d lockstep (docs/d2d-determinism.md), wall on Game.exe as before
		ex.Hooks.OnStart = func(s task.Step) { r.noTP.Store(s.Constraints.NoTownPortal) }
		ex.Hooks.OnEnd = func(s task.Step, res task.Result) {
			r.noTP.Store(false)
			if res.Outcome == task.OutInterrupted {
				return // the guard's safety and trip interrupts are routine
			}
			r.e.Log.Info("task step", "step", s.ID, "outcome", res.Outcome.String(), "kind", res.Kind.String(), "err", res.Err, "attention", res.Attention)
		}
		l.task = taskState{ex: ex, gen: c.gen}
		r.obMu.Lock()
		r.ob.taskRep = taskReport{}
		r.obMu.Unlock()
	}
	ex := l.task.ex
	env := taskEnv{r}
	w, err := env.World(ctx)
	if err != nil {
		return false
	}
	plan := camp.Plan(w, ex)
	r.noteAndariel(w)
	step, _, hasStep := ex.Peek(plan, w)
	st, act := StateTraveling, "Quest: planning"
	if hasStep {
		st, act = phaseState(step.Phase)
		act += " (" + step.Title + ")"
	}
	if plan.Done {
		return r.campaignFinished(ctx, snap, c, w)
	}
	var res task.Result
	l.task.err = nil
	ok := r.do(ctx, c, st, act, 40*time.Minute, "task step", func(ctx context.Context) error {
		res, _ = ex.Tick(ctx, camp)
		if l.task.err != nil {
			return l.task.err
		}
		if res.Outcome == task.OutInterrupted {
			// The guard cancelled the step. Returning nil here made do() read it as a success and drop the cancel cause,
			// so no potion, portal or chicken was ever carried out during a quest walk (G1 attempts 11 and 12, V).
			var si safetyInterrupt
			if cause := context.Cause(ctx); errors.As(cause, &si) {
				return cause
			}
		}
		// A failing step counts for the bot's own failure ladder too (close panels, town, a fresh game): the executor
		// only retries and blocks quests, it cannot get a character out of a place it cannot leave (LIVE: the Den).
		r.noteStepRefused(res)
		switch res.Outcome {
		case task.OutRetry, task.OutBlocked:
			if planningFault(res) {
				return r.planningBlock(ctx, res)
			}
			return res.Err
		case task.OutRan, task.OutDone, task.OutSkipped:
			l.task.blocks = 0
		}
		return nil
	})
	if w2, werr := env.World(ctx); werr == nil {
		r.setTaskReport(c.gen, camp.Report(w2, ex))
	}
	switch res.Outcome {
	case task.OutDone:
		return r.campaignFinished(ctx, snap, c, w)
	case task.OutBlocked:
		if res.Stuck {
			r.selfPause(res.Attention)
			return true
		}
		return false // a blocked quest is routed around next tick; do not spin on it
	case task.OutWait:
		return false
	}
	return ok
}

// noteAndariel logs the Andariel gate verdict (level and kit) when it changes, from clvl 8 on, until she is dead.
func (r *Runner) noteAndariel(w task.World) {
	if w.Act > 1 || w.Level < 8 || (w.Difficulty != "" && w.Difficulty != "normal") || task.QuestDone(w, task.QAndariel) || w.Killed[task.BossAndy] {
		return
	}
	line := task.AndarielLine(w)
	if line == r.loop.task.andyLine {
		return
	}
	r.loop.task.andyLine = line
	r.e.Log.Info(line)
}

// noteStepRefused logs, once per step and reason, why the executor did not run a step it picked (a Ready fault: gear gate,
// banned area, missing potions). Without it a gated quest step (Talk to Charsi, Go to the Barracks) only showed as a
// state flicker at every new game (G1 attempt 6).
func (r *Runner) noteStepRefused(res task.Result) {
	if res.Err == nil || res.StepID == "" || (res.Outcome != task.OutRetry && res.Outcome != task.OutBlocked && res.Outcome != task.OutSkipped) {
		return
	}
	key := res.StepID + ": " + res.Err.Error()
	l := &r.loop.task
	if l.refused == nil {
		l.refused = map[string]bool{}
	}
	if l.refused[key] {
		return
	}
	l.refused[key] = true
	r.e.Log.Info("quest step skipped", "step", res.StepID, "outcome", res.Outcome.String(), "kind", res.Kind.String(), "why", res.Err)
}

// campaignFinished: nothing left to do. The goal manager sees Current >= Target and ends the goal; until then the
// character waits at home.
func (r *Runner) campaignFinished(ctx context.Context, snap *d2.Snapshot, c control, w task.World) bool {
	if !r.loop.task.done {
		r.loop.task.done = true
		r.e.Log.Info("campaign finished", "level", w.Level)
	}
	return r.rest(ctx, snap, c)
}

// ---- Env -----------------------------------------------------------------------------------------------

type taskEnv struct{ r *Runner }

// World assembles one tick's read-only view (docs/task.md section 3.3).
func (e taskEnv) World(ctx context.Context) (task.World, error) {
	r := e.r
	if err := ctx.Err(); err != nil {
		return task.World{}, err
	}
	s := r.latest()
	if s == nil {
		if s = r.poll(); s == nil {
			return task.World{}, errors.New("task: not in a game")
		}
	}
	p := s.Player
	l := &r.loop
	w := task.World{
		Act: p.Act + 1, Difficulty: difficultyName(p.Difficulty), Area: task.Area(p.Area), InTown: p.InTown(),
		Dead: isDead(p), Level: p.Level, Gold: p.Gold,
		WPs: map[task.Area]bool{}, Quests: map[string]task.QuestState{}, Killed: map[string]bool{}, Items: map[string]int{},
		Cleared: map[task.Area]bool{}, Swept: map[task.Area]bool{},
	}
	w.Potions, _, _ = r.beltAndInv(s)
	in := r.resInput(s)
	w.Potions += in.Belt.PackHP + in.Belt.PackRV // World.Potions says belt and pack
	w.HasTP, w.ManaPotions = in.HasTP, in.Belt.MP
	w.AndarielMinLevel, w.Antidotes = r.e.Tune.Boss.AndarielMinLevel, r.antidotes(s)
	if task.MercAlive != nil {
		w.MercAlive, w.MercKnown = task.MercAlive()
	}
	w.Caster = p.Class == d2.Sorceress || p.Class == d2.Necromancer || p.Class == d2.Druid
	bt := r.e.Tune.Boss
	w.BRHigh = p.Class == d2.Sorceress || p.Class == d2.Druid || p.Class == d2.Amazon
	w.BR = task.BRKnobs{CasterLevel: bt.BRCasterLevel, MeleeLevel: bt.BRMeleeLevel, Potions: bt.BRPotions, ManaPotions: bt.BRManaPotions, TPCharges: bt.BRTPCharges, Gold: bt.BRGold}
	w.BRWaitLevel = l.br.waitLvl
	if g := r.gearVerdict(s); g.Gated {
		w.Gated, w.GatedWhy, w.SafeArea = true, g.Reason, task.Area(r.safeFarmArea(s))
	}
	for a := range r.avoidedNow() {
		if w.Banned == nil {
			w.Banned = map[task.Area]bool{}
		}
		w.Banned[task.Area(a)] = true
	}
	w.HasMerc = d2.FindMerc(s).Alive
	r.fillRoute(&w, p.Class)
	w.DenMinLevel = r.denFloor(s)
	w.NoTownPortal = r.noTP.Load()
	if c, ok := r.d.Mover.(interface{ CanTeleport() bool }); ok {
		w.CanTeleport = c.CanTeleport()
	}
	if td := r.d.Task; td != nil && td.Quests != nil {
		if v, ok := td.Quests.Read(int(p.Difficulty)); ok {
			for q, st := range v.Quests {
				w.Quests[q] = st
			}
			for a, on := range v.WPs {
				w.WPs[task.Area(a)] = on
			}
		}
	}
	w.SaveOnly = r.d.Task != nil && r.d.Task.Quests != nil && !r.d.Task.QuestsLive
	if len(w.Quests) == 0 {
		w.Quests = nil // no reader: proxies only (World.Quests nil means exactly that)
	}
	r.syncGames() // the cleared set read below belongs to this game (the task path never passes runGoal)
	r.syncTaskGame()
	for a, on := range l.task.wp {
		w.WPs[task.Area(a)] = w.WPs[task.Area(a)] || on
	}
	l.wp.init()
	for k := range l.wp.touched {
		if k.diff == int(p.Difficulty) {
			w.WPs[task.Area(k.area)] = true
		}
	}
	for _, boss := range campaignBosses {
		if l.task.killed[boss] || r.killedBoss(boss) {
			w.Killed[boss] = true
		}
	}
	denOpen := false // the quest word is the truth for the Den: an exhausted sweep is not "every monster dead"
	if met, known := r.questObjectiveMet(task.QDen); known && !met {
		denOpen = true
	}
	for a, on := range l.cleared {
		if on && !(denOpen && a == areaDenOfEvil) {
			w.Cleared[task.Area(a)] = true
		} else if on {
			w.Swept[task.Area(a)] = true
		}
	}
	r.scanQuestItems(s, w.Items)
	for k, v := range l.task.pseudo {
		w.Items[k] = v
	}
	if w.HasMerc {
		w.Items[task.ItemMercenary] = 1
	}
	return w, nil
}

// syncTaskGame forgets per-game campaign memory (kills, own waypoint touches) when a new game started.
func (r *Runner) syncTaskGame() {
	r.obMu.Lock()
	n := r.ob.newGames
	r.obMu.Unlock()
	t := &r.loop.task
	if t.wp == nil || t.killed == nil || t.scene != n {
		if t.scene != n {
			if t.ex != nil {
				t.ex.ForgetFails()
			}
			r.killMu.Lock()
			r.killedNames = map[string]bool{}
			r.killMu.Unlock()
		}
		t.wp, t.killed, t.scene = map[int]bool{}, map[string]bool{}, n
		t.pseudo = map[string]int{}
	}
}

// campaignBosses are the uniques whose kill the World reports (the names the steps pass to KillBoss).
var campaignBosses = []string{task.BossBR, task.BossSmith, task.BossCount, task.BossAndy,
	task.BossRadament, task.BossFangskin, task.BossSummoner, task.BossDuriel, task.BossCouncil, task.BossMephisto,
	task.BossIzual, task.BossHephasto, task.BossDiablo, task.BossShenk, task.BossNihlathak, task.BossAncients, task.BossBaal}

// questItemNames are the pack items the steps ask about, by display name (the Act 2 and 3 names are V items.txt Display
// for the Act 2 ones, U for the Act 3 ones, docs/act3.md).
var questItemNames = []string{task.ItemMalus, task.ItemBark, task.ItemCainSc,
	task.ItemCube, task.ItemShaft, task.ItemStaffTop, task.ItemStaff, task.ItemBook, task.ItemScroll,
	task.ItemEye, task.ItemHeart, task.ItemBrain, task.ItemFlail, task.ItemWill, task.ItemJade, task.ItemGolden,
	task.ItemGidbinnRaw, task.ItemLamTome,
	// Acts 4 and 5 (names U until a live run read them, codes mss, hfh, ice, tr2 are V)
	task.ItemSoulstone, task.ItemHammer, task.ItemMalahPotion, task.ItemResScroll}

func (r *Runner) scanQuestItems(s *d2.Snapshot, out map[string]int) {
	if r.d.Catalog == nil || s.Inventory == nil {
		return
	}
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		// the game wears a quest weapon picked up from the ground (docs/act2.md P1), so worn counts as held
		if u.Item == nil || u.Item.Where != d2.ContInventory && u.Item.Where != d2.ContEquipment {
			continue
		}
		m, ok := r.d.Catalog.Item(u.Class)
		if !ok {
			continue
		}
		for _, n := range questItemNames {
			if m.Name == n {
				out[n]++
			}
		}
	}
}

// ---- Actions -------------------------------------------------------------------------------------------

type taskActs struct {
	r *Runner
	c control
}

// tag maps the bot's errors to the task layer's kinds. Sentinels that r.do must see (an opened portal, a left game)
// are remembered for r.do. An opened portal travels as KindFatal (an interrupt, no failure counted), a chicken exit as KindDeath.
func (a taskActs) tag(err error) error {
	if err == nil {
		return nil
	}
	r := a.r
	switch {
	case errors.Is(err, ErrLeftGame):
		// A chicken exit is a failed attempt: three of them in one step block the quest (like deaths), so a boss that is
		// too strong does not loop forever through fresh games.
		r.loop.task.err = err
		return task.Wrap(task.KindDeath, err)
	case errors.Is(err, ErrPortalOpened):
		r.loop.task.err = err
		return task.Wrap(task.KindFatal, err)
	case errors.Is(err, errAreaGated):
		return task.Wrap(task.KindDanger, err)
	}
	if s := r.latest(); s != nil && isDead(s.Player) {
		return task.Wrap(task.KindDeath, err)
	}
	if strings.Contains(err.Error(), "no route") {
		return task.Wrap(task.KindNotFound, err)
	}
	return err
}

func (a taskActs) Travel(ctx context.Context, ar task.Area) error {
	return a.tag(a.r.travelTo(ctx, int(ar)))
}

// ClearAround implements task.StallRecoverer: a bounded fight around the character frees a walk stuck in a pack.
func (a taskActs) ClearAround(ctx context.Context) error {
	_, err := a.r.d.Fighter.Clear(ctx, ClearOpts{Radius: 40})
	return a.tag(err)
}

// TeleportHopper is the optional part of a Mover that can hop into an area by teleport (needs the Teleport key).
type TeleportHopper interface {
	TeleportToArea(ctx context.Context, levelNo int) error
}

// TeleportToward implements task.StallRecoverer: without a hopping Mover it is task.ErrNoTeleport, so the caller
// falls back to Clear and retry.
func (a taskActs) TeleportToward(ctx context.Context, ar task.Area) error {
	h, ok := a.r.d.Mover.(TeleportHopper)
	if !ok {
		return task.ErrNoTeleport
	}
	return a.tag(h.TeleportToArea(ctx, int(ar)))
}

func (a taskActs) ToTown(ctx context.Context) error {
	if s := a.r.latest(); s != nil && s.Player.InTown() {
		return nil
	}
	if err := a.r.d.Town.ToTown(ctx); err != nil {
		if a.r.resInput(a.r.latest()).HasTP {
			return a.tag(err)
		}
		return task.Wrap(task.KindResource, err)
	}
	return nil
}

// Clear sweeps the area until exploration is exhausted (docs/task.md 3.4: ErrExplored is success).
func (a taskActs) Clear(ctx context.Context, ar task.Area) error {
	r := a.r
	area := int(ar)
	if cur := r.curArea(); cur != area {
		if err := r.travelTo(ctx, area); err != nil {
			return a.tag(err)
		}
	}
	r.obMu.Lock()
	r.ob.targetArea = area
	r.obMu.Unlock()
	defer func() {
		r.obMu.Lock()
		r.ob.targetArea = 0
		r.obMu.Unlock()
	}()
	hunts := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !r.loop.cleared[area] {
			if err := r.sweepPass(ctx, area); err != nil {
				return a.tag(err)
			}
			continue
		}
		// Exploration ran out. That is not "every monster is dead": the fighter sets targets aside (no shot, unreachable)
		// and the quest needs them all (LIVE: the Den was declared cleared with 11 monsters near). Walk to the nearest
		// leftover and fight again, a bounded number of times.
		s := r.latest()
		if s == nil {
			return nil
		}
		left := r.aliveHostiles(s)
		if len(left) == 0 {
			return nil
		}
		if hunts >= 25 {
			r.e.Log.Warn("campaign clear: leftovers given up", "area", area, "left", len(left))
			return nil
		}
		hunts++
		r.loop.cleared[area] = false
		r.loop.futile = false
		u := left[0]
		r.e.Log.Info("campaign clear: hunting leftovers", "area", area, "left", len(left), "x", u.X, "y", u.Y, "hunt", hunts)
		if err := r.d.Mover.GoTo(ctx, input.Point{X: u.X, Y: u.Y}); err != nil && ctx.Err() == nil {
			r.e.Log.Info("campaign clear: cannot reach a leftover", "err", err)
		}
		r.pause(ctx, time.Duration(r.e.Tune.Bot.TickMs)*time.Millisecond)
	}
}

// aliveHostiles lists the living hostile monsters in the snapshot, nearest first.
func (r *Runner) aliveHostiles(s *d2.Snapshot) []d2.Unit {
	var out []d2.Unit
	for i := range s.Monsters {
		if u := s.Monsters[i]; r.hostile(s, &u) {
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di := max(abs(out[i].X-s.Player.X), abs(out[i].Y-s.Player.Y))
		dj := max(abs(out[j].X-s.Player.X), abs(out[j].Y-s.Player.Y))
		return di < dj
	})
	return out
}

// bossRestartGap is the least time between two starts of the same boss run.
const bossRestartGap = 30 * time.Second

// KillBoss finds the named unit (exploring until it shows), kills it and picks up what it dropped.
func (a taskActs) KillBoss(ctx context.Context, name string) (ret error) {
	r := a.r
	if name == boss.NameBloodRaven {
		worldFn := taskEnv{r}.World
		if r.loop.brWorldFn != nil {
			worldFn = r.loop.brWorldFn
		}
		w, err := worldFn(ctx)
		if err != nil {
			return a.tag(err)
		}
		if err := r.brAttempt(w); err != nil {
			if s := r.latest(); s != nil && !s.Player.InTown() && r.d.Town != nil && ctx.Err() == nil && !strings.Contains(err.Error(), "attempts spent") {
				// the kit ran short on the way (or the level wait applies): restock from town, where the planner's gate sees it
				_ = r.d.Town.ToTown(ctx)
			}
			return a.tag(err)
		}
		defer func() { r.brFinish(ret) }()
	}
	if td := r.d.Task; td != nil && td.Boss != nil {
		// At most one start per bossRestartGap and name: an aborted run (no kill) must not be restarted by the next
		// tick, 181 such starts of about 1 s each were logged in 87 minutes.
		if last, ok := r.bossStart[name]; ok {
			if wait := bossRestartGap - r.e.Clock.Now().Sub(last); wait > 0 {
				r.pause(ctx, wait)
				if ctx.Err() != nil {
					return a.tag(ctx.Err())
				}
			}
		}
		if r.bossStart == nil {
			r.bossStart = map[string]time.Time{}
		}
		r.bossStart[name] = r.e.Clock.Now()
		r.setTargetArea(0) // the leveling target of an earlier slice must not look like a border crossing
		r.bossRun.Store(true)
		err := td.Boss(ctx, name)
		r.bossRun.Store(false)
		if err != nil {
			if ctx.Err() != nil {
				r.e.Log.Info("boss run cancelled from outside", "boss", name, "cause", context.Cause(ctx), "ran", r.e.Clock.Now().Sub(r.bossStart[name]).Round(time.Second).String())
			}
			return a.tag(err)
		}
		delete(r.bossStart, name)
		r.noteKill(name)
		r.syncTaskGame()
		r.loop.task.killed[name] = true
		return nil
	}
	want := normName(strings.TrimPrefix(name, "The "))
	find := func() (uint32, bool) {
		s := r.latest()
		if s == nil || r.d.Catalog == nil {
			return 0, false
		}
		for i := range s.Monsters {
			u := &s.Monsters[i]
			if !u.Alive() {
				continue
			}
			if m, ok := r.d.Catalog.Monster(u.Class); ok && normName(strings.TrimPrefix(m.Name, "The ")) == want {
				return u.ID, true
			}
		}
		return 0, false
	}
	id, ok := find()
	if !ok {
		if err := r.d.Mover.Explore(ctx, func() bool { _, seen := find(); return seen }); err != nil {
			return a.tag(err)
		}
		if id, ok = find(); !ok {
			if r.killedBoss(name) {
				return nil
			}
			return task.Errorf(task.KindNotFound, "%s not found after exploring the area", name)
		}
	}
	if err := r.d.Fighter.Kill(ctx, id); err != nil {
		return a.tag(err)
	}
	r.noteKill(name)
	r.syncTaskGame()
	r.loop.task.killed[name] = true
	if r.d.Loot != nil {
		if _, err := r.d.Loot.PickUp(ctx, PickOpts{Radius: 30}); err != nil {
			return a.tag(fmt.Errorf("pick up after %s: %w", name, err))
		}
	}
	return nil
}

func (a taskActs) Talk(ctx context.Context, npc, pick string) error {
	td := a.r.d.Task
	if td == nil || td.Dialog == nil {
		return task.Errorf(task.KindUI, "no dialog capability: cannot talk to %s", npc)
	}
	return a.tag(td.Dialog.Talk(ctx, npc, pick))
}

func (a taskActs) Interact(ctx context.Context, object string) error {
	td := a.r.d.Task
	if td == nil || td.Objects == nil {
		return task.Errorf(task.KindUI, "no object interaction capability: cannot use %s", object)
	}
	// Decide Anya's state from the pack before the click: the thaw consumes Malah's potion.
	holdsPotion := false
	if object == AnyaObject {
		held := map[string]int{}
		if s := a.r.latest(); s != nil {
			a.r.scanQuestItems(s, held)
		}
		holdsPotion = held[task.ItemMalahPotion] > 0
	}
	if err := a.tag(td.Objects.Interact(ctx, object)); err != nil {
		return err
	}
	if object == AnyaObject {
		// the same click touches her, and thaws her once Malah's potion is in the pack
		if holdsPotion {
			a.setPseudo(task.ItemAnyaThawed)
		} else {
			a.setPseudo(task.ItemAnyaTouched)
		}
		return nil
	}
	if object == TreeObject {
		// the click only drops the scroll (U); the task node has no pick-up of its own
		if err := a.PickUp(ctx, task.ItemBark); err != nil {
			return err
		}
	}
	a.setPseudo(interactPseudo[object])
	return nil
}

// TreeObject is the Tree of Inifuss in the Dark Wood.
const TreeObject = "Tree of Inifuss"

// AnyaObject is the frozen Anya of the Frozen River (object 558, V "fana").
const AnyaObject = "Frozen Anya"

// The two object names of the Hellforge: put the Soulstone on it, then smash it with the Hammer in hand.
const (
	ForgePlaceObject = "Hellforge: place the Soulstone"
	ForgeSmashObject = "Hellforge: smash"
)

// UseForge implements task.ForgeUser: the Soulstone goes on the forge, the Hammer is swapped in and the forge clicked
// (U: the real procedure, docs/act4.md A4-6; the quest word 26 is the proof, the pseudo item only stops a repeat).
func (a taskActs) UseForge(ctx context.Context, soulstone, hammer string) error {
	td := a.r.d.Task
	if td == nil || td.Objects == nil {
		return task.Errorf(task.KindUI, "no object interaction capability: cannot use the Hellforge")
	}
	if err := a.tag(td.Objects.Interact(ctx, ForgePlaceObject)); err != nil {
		return err
	}
	err := a.UseWeapon(ctx, hammer, func(ctx context.Context) error {
		return a.tag(td.Objects.Interact(ctx, ForgeSmashObject))
	})
	if err == nil {
		a.setPseudo(task.ItemForgeUsed)
	}
	return err
}

// UseItem implements task.ItemUser: right-click the named pack item through the object adapter ("item:<name>").
func (a taskActs) UseItem(ctx context.Context, item string) error {
	return a.Interact(ctx, "item:"+item)
}

// interactPseudo names the pseudo item an object click proves (the world cannot see these effects in the pack).
var interactPseudo = map[string]string{
	"tal-orifice":                task.ItemStaffPlaced,
	"cairn-stones":               task.ItemPortal, // the adapter walks through the portal it opened
	"item:" + task.ItemBook:      task.ItemBookRead,
	"item:" + task.ItemResScroll: task.ItemScrollRead,
}

func (a taskActs) setPseudo(name string) {
	if name == "" {
		return
	}
	a.r.syncTaskGame()
	a.r.loop.task.pseudo[name] = 1
}

// Transmute implements task.Transmuter over TaskDeps.Cube.
func (a taskActs) Transmute(ctx context.Context, inputs []string) error {
	td := a.r.d.Task
	if td == nil || td.Cube == nil {
		return task.Errorf(task.KindPrereq, "the bot has no Horadric Cube capability")
	}
	return a.tag(td.Cube(ctx, inputs))
}

// HireMerc implements task.MercHirer over TaskDeps.HireMerc.
func (a taskActs) HireMerc(ctx context.Context, act int, aura string) error {
	td := a.r.d.Task
	if td == nil || td.HireMerc == nil {
		return task.Errorf(task.KindPrereq, "the bot has no mercenary hire capability")
	}
	return a.tag(td.HireMerc(ctx, act, aura))
}

// ClaimFreeMerc implements task.MercClaimer over TaskDeps.ClaimMerc. Without a backend it is a logged no-op, never an error.
func (a taskActs) ClaimFreeMerc(ctx context.Context) error {
	td := a.r.d.Task
	if td == nil || td.ClaimMerc == nil {
		a.r.e.Log.Info("merc: claim step skipped (no backend)")
		return nil
	}
	if err := td.ClaimMerc(ctx); err != nil && !errors.Is(err, task.ErrClaimUnsupported) {
		return a.tag(err)
	}
	return nil
}

// SmashOrb implements task.OrbSmasher: swap the quest weapon in, click the Compelling Orb (the object adapter keeps
// clicking until its mode changes), swap back. The orb is only ever clicked through Interact.
func (a taskActs) SmashOrb(ctx context.Context, weapon string) error {
	err := a.UseWeapon(ctx, weapon, func(ctx context.Context) error {
		td := a.r.d.Task
		if td == nil || td.Objects == nil {
			return task.Errorf(task.KindUI, "no object interaction capability: cannot smash the orb")
		}
		return a.tag(td.Objects.Interact(ctx, OrbObject))
	})
	if err == nil {
		a.setPseudo(task.ItemOrbSmashed)
	}
	return err
}

// OrbObject is the object name the adapter resolves to the Compelling Orb (object 404, V objects.txt).
const OrbObject = "Compelling Orb"

func (a taskActs) PickUp(ctx context.Context, item string) error {
	if a.r.d.Loot == nil {
		return task.Errorf(task.KindNotFound, "no looter: cannot pick up %s", item)
	}
	_, err := a.r.d.Loot.PickUp(ctx, PickOpts{Radius: 30, Names: []string{item}})
	return a.tag(err)
}

func (a taskActs) ActivateWaypoint(ctx context.Context, ar task.Area) error {
	r := a.r
	td := r.d.Task
	if td == nil || td.Waypoint == nil {
		return task.Errorf(task.KindNotFound, "no waypoint capability")
	}
	if cur := r.curArea(); cur != int(ar) {
		if err := r.travelTo(ctx, int(ar)); err != nil {
			return a.tag(err)
		}
	}
	if err := td.Waypoint(ctx); err != nil {
		return a.tag(err)
	}
	r.syncTaskGame()
	r.loop.task.wp[int(ar)] = true
	return nil
}

func (a taskActs) Chores(ctx context.Context) error {
	_, err := a.r.d.Town.Chores(ctx, a.r.fullChores("task"))
	if err != nil {
		return task.Wrap(task.KindResource, err)
	}
	return nil
}

// noAreaMsg is the text of LevelTo's no-target error; recovery treats it as a planning block.
const noAreaMsg = "no eligible reachable leveling area"

// LevelTo is one bounded slice of the reach_level loop toward level n: pick the planner's area, walk there, sweep once.
func (a taskActs) LevelTo(ctx context.Context, n int) error {
	r := a.r
	s := r.latest()
	if s == nil {
		return task.Errorf(task.KindUI, "not in a game")
	}
	r.syncGames() // a fresh game forgets its predecessor's cleared areas, else freshGameToLevel restarts in a loop
	c := a.c
	c.spec = goalSpec{kind: goals.ReachLevel, level: n, label: fmt.Sprintf("reach level %d", n)}
	// the planner reads the goal's level through c.goal.Params.Level
	g := *c.goal
	g.Type, g.Params.Level = goals.ReachLevel, n
	c.goal = &g
	var target int
	ok := false
	for try := 0; try < 8; try++ {
		if target, ok = r.target(s, c); !ok {
			break
		}
		why := r.unreachable(ctx, s, target)
		if why == "" {
			break
		}
		// never stand still on a target the character cannot get to: avoid it for a while and pick the next best area
		r.skipArea(target, r.navSkip(), whyUnreachable)
		r.loop.stepArea = 0
		r.e.Log.Warn("leveling target not reachable, picking another", "area", target, "name", r.areaName(target), "why", why)
		ok = false
	}
	if !ok {
		// Every candidate was unreachable from where the character stands (G1 attempt 17: a dead-end tower cellar after a
		// failed corpse run): get out of the guarded, banned or recovery spot first, the next pick starts from town.
		if cur := s.Player.Area; !s.Player.InTown() && r.d.Town != nil && (r.recoveryActive() || r.areaGuardWhy(s, cur) != "" || r.banned(cur)) {
			r.e.Log.Info(fmt.Sprintf("recovery: leaving %s first", r.areaName(cur)), "area", cur)
			return a.tag(r.d.Town.ToTown(ctx))
		}
		if done, err := r.freshGameToLevel(ctx, s); done {
			return a.tag(err)
		}
		return task.Errorf(task.KindNotFound, noAreaMsg+" for level %d", s.Player.Level)
	}
	if r.d.Plan.IsTown(target) {
		return nil
	}
	if s.Player.Area != target {
		// target() published the area for the guard, which in the clearing state cancels any action running outside it
		// ("exploring crossed a border: walk back"). A walk to the target is exactly that, so the guard must not see the
		// target until the character is there (LIVE soak 2026-10-29: Stony Field to Dark Wood was cancelled after every
		// waypoint try, never reached nav, stalled 65 s each time).
		r.setTargetArea(0)
		err := r.travelTo(ctx, target)
		if err != nil && ctx.Err() == nil && strings.Contains(err.Error(), "no exit to that level") {
			// nav cannot enter this area (a dungeon entrance it has not seen, LIVE: Cold Plains to Cave Level 1): let the
			// planner pick another area for a while instead of failing the same walk over and over
			r.skipArea(target, r.navSkip(), whyUnreachable)
			r.e.Log.Warn("leveling area unreachable", "area", target, "name", r.areaName(target), "err", err)
		}
		return a.tag(err)
	}
	r.setTargetArea(target)
	return a.tag(r.sweepPass(ctx, target))
}

func (r *Runner) setTargetArea(a int) {
	r.obMu.Lock()
	r.ob.targetArea = a
	r.obMu.Unlock()
}

// freshGameToLevel is the leveling slice's answer to "every area worth farming is cleared in this game": walk home and
// start a new game (monsters respawn) rather than farm areas far below the character for no experience. done is false
// when that is not possible (no out-of-game flow, or nothing was cleared, so the lack of a target has another cause).
func (r *Runner) freshGameToLevel(ctx context.Context, s *d2.Snapshot) (done bool, err error) {
	if r.d.Cycler == nil || len(r.loop.cleared) == 0 {
		return false, nil
	}
	if !s.Player.InTown() {
		return true, r.d.Town.ToTown(ctx)
	}
	r.e.Log.Info("leveling: everything worth farming is cleared, starting a fresh game", "level", s.Player.Level, "cleared", len(r.loop.cleared))
	err = r.oogRun(func(ctx context.Context) error {
		return r.d.Cycler.NewGame(ctx, difficultyName(s.Player.Difficulty))
	})(ctx)
	if err == nil {
		// d2d restarts the process without a menu poll, so the observer's newGames never moves (live: restarts 0
		// after 38 fresh games) and syncGames cannot forget: the cleared set belongs to the old game.
		r.loop.cleared = map[int]bool{}
		// Same reason (soakfull hotfix): the per-game sweep, buff and step state is the old game's too.
		r.loop.sweepKills = nil
		r.loop.needBuff = true
		r.loop.stepArea = 0
	}
	return true, err
}

var _ task.Actions = taskActs{}
var _ task.StallRecoverer = taskActs{}
var _ task.WeaponUser = taskActs{}
var _ task.Transmuter = taskActs{}
var _ task.MercHirer = taskActs{}
var _ task.OrbSmasher = taskActs{}
var _ task.Env = taskEnv{}

// unreachable says why the character cannot get to the area now ("" = it can): there must be a route from where it
// stands, and a route longer than a few hops needs an activated waypoint on the way (the walk would be a long trek
// through areas it has never seen, which stalls on missing exits).
func (r *Runner) unreachable(ctx context.Context, s *d2.Snapshot, target int) string {
	cur := s.Player.Area
	if cur == target {
		return ""
	}
	hops, err := r.d.Mover.Route(cur, target)
	if err != nil || len(hops) == 0 {
		return fmt.Sprintf("no route from %s: %v", r.areaName(cur), err)
	}
	if err := r.routeAllowed(s, cur, target, hops); err != nil {
		return err.Error() // a pick that needs a guarded or banned area on the way is no pick
	}
	const walkable = 3
	if len(hops) <= walkable {
		return ""
	}
	w, werr := taskEnv{r}.World(ctx)
	if werr != nil {
		return ""
	}
	if w.WPs[task.Area(cur)] || s.Player.InTown() {
		for _, h := range hops {
			if w.WPs[task.Area(h)] {
				return ""
			}
		}
	}
	return fmt.Sprintf("%d hops and no activated waypoint on the way", len(hops))
}

// UseWeapon implements task.WeaponUser over the town's hand (town.Town.UseWeapon); a Townsman without it is a prereq fault.
func (a taskActs) UseWeapon(ctx context.Context, item string, fn func(ctx context.Context) error) error {
	t, ok := a.r.d.Town.(interface {
		UseWeapon(ctx context.Context, item string, fn func(ctx context.Context) error) error
	})
	if !ok {
		return task.Errorf(task.KindPrereq, "the town module cannot swap weapons")
	}
	return a.tag(t.UseWeapon(ctx, item, fn))
}

// planningFault is a step result that says "the plan has nothing runnable for this character now" (no eligible area, the
// quest is not ready, the gate refuses the area) as opposed to a game or input fault. It must never climb the failsafe
// ladder: its rungs 3 and 4 restart the game, which fixes nothing about a plan (LIVE G1 attempt 4: four restarts in five
// minutes from level.14 not_found).
func planningFault(res task.Result) bool {
	if res.Stuck {
		return false // the executor's own "nothing runnable at all" pause
	}
	switch res.Kind {
	case task.KindNotFound, task.KindDanger:
		return true
	}
	return res.Err != nil && strings.Contains(res.Err.Error(), "not ready")
}

// Planning blocks: wait in town between them, relax the area bans, and after planBlockMax in a row pause for attention
// (the pause resumes by itself).
const (
	planBlockMax  = 4
	planBlockWait = 2 * time.Minute
)

// planningBlock handles one planning fault (waiting only when no area is eligible): walk home, wait planBlockWait in town, then let the planner try again with the
// character's own safe farm area (the gear gate's SafeArea) allowed again. It returns nil so the failsafe ladder never
// sees the block; it never restarts the game.
func (r *Runner) planningBlock(ctx context.Context, res task.Result) error {
	l := &r.loop
	if w := r.banWait(); w != "" && res.Kind == task.KindNotFound {
		// every way out is banned: that is waiting, not a failure; it counts toward no pause and raises no error. It
		// never lasts longer than the idle cap: then the least bad area is farmed anyway (areaskip.go).
		if r.idleBreak(false) {
			return nil
		}
		if s := r.latest(); s != nil && !s.Player.InTown() && r.d.Town != nil {
			if err := r.d.Town.ToTown(ctx); err != nil {
				return err
			}
		}
		r.setState(StateInTown, w)
		r.pause(ctx, 5*time.Second)
		return nil
	}
	l.task.blocks++
	if res.Kind != task.KindNotFound {
		return nil // a quest that is not ready: the executor routes around it on the next tick, no waiting needed
	}
	r.e.Log.Warn("planning blocked", "step", res.StepID, "kind", res.Kind.String(), "err", res.Err, "blocks", l.task.blocks)
	s := r.latest()
	if s == nil {
		return nil
	}
	if !s.Player.InTown() && r.d.Town != nil {
		if err := r.d.Town.ToTown(ctx); err != nil {
			return err // a walk home that fails is a real fault
		}
	}
	if l.task.blocks >= planBlockMax {
		l.task.blocks = 0
		r.selfPause(fmt.Sprintf("needs attention: %v. Waiting in town; retrying by itself.", res.Err))
		return nil
	}
	if r.gearVerdict(s).Gated {
		if a := r.safeFarmArea(s); a != 0 {
			r.obMu.Lock()
			delete(r.avoid, a)
			r.obMu.Unlock()
			delete(l.cleared, a)
		}
	}
	r.setState(StateInTown, "Planning blocked: waiting in town")
	for waited := time.Duration(0); waited < planBlockWait && ctx.Err() == nil; waited += 5 * time.Second {
		r.pause(ctx, 5*time.Second)
		r.rearm()
	}
	return nil
}
